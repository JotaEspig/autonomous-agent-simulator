// Package script runs model logic: Lua scripts (decisions, rules, metrics, stop
// conditions) and native expressions (propensity and eligibility).
package script

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/rng"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/world"
)

type ResultKind uint8

const (
	ResultNil ResultKind = iota
	ResultBool
	ResultNumber
	ResultEffects
)

// Result is what a script function returned, already converted to Go.
type Result struct {
	Kind    ResultKind
	Bool    bool
	Number  float64
	Effects []*world.Effect
	Reason  string // optional second return value (e.g. why a precondition failed)
}

// Truthy follows Lua semantics: only nil and false are false.
func (r Result) Truthy() bool { return r.Kind != ResultNil && !(r.Kind == ResultBool && !r.Bool) }

// Func is a resolved script function. The zero value means "not defined".
// Handles are valid for every Runtime created from the same Compiled.
type Func int32

// Runtime is the contract between the engine and a scripting backend. The engine
// only talks to this interface, so the gopher-lua implementation can be replaced
// (e.g. by LuaJIT) without touching the engine. A Runtime is used by one goroutine.
type Runtime interface {
	// Lookup resolves a global function of a script; it returns 0 when undefined.
	Lookup(script, fn string) Func
	// Functions lists the global functions a script defines, sorted.
	Functions(script string) []string
	// CallAgent calls fn(self, world). Privileged calls (agent-scoped rules) may read
	// private data of any agent and hidden context fields. Calling 0 returns nil.
	CallAgent(fn Func, self world.AgentID, src *rng.Source, privileged bool) (Result, error)
	// CallWorld calls fn(world) with full privileges (world rules, metrics, stop).
	CallWorld(fn Func, src *rng.Source) (Result, error)
	Close()
}

// Compiled holds every script of a model compiled to bytecode once; the bytecode is
// shared by all runtimes.
type Compiled struct {
	protos map[string]*lua.FunctionProto
}

// Compile parses and compiles every script referenced by the model.
func Compile(m *model.Model) (*Compiled, error) {
	c := &Compiled{protos: map[string]*lua.FunctionProto{}}
	errs := &model.Errors{}
	for _, path := range m.Scripts {
		f, err := os.Open(filepath.Join(m.Dir, path))
		if err != nil {
			errs.Addf(path, "", "%v", err)
			continue
		}
		chunk, err := parse.Parse(f, path)
		f.Close()
		if err != nil {
			errs.Addf("", "", "%v", err)
			continue
		}
		proto, err := lua.Compile(chunk, path)
		if err != nil {
			errs.Addf("", "", "%v", err)
			continue
		}
		c.protos[path] = proto
	}
	if errs.Len() > 0 {
		return nil, errs
	}
	return c, nil
}

type luaScript struct {
	fns map[string]Func
}

type luaFunc struct {
	script, name string
	fn           *lua.LFunction
}

type agentRef struct {
	id  world.AgentID
	loc world.Loc
	own bool
}

// LuaRuntime is the gopher-lua backend.
type LuaRuntime struct {
	L *lua.LState
	w *world.World

	scripts map[string]*luaScript
	funcs   []luaFunc // indexed by Func; entry 0 is unused

	// per-call state
	src        *rng.Source
	privileged bool
	hasSelf    bool
	self       agentRef

	selfUD   *lua.LUserData
	worldUD  *lua.LUserData
	ctxUD    *lua.LUserData
	mapUDs   []*lua.LUserData
	agentMT  *lua.LTable
	effectMT *lua.LTable

	agentMethods map[string]*lua.LFunction
	worldMethods map[string]*lua.LFunction
}

var _ Runtime = (*LuaRuntime)(nil)

var removedGlobals = []string{"dofile", "loadfile", "load", "loadstring", "require", "module",
	"getfenv", "setfenv", "collectgarbage", "newproxy", "next", "_printregs"}

