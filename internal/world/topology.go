package world

import (
	"math"
	"slices"
	"sort"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
)

func insertSorted(s []AgentID, id AgentID) ([]AgentID, bool) {
	i, found := slices.BinarySearch(s, id)
	if found {
		return s, false
	}
	return slices.Insert(s, i, id), true
}

func removeSorted(s []AgentID, id AgentID) ([]AgentID, bool) {
	i, found := slices.BinarySearch(s, id)
	if !found {
		return s, false
	}
	return slices.Delete(s, i, i+1), true
}

// Graph stores one relation as sorted adjacency lists. Undirected relations keep
// each edge in both endpoints' lists.
type Graph struct {
	Relation *model.Relation
	out, in  map[AgentID][]AgentID
}

func newGraph(r *model.Relation) *Graph {
	return &Graph{Relation: r, out: map[AgentID][]AgentID{}, in: map[AgentID][]AgentID{}}
}

func (g *Graph) add(m map[AgentID][]AgentID, a, b AgentID) bool {
	s, ok := insertSorted(m[a], b)
	m[a] = s
	return ok
}

func (g *Graph) del(m map[AgentID][]AgentID, a, b AgentID) bool {
	s, ok := removeSorted(m[a], b)
	if len(s) == 0 {
		delete(m, a)
	} else {
		m[a] = s
	}
	return ok
}

func (g *Graph) link(a, b AgentID) {
	if g.Relation.Directed {
		g.add(g.out, a, b)
		g.add(g.in, b, a)
		return
	}
	g.add(g.out, a, b)
	g.add(g.out, b, a)
}

func (g *Graph) unlink(a, b AgentID) {
	if g.Relation.Directed {
		g.del(g.out, a, b)
		g.del(g.in, b, a)
		return
	}
	g.del(g.out, a, b)
	g.del(g.out, b, a)
}

// Out returns the neighbors of a (outgoing edges for directed relations). The slice
// is shared; callers must not modify it.
func (g *Graph) Out(a AgentID) []AgentID { return g.out[a] }

// In returns incoming neighbors (equal to Out for undirected relations).
func (g *Graph) In(a AgentID) []AgentID {
	if !g.Relation.Directed {
		return g.out[a]
	}
	return g.in[a]
}

func (g *Graph) removeAgent(id AgentID) {
	for _, n := range slices.Clone(g.out[id]) {
		g.unlink(id, n)
	}
	for _, n := range slices.Clone(g.in[id]) {
		g.unlink(n, id)
	}
}

// Edges lists every edge once, sorted. Undirected edges are reported with a < b.
func (g *Graph) Edges() [][2]AgentID {
	keys := make([]AgentID, 0, len(g.out))
	for k := range g.out {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var edges [][2]AgentID
	for _, a := range keys {
		for _, b := range g.out[a] {
			if g.Relation.Directed || a < b {
				edges = append(edges, [2]AgentID{a, b})
			}
		}
	}
	return edges
}

// Groups stores memberships for one group kind.
type Groups struct {
	Kind    *model.GroupKind
	members map[string][]AgentID
	of      map[AgentID][]string
}

func newGroups(k *model.GroupKind) *Groups {
	return &Groups{Kind: k, members: map[string][]AgentID{}, of: map[AgentID][]string{}}
}

func (g *Groups) join(id AgentID, name string) {
	s, added := insertSorted(g.members[name], id)
	if !added {
		return
	}
	g.members[name] = s
	names := g.of[id]
	i, _ := slices.BinarySearch(names, name)
	g.of[id] = slices.Insert(names, i, name)
}

func (g *Groups) leave(id AgentID, name string) {
	s, removed := removeSorted(g.members[name], id)
	if !removed {
		return
	}
	if len(s) == 0 {
		delete(g.members, name)
	} else {
		g.members[name] = s
	}
	names := g.of[id]
	if i, ok := slices.BinarySearch(names, name); ok {
		names = slices.Delete(names, i, i+1)
	}
	if len(names) == 0 {
		delete(g.of, id)
	} else {
		g.of[id] = names
	}
}

func (g *Groups) removeAgent(id AgentID) {
	for _, name := range slices.Clone(g.of[id]) {
		g.leave(id, name)
	}
}

// Members returns the sorted members of a group. The slice is shared.
func (g *Groups) Members(name string) []AgentID { return g.members[name] }

// Of returns the sorted names of the groups an agent belongs to. The slice is shared.
func (g *Groups) Of(id AgentID) []string { return g.of[id] }

// Names returns every non-empty group name, sorted.
func (g *Groups) Names() []string {
	names := make([]string, 0, len(g.members))
	for n := range g.members {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

type cellKey struct{ x, y int32 }

// SpatialIndex is a uniform grid over the 2D space; each cell lists the agents in it.
type SpatialIndex struct {
	Space *model.Space
	cells map[cellKey][]AgentID
}

func newSpatialIndex(sp *model.Space) *SpatialIndex {
	return &SpatialIndex{Space: sp, cells: map[cellKey][]AgentID{}}
}

func (s *SpatialIndex) cell(x, y float64) cellKey {
	return cellKey{int32(x / s.Space.CellSize), int32(y / s.Space.CellSize)}
}

// Clamp keeps a position inside the space bounds.
func (s *SpatialIndex) Clamp(x, y float64) (float64, float64) {
	return math.Min(math.Max(x, 0), s.Space.Width), math.Min(math.Max(y, 0), s.Space.Height)
}

func (s *SpatialIndex) insert(id AgentID, x, y float64) {
	k := s.cell(x, y)
	s.cells[k], _ = insertSorted(s.cells[k], id)
}

func (s *SpatialIndex) remove(id AgentID, x, y float64) {
	k := s.cell(x, y)
	c, _ := removeSorted(s.cells[k], id)
	if len(c) == 0 {
		delete(s.cells, k)
	} else {
		s.cells[k] = c
	}
}

// nearby visits every agent within radius r of (x, y).
func (s *SpatialIndex) nearby(x, y, r float64, visit func(AgentID)) {
	lo, hi := s.cell(x-r, y-r), s.cell(x+r, y+r)
	for cx := lo.x; cx <= hi.x; cx++ {
		for cy := lo.y; cy <= hi.y; cy++ {
			for _, id := range s.cells[cellKey{cx, cy}] {
				visit(id)
			}
		}
	}
}
