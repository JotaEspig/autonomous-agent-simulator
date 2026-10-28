package world

import (
	"fmt"
	"strings"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
)

// Op is the kind of change an effect requests. Effects are the only way scripts
// change the world: scripts compute them, the engine validates and applies them.
type Op uint8

const (
	OpSet Op = iota + 1
	OpAdd
	OpTransfer
	OpClear
	OpDelete
	OpRemember
	OpSpawn
	OpRemove
	OpLink
	OpUnlink
	OpJoin
	OpLeave
	OpMove
	OpEmit
)

var opNames = [...]string{"", "set", "add", "transfer", "clear", "delete", "remember", "spawn", "remove", "link", "unlink", "join", "leave", "move", "emit"}

func (o Op) String() string { return opNames[o] }

// isValue reports whether the op changes a numeric/scalar value and may be rejected
// by constraints; those ops run first, with undo, inside a transaction.
func (o Op) isValue() bool { return o == OpSet || o == OpAdd || o == OpTransfer }

// Ref addresses a value: a context field (optionally a map entry) or an agent attribute.
type Ref struct {
	Field  *model.Field
	Agent  AgentID
	Key    string
	HasKey bool
}

func (r Ref) IsContext() bool { return r.Field.Owner == nil }

func (r Ref) String() string {
	if r.IsContext() {
		if r.HasKey {
			return "ctx." + r.Field.Name + "." + r.Key
		}
		return "ctx." + r.Field.Name
	}
	return fmt.Sprintf("agent.%d.%s", r.Agent, r.Field.Name)
}

// Endpoint designates an agent: an existing one, or one spawned earlier in the
// same transaction (Spawn points at that spawn effect).
type Endpoint struct {
	ID    AgentID
	Spawn *Effect
}

func (e Endpoint) resolve() AgentID {
	if e.Spawn != nil {
		return e.Spawn.spawned
	}
	return e.ID
}

func (e Endpoint) String() string {
	if e.Spawn != nil {
		if e.Spawn.spawned != 0 {
			return fmt.Sprintf("%d", e.Spawn.spawned)
		}
		return "<spawned " + e.Spawn.Type.Name + ">"
	}
	return fmt.Sprintf("%d", e.ID)
}

type AttrValue struct {
	Field *model.Field
	Value model.Value
}

type GroupName struct {
	Kind *model.GroupKind
	Name string
}

// Effect is one requested change. Which fields are used depends on Op.
type Effect struct {
	Op     Op
	Target Ref         // set, add, transfer (destination), clear, delete
	Source Ref         // transfer (origin)
	Value  model.Value // set value, add delta, transfer amount, remember value

	Agent Endpoint // remember, remove, join, leave, move; link/unlink origin
	Other Endpoint // link/unlink destination

	Memory    string
	Relation  *model.Relation
	GroupKind *model.GroupKind
	Group     string
	X, Y      float64

	// spawn
	Type   *model.AgentType
	Attrs  []AttrValue
	HasPos bool
	Groups []GroupName

	// emit
	Name string
	Data map[string]any

	spawned AgentID
}

// Spawned returns the ID assigned by a successful spawn effect.
func (e *Effect) Spawned() AgentID { return e.spawned }

func (e *Effect) String() string {
	switch e.Op {
	case OpSet:
		return fmt.Sprintf("set %s = %s", e.Target, e.Value)
	case OpAdd:
		return fmt.Sprintf("add %s += %s", e.Target, e.Value)
	case OpTransfer:
		return fmt.Sprintf("transfer %s from %s to %s", e.Value, e.Source, e.Target)
	case OpClear:
		return fmt.Sprintf("clear %s", e.Target)
	case OpDelete:
		return fmt.Sprintf("delete %s", e.Target)
	case OpRemember:
		return fmt.Sprintf("remember %s %s for %s", e.Memory, e.Value, e.Agent)
	case OpSpawn:
		var parts []string
		for _, a := range e.Attrs {
			parts = append(parts, a.Field.Name+"="+a.Value.String())
		}
		return fmt.Sprintf("spawn %s %s {%s}", e.Type.Name, Endpoint{Spawn: e}, strings.Join(parts, ", "))
	case OpRemove:
		return fmt.Sprintf("remove %s", e.Agent)
	case OpLink:
		return fmt.Sprintf("link %s %s -> %s", e.Relation.Name, e.Agent, e.Other)
	case OpUnlink:
		return fmt.Sprintf("unlink %s %s -> %s", e.Relation.Name, e.Agent, e.Other)
	case OpJoin:
		return fmt.Sprintf("join %s %s/%s", e.Agent, e.GroupKind.Name, e.Group)
	case OpLeave:
		return fmt.Sprintf("leave %s %s/%s", e.Agent, e.GroupKind.Name, e.Group)
	case OpMove:
		return fmt.Sprintf("move %s to (%g, %g)", e.Agent, e.X, e.Y)
	case OpEmit:
		return fmt.Sprintf("emit %s", e.Name)
	}
	return "unknown effect"
}
