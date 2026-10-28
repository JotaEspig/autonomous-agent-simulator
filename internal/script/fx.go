package script

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/world"
)

// keyRule says whether a path must, may not, or may address a map entry.
type keyRule uint8

const (
	keyRequiredForMaps keyRule = iota
	keyForbidden
)

// parsePath resolves a value path:
//
//	self.<attr>          the calling agent's attribute
//	ctx.<field>          a scalar context field (or a whole map, for fx.clear)
//	ctx.<field>.<key>    an entry of a map context field
//	agent.<id>.<attr>    any agent's attribute (rules only; checked when applied)
func (r *LuaRuntime) parsePath(L *lua.LState, n int, rule keyRule) world.Ref {
	path := L.CheckString(n)
	head, rest, _ := strings.Cut(path, ".")
	switch head {
	case "self":
		if !r.hasSelf {
			L.ArgError(n, "\"self.\" paths are only available in agent scripts; use \"agent.<id>.<attr>\"")
		}
		return r.attrRef(L, n, r.self.id, rest)
	case "agent":
		idStr, attr, ok := strings.Cut(rest, ".")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if !ok || err != nil {
			L.ArgError(n, fmt.Sprintf("invalid path %q (expected agent.<id>.<attr>)", path))
		}
		return r.attrRef(L, n, world.AgentID(id), attr)
	case "ctx":
		name, key, hasKey := strings.Cut(rest, ".")
		f := r.w.Model.ContextIndex[name]
		if f == nil {
			L.ArgError(n, fmt.Sprintf("unknown context field %q", name))
		}
		switch {
		case hasKey && !f.IsMap:
			L.ArgError(n, fmt.Sprintf("context field %q is not a map", name))
		case hasKey && rule == keyForbidden:
			L.ArgError(n, "this operation applies to a whole map, not an entry")
		case !hasKey && f.IsMap && rule == keyRequiredForMaps:
			L.ArgError(n, fmt.Sprintf("context field %q is a map; address an entry with ctx.%s.<key>", name, name))
		}
		return world.Ref{Field: f, Key: key, HasKey: hasKey}
	}
	L.ArgError(n, fmt.Sprintf("invalid path %q (expected self.<attr>, ctx.<field>[.<key>] or agent.<id>.<attr>)", path))
	return world.Ref{}
}

func (r *LuaRuntime) attrRef(L *lua.LState, n int, id world.AgentID, attr string) world.Ref {
	t := r.w.TypeOf(id)
	if t == nil {
		L.ArgError(n, fmt.Sprintf("agent %d does not exist", id))
	}
	f := t.AttrIndex[attr]
	if f == nil {
		L.ArgError(n, fmt.Sprintf("agent type %q has no attribute %q", t.Name, attr))
	}
	return world.Ref{Field: f, Agent: id}
}

func (r *LuaRuntime) value(L *lua.LState, n int) model.Value {
	v, err := fromLua(L.CheckAny(n))
	if err != nil {
		L.ArgError(n, err.Error())
	}
	return v
}

// endpoint reads an agent argument: an id, an agent proxy, a spawn effect, or
// (when absent) the calling agent.
func (r *LuaRuntime) endpoint(L *lua.LState, n int) world.Endpoint {
	switch v := L.Get(n).(type) {
	case lua.LNumber:
		return world.Endpoint{ID: world.AgentID(v)}
	case *lua.LUserData:
		switch x := v.Value.(type) {
		case *agentRef:
			return world.Endpoint{ID: x.id}
		case *world.Effect:
			if x.Op == world.OpSpawn {
				return world.Endpoint{Spawn: x}
			}
		}
	case *lua.LNilType:
		if r.hasSelf {
			return world.Endpoint{ID: r.self.id}
		}
		L.ArgError(n, "an agent is required outside agent scripts")
	}
	L.ArgError(n, "expected an agent id, an agent, or the result of fx.spawn")
	return world.Endpoint{}
}

func (r *LuaRuntime) push(L *lua.LState, e *world.Effect) int {
	ud := L.NewUserData()
	ud.Value = e
	ud.Metatable = r.effectMT
	L.Push(ud)
	return 1
}

func luaToAny(v lua.LValue, depth int) any {
	switch x := v.(type) {
	case lua.LNumber:
		return float64(x)
	case lua.LString:
		return string(x)
	case lua.LBool:
		return bool(x)
	case *lua.LUserData:
		if ref, ok := x.Value.(*agentRef); ok {
			return uint64(ref.id)
		}
	case *lua.LTable:
		if depth > 8 {
			return "<too deep>"
		}
		if n := x.Len(); n > 0 {
			list := make([]any, n)
			for i := range list {
				list[i] = luaToAny(x.RawGetInt(i+1), depth+1)
			}
			return list
		}
		m := map[string]any{}
		x.ForEach(func(k, val lua.LValue) { m[k.String()] = luaToAny(val, depth+1) })
		return m
	}
	return nil
}

