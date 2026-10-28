package world

import (
	"fmt"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
)

// Actor is whoever produced a list of effects.
type Actor struct {
	ID   AgentID     // the acting agent; 0 for world-scoped rules
	Rule *model.Rule // set when a rule produced the effects (rules may change anything)
}

func (a Actor) privileged() bool { return a.Rule != nil }

func (a Actor) key() uint64 {
	if a.Rule != nil && a.ID == 0 {
		return 1<<63 | uint64(a.Rule.Index)
	}
	return uint64(a.ID)
}

func (a Actor) String() string {
	switch {
	case a.Rule != nil && a.ID != 0:
		return fmt.Sprintf("rule %q (agent %d)", a.Rule.Name, a.ID)
	case a.Rule != nil:
		return fmt.Sprintf("rule %q", a.Rule.Name)
	}
	return fmt.Sprintf("agent %d", a.ID)
}

// ModelError reports effects that can never be valid: wrong types, permission
// violations, unknown memories. These are bugs in the model, so the run stops.
type ModelError struct{ Msg string }

func (e *ModelError) Error() string { return e.Msg }

func modelErr(format string, args ...any) error {
	return &ModelError{Msg: fmt.Sprintf(format, args...)}
}

type writeKey struct {
	field *model.Field
	key   string
}

type undoEntry struct {
	ref     Ref
	old     model.Value
	existed bool

	isWriter bool
	wkey     writeKey
	wprev    uint64
	whad     bool
}

// UndoBuffer holds rollback data for a transaction. Each goroutine applying local
// transactions concurrently needs its own.
type UndoBuffer struct{ entries []undoEntry }