// NewRuntime creates a sandboxed Lua state with every script loaded.
func (c *Compiled) NewRuntime(w *world.World) (*LuaRuntime, error) {
	L := lua.NewState(lua.Options{SkipOpenLibs: true, MinimizeStackMemory: true})
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{{lua.BaseLibName, lua.OpenBase}, {lua.TabLibName, lua.OpenTable}, {lua.StringLibName, lua.OpenString}, {lua.MathLibName, lua.OpenMath}} {
		L.Push(L.NewFunction(lib.fn))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	for _, name := range removedGlobals {
		L.SetGlobal(name, lua.LNil)
	}
	r := &LuaRuntime{L: L, w: w, scripts: map[string]*luaScript{}, funcs: []luaFunc{{}}}
	r.install()

	paths := make([]string, 0, len(c.protos))
	for p := range c.protos {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := r.load(path, c.protos[path]); err != nil {
			L.Close()
			return nil, err
		}
	}
	return r, nil
}

// load runs a script's top level inside its own environment table, so scripts can
// all define e.g. propensity() without clashing, then freezes that environment.
// The environment starts as a copy of the sandboxed globals (rather than inheriting
// them through __index) so global lookups such as fx or math stay raw table hits.
func (r *LuaRuntime) load(path string, proto *lua.FunctionProto) error {
	L := r.L
	fn := L.NewFunctionFromProto(proto)
	env := L.NewTable()
	builtin := map[string]bool{}
	L.G.Global.ForEach(func(k, v lua.LValue) {
		if k.String() == "_G" {
			return // would let scripts write shared globals
		}
		env.RawSet(k, v)
		builtin[k.String()] = true
	})
	fn.Env = env
	L.Push(fn)
	if err := L.PCall(0, 0, nil); err != nil {
		return fmt.Errorf("%s: loading script: %s", path, luaMessage(err))
	}
	s := &luaScript{fns: map[string]Func{}}
	var names []string
	env.ForEach(func(k, v lua.LValue) {
		if _, ok := v.(*lua.LFunction); ok {
			if name, ok := k.(lua.LString); ok && !builtin[string(name)] {
				names = append(names, string(name))
			}
		}
	})
	sort.Strings(names)
	for _, name := range names {
		s.fns[name] = Func(len(r.funcs))
		r.funcs = append(r.funcs, luaFunc{script: path, name: name, fn: env.RawGetString(name).(*lua.LFunction)})
	}
	mt := L.NewTable()
	L.SetMetatable(env, mt)
	mt.RawSetString("__newindex", L.NewFunction(func(L *lua.LState) int {
		L.RaiseError("scripts must not keep state in globals between calls (tried to set %q); "+
			"use local variables, or store state in attributes, memory or the context", L.ToString(2))
		return 0
	}))
	r.scripts[path] = s
	return nil
}

func luaMessage(err error) string {
	if ae, ok := err.(*lua.ApiError); ok {
		msg := ae.Object.String()
		if ae.StackTrace != "" {
			msg += "\n" + ae.StackTrace
		}
		return msg
	}
	return err.Error()
}

func (r *LuaRuntime) Lookup(script, fn string) Func {
	if s := r.scripts[script]; s != nil {
		return s.fns[fn]
	}
	return 0
}

func (r *LuaRuntime) Functions(script string) []string {
	var names []string
	if s := r.scripts[script]; s != nil {
		for n := range s.fns {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

func (r *LuaRuntime) Close() { r.L.Close() }

func (r *LuaRuntime) CallAgent(fn Func, self world.AgentID, src *rng.Source, privileged bool) (Result, error) {
	if fn == 0 {
		return Result{}, nil
	}
	loc, ok := r.w.Locate(self)
	if !ok {
		return Result{}, fmt.Errorf("agent %d does not exist", self)
	}
	r.self = agentRef{id: self, loc: loc, own: true}
	r.selfUD.Value = &r.self
	r.hasSelf, r.privileged, r.src = true, privileged, src
	return r.call(fn, r.selfUD, r.worldUD)
}

func (r *LuaRuntime) CallWorld(fn Func, src *rng.Source) (Result, error) {
	if fn == 0 {
		return Result{}, nil
	}
	r.hasSelf, r.privileged, r.src = false, true, src
	return r.call(fn, r.worldUD)
}

func (r *LuaRuntime) call(fn Func, args ...lua.LValue) (Result, error) {
	L := r.L
	lf := r.funcs[fn]
	script, name := lf.script, lf.name
	L.Push(lf.fn)
	for _, a := range args {
		L.Push(a)
	}
	if err := L.PCall(len(args), 2, nil); err != nil {
		return Result{}, fmt.Errorf("%s: %s(): %s", script, name, luaMessage(err))
	}
	ret, reason := L.Get(-2), L.Get(-1)
	L.Pop(2)
	res := Result{}
	if s, ok := reason.(lua.LString); ok {
		res.Reason = string(s)
	}
	switch v := ret.(type) {
	case *lua.LNilType:
	case lua.LBool:
		res.Kind, res.Bool = ResultBool, bool(v)
	case lua.LNumber:
		res.Kind, res.Number = ResultNumber, float64(v)
	default:
		res.Kind = ResultEffects
		var err error
		if res.Effects, err = collectEffects(v, nil); err != nil {
			return Result{}, fmt.Errorf("%s: %s(): %v", script, name, err)
		}
	}
	return res, nil
}

// collectEffects flattens an effect or a (possibly nested) list of effects.
func collectEffects(v lua.LValue, out []*world.Effect) ([]*world.Effect, error) {
	switch x := v.(type) {
	case *lua.LUserData:
		if e, ok := x.Value.(*world.Effect); ok {
			return append(out, e), nil
		}
	case *lua.LTable:
		n := x.Len()
		for i := 1; i <= n; i++ {
			var err error
			if out, err = collectEffects(x.RawGetInt(i), out); err != nil {
				return nil, fmt.Errorf("element %d: %v", i, err)
			}
		}
		return out, nil
	}
	return nil, fmt.Errorf("returned %s; expected nil, a boolean, a number, an effect (fx.*) or a list of effects", v.Type())
}

func toLua(v model.Value) lua.LValue {
	switch v.Kind {
	case model.KindInt:
		return lua.LNumber(v.I)
	case model.KindFloat:
		return lua.LNumber(v.F)
	case model.KindBool:
		return lua.LBool(v.B)
	case model.KindString:
		return lua.LString(v.S)
	}
	return lua.LNil
}

func fromLua(v lua.LValue) (model.Value, error) {
	switch x := v.(type) {
	case lua.LNumber:
		return model.Float(float64(x)), nil
	case lua.LString:
		return model.String(string(x)), nil
	case lua.LBool:
		return model.Bool(bool(x)), nil
	}
	return model.Value{}, fmt.Errorf("expected a number, string or boolean, got %s", v.Type())
}

// keyString converts a map key (string or integral number) to the stored string key.
func keyString(L *lua.LState, v lua.LValue) string {
	switch x := v.(type) {
	case lua.LString:
		return string(x)
	case lua.LNumber:
		if float64(x) == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(float64(x), 'g', -1, 64)
	}
	L.RaiseError("map keys must be strings or numbers, got %s", v.Type())
	return ""
}

func idList(L *lua.LState, ids []world.AgentID) *lua.LTable {
	t := L.CreateTable(len(ids), 0)
	for i, id := range ids {
		t.RawSetInt(i+1, lua.LNumber(id))
	}
	return t
}

func stringList(L *lua.LState, ss []string) *lua.LTable {
	t := L.CreateTable(len(ss), 0)
	for i, s := range ss {
		t.RawSetInt(i+1, lua.LString(s))
	}
	return t
}

// sortedPairs replaces pairs(): gopher-lua stores string keys in a Go map whose
// iteration order is random, which would make runs irreproducible. Keys are visited
// numbers first (ascending), then strings (ascending), then anything else.
func sortedPairs(L *lua.LState) int {
	t := L.CheckTable(1)
	var keys []lua.LValue
	t.ForEach(func(k, _ lua.LValue) { keys = append(keys, k) })
	rank := func(v lua.LValue) int {
		switch v.(type) {
		case lua.LNumber:
			return 0
		case lua.LString:
			return 1
		}
		return 2
	}
	sort.SliceStable(keys, func(i, j int) bool {
		ri, rj := rank(keys[i]), rank(keys[j])
		if ri != rj {
			return ri < rj
		}
		switch a := keys[i].(type) {
		case lua.LNumber:
			return a < keys[j].(lua.LNumber)
		case lua.LString:
			return a < keys[j].(lua.LString)
		}
		return false
	})
	i := 0
	L.Push(L.NewFunction(func(L *lua.LState) int {
		if i >= len(keys) {
			L.Push(lua.LNil)
			return 1
		}
		k := keys[i]
		i++
		L.Push(k)
		L.Push(t.RawGet(k))
		return 2
	}))
	L.Push(t)
	L.Push(lua.LNil)
	return 3
}

func (r *LuaRuntime) install() {
	L := r.L
	L.SetGlobal("pairs", L.NewFunction(sortedPairs))
	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		parts := make([]string, L.GetTop())
		for i := range parts {
			parts[i] = L.ToStringMeta(L.Get(i + 1)).String()
		}
		fmt.Fprintf(os.Stderr, "[lua tick %d] %s\n", r.w.Tick, strings.Join(parts, "\t"))
		return 0
	}))
	if math, ok := L.GetGlobal("math").(*lua.LTable); ok {
		deny := L.NewFunction(func(L *lua.LState) int {
			L.RaiseError("math.random is disabled to keep runs reproducible; use world:rand(), world:randint() or world:normal()")
			return 0
		})
		math.RawSetString("random", deny)
		math.RawSetString("randomseed", deny)
	}

	r.agentMT = L.NewTypeMetatable("agent")
	r.agentMT.RawSetString("__index", L.NewFunction(r.agentIndex))
	r.agentMT.RawSetString("__newindex", L.NewFunction(readOnly("agents")))
	r.agentMT.RawSetString("__tostring", L.NewFunction(func(L *lua.LState) int {
		ref := L.CheckUserData(1).Value.(*agentRef)
		L.Push(lua.LString(fmt.Sprintf("agent<%d>", ref.id)))
		return 1
	}))
	r.agentMT.RawSetString("__eq", L.NewFunction(func(L *lua.LState) int {
		a, _ := L.CheckUserData(1).Value.(*agentRef)
		b, _ := L.CheckUserData(2).Value.(*agentRef)
		L.Push(lua.LBool(a != nil && b != nil && a.id == b.id))
		return 1
	}))
	r.agentMethods = r.makeAgentMethods()

	worldMT := L.NewTypeMetatable("world")
	worldMT.RawSetString("__index", L.NewFunction(r.worldIndex))
	worldMT.RawSetString("__newindex", L.NewFunction(readOnly("the world")))
	r.worldMethods = r.makeWorldMethods()

	ctxMT := L.NewTypeMetatable("context")
	ctxMT.RawSetString("__index", L.NewFunction(r.ctxIndex))
	ctxMT.RawSetString("__newindex", L.NewFunction(readOnly("the context")))

	mapMT := L.NewTypeMetatable("contextmap")
	mapMT.RawSetString("__index", L.NewFunction(r.mapIndex))
	mapMT.RawSetString("__newindex", L.NewFunction(readOnly("the context")))
	mapMT.RawSetString("__len", L.NewFunction(func(L *lua.LState) int {
		f := L.CheckUserData(1).Value.(*model.Field)
		L.Push(lua.LNumber(r.w.Context.MapLen(f)))
		return 1
	}))

	r.effectMT = L.NewTypeMetatable("effect")
	r.effectMT.RawSetString("__index", L.NewFunction(func(L *lua.LState) int {
		e := L.CheckUserData(1).Value.(*world.Effect)
		if L.CheckString(2) == "op" {
			L.Push(lua.LString(e.Op.String()))
			return 1
		}
		L.Push(lua.LNil)
		return 1
	}))
	r.effectMT.RawSetString("__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(L.CheckUserData(1).Value.(*world.Effect).String()))
		return 1
	}))

	r.selfUD = L.NewUserData()
	r.selfUD.Metatable = r.agentMT
	r.worldUD = L.NewUserData()
	r.worldUD.Metatable = worldMT
	r.ctxUD = L.NewUserData()
	r.ctxUD.Metatable = ctxMT
	for _, f := range r.w.Model.Context {
		ud := L.NewUserData()
		ud.Value = f
		ud.Metatable = mapMT
		r.mapUDs = append(r.mapUDs, ud)
	}
	L.SetGlobal("fx", r.makeFx())
}

