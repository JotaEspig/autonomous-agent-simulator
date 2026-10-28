// Package model loads, validates and compiles a model directory (schema, context,
// agents and script references) into the immutable description the engine runs.
package model

// Model is the compiled, validated form of a model directory. It is immutable once
// loaded and safe to share between concurrent simulations.
type Model struct {
	Name        string
	Description string
	Dir         string

	MaxTicks int
	Stop     string // script path relative to Dir; empty when absent

	Context      []*Field
	ContextIndex map[string]*Field

	Types     []*AgentType
	TypeIndex map[string]*AgentType

	Relations     []*Relation
	RelationIndex map[string]*Relation

	GroupKinds []*GroupKind
	GroupIndex map[string]*GroupKind

	Space *Space

	Decisions     []*Decision
	DecisionIndex map[string]*Decision

	Rules   []*Rule
	Metrics []*Metric
	Output  Output

	Populations []*Population

	// Scripts lists every script referenced by the model, sorted.
	Scripts []string
}

// MergePolicy decides what happens when different actors set the same context
// value during the same phase.
type MergePolicy uint8

const (
	MergeError MergePolicy = iota // abort: two writers are a modeling bug
	MergeLast                     // the last write in the (shuffled) apply order wins
	MergeFirst                    // the first write wins; later ones are rejected
)

// Field is a context field or an agent attribute.
type Field struct {
	Name        string
	Description string
	Kind        Kind
	IsMap       bool // context only: a string-keyed map of Kind values
	Default     Value
	DefaultMap  map[string]Value
	Min, Max    *float64
	Clamp       bool
	Enum        []string
	Public      bool
	Merge       MergePolicy

	// Slot is the column index among attributes of the same kind (agent attributes)
	// or the field index (context fields).
	Slot int
	// Owner is the agent type declaring the attribute; nil for context fields.
	Owner *AgentType
}

// Memory is a bounded, per-agent history of values (a ring buffer).
type Memory struct {
	Name        string
	Description string
	Kind        Kind
	Capacity    int
	Slot        int
}

type Activation struct {
	Every          int
	Probability    float64
	HasProbability bool
}

type ChoiceMode uint8

const (
	ChoiceWeighted ChoiceMode = iota // roulette selection proportional to propensity
	ChoiceSoftmax                    // roulette over exp(propensity / temperature)
	ChoiceMax                        // highest propensity, ties broken at random
)

type Choice struct {
	Mode        ChoiceMode
	Temperature float64
	HasIdle     bool
	IdleConst   float64
	IdleExpr    string
}

type AgentType struct {
	Name        string
	Description string
	Index       int

	Attrs     []*Field
	AttrIndex map[string]*Field
	// KindCount is the number of attributes per Kind (the number of columns).
	KindCount [kindCount]int

	Memories    []*Memory
	MemoryIndex map[string]*Memory

	Decisions  []*Decision
	Activation Activation
	Choice     Choice
	Spatial    bool
}

type Relation struct {
	Name     string
	Index    int
	Directed bool
}

type GroupKind struct {
	Name  string
	Index int
}

type Space struct {
	Width, Height, CellSize float64
}

type Decision struct {
	Name        string
	Description string
	Index       int
	Script      string
	Eligible    string // optional expression
	Propensity  string // optional expression
}

type Phase uint8

const (
	PhaseInit Phase = iota // once, before the first tick
	PhasePre               // every tick, before decisions
	PhasePost              // every tick, after decisions
)

func (p Phase) String() string { return [...]string{"init", "pre", "post"}[p] }

type Rule struct {
	Name   string
	Index  int
	Script string
	Phase  Phase
	Every  int
	// Type is set for agent-scoped rules, which run once per agent of that type.
	Type *AgentType
}

type AggKind uint8

const (
	AggCount AggKind = iota
	AggSum
	AggMean
	AggMin
	AggMax
	AggStd
	AggGini
)

var aggNames = map[string]AggKind{
	"count": AggCount, "sum": AggSum, "mean": AggMean, "min": AggMin,
	"max": AggMax, "std": AggStd, "gini": AggGini,
}

// ParseAgg converts an aggregate name into an AggKind.
func ParseAgg(s string) (AggKind, bool) {
	k, ok := aggNames[s]
	return k, ok
}

type MetricSource uint8

const (
	MetricAggregate MetricSource = iota
	MetricContext
	MetricAgent
	MetricScript
)

type Metric struct {
	Name    string
	Source  MetricSource
	Agg     AggKind
	Type    *AgentType
	Attr    *Field
	Context *Field
	Agent   string
	Script  string
}

type LogLevel uint8

const (
	LogNone      LogLevel = iota
	LogEvents             // emits, spawns, removals
	LogDecisions          // + every decision with propensities and outcome
	LogEffects            // + every applied effect
)

// ParseLogLevel converts a log level name into a LogLevel.
func ParseLogLevel(s string) (LogLevel, bool) {
	switch s {
	case "none":
		return LogNone, true
	case "events":
		return LogEvents, true
	case "decisions":
		return LogDecisions, true
	case "effects":
		return LogEffects, true
	}
	return LogNone, false
}

type Output struct {
	Log           LogLevel
	Sample        float64
	SnapshotEvery int
	SnapshotFinal bool
}

// Population is one agent file: a single named agent or a generated population.
type Population struct {
	File   string
	Type   *AgentType
	Name   string
	Count  int
	Attrs  []AttrGen
	X, Y   *Generator
	Groups []GroupGen
	Links  []Link
}

type AttrGen struct {
	Field *Field
	Gen   Generator
}

type GroupGen struct {
	Kind *GroupKind
	Gen  Generator
}

type Link struct {
	Relation *Relation
	To       string
}