// IsLocal reports whether a transaction only touches the actor's own attributes
// and memory. Agents cannot change other agents, so during the decision phase such
// transactions commute with every other transaction and can be applied in
// parallel with ApplyLocal without changing the result.
func IsLocal(a Actor, effs []*Effect) bool {
	if a.privileged() {
		return false
	}
	self := func(r Ref) bool { return !r.IsContext() && r.Agent == a.ID }
	for _, e := range effs {
		switch e.Op {
		case OpSet, OpAdd:
			if !self(e.Target) {
				return false
			}
		case OpTransfer:
			if !self(e.Target) || !self(e.Source) {
				return false
			}
		case OpRemember:
			if e.Agent.Spawn != nil || e.Agent.ID != a.ID {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// ApplyLocal applies a transaction for which IsLocal is true. It is safe to call
// concurrently for different actors, each with its own UndoBuffer, as long as no
// other kind of transaction runs at the same time. The Hook is not called.
func (w *World) ApplyLocal(a Actor, effs []*Effect, undo *UndoBuffer) (string, error) {
	return w.apply(a, effs, &undo.entries, false)
}

// Apply validates and applies effects as one transaction: either all of them take
// effect or none does. A non-empty reason means the transaction was rejected by
// the current state (a constraint, a missing agent, a merge policy); a non-nil
// error means the model itself is wrong.
//
// Value changes (set, add, transfer) run first and are rolled back on rejection;
// structural changes (spawn, remove, link, memory, ...) are fully checked up front
// and then run in the order given.
func (w *World) Apply(a Actor, effs []*Effect) (string, error) {
	return w.apply(a, effs, &w.undo, true)
}

func (w *World) apply(a Actor, effs []*Effect, undo *[]undoEntry, hook bool) (string, error) {
	if len(effs) == 0 {
		return "", nil
	}
	if !a.privileged() && !w.Exists(a.ID) {
		return "the agent no longer exists", nil
	}
	for i, e := range effs {
		if reason, err := w.check(a, e, effs[:i]); err != nil || reason != "" {
			return reason, err
		}
	}
	*undo = (*undo)[:0]
	for _, e := range effs {
		if !e.Op.isValue() {
			continue
		}
		if reason, err := w.applyValue(a, e, undo); err != nil || reason != "" {
			w.rollback(undo)
			return reason, err
		}
	}
	for _, e := range effs {
		if !e.Op.isValue() {
			w.applyStructural(a, e)
		}
		if hook && w.Hook != nil {
			w.Hook(a, e)
		}
	}
	return "", nil
}

func (w *World) checkEndpoint(a Actor, ep Endpoint, prior []*Effect, what string) (string, error) {
	if ep.Spawn != nil {
		for _, p := range prior {
			if p == ep.Spawn {
				return "", nil
			}
		}
		return "", modelErr("%s: %s refers to a spawned agent whose spawn effect is not earlier in the same list of effects", a, what)
	}
	if !a.privileged() && ep.ID != a.ID {
		return "", modelErr("%s: %s targets agent %d; agents may only affect themselves, the context and agents they spawn (use a rule to affect other agents)", a, what, ep.ID)
	}
	if !w.Exists(ep.ID) {
		return fmt.Sprintf("agent %d no longer exists", ep.ID), nil
	}
	return "", nil
}

func (w *World) checkRef(a Actor, r Ref, what string) (string, error) {
	if r.IsContext() {
		return "", nil
	}
	if !a.privileged() && r.Agent != a.ID {
		return "", modelErr("%s: %s targets %s; agents may only change their own attributes and the context (use a rule to affect other agents)", a, what, r)
	}
	if w.TypeOf(r.Agent) == nil {
		return fmt.Sprintf("agent %d no longer exists", r.Agent), nil
	}
	return "", nil
}

func (w *World) check(a Actor, e *Effect, prior []*Effect) (string, error) {
	what := e.Op.String()
	switch e.Op {
	case OpSet, OpAdd, OpTransfer:
		if reason, err := w.checkRef(a, e.Target, what); err != nil || reason != "" {
			return reason, err
		}
		f := e.Target.Field
		if e.Op == OpSet {
			v, err := model.Coerce(e.Value, f.Kind)
			if err != nil {
				return "", modelErr("%s: set %s: %v", a, e.Target, err)
			}
			e.Value = v
			return "", nil
		}
		if !f.Kind.Numeric() {
			return "", modelErr("%s: %s %s: field is %s, not numeric", a, what, e.Target, f.Kind)
		}
		v, err := model.Coerce(e.Value, f.Kind)
		if err != nil {
			return "", modelErr("%s: %s %s: %v", a, what, e.Target, err)
		}
		e.Value = v
		if e.Op == OpTransfer {
			if reason, err := w.checkRef(a, e.Source, what); err != nil || reason != "" {
				return reason, err
			}
			if e.Source.Field.Kind != f.Kind {
				return "", modelErr("%s: transfer between %s (%s) and %s (%s): both sides must have the same type", a, e.Source, e.Source.Field.Kind, e.Target, f.Kind)
			}
			if e.Value.AsFloat() < 0 {
				return "", modelErr("%s: transfer amount must be >= 0 (got %s)", a, e.Value)
			}
		}
	case OpRemember:
		if reason, err := w.checkEndpoint(a, e.Agent, prior, what); err != nil || reason != "" {
			return reason, err
		}
		t := e.endpointType(w, e.Agent)
		mem := t.MemoryIndex[e.Memory]
		if mem == nil {
			return "", modelErr("%s: agent type %q has no memory %q", a, t.Name, e.Memory)
		}
		v, err := model.Coerce(e.Value, mem.Kind)
		if err != nil {
			return "", modelErr("%s: remember %s: %v", a, e.Memory, err)
		}
		e.Value = v
	case OpSpawn:
		for i, av := range e.Attrs {
			v, err := model.Coerce(av.Value, av.Field.Kind)
			if err != nil {
				return "", modelErr("%s: spawn %s: attribute %s: %v", a, e.Type.Name, av.Field.Name, err)
			}
			if err := av.Field.CheckConstraints(v); err != nil {
				if !av.Field.Clamp {
					return fmt.Sprintf("spawn %s: attribute %s: %v", e.Type.Name, av.Field.Name, err), nil
				}
				v = clampValue(av.Field, v)
			}
			e.Attrs[i].Value = v
		}
		if e.HasPos && !e.Type.Spatial {
			return "", modelErr("%s: spawn %s: a position was given but the type is not spatial", a, e.Type.Name)
		}
	case OpRemove, OpJoin, OpLeave:
		return w.checkEndpoint(a, e.Agent, prior, what)
	case OpMove:
		if reason, err := w.checkEndpoint(a, e.Agent, prior, what); err != nil || reason != "" {
			return reason, err
		}
		if t := e.endpointType(w, e.Agent); !t.Spatial {
			return "", modelErr("%s: move: agent type %q is not spatial", a, t.Name)
		}
	case OpLink, OpUnlink:
		if reason, err := w.checkEndpoint(a, e.Agent, prior, what); err != nil || reason != "" {
			return reason, err
		}
		return w.checkEndpoint(a, e.Other, prior, what)
	}
	return "", nil
}

func (e *Effect) endpointType(w *World, ep Endpoint) *model.AgentType {
	if ep.Spawn != nil {
		return ep.Spawn.Type
	}
	return w.TypeOf(ep.ID)
}

func (w *World) read(r Ref) (model.Value, bool) {
	if r.IsContext() {
		if r.Field.IsMap {
			v, ok := w.Context.maps[r.Field.Slot][r.Key]
			if !ok {
				return model.Zero(r.Field.Kind), false
			}
			return v, true
		}
		return w.Context.scalars[r.Field.Slot], true
	}
	l := w.locs[r.Agent]
	return w.Stores[l.Type].Get(int(l.Index), r.Field), true
}

func (w *World) write(r Ref, v model.Value) {
	if r.IsContext() {
		if r.Field.IsMap {
			w.Context.maps[r.Field.Slot][r.Key] = v
			return
		}
		w.Context.scalars[r.Field.Slot] = v
		return
	}
	l := w.locs[r.Agent]
	w.Stores[l.Type].set(int(l.Index), r.Field, v)
}

// store writes v after enforcing the field's constraints, recording undo data.
func (w *World) store(r Ref, v model.Value, undo *[]undoEntry) string {
	if err := r.Field.CheckConstraints(v); err != nil {
		if !r.Field.Clamp {
			return fmt.Sprintf("%s: %v", r, err)
		}
		v = clampValue(r.Field, v)
	}
	old, existed := w.read(r)
	*undo = append(*undo, undoEntry{ref: r, old: old, existed: existed})
	w.write(r, v)
	return ""
}

func add(a, b model.Value) model.Value {
	if a.Kind == model.KindInt {
		return model.Int(a.I + b.I)
	}
	return model.Float(a.F + b.F)
}

func neg(v model.Value) model.Value {
	if v.Kind == model.KindInt {
		return model.Int(-v.I)
	}
	return model.Float(-v.F)
}

func (w *World) applyValue(a Actor, e *Effect, undo *[]undoEntry) (string, error) {
	switch e.Op {
	case OpSet:
		if e.Target.IsContext() {
			k := writeKey{field: e.Target.Field, key: e.Target.Key}
			if prev, ok := w.written[k]; ok && prev != a.key() {
				switch e.Target.Field.Merge {
				case model.MergeError:
					return "", modelErr("%s was set by two different actors in the same phase (the second was %s). "+
						"Partition the field (a map keyed by agent id), change it with \"add\", or declare \"merge\": \"last\" or \"first\"", e.Target, a)
				case model.MergeFirst:
					return fmt.Sprintf("%s was already set by another actor in this phase", e.Target), nil
				}
			}
			prev, had := w.written[k]
			*undo = append(*undo, undoEntry{isWriter: true, wkey: k, wprev: prev, whad: had})
			w.written[k] = a.key()
		}
		return w.store(e.Target, e.Value, undo), nil
	case OpAdd:
		cur, _ := w.read(e.Target)
		return w.store(e.Target, add(cur, e.Value), undo), nil
	case OpTransfer:
		src, _ := w.read(e.Source)
		if reason := w.store(e.Source, add(src, neg(e.Value)), undo); reason != "" {
			return reason, nil
		}
		dst, _ := w.read(e.Target)
		return w.store(e.Target, add(dst, e.Value), undo), nil
	}
	return "", nil
}

func (w *World) rollback(undo *[]undoEntry) {
	for i := len(*undo) - 1; i >= 0; i-- {
		u := (*undo)[i]
		switch {
		case u.isWriter && u.whad:
			w.written[u.wkey] = u.wprev
		case u.isWriter:
			delete(w.written, u.wkey)
		case !u.existed:
			delete(w.Context.maps[u.ref.Field.Slot], u.ref.Key)
		default:
			w.write(u.ref, u.old)
		}
	}
	*undo = (*undo)[:0]
}

func (w *World) applyStructural(a Actor, e *Effect) {
	switch e.Op {
	case OpClear:
		clear(w.Context.maps[e.Target.Field.Slot])
	case OpDelete:
		delete(w.Context.maps[e.Target.Field.Slot], e.Target.Key)
	case OpRemember:
		id := e.Agent.resolve()
		l := w.locs[id]
		st := w.Stores[l.Type]
		st.remember(int(l.Index), st.Type.MemoryIndex[e.Memory], e.Value)
	case OpSpawn:
		id, idx := w.spawn(e.Type, "")
		st := w.Stores[e.Type.Index]
		for _, av := range e.Attrs {
			st.set(idx, av.Field, av.Value)
		}
		if e.Type.Spatial {
			x, y := e.X, e.Y
			if !e.HasPos {
				x, y, _ = w.Position(a.ID)
			}
			w.place(id, x, y)
		}
		for _, g := range e.Groups {
			w.Groups[g.Kind.Index].join(id, g.Name)
		}
		e.spawned = id
	case OpRemove:
		w.remove(e.Agent.resolve())
	case OpLink:
		w.Graphs[e.Relation.Index].link(e.Agent.resolve(), e.Other.resolve())
	case OpUnlink:
		w.Graphs[e.Relation.Index].unlink(e.Agent.resolve(), e.Other.resolve())
	case OpJoin:
		w.Groups[e.GroupKind.Index].join(e.Agent.resolve(), e.Group)
	case OpLeave:
		w.Groups[e.GroupKind.Index].leave(e.Agent.resolve(), e.Group)
	case OpMove:
		w.place(e.Agent.resolve(), e.X, e.Y)
	}
}
