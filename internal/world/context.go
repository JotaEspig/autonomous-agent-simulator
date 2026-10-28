package world

import (
	"sort"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
)

// Context is the shared environment: scalar fields and string-keyed map fields,
// indexed by Field.Slot.
type Context struct {
	scalars []model.Value
	maps    []map[string]model.Value
}

func newContext(m *model.Model) *Context {
	c := &Context{scalars: make([]model.Value, len(m.Context)), maps: make([]map[string]model.Value, len(m.Context))}
	for _, f := range m.Context {
		if f.IsMap {
			c.maps[f.Slot] = make(map[string]model.Value, len(f.DefaultMap))
			for k, v := range f.DefaultMap {
				c.maps[f.Slot][k] = v
			}
			continue
		}
		c.scalars[f.Slot] = f.Default
	}
	return c
}

// Get returns a scalar field.
func (c *Context) Get(f *model.Field) model.Value { return c.scalars[f.Slot] }

// MapGet returns an entry of a map field.
func (c *Context) MapGet(f *model.Field, key string) (model.Value, bool) {
	v, ok := c.maps[f.Slot][key]
	return v, ok
}

// MapKeys returns the keys of a map field in sorted order, for deterministic iteration.
func (c *Context) MapKeys(f *model.Field) []string {
	m := c.maps[f.Slot]
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (c *Context) MapLen(f *model.Field) int { return len(c.maps[f.Slot]) }
