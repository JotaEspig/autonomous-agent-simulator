package model

// This file mirrors the JSON files written by modelers. Unknown keys are rejected
// when decoding, so typos surface as load errors instead of silently ignored settings.
// Every object accepts a "description" key for documentation.

// SchemaSpec is the content of a <name>.schema.json file.
type SchemaSpec struct {
	Schema      string                   `json:"$schema"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Simulation  SimulationSpec           `json:"simulation"`
	Context     map[string]FieldSpec     `json:"context"`
	AgentTypes  map[string]AgentTypeSpec `json:"agentTypes"`
	Relations   map[string]RelationSpec  `json:"relations"`
	Groups      map[string]GroupSpec     `json:"groups"`
	Space       *SpaceSpec               `json:"space"`
	Decisions   map[string]DecisionSpec  `json:"decisions"`
	Rules       []RuleSpec               `json:"rules"`
	Metrics     []MetricSpec             `json:"metrics"`
	Output      OutputSpec               `json:"output"`
}

type SimulationSpec struct {
	Description string `json:"description"`
	MaxTicks    int    `json:"maxTicks"`
	Stop        string `json:"stop"`
}

// FieldSpec declares a context field or an agent attribute.
type FieldSpec struct {
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Of          string   `json:"of"`
	Default     any      `json:"default"`
	Min         *float64 `json:"min"`
	Max         *float64 `json:"max"`
	Clamp       bool     `json:"clamp"`
	Values      []string `json:"values"`
	Visibility  string   `json:"visibility"`
	Merge       string   `json:"merge"`
}

type MemorySpec struct {
	Description string `json:"description"`
	Type        string `json:"type"`
	Capacity    int    `json:"capacity"`
}

type ActivationSpec struct {
	Description string   `json:"description"`
	Every       int      `json:"every"`
	Probability *float64 `json:"probability"`
}

type ChoiceSpec struct {
	Description string  `json:"description"`
	Mode        string  `json:"mode"`
	Temperature float64 `json:"temperature"`
	// Idle is the weight of doing nothing: a number or an expression.
	Idle any `json:"idle"`
}

type AgentTypeSpec struct {
	Description string                `json:"description"`
	Attributes  map[string]FieldSpec  `json:"attributes"`
	Memory      map[string]MemorySpec `json:"memory"`
	Decisions   []string              `json:"decisions"`
	Activation  ActivationSpec        `json:"activation"`
	Choice      ChoiceSpec            `json:"choice"`
	Spatial     bool                  `json:"spatial"`
}

type RelationSpec struct {
	Description string `json:"description"`
	Directed    bool   `json:"directed"`
}

type GroupSpec struct {
	Description string `json:"description"`
}

type SpaceSpec struct {
	Description string  `json:"description"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	CellSize    float64 `json:"cellSize"`
}

type DecisionSpec struct {
	Description string `json:"description"`
	Script      string `json:"script"`
	Eligible    string `json:"eligible"`
	Propensity  string `json:"propensity"`
}

type RuleSpec struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Script      string `json:"script"`
	Phase       string `json:"phase"`
	Every       int    `json:"every"`
	AgentType   string `json:"agentType"`
}

type MetricSpec struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Agg         string `json:"agg"`
	AgentType   string `json:"agentType"`
	Attribute   string `json:"attribute"`
	Context     string `json:"context"`
	Agent       string `json:"agent"`
	Script      string `json:"script"`
}

type OutputSpec struct {
	Description   string   `json:"description"`
	Log           string   `json:"log"`
	Sample        *float64 `json:"sample"`
	SnapshotEvery int      `json:"snapshotEvery"`
	SnapshotFinal *bool    `json:"snapshotFinal"`
}

// AgentFileSpec is the content of a <name>.agent.json file. It describes one agent or,
// with count > 1, a population whose attributes may be drawn from distributions.
type AgentFileSpec struct {
	Schema      string         `json:"$schema"`
	Description string         `json:"description"`
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Count       int            `json:"count"`
	Attributes  map[string]any `json:"attributes"`
	Position    map[string]any `json:"position"`
	Groups      map[string]any `json:"groups"`
	Links       []LinkSpec     `json:"links"`
}

type LinkSpec struct {
	Relation string `json:"relation"`
	To       string `json:"to"`
}