func readOnly(what string) lua.LGFunction {
	return func(L *lua.LState) int {
		L.RaiseError("%s cannot be modified directly (tried to set %q); return effects built with fx.* instead", what, L.ToString(2))
		return 0
	}
}

func (r *LuaRuntime) newAgent(id world.AgentID) lua.LValue {
	loc, ok := r.w.Locate(id)
	if !ok {
		return lua.LNil
	}
	ud := r.L.NewUserData()
	ud.Value = &agentRef{id: id, loc: loc, own: r.hasSelf && id == r.self.id}
	ud.Metatable = r.agentMT
	return ud
}

func (r *LuaRuntime) agentIndex(L *lua.LState) int {
	ref := L.CheckUserData(1).Value.(*agentRef)
	key := L.CheckString(2)
	st := r.w.Stores[ref.loc.Type]
	switch key {
	case "id":
		L.Push(lua.LNumber(ref.id))
		return 1
	case "type":
		L.Push(lua.LString(st.Type.Name))
		return 1
	case "name":
		if n := r.w.Name(ref.id); n != "" {
			L.Push(lua.LString(n))
		} else {
			L.Push(lua.LNil)
		}
		return 1
	case "x", "y":
		if !st.Type.Spatial {
			L.RaiseError("agent type %q is not spatial", st.Type.Name)
		}
		if key == "x" {
			L.Push(lua.LNumber(st.X[ref.loc.Index]))
		} else {
			L.Push(lua.LNumber(st.Y[ref.loc.Index]))
		}
		return 1
	}
	if m, ok := r.agentMethods[key]; ok {
		L.Push(m)
		return 1
	}
	f := st.Type.AttrIndex[key]
	if f == nil {
		L.RaiseError("agent type %q has no attribute %q", st.Type.Name, key)
	}
	if !f.Public && !ref.own && !r.privileged {
		L.RaiseError("attribute %q of agent type %q is private; agents only see public attributes of others", key, st.Type.Name)
	}
	L.Push(toLua(st.Get(int(ref.loc.Index), f)))
	return 1
}

