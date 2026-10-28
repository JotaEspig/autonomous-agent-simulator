package world

import "github.com/JotaEspig/autonomous-agent-simulator/internal/model"

// AgentID identifies an agent for the whole run. IDs start at 1 and are never reused.
type AgentID uint64

// Store holds every agent of one type in columns (struct of arrays): one typed slice
// per attribute. Index i of every column belongs to the agent IDs[i]. Removal swaps
// the last agent into the freed slot, so indices are only stable within a phase.
type Store struct {
	Type *model.AgentType
	IDs  []AgentID

	ints   [][]int64
	floats [][]float64
	bools  [][]bool
	strs   [][]string
	X, Y   []float64
	mems   []*memColumn
}

func newStore(t *model.AgentType) *Store {
	s := &Store{
		Type:   t,
		ints:   make([][]int64, t.KindCount[model.KindInt]),
		floats: make([][]float64, t.KindCount[model.KindFloat]),
		bools:  make([][]bool, t.KindCount[model.KindBool]),
		strs:   make([][]string, t.KindCount[model.KindString]),
		mems:   make([]*memColumn, len(t.Memories)),
	}
	for i, m := range t.Memories {
		s.mems[i] = &memColumn{mem: m}
	}
	return s
}

func (s *Store) Len() int { return len(s.IDs) }

// Get reads attribute f of the agent at index i.
func (s *Store) Get(i int, f *model.Field) model.Value {
	switch f.Kind {
	case model.KindInt:
		return model.Int(s.ints[f.Slot][i])
	case model.KindFloat:
		return model.Float(s.floats[f.Slot][i])
	case model.KindBool:
		return model.Bool(s.bools[f.Slot][i])
	case model.KindString:
		return model.String(s.strs[f.Slot][i])
	}
	return model.Value{}
}

// Float reads a numeric or bool attribute as float64 without building a Value.
func (s *Store) Float(i int, f *model.Field) float64 {
	switch f.Kind {
	case model.KindInt:
		return float64(s.ints[f.Slot][i])
	case model.KindFloat:
		return s.floats[f.Slot][i]
	case model.KindBool:
		if s.bools[f.Slot][i] {
			return 1
		}
	}
	return 0
}

func (s *Store) set(i int, f *model.Field, v model.Value) {
	switch f.Kind {
	case model.KindInt:
		s.ints[f.Slot][i] = v.I
	case model.KindFloat:
		s.floats[f.Slot][i] = v.F
	case model.KindBool:
		s.bools[f.Slot][i] = v.B
	case model.KindString:
		s.strs[f.Slot][i] = v.S
	}
}

// push appends an agent initialized with attribute defaults and returns its index.
func (s *Store) push(id AgentID) int {
	s.IDs = append(s.IDs, id)
	for _, f := range s.Type.Attrs {
		switch f.Kind {
		case model.KindInt:
			s.ints[f.Slot] = append(s.ints[f.Slot], f.Default.I)
		case model.KindFloat:
			s.floats[f.Slot] = append(s.floats[f.Slot], f.Default.F)
		case model.KindBool:
			s.bools[f.Slot] = append(s.bools[f.Slot], f.Default.B)
		case model.KindString:
			s.strs[f.Slot] = append(s.strs[f.Slot], f.Default.S)
		}
	}
	if s.Type.Spatial {
		s.X = append(s.X, 0)
		s.Y = append(s.Y, 0)
	}
	for _, m := range s.mems {
		m.grow()
	}
	return len(s.IDs) - 1
}

// swapRemove deletes index i by moving the last agent into it. It returns the ID of
// the moved agent, if any, so the caller can update its location.
func (s *Store) swapRemove(i int) (AgentID, bool) {
	last := len(s.IDs) - 1
	var moved AgentID
	ok := i != last
	if ok {
		moved = s.IDs[last]
		s.IDs[i] = moved
		for _, c := range s.ints {
			c[i] = c[last]
		}
		for _, c := range s.floats {
			c[i] = c[last]
		}
		for _, c := range s.bools {
			c[i] = c[last]
		}
		for _, c := range s.strs {
			c[i] = c[last]
		}
		if s.Type.Spatial {
			s.X[i], s.Y[i] = s.X[last], s.Y[last]
		}
		for _, m := range s.mems {
			m.move(i, last)
		}
	}
	s.IDs = s.IDs[:last]
	for k := range s.ints {
		s.ints[k] = s.ints[k][:last]
	}
	for k := range s.floats {
		s.floats[k] = s.floats[k][:last]
	}
	for k := range s.bools {
		s.bools[k] = s.bools[k][:last]
	}
	for k := range s.strs {
		s.strs[k][last] = ""
		s.strs[k] = s.strs[k][:last]
	}
	if s.Type.Spatial {
		s.X, s.Y = s.X[:last], s.Y[:last]
	}
	for _, m := range s.mems {
		m.truncate(last)
	}
	return moved, ok
}

