// Package world holds the simulation state (agents in columnar stores, the context
// and the topology) and applies effects to it atomically.
package world

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"sync"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/rng"
)

// Loc is where an agent lives: its type and its index in that type's store.
type Loc struct {
	Type  int32
	Index int32
}

// World is the full simulation state. Reads are safe from many goroutines while no
// effects are being applied; the engine guarantees that by alternating read phases
// (scripts, in parallel) and write phases (Apply, sequential).
type World struct {
	Model   *model.Model
	Tick    int
	Stores  []*Store
	Context *Context
	Graphs  []*Graph
	Groups  []*Groups
	Space   *SpatialIndex

	// Hook, when set, observes every effect applied successfully.
	Hook func(Actor, *Effect)

	// locs is indexed by AgentID (IDs are dense and never reused); removed agents
	// have Type -1. A slice avoids hashing on the hottest lookup in the engine.
	locs   []Loc
	alive  int
	nextID AgentID
	names  map[AgentID]string
	byName map[string]AgentID

	written map[writeKey]uint64
	undo    []undoEntry

	aggMu sync.RWMutex
	agg   map[aggKey]float64
}

func New(m *model.Model) *World {
	w := &World{
		Model: m, Context: newContext(m), locs: []Loc{{Type: -1}}, nextID: 1,
		names: map[AgentID]string{}, byName: map[string]AgentID{},
		written: map[writeKey]uint64{}, agg: map[aggKey]float64{},
	}
	for _, t := range m.Types {
		w.Stores = append(w.Stores, newStore(t))
	}
	for _, r := range m.Relations {
		w.Graphs = append(w.Graphs, newGraph(r))
	}
	for _, g := range m.GroupKinds {
		w.Groups = append(w.Groups, newGroups(g))
	}
	if m.Space != nil {
		w.Space = newSpatialIndex(m.Space)
	}
	return w
}

// Populate creates the agents described by the model's agent files.
func (w *World) Populate(seed uint64) error {
	src := rng.New()
	for pi, p := range w.Model.Populations {
		st := w.Stores[p.Type.Index]
		for n := 0; n < p.Count; n++ {
			src.Reset(seed, rng.StreamPopulation, uint64(pi), uint64(n))
			name := ""
			if p.Count == 1 {
				name = p.Name
			}
			id, idx := w.spawn(p.Type, name)
			for _, ag := range p.Attrs {
				v := ag.Gen.Sample(src)
				if err := ag.Field.CheckConstraints(v); err != nil {
					if !ag.Field.Clamp {
						return fmt.Errorf("%s: attributes.%s: generated %v", p.File, ag.Field.Name, err)
					}
					v = clampValue(ag.Field, v)
				}
				st.set(idx, ag.Field, v)
			}
			if p.Type.Spatial {
				x, y := w.Space.Space.Width*src.Float(), w.Space.Space.Height*src.Float()
				if p.X != nil {
					x = p.X.Sample(src).AsFloat()
				}
				if p.Y != nil {
					y = p.Y.Sample(src).AsFloat()
				}
				w.place(id, x, y)
			}
			for _, gg := range p.Groups {
				w.Groups[gg.Kind.Index].join(id, gg.Gen.Sample(src).S)
			}
		}
	}
	for _, p := range w.Model.Populations {
		for _, ln := range p.Links {
			w.Graphs[ln.Relation.Index].link(w.byName[p.Name], w.byName[ln.To])
		}
	}
	return nil
}

func (w *World) spawn(t *model.AgentType, name string) (AgentID, int) {
	id := w.nextID
	w.nextID++
	idx := w.Stores[t.Index].push(id)
	w.locs = append(w.locs, Loc{Type: int32(t.Index), Index: int32(idx)})
	w.alive++
	if name != "" {
		w.names[id] = name
		w.byName[name] = id
	}
	if t.Spatial {
		w.Space.insert(id, 0, 0)
	}
	return id, idx
}

func (w *World) place(id AgentID, x, y float64) {
	l := w.locs[id]
	st := w.Stores[l.Type]
	x, y = w.Space.Clamp(x, y)
	w.Space.remove(id, st.X[l.Index], st.Y[l.Index])
	st.X[l.Index], st.Y[l.Index] = x, y
	w.Space.insert(id, x, y)
}

func (w *World) remove(id AgentID) {
	l, ok := w.Locate(id)
	if !ok {
		return
	}
	st := w.Stores[l.Type]
	for _, g := range w.Graphs {
		g.removeAgent(id)
	}
	for _, g := range w.Groups {
		g.removeAgent(id)
	}
	if st.Type.Spatial {
		w.Space.remove(id, st.X[l.Index], st.Y[l.Index])
	}
	if moved, ok := st.swapRemove(int(l.Index)); ok {
		w.locs[moved] = l
	}
	w.locs[id] = Loc{Type: -1}
	w.alive--
	if name, ok := w.names[id]; ok {
		delete(w.byName, name)
		delete(w.names, id)
	}
}

// Locate returns where an agent lives.
func (w *World) Locate(id AgentID) (Loc, bool) {
	if id >= AgentID(len(w.locs)) {
		return Loc{}, false
	}
	l := w.locs[id]
	return l, l.Type >= 0
}

func (w *World) Exists(id AgentID) bool {
	_, ok := w.Locate(id)
	return ok
}

// TypeOf returns the agent's type, or nil if it does not exist.
func (w *World) TypeOf(id AgentID) *model.AgentType {
	l, ok := w.Locate(id)
	if !ok {
		return nil
	}
	return w.Stores[l.Type].Type
}

// Name returns the name given in the agent file, or "" for unnamed agents.
func (w *World) Name(id AgentID) string { return w.names[id] }