func (r *LuaRuntime) checkAgent(L *lua.LState) (*agentRef, *world.Store) {
	ud := L.CheckUserData(1)
	ref, ok := ud.Value.(*agentRef)
	if !ok {
		L.ArgError(1, "expected an agent (call methods with ':', e.g. self:last(\"moves\"))")
	}
	return ref, r.w.Stores[ref.loc.Type]
}

func (r *LuaRuntime) checkMemory(L *lua.LState, ref *agentRef, st *world.Store) *model.Memory {
	name := L.CheckString(2)
	mem := st.Type.MemoryIndex[name]
	if mem == nil {
		L.RaiseError("agent type %q has no memory %q", st.Type.Name, name)
	}
	if !ref.own && !r.privileged {
		L.RaiseError("memories are private; an agent can only read its own")
	}
	return mem
}

func (r *LuaRuntime) makeAgentMethods() map[string]*lua.LFunction {
	L := r.L
	recall := func(L *lua.LState, back int) int {
		ref, st := r.checkAgent(L)
		mem := r.checkMemory(L, ref, st)
		v, ok := st.Recall(int(ref.loc.Index), mem, back)
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(toLua(v))
		return 1
	}
	return map[string]*lua.LFunction{
		// last(memory): most recent entry, or nil.
		"last": L.NewFunction(func(L *lua.LState) int { return recall(L, 0) }),
		// recall(memory, n): the n-th most recent entry (1 = last), or nil.
		"recall": L.NewFunction(func(L *lua.LState) int { return recall(L, L.CheckInt(3)-1) }),
		// memlen(memory): number of entries held.
		"memlen": L.NewFunction(func(L *lua.LState) int {
			ref, st := r.checkAgent(L)
			mem := r.checkMemory(L, ref, st)
			L.Push(lua.LNumber(st.MemLen(int(ref.loc.Index), mem)))
			return 1
		}),
		// count(memory, value): how many entries equal value.
		"count": L.NewFunction(func(L *lua.LState) int {
			ref, st := r.checkAgent(L)
			mem := r.checkMemory(L, ref, st)
			want, err := fromLua(L.CheckAny(3))
			if err != nil {
				L.ArgError(3, err.Error())
			}
			n := 0
			for i := 0; i < st.MemLen(int(ref.loc.Index), mem); i++ {
				if v, _ := st.Recall(int(ref.loc.Index), mem, i); v.Equal(want) {
					n++
				}
			}
			L.Push(lua.LNumber(n))
			return 1
		}),
		// mean(memory): average of a numeric memory (0 when empty).
		"mean": L.NewFunction(func(L *lua.LState) int {
			ref, st := r.checkAgent(L)
			mem := r.checkMemory(L, ref, st)
			n := st.MemLen(int(ref.loc.Index), mem)
			s := 0.0
			for i := 0; i < n; i++ {
				v, _ := st.Recall(int(ref.loc.Index), mem, i)
				s += v.AsFloat()
			}
			if n > 0 {
				s /= float64(n)
			}
			L.Push(lua.LNumber(s))
			return 1
		}),
	}
}