// makeFx builds the global fx table: constructors for every effect scripts can return.
func (r *LuaRuntime) makeFx() *lua.LTable {
	L := r.L
	fns := map[string]lua.LGFunction{
		// fx.set(path, value)
		"set": func(L *lua.LState) int {
			return r.push(L, &world.Effect{Op: world.OpSet, Target: r.parsePath(L, 1, keyRequiredForMaps), Value: r.value(L, 2)})
		},
		// fx.add(path, delta): commutative; never conflicts with other writers.
		"add": func(L *lua.LState) int {
			return r.push(L, &world.Effect{Op: world.OpAdd, Target: r.parsePath(L, 1, keyRequiredForMaps), Value: r.value(L, 2)})
		},
		// fx.transfer(from, to, amount): moves a quantity, conserving the total.
		"transfer": func(L *lua.LState) int {
			return r.push(L, &world.Effect{Op: world.OpTransfer, Source: r.parsePath(L, 1, keyRequiredForMaps),
				Target: r.parsePath(L, 2, keyRequiredForMaps), Value: r.value(L, 3)})
		},
		// fx.clear("ctx.<map>")
		"clear": func(L *lua.LState) int {
			ref := r.parsePath(L, 1, keyForbidden)
			if !ref.IsContext() || !ref.Field.IsMap {
				L.ArgError(1, "fx.clear expects a map context field")
			}
			return r.push(L, &world.Effect{Op: world.OpClear, Target: ref})
		},
		// fx.delete("ctx.<map>.<key>")
		"delete": func(L *lua.LState) int {
			ref := r.parsePath(L, 1, keyRequiredForMaps)
			if !ref.HasKey {
				L.ArgError(1, "fx.delete expects a map entry (ctx.<field>.<key>)")
			}
			return r.push(L, &world.Effect{Op: world.OpDelete, Target: ref})
		},
		// fx.remember(memory, value [, agent])
		"remember": func(L *lua.LState) int {
			return r.push(L, &world.Effect{Op: world.OpRemember, Memory: L.CheckString(1), Value: r.value(L, 2), Agent: r.endpoint(L, 3)})
		},
		// fx.spawn(type [, {attr = value}] [, {x =, y =, groups = {kind = name}}])
		"spawn": func(L *lua.LState) int {
			t := r.checkType(L, 1)
			e := &world.Effect{Op: world.OpSpawn, Type: t}
			if attrs, ok := L.Get(2).(*lua.LTable); ok {
				var names []string
				attrs.ForEach(func(k, _ lua.LValue) { names = append(names, k.String()) })
				sort.Strings(names)
				for _, name := range names {
					f := t.AttrIndex[name]
					if f == nil {
						L.ArgError(2, fmt.Sprintf("agent type %q has no attribute %q", t.Name, name))
					}
					v, err := fromLua(attrs.RawGetString(name))
					if err != nil {
						L.ArgError(2, fmt.Sprintf("%s: %v", name, err))
					}
					e.Attrs = append(e.Attrs, world.AttrValue{Field: f, Value: v})
				}
			}
			if opts, ok := L.Get(3).(*lua.LTable); ok {
				x, y := opts.RawGetString("x"), opts.RawGetString("y")
				if x != lua.LNil || y != lua.LNil {
					xn, okx := x.(lua.LNumber)
					yn, oky := y.(lua.LNumber)
					if !okx || !oky {
						L.ArgError(3, "x and y must both be numbers")
					}
					e.HasPos, e.X, e.Y = true, float64(xn), float64(yn)
				}
				if groups, ok := opts.RawGetString("groups").(*lua.LTable); ok {
					groups.ForEach(func(k, v lua.LValue) {
						kind := r.w.Model.GroupIndex[k.String()]
						if kind == nil {
							L.ArgError(3, fmt.Sprintf("unknown group kind %q", k.String()))
						}
						e.Groups = append(e.Groups, world.GroupName{Kind: kind, Name: v.String()})
					})
					sort.Slice(e.Groups, func(i, j int) bool { return e.Groups[i].Kind.Index < e.Groups[j].Kind.Index })
				}
			}
			return r.push(L, e)
		},
		// fx.remove([agent])
		"remove": func(L *lua.LState) int {
			return r.push(L, &world.Effect{Op: world.OpRemove, Agent: r.endpoint(L, 1)})
		},
		// fx.link(relation, a, b) / fx.unlink(relation, a, b)
		"link": func(L *lua.LState) int {
			g := r.checkRelation(L, 1)
			return r.push(L, &world.Effect{Op: world.OpLink, Relation: g.Relation, Agent: r.endpoint(L, 2), Other: r.endpoint(L, 3)})
		},
		"unlink": func(L *lua.LState) int {
			g := r.checkRelation(L, 1)
			return r.push(L, &world.Effect{Op: world.OpUnlink, Relation: g.Relation, Agent: r.endpoint(L, 2), Other: r.endpoint(L, 3)})
		},
		// fx.join(kind, name [, agent]) / fx.leave(kind, name [, agent])
		"join": func(L *lua.LState) int {
			k := r.checkGroupKind(L, 1)
			return r.push(L, &world.Effect{Op: world.OpJoin, GroupKind: k, Group: L.CheckString(2), Agent: r.endpoint(L, 3)})
		},
		"leave": func(L *lua.LState) int {
			k := r.checkGroupKind(L, 1)
			return r.push(L, &world.Effect{Op: world.OpLeave, GroupKind: k, Group: L.CheckString(2), Agent: r.endpoint(L, 3)})
		},
		// fx.move(x, y [, agent])
		"move": func(L *lua.LState) int {
			return r.push(L, &world.Effect{Op: world.OpMove, X: float64(L.CheckNumber(1)), Y: float64(L.CheckNumber(2)), Agent: r.endpoint(L, 3)})
		},
		// fx.emit(name [, data]): records an event in the log; changes nothing.
		"emit": func(L *lua.LState) int {
			e := &world.Effect{Op: world.OpEmit, Name: L.CheckString(1)}
			if t, ok := L.Get(2).(*lua.LTable); ok {
				if m, ok := luaToAny(t, 0).(map[string]any); ok {
					e.Data = m
				} else {
					e.Data = map[string]any{"value": luaToAny(t, 0)}
				}
			}
			return r.push(L, e)
		},
	}
	t := L.NewTable()
	for name, f := range fns {
		t.RawSetString(name, L.NewFunction(f))
	}
	return t
}