// Find resolves an agent name.
func (w *World) Find(name string) (AgentID, bool) {
	id, ok := w.byName[name]
	return id, ok
}

// Get reads an attribute of an agent by ID.
func (w *World) Get(id AgentID, f *model.Field) (model.Value, bool) {
	l, ok := w.Locate(id)
	if !ok || w.Stores[l.Type].Type != f.Owner {
		return model.Value{}, false
	}
	return w.Stores[l.Type].Get(int(l.Index), f), true
}

// Position returns the agent's coordinates; ok is false for non-spatial agents.
func (w *World) Position(id AgentID) (x, y float64, ok bool) {
	l, found := w.Locate(id)
	if !found || !w.Stores[l.Type].Type.Spatial {
		return 0, 0, false
	}
	st := w.Stores[l.Type]
	return st.X[l.Index], st.Y[l.Index], true
}

// Population returns the total number of agents.
func (w *World) Population() int { return w.alive }

// Nearby returns, sorted, the agents within radius r of (x, y), optionally only of
// type t and excluding one agent.
func (w *World) Nearby(x, y, r float64, t *model.AgentType, exclude AgentID) []AgentID {
	if w.Space == nil {
		return nil
	}
	var out []AgentID
	r2 := r * r
	w.Space.nearby(x, y, r, func(id AgentID) {
		if id == exclude {
			return
		}
		l := w.locs[id]
		st := w.Stores[l.Type]
		if t != nil && st.Type != t {
			return
		}
		dx, dy := st.X[l.Index]-x, st.Y[l.Index]-y
		if dx*dx+dy*dy <= r2 {
			out = append(out, id)
		}
	})
	slices.Sort(out)
	return out
}

// BeginPhase resets per-phase bookkeeping. Call it before each read phase so cached
// aggregates reflect the latest state and context write conflicts are tracked anew.
func (w *World) BeginPhase() {
	clear(w.written)
	w.aggMu.Lock()
	clear(w.agg)
	w.aggMu.Unlock()
}

type aggKey struct {
	agg   model.AggKind
	typ   int32
	field *model.Field
	group int32
	name  string
	attr  string
}

func (w *World) cached(k aggKey, compute func() float64) float64 {
	w.aggMu.RLock()
	v, ok := w.agg[k]
	w.aggMu.RUnlock()
	if ok {
		return v
	}
	v = compute()
	w.aggMu.Lock()
	w.agg[k] = v
	w.aggMu.Unlock()
	return v
}

// Aggregate computes agg over attribute f of every agent of type t. Results are
// memoized until the next BeginPhase, so thousands of agents asking for the same
// mean cost a single scan.
func (w *World) Aggregate(agg model.AggKind, t *model.AgentType, f *model.Field) float64 {
	st := w.Stores[t.Index]
	if agg == model.AggCount {
		return float64(st.Len())
	}
	return w.cached(aggKey{agg: agg, typ: int32(t.Index), field: f, group: -1}, func() float64 {
		return reduce(agg, st.Len(), func(i int) float64 { return st.Float(i, f) })
	})
}

// GroupAggregate computes agg over attribute attr of the members of a group. Members
// whose type lacks the attribute are skipped.
func (w *World) GroupAggregate(agg model.AggKind, kind *model.GroupKind, name, attr string) float64 {
	return w.cached(aggKey{agg: agg, typ: -1, group: int32(kind.Index), name: name, attr: attr}, func() float64 {
		var vals []float64
		for _, id := range w.Groups[kind.Index].Members(name) {
			l := w.locs[id]
			st := w.Stores[l.Type]
			if agg == model.AggCount {
				vals = append(vals, 0)
				continue
			}
			if f := st.Type.AttrIndex[attr]; f != nil {
				vals = append(vals, st.Float(int(l.Index), f))
			}
		}
		return reduce(agg, len(vals), func(i int) float64 { return vals[i] })
	})
}

func reduce(agg model.AggKind, n int, at func(int) float64) float64 {
	if agg == model.AggCount {
		return float64(n)
	}
	if n == 0 {
		if agg == model.AggSum {
			return 0
		}
		return math.NaN()
	}
	switch agg {
	case model.AggSum, model.AggMean:
		s := 0.0
		for i := 0; i < n; i++ {
			s += at(i)
		}
		if agg == model.AggMean {
			return s / float64(n)
		}
		return s
	case model.AggMin, model.AggMax:
		best := at(0)
		for i := 1; i < n; i++ {
			v := at(i)
			if (agg == model.AggMin && v < best) || (agg == model.AggMax && v > best) {
				best = v
			}
		}
		return best
	case model.AggStd:
		mean, m2 := 0.0, 0.0
		for i := 0; i < n; i++ {
			x := at(i)
			d := x - mean
			mean += d / float64(i+1)
			m2 += d * (x - mean)
		}
		return math.Sqrt(m2 / float64(n))
	case model.AggGini:
		vals := make([]float64, n)
		total := 0.0
		for i := range vals {
			vals[i] = at(i)
			total += vals[i]
		}
		if total == 0 {
			return 0
		}
		sort.Float64s(vals)
		weighted := 0.0
		for i, v := range vals {
			weighted += float64(i+1) * v
		}
		fn := float64(n)
		return 2*weighted/(fn*total) - (fn+1)/fn
	}
	return math.NaN()
}

func clampValue(f *model.Field, v model.Value) model.Value {
	x := v.AsFloat()
	if f.Min != nil && x < *f.Min {
		x = *f.Min
	}
	if f.Max != nil && x > *f.Max {
		x = *f.Max
	}
	if f.Kind == model.KindInt {
		return model.Int(int64(x))
	}
	return model.Float(x)
}