func (r *LuaRuntime) worldIndex(L *lua.LState) int {
	key := L.CheckString(2)
	switch key {
	case "tick":
		L.Push(lua.LNumber(r.w.Tick))
		return 1
	case "ctx":
		L.Push(r.ctxUD)
		return 1
	}
	if m, ok := r.worldMethods[key]; ok {
		L.Push(m)
		return 1
	}
	L.RaiseError("world has no field or method %q", key)
	return 0
}

func (r *LuaRuntime) contextField(L *lua.LState, name string) *model.Field {
	f := r.w.Model.ContextIndex[name]
	if f == nil {
		L.RaiseError("unknown context field %q", name)
	}
	if !f.Public && !r.privileged {
		L.RaiseError("context field %q is hidden from agents", name)
	}
	return f
}

func (r *LuaRuntime) ctxIndex(L *lua.LState) int {
	f := r.contextField(L, L.CheckString(2))
	if f.IsMap {
		L.Push(r.mapUDs[f.Slot])
		return 1
	}
	L.Push(toLua(r.w.Context.Get(f)))
	return 1
}

func (r *LuaRuntime) mapIndex(L *lua.LState) int {
	f := L.CheckUserData(1).Value.(*model.Field)
	v, ok := r.w.Context.MapGet(f, keyString(L, L.Get(2)))
	if !ok {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(toLua(v))
	return 1
}

func (r *LuaRuntime) checkType(L *lua.LState, n int) *model.AgentType {
	name := L.CheckString(n)
	t := r.w.Model.TypeIndex[name]
	if t == nil {
		L.ArgError(n, fmt.Sprintf("unknown agent type %q", name))
	}
	return t
}

func (r *LuaRuntime) checkAttr(L *lua.LState, t *model.AgentType, n int) *model.Field {
	name := L.CheckString(n)
	f := t.AttrIndex[name]
	if f == nil {
		L.ArgError(n, fmt.Sprintf("agent type %q has no attribute %q", t.Name, name))
	}
	if !f.Public && !r.privileged {
		L.ArgError(n, fmt.Sprintf("attribute %q is private; agents can only aggregate public attributes", name))
	}
	return f
}

// optID reads an optional agent id argument, defaulting to self.
func (r *LuaRuntime) optID(L *lua.LState, n int) world.AgentID {
	switch v := L.Get(n).(type) {
	case lua.LNumber:
		return world.AgentID(v)
	case *lua.LUserData:
		if ref, ok := v.Value.(*agentRef); ok {
			return ref.id
		}
	case *lua.LNilType:
		if r.hasSelf {
			return r.self.id
		}
		L.ArgError(n, "an agent id is required outside agent scripts")
	}
	L.ArgError(n, "expected an agent id")
	return 0
}

func (r *LuaRuntime) checkRelation(L *lua.LState, n int) *world.Graph {
	name := L.CheckString(n)
	rel := r.w.Model.RelationIndex[name]
	if rel == nil {
		L.ArgError(n, fmt.Sprintf("unknown relation %q", name))
	}
	return r.w.Graphs[rel.Index]
}

func (r *LuaRuntime) checkGroupKind(L *lua.LState, n int) *model.GroupKind {
	name := L.CheckString(n)
	k := r.w.Model.GroupIndex[name]
	if k == nil {
		L.ArgError(n, fmt.Sprintf("unknown group kind %q", name))
	}
	return k
}

func (r *LuaRuntime) makeWorldMethods() map[string]*lua.LFunction {
	L := r.L
	fn := func(f lua.LGFunction) *lua.LFunction { return L.NewFunction(f) }
	agg := func(kind model.AggKind) *lua.LFunction {
		return fn(func(L *lua.LState) int {
			t := r.checkType(L, 2)
			var f *model.Field
			if kind != model.AggCount {
				f = r.checkAttr(L, t, 3)
			}
			L.Push(lua.LNumber(r.w.Aggregate(kind, t, f)))
			return 1
		})
	}
	groupAgg := func(kind model.AggKind) *lua.LFunction {
		return fn(func(L *lua.LState) int {
			k := r.checkGroupKind(L, 2)
			name := L.CheckString(3)
			attr := ""
			if kind != model.AggCount {
				attr = L.CheckString(4)
			}
			L.Push(lua.LNumber(r.w.GroupAggregate(kind, k, name, attr)))
			return 1
		})
	}
	return map[string]*lua.LFunction{
		"rand": fn(func(L *lua.LState) int {
			L.Push(lua.LNumber(r.src.Float()))
			return 1
		}),
		"randint": fn(func(L *lua.LState) int {
			lo, hi := L.CheckInt(2), L.CheckInt(3)
			if hi < lo {
				L.ArgError(3, "hi must be >= lo")
			}
			L.Push(lua.LNumber(lo + r.src.IntN(hi-lo+1)))
			return 1
		}),
		"normal": fn(func(L *lua.LState) int {
			L.Push(lua.LNumber(L.OptNumber(2, 0) + lua.LNumber(r.src.Norm())*L.OptNumber(3, 1)))
			return 1
		}),
		"chance": fn(func(L *lua.LState) int {
			L.Push(lua.LBool(r.src.Float() < float64(L.CheckNumber(2))))
			return 1
		}),
		"pick": fn(func(L *lua.LState) int {
			t := L.CheckTable(2)
			if n := t.Len(); n > 0 {
				L.Push(t.RawGetInt(1 + r.src.IntN(n)))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		}),
		"shuffle": fn(func(L *lua.LState) int {
			t := L.CheckTable(2)
			n := t.Len()
			r.src.Shuffle(n, func(i, j int) {
				a, b := t.RawGetInt(i+1), t.RawGetInt(j+1)
				t.RawSetInt(i+1, b)
				t.RawSetInt(j+1, a)
			})
			L.Push(t)
			return 1
		}),
		"agent": fn(func(L *lua.LState) int {
			L.Push(r.newAgent(r.optID(L, 2)))
			return 1
		}),
		"exists": fn(func(L *lua.LState) int {
			L.Push(lua.LBool(r.w.Exists(r.optID(L, 2))))
			return 1
		}),
		"find": fn(func(L *lua.LState) int {
			if id, ok := r.w.Find(L.CheckString(2)); ok {
				L.Push(lua.LNumber(id))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		}),
		"neighbors": fn(func(L *lua.LState) int {
			g := r.checkRelation(L, 2)
			L.Push(idList(L, g.Out(r.optID(L, 3))))
			return 1
		}),
		"incoming": fn(func(L *lua.LState) int {
			g := r.checkRelation(L, 2)
			L.Push(idList(L, g.In(r.optID(L, 3))))
			return 1
		}),
		"nearby": fn(func(L *lua.LState) int {
			radius := float64(L.CheckNumber(2))
			var t *model.AgentType
			if L.Get(3) != lua.LNil {
				t = r.checkType(L, 3)
			}
			id := r.optID(L, 4)
			x, y, ok := r.w.Position(id)
			if !ok {
				L.RaiseError("agent %d is not spatial", id)
			}
			L.Push(idList(L, r.w.Nearby(x, y, radius, t, id)))
			return 1
		}),
		"members": fn(func(L *lua.LState) int {
			k := r.checkGroupKind(L, 2)
			L.Push(idList(L, r.w.Groups[k.Index].Members(L.CheckString(3))))
			return 1
		}),
		"groups": fn(func(L *lua.LState) int {
			k := r.checkGroupKind(L, 2)
			L.Push(stringList(L, r.w.Groups[k.Index].Of(r.optID(L, 3))))
			return 1
		}),
		"count": agg(model.AggCount),
		"sum":   agg(model.AggSum),
		"mean":  agg(model.AggMean),
		"min":   agg(model.AggMin),
		"max":   agg(model.AggMax),
		"std":   agg(model.AggStd),
		"gini":  agg(model.AggGini),

		"group_count": groupAgg(model.AggCount),
		"group_sum":   groupAgg(model.AggSum),
		"group_mean":  groupAgg(model.AggMean),
		"group_min":   groupAgg(model.AggMin),
		"group_max":   groupAgg(model.AggMax),

		"sample": fn(func(L *lua.LState) int {
			t := r.checkType(L, 2)
			k := L.CheckInt(3)
			ids := r.w.Stores[t.Index].IDs
			n := len(ids)
			if k > n {
				k = n
			}
			// Floyd's algorithm: k distinct indices without materializing a permutation.
			chosen := make(map[int]bool, k)
			picked := make([]world.AgentID, 0, k)
			for j := n - k; j < n; j++ {
				i := r.src.IntN(j + 1)
				if chosen[i] {
					i = j
				}
				chosen[i] = true
				picked = append(picked, ids[i])
			}
			L.Push(idList(L, picked))
			return 1
		}),
		"ids": fn(func(L *lua.LState) int {
			if !r.privileged {
				L.RaiseError("world:ids() is only available to rules; agents observe through neighbors, nearby, members or sample")
			}
			t := r.checkType(L, 2)
			L.Push(idList(L, r.w.Stores[t.Index].IDs))
			return 1
		}),
		"keys": fn(func(L *lua.LState) int {
			f := r.contextField(L, L.CheckString(2))
			if !f.IsMap {
				L.ArgError(2, "not a map field")
			}
			L.Push(stringList(L, r.w.Context.MapKeys(f)))
			return 1
		}),
	}
}