// Recall returns the memory entry `back` steps in the past (0 is the most recent).
func (s *Store) Recall(i int, m *model.Memory, back int) (model.Value, bool) {
	return s.mems[m.Slot].get(i, back)
}

// MemLen returns how many entries memory m currently holds for agent i.
func (s *Store) MemLen(i int, m *model.Memory) int { return int(s.mems[m.Slot].size[i]) }

func (s *Store) remember(i int, m *model.Memory, v model.Value) { s.mems[m.Slot].push(i, v) }

// memColumn stores one bounded memory for all agents of a type in flat typed
// buffers: agent i owns positions [i*capacity, (i+1)*capacity).
type memColumn struct {
	mem    *model.Memory
	head   []int32 // next write position within the agent's block
	size   []int32
	ints   []int64
	floats []float64
	bools  []bool
	strs   []string
}

func (c *memColumn) grow() {
	c.head = append(c.head, 0)
	c.size = append(c.size, 0)
	n := c.mem.Capacity
	switch c.mem.Kind {
	case model.KindInt:
		c.ints = append(c.ints, make([]int64, n)...)
	case model.KindFloat:
		c.floats = append(c.floats, make([]float64, n)...)
	case model.KindBool:
		c.bools = append(c.bools, make([]bool, n)...)
	case model.KindString:
		c.strs = append(c.strs, make([]string, n)...)
	}
}

func (c *memColumn) push(i int, v model.Value) {
	n := c.mem.Capacity
	pos := i*n + int(c.head[i])
	switch c.mem.Kind {
	case model.KindInt:
		c.ints[pos] = v.I
	case model.KindFloat:
		c.floats[pos] = v.F
	case model.KindBool:
		c.bools[pos] = v.B
	case model.KindString:
		c.strs[pos] = v.S
	}
	c.head[i] = int32((int(c.head[i]) + 1) % n)
	if int(c.size[i]) < n {
		c.size[i]++
	}
}

func (c *memColumn) get(i, back int) (model.Value, bool) {
	if back < 0 || back >= int(c.size[i]) {
		return model.Value{}, false
	}
	n := c.mem.Capacity
	pos := i*n + (int(c.head[i])-1-back+2*n)%n
	switch c.mem.Kind {
	case model.KindInt:
		return model.Int(c.ints[pos]), true
	case model.KindFloat:
		return model.Float(c.floats[pos]), true
	case model.KindBool:
		return model.Bool(c.bools[pos]), true
	default:
		return model.String(c.strs[pos]), true
	}
}

func (c *memColumn) move(dst, src int) {
	n := c.mem.Capacity
	c.head[dst], c.size[dst] = c.head[src], c.size[src]
	d, s := dst*n, src*n
	switch c.mem.Kind {
	case model.KindInt:
		copy(c.ints[d:d+n], c.ints[s:s+n])
	case model.KindFloat:
		copy(c.floats[d:d+n], c.floats[s:s+n])
	case model.KindBool:
		copy(c.bools[d:d+n], c.bools[s:s+n])
	case model.KindString:
		copy(c.strs[d:d+n], c.strs[s:s+n])
	}
}

func (c *memColumn) truncate(agents int) {
	c.head, c.size = c.head[:agents], c.size[:agents]
	n := agents * c.mem.Capacity
	switch c.mem.Kind {
	case model.KindInt:
		c.ints = c.ints[:n]
	case model.KindFloat:
		c.floats = c.floats[:n]
	case model.KindBool:
		c.bools = c.bools[:n]
	case model.KindString:
		clear(c.strs[n:])
		c.strs = c.strs[:n]
	}
}
