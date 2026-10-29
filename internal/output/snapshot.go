package output

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/world"
)

type snapshotAgent struct {
	ID         uint64           `json:"id"`
	Name       string           `json:"name,omitempty"`
	X          *float64         `json:"x,omitempty"`
	Y          *float64         `json:"y,omitempty"`
	Attributes map[string]any   `json:"attributes"`
	Memory     map[string][]any `json:"memory,omitempty"`
}

// WriteSnapshot streams the full state as JSON, one agent at a time, so snapshots
// of large populations never need the whole document in memory. Agents are sorted
// by ID and memories are listed most recent first.
func WriteSnapshot(out io.Writer, w *world.World) error {
	enc := json.NewEncoder(out)
	write := func(s string) error {
		_, err := io.WriteString(out, s)
		return err
	}
	if err := write(`{"tick":`); err != nil {
		return err
	}
	if err := enc.Encode(w.Tick); err != nil {
		return err
	}

	ctx := map[string]any{}
	for _, f := range w.Model.Context {
		if f.IsMap {
			m := map[string]any{}
			for _, k := range w.Context.MapKeys(f) {
				v, _ := w.Context.MapGet(f, k)
				m[k] = v.Any()
			}
			ctx[f.Name] = m
			continue
		}
		ctx[f.Name] = w.Context.Get(f).Any()
	}
	if err := write(`,"context":`); err != nil {
		return err
	}
	if err := enc.Encode(ctx); err != nil {
		return err
	}

	if err := write(`,"agents":{`); err != nil {
		return err
	}
	for ti, st := range w.Stores {
		if ti > 0 {
			if err := write(","); err != nil {
				return err
			}
		}
		name, _ := json.Marshal(st.Type.Name)
		if err := write(string(name) + ":["); err != nil {
			return err
		}
		order := make([]int, st.Len())
		for i := range order {
			order[i] = i
		}
		sort.Slice(order, func(a, b int) bool { return st.IDs[order[a]] < st.IDs[order[b]] })
		for n, i := range order {
			if n > 0 {
				if err := write(","); err != nil {
					return err
				}
			}
			a := snapshotAgent{ID: uint64(st.IDs[i]), Name: w.Name(st.IDs[i]), Attributes: map[string]any{}}
			for _, f := range st.Type.Attrs {
				a.Attributes[f.Name] = st.Get(i, f).Any()
			}
			if st.Type.Spatial {
				x, y := st.X[i], st.Y[i]
				a.X, a.Y = &x, &y
			}
			if len(st.Type.Memories) > 0 {
				a.Memory = map[string][]any{}
				for _, m := range st.Type.Memories {
					vals := make([]any, st.MemLen(i, m))
					for k := range vals {
						v, _ := st.Recall(i, m, k)
						vals[k] = v.Any()
					}
					a.Memory[m.Name] = vals
				}
			}
			if err := enc.Encode(a); err != nil {
				return err
			}
		}
		if err := write("]"); err != nil {
			return err
		}
	}

	rels := map[string][][2]world.AgentID{}
	for _, g := range w.Graphs {
		rels[g.Relation.Name] = g.Edges()
	}
	if err := write(`},"relations":`); err != nil {
		return err
	}
	if err := enc.Encode(rels); err != nil {
		return err
	}
	groups := map[string]map[string][]world.AgentID{}
	for _, g := range w.Groups {
		m := map[string][]world.AgentID{}
		for _, name := range g.Names() {
			m[name] = g.Members(name)
		}
		groups[g.Kind.Name] = m
	}
	if err := write(`,"groups":`); err != nil {
		return err
	}
	if err := enc.Encode(groups); err != nil {
		return err
	}
	return write("}\n")
}
