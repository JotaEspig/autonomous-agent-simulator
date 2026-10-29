package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Errors collects every problem found while loading a model, so a modeler can fix
// them all in one pass instead of one per run.
type Errors struct {
	list []string
}

func (e *Errors) Addf(file, path, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	switch {
	case file != "" && path != "":
		msg = file + ": " + path + ": " + msg
	case file != "":
		msg = file + ": " + msg
	}
	e.list = append(e.list, msg)
}

func (e *Errors) Len() int { return len(e.list) }

func (e *Errors) Error() string {
	return fmt.Sprintf("model has %d error(s):\n  - %s", len(e.list), strings.Join(e.list, "\n  - "))
}

func (e *Errors) err() error {
	if len(e.list) == 0 {
		return nil
	}
	return e
}

var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// reserved names cannot be used as attributes: they collide with the agent proxy
// exposed to scripts and with expression identifiers.
var reserved = map[string]bool{
	"id": true, "type": true, "name": true, "x": true, "y": true,
	"last": true, "recall": true, "memlen": true, "count": true, "mean": true,
	"ctx": true, "tick": true, "sigmoid": true, "clamp": true, "exp": true,
	"log": true, "sqrt": true, "pow": true,
}

type files struct {
	schema  string
	context string
	agents  []string
}

func discover(dir string) (files, error) {
	var f files
	var schemas, contexts []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != dir && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		switch name := d.Name(); {
		case strings.HasSuffix(name, ".schema.json"):
			schemas = append(schemas, path)
		case strings.HasSuffix(name, ".context.json"):
			contexts = append(contexts, path)
		case strings.HasSuffix(name, ".agent.json"):
			f.agents = append(f.agents, path)
		}
		return nil
	})
	if err != nil {
		return f, err
	}
	if len(schemas) != 1 {
		return f, fmt.Errorf("%s: expected exactly one *.schema.json file, found %d", dir, len(schemas))
	}
	if len(contexts) > 1 {
		return f, fmt.Errorf("%s: expected at most one *.context.json file, found %d", dir, len(contexts))
	}
	f.schema = schemas[0]
	if len(contexts) == 1 {
		f.context = contexts[0]
	}
	sort.Strings(f.agents)
	return f, nil
}

func decodeStrict(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	dec.UseNumber()
	if err := dec.Decode(v); err != nil {
		var syn *json.SyntaxError
		if errors.As(err, &syn) {
			line := 1 + bytes.Count(data[:syn.Offset], []byte("\n"))
			return fmt.Errorf("line %d: %v", line, err)
		}
		return err
	}
	return nil
}

// Load reads and validates the model in dir.
func Load(dir string) (*Model, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	fl, err := discover(abs)
	if err != nil {
		return nil, err
	}
	l := &loader{dir: abs, errs: &Errors{}, scripts: map[string]bool{}}

	var spec SchemaSpec
	if err := decodeStrict(fl.schema, &spec); err != nil {
		return nil, fmt.Errorf("%s: %v", l.rel(fl.schema), err)
	}
	m := l.compileSchema(l.rel(fl.schema), &spec)
	if fl.context != "" {
		l.loadContext(m, fl.context)
	}
	for _, p := range fl.agents {
		l.loadAgent(m, p)
	}
	l.resolveLinks(m)
	for s := range l.scripts {
		m.Scripts = append(m.Scripts, s)
	}
	sort.Strings(m.Scripts)
	return m, l.errs.err()
}

type loader struct {
	dir     string
	errs    *Errors
	scripts map[string]bool
}

func (l *loader) rel(p string) string {
	if r, err := filepath.Rel(l.dir, p); err == nil {
		return r
	}
	return p
}

func (l *loader) script(file, path, ref string) string {
	if ref == "" {
		l.errs.Addf(file, path, "a script path is required")
		return ""
	}
	clean := filepath.Clean(filepath.FromSlash(ref))
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		l.errs.Addf(file, path, "script %q must be a relative path inside the model directory", ref)
		return ""
	}
	if _, err := os.Stat(filepath.Join(l.dir, clean)); err != nil {
		l.errs.Addf(file, path, "script %q not found", ref)
		return ""
	}
	l.scripts[clean] = true
	return clean
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (l *loader) compileSchema(file string, s *SchemaSpec) *Model {
	m := &Model{
		Name: s.Name, Description: s.Description, Dir: l.dir,
		ContextIndex: map[string]*Field{}, TypeIndex: map[string]*AgentType{},
		RelationIndex: map[string]*Relation{}, GroupIndex: map[string]*GroupKind{},
		DecisionIndex: map[string]*Decision{},
	}
	if m.Name == "" {
		l.errs.Addf(file, "name", "the model needs a name")
	}
	m.MaxTicks = s.Simulation.MaxTicks
	if m.MaxTicks <= 0 {
		l.errs.Addf(file, "simulation.maxTicks", "must be > 0 (got %d)", m.MaxTicks)
	}
	if s.Simulation.Stop != "" {
		m.Stop = l.script(file, "simulation.stop", s.Simulation.Stop)
	}

	for _, name := range sortedKeys(s.Context) {
		path := "context." + name
		f := l.compileField(file, path, name, s.Context[name], nil)
		f.Slot = len(m.Context)
		m.Context = append(m.Context, f)
		m.ContextIndex[name] = f
	}

	for _, name := range sortedKeys(s.Relations) {
		if !identRe.MatchString(name) {
			l.errs.Addf(file, "relations."+name, "relation names must be identifiers")
		}
		r := &Relation{Name: name, Index: len(m.Relations), Directed: s.Relations[name].Directed}
		m.Relations = append(m.Relations, r)
		m.RelationIndex[name] = r
	}
	for _, name := range sortedKeys(s.Groups) {
		if !identRe.MatchString(name) {
			l.errs.Addf(file, "groups."+name, "group kind names must be identifiers")
		}
		g := &GroupKind{Name: name, Index: len(m.GroupKinds)}
		m.GroupKinds = append(m.GroupKinds, g)
		m.GroupIndex[name] = g
	}
	if sp := s.Space; sp != nil {
		if sp.Width <= 0 || sp.Height <= 0 {
			l.errs.Addf(file, "space", "width and height must be > 0")
		}
		cell := sp.CellSize
		if cell <= 0 {
			cell = math.Max(sp.Width, sp.Height) / 100
		}
		m.Space = &Space{Width: sp.Width, Height: sp.Height, CellSize: cell}
	}

	for _, name := range sortedKeys(s.Decisions) {
		ds := s.Decisions[name]
		path := "decisions." + name
		if !identRe.MatchString(name) {
			l.errs.Addf(file, path, "decision names must be identifiers")
		}
		d := &Decision{
			Name: name, Description: ds.Description, Index: len(m.Decisions),
			Script: l.script(file, path+".script", ds.Script), Eligible: ds.Eligible, Propensity: ds.Propensity,
		}
		m.Decisions = append(m.Decisions, d)
		m.DecisionIndex[name] = d
	}

	if len(s.AgentTypes) == 0 {
		l.errs.Addf(file, "agentTypes", "declare at least one agent type")
	}
	for _, name := range sortedKeys(s.AgentTypes) {
		t := l.compileType(file, m, name, s.AgentTypes[name])
		t.Index = len(m.Types)
		m.Types = append(m.Types, t)
		m.TypeIndex[name] = t
	}

	for i, rs := range s.Rules {
		path := fmt.Sprintf("rules[%d]", i)
		r := &Rule{Name: rs.Name, Index: i, Script: l.script(file, path+".script", rs.Script), Every: rs.Every}
		if r.Name == "" {
			r.Name = strings.TrimSuffix(filepath.Base(rs.Script), filepath.Ext(rs.Script))
		}
		switch rs.Phase {
		case "", "post":
			r.Phase = PhasePost
		case "pre":
			r.Phase = PhasePre
		case "init":
			r.Phase = PhaseInit
		default:
			l.errs.Addf(file, path+".phase", "unknown phase %q (expected init, pre or post)", rs.Phase)
		}
		if r.Every == 0 {
			r.Every = 1
		} else if r.Every < 0 {
			l.errs.Addf(file, path+".every", "must be >= 1")
		}
		if rs.AgentType != "" {
			if r.Type = m.TypeIndex[rs.AgentType]; r.Type == nil {
				l.errs.Addf(file, path+".agentType", "unknown agent type %q", rs.AgentType)
			}
		}
		m.Rules = append(m.Rules, r)
	}

	seen := map[string]bool{}
	for i, ms := range s.Metrics {
		m.Metrics = append(m.Metrics, l.compileMetric(file, fmt.Sprintf("metrics[%d]", i), m, ms, seen))
	}

	m.Output = Output{Log: LogEvents, Sample: 1, SnapshotEvery: s.Output.SnapshotEvery, SnapshotFinal: true}
	if s.Output.Log != "" {
		lvl, ok := ParseLogLevel(s.Output.Log)
		if !ok {
			l.errs.Addf(file, "output.log", "unknown level %q (expected none, events, decisions or effects)", s.Output.Log)
		}
		m.Output.Log = lvl
	}
	if s.Output.Sample != nil {
		if *s.Output.Sample < 0 || *s.Output.Sample > 1 {
			l.errs.Addf(file, "output.sample", "must be in [0, 1]")
		}
		m.Output.Sample = *s.Output.Sample
	}
	if s.Output.SnapshotFinal != nil {
		m.Output.SnapshotFinal = *s.Output.SnapshotFinal
	}
	return m
}

func (l *loader) compileField(file, path, name string, s FieldSpec, owner *AgentType) *Field {
	f := &Field{Name: name, Description: s.Description, Owner: owner, Min: s.Min, Max: s.Max, Clamp: s.Clamp, Enum: s.Values}
	isContext := owner == nil
	if !identRe.MatchString(name) {
		l.errs.Addf(file, path, "names must be identifiers (letters, digits and _)")
	}
	if !isContext && reserved[name] {
		l.errs.Addf(file, path, "%q is reserved; choose another attribute name", name)
	}
	typeName := s.Type
	if s.Type == "map" {
		if !isContext {
			l.errs.Addf(file, path+".type", "map attributes are only allowed in the context; use memory for per-agent histories")
		}
		f.IsMap = true
		typeName = s.Of
	} else if s.Of != "" {
		l.errs.Addf(file, path+".of", "\"of\" only applies to \"type\": \"map\"")
	}
	k, ok := ParseKind(typeName)
	if !ok {
		l.errs.Addf(file, path+".type", "unknown type %q (expected int, float, bool, string, or map with \"of\")", typeName)
		k = KindFloat
	}
	f.Kind = k
	if (f.Min != nil || f.Max != nil || f.Clamp) && !k.Numeric() {
		l.errs.Addf(file, path, "min, max and clamp only apply to numeric fields")
	}
	if f.Min != nil && f.Max != nil && *f.Min > *f.Max {
		l.errs.Addf(file, path, "min must be <= max")
	}
	if len(f.Enum) > 0 && k != KindString {
		l.errs.Addf(file, path+".values", "values only apply to string fields")
	}

	switch s.Visibility {
	case "":
		f.Public = isContext
	case "public":
		f.Public = true
	case "private":
		if isContext {
			l.errs.Addf(file, path+".visibility", "context fields are \"public\" or \"hidden\"")
		}
	case "hidden":
		if !isContext {
			l.errs.Addf(file, path+".visibility", "agent attributes are \"public\" or \"private\"")
		}
	default:
		l.errs.Addf(file, path+".visibility", "unknown visibility %q", s.Visibility)
	}

	switch s.Merge {
	case "", "error":
		f.Merge = MergeError
	case "last":
		f.Merge = MergeLast
	case "first":
		f.Merge = MergeFirst
	default:
		l.errs.Addf(file, path+".merge", "unknown merge policy %q (expected error, last or first)", s.Merge)
	}
	if s.Merge != "" && !isContext {
		l.errs.Addf(file, path+".merge", "merge policies only apply to context fields")
	}

	if f.IsMap {
		f.DefaultMap = map[string]Value{}
		if s.Default != nil {
			obj, ok := s.Default.(map[string]any)
			if !ok {
				l.errs.Addf(file, path+".default", "a map default must be an object")
			}
			for key, raw := range obj {
				if v, err := l.checkValue(f, raw); err != nil {
					l.errs.Addf(file, path+".default."+key, "%v", err)
				} else {
					f.DefaultMap[key] = v
				}
			}
		}
		return f
	}
	f.Default = Zero(k)
	if k == KindString && len(f.Enum) > 0 {
		f.Default = String(f.Enum[0])
	}
	if s.Default != nil {
		if v, err := l.checkValue(f, s.Default); err != nil {
			l.errs.Addf(file, path+".default", "%v", err)
		} else {
			f.Default = v
		}
	} else if k.Numeric() && f.Min != nil && f.Default.AsFloat() < *f.Min {
		f.Default, _ = Coerce(Float(*f.Min), k)
	}
	return f
}

// CheckConstraints validates v (already of the field's kind) against enum, min and max.
func (f *Field) CheckConstraints(v Value) error {
	if len(f.Enum) > 0 {
		for _, e := range f.Enum {
			if v.S == e {
				return nil
			}
		}
		return fmt.Errorf("%s is not one of %v", v, f.Enum)
	}
	if f.Kind.Numeric() {
		x := v.AsFloat()
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return fmt.Errorf("%s is not a finite number", v)
		}
		if f.Min != nil && x < *f.Min {
			return fmt.Errorf("%s is below the minimum %g", v, *f.Min)
		}
		if f.Max != nil && x > *f.Max {
			return fmt.Errorf("%s is above the maximum %g", v, *f.Max)
		}
	}
	return nil
}

func (l *loader) checkValue(f *Field, raw any) (Value, error) {
	v, err := FromJSON(raw)
	if err != nil {
		return v, err
	}
	if v, err = Coerce(v, f.Kind); err != nil {
		return v, err
	}
	return v, f.CheckConstraints(v)
}

func (l *loader) compileType(file string, m *Model, name string, s AgentTypeSpec) *AgentType {
	path := "agentTypes." + name
	t := &AgentType{Name: name, Description: s.Description, AttrIndex: map[string]*Field{}, MemoryIndex: map[string]*Memory{}, Spatial: s.Spatial}
	if !identRe.MatchString(name) {
		l.errs.Addf(file, path, "agent type names must be identifiers")
	}
	for _, an := range sortedKeys(s.Attributes) {
		f := l.compileField(file, path+".attributes."+an, an, s.Attributes[an], t)
		f.Slot = t.KindCount[f.Kind]
		t.KindCount[f.Kind]++
		t.Attrs = append(t.Attrs, f)
		t.AttrIndex[an] = f
	}
	for _, mn := range sortedKeys(s.Memory) {
		ms := s.Memory[mn]
		mp := path + ".memory." + mn
		if !identRe.MatchString(mn) {
			l.errs.Addf(file, mp, "memory names must be identifiers")
		}
		k, ok := ParseKind(ms.Type)
		if !ok {
			l.errs.Addf(file, mp+".type", "unknown type %q (expected int, float, bool or string)", ms.Type)
		}
		if ms.Capacity <= 0 {
			l.errs.Addf(file, mp+".capacity", "must be > 0: memories are bounded so long runs keep a fixed footprint")
		}
		mem := &Memory{Name: mn, Description: ms.Description, Kind: k, Capacity: ms.Capacity, Slot: len(t.Memories)}
		t.Memories = append(t.Memories, mem)
		t.MemoryIndex[mn] = mem
	}
	seen := map[string]bool{}
	for i, dn := range s.Decisions {
		d := m.DecisionIndex[dn]
		if d == nil {
			l.errs.Addf(file, fmt.Sprintf("%s.decisions[%d]", path, i), "unknown decision %q", dn)
			continue
		}
		if seen[dn] {
			l.errs.Addf(file, fmt.Sprintf("%s.decisions[%d]", path, i), "decision %q listed twice", dn)
		}
		seen[dn] = true
		t.Decisions = append(t.Decisions, d)
	}

	a := s.Activation
	t.Activation.Every = a.Every
	if a.Every == 0 {
		t.Activation.Every = 1
	} else if a.Every < 0 {
		l.errs.Addf(file, path+".activation.every", "must be >= 1")
	}
	if a.Probability != nil {
		if *a.Probability < 0 || *a.Probability > 1 {
			l.errs.Addf(file, path+".activation.probability", "must be in [0, 1]")
		}
		t.Activation.Probability, t.Activation.HasProbability = *a.Probability, true
	}

	c := s.Choice
	switch c.Mode {
	case "", "weighted":
		t.Choice.Mode = ChoiceWeighted
	case "softmax":
		t.Choice.Mode = ChoiceSoftmax
		t.Choice.Temperature = c.Temperature
		if c.Temperature <= 0 {
			l.errs.Addf(file, path+".choice.temperature", "softmax requires a temperature > 0")
		}
	case "max":
		t.Choice.Mode = ChoiceMax
	default:
		l.errs.Addf(file, path+".choice.mode", "unknown mode %q (expected weighted, softmax or max)", c.Mode)
	}
	switch idle := c.Idle.(type) {
	case nil:
	case string:
		t.Choice.HasIdle, t.Choice.IdleExpr = true, idle
	default:
		v, err := FromJSON(idle)
		if err != nil || !v.Kind.Numeric() || v.AsFloat() < 0 {
			l.errs.Addf(file, path+".choice.idle", "must be a number >= 0 or an expression string")
		}
		t.Choice.HasIdle, t.Choice.IdleConst = true, v.AsFloat()
	}
	if s.Spatial && m.Space == nil {
		l.errs.Addf(file, path+".spatial", "spatial agent types require a top-level \"space\" declaration")
	}
	return t
}

func (l *loader) compileMetric(file, path string, m *Model, s MetricSpec, seen map[string]bool) *Metric {
	mt := &Metric{Name: s.Name}
	if s.Name == "" {
		l.errs.Addf(file, path+".name", "metrics need a name")
	} else if seen[s.Name] {
		l.errs.Addf(file, path+".name", "duplicate metric name %q", s.Name)
	}
	seen[s.Name] = true
	sources := 0
	for _, set := range []bool{s.Agg != "", s.Context != "", s.Agent != "", s.Script != ""} {
		if set {
			sources++
		}
	}
	if sources != 1 {
		l.errs.Addf(file, path, "set exactly one of \"agg\", \"context\", \"agent\" or \"script\"")
		return mt
	}
	switch {
	case s.Agg != "":
		mt.Source = MetricAggregate
		agg, ok := ParseAgg(s.Agg)
		if !ok {
			l.errs.Addf(file, path+".agg", "unknown aggregate %q (expected count, sum, mean, min, max, std or gini)", s.Agg)
		}
		mt.Agg = agg
		if mt.Type = m.TypeIndex[s.AgentType]; mt.Type == nil {
			l.errs.Addf(file, path+".agentType", "unknown agent type %q", s.AgentType)
			return mt
		}
		if agg != AggCount {
			mt.Attr = mt.Type.AttrIndex[s.Attribute]
			if mt.Attr == nil {
				l.errs.Addf(file, path+".attribute", "agent type %q has no attribute %q", s.AgentType, s.Attribute)
			} else if mt.Attr.Kind == KindString {
				l.errs.Addf(file, path+".attribute", "cannot aggregate string attribute %q", s.Attribute)
			}
		}
	case s.Context != "":
		mt.Source = MetricContext
		if mt.Context = m.ContextIndex[s.Context]; mt.Context == nil {
			l.errs.Addf(file, path+".context", "unknown context field %q", s.Context)
		} else if mt.Context.IsMap || mt.Context.Kind == KindString {
			l.errs.Addf(file, path+".context", "context metrics need a numeric or bool scalar field")
		}
	case s.Agent != "":
		mt.Source = MetricAgent
		mt.Agent = s.Agent
		if s.Attribute == "" {
			l.errs.Addf(file, path+".attribute", "agent metrics need an attribute")
		}
		mt.Attr = &Field{Name: s.Attribute}
	case s.Script != "":
		mt.Source = MetricScript
		mt.Script = l.script(file, path+".script", s.Script)
	}
	return mt
}

func (l *loader) loadContext(m *Model, path string) {
	file := l.rel(path)
	var raw map[string]any
	if err := decodeStrict(path, &raw); err != nil {
		l.errs.Addf(file, "", "%v", err)
		return
	}
	for _, key := range sortedKeys(raw) {
		if key == "$schema" || key == "description" {
			continue
		}
		f := m.ContextIndex[key]
		if f == nil {
			l.errs.Addf(file, key, "unknown context field (declare it under \"context\" in the schema)")
			continue
		}
		if f.IsMap {
			obj, ok := raw[key].(map[string]any)
			if !ok {
				l.errs.Addf(file, key, "expected an object for map field")
				continue
			}
			f.DefaultMap = map[string]Value{}
			for mk, mv := range obj {
				v, err := l.checkValue(f, mv)
				if err != nil {
					l.errs.Addf(file, key+"."+mk, "%v", err)
					continue
				}
				f.DefaultMap[mk] = v
			}
			continue
		}
		v, err := l.checkValue(f, raw[key])
		if err != nil {
			l.errs.Addf(file, key, "%v", err)
			continue
		}
		f.Default = v
	}
}

func (l *loader) loadAgent(m *Model, path string) {
	file := l.rel(path)
	var s AgentFileSpec
	if err := decodeStrict(path, &s); err != nil {
		l.errs.Addf(file, "", "%v", err)
		return
	}
	t := m.TypeIndex[s.Type]
	if t == nil {
		l.errs.Addf(file, "type", "unknown agent type %q", s.Type)
		return
	}
	p := &Population{File: file, Type: t, Name: s.Name, Count: s.Count}
	if p.Count == 0 {
		p.Count = 1
	} else if p.Count < 0 {
		l.errs.Addf(file, "count", "must be >= 1")
	}
	for _, an := range sortedKeys(s.Attributes) {
		f := t.AttrIndex[an]
		if f == nil {
			l.errs.Addf(file, "attributes."+an, "agent type %q has no attribute %q", t.Name, an)
			continue
		}
		g, err := ParseGenerator(s.Attributes[an], f.Kind)
		if err == nil && g.Dist == "" {
			err = f.CheckConstraints(g.Const)
		}
		if err != nil {
			l.errs.Addf(file, "attributes."+an, "%v", err)
			continue
		}
		p.Attrs = append(p.Attrs, AttrGen{Field: f, Gen: g})
	}
	if len(s.Position) > 0 {
		if !t.Spatial {
			l.errs.Addf(file, "position", "agent type %q is not spatial", t.Name)
		}
		for key, raw := range s.Position {
			g, err := ParseGenerator(raw, KindFloat)
			if err != nil {
				l.errs.Addf(file, "position."+key, "%v", err)
				continue
			}
			switch key {
			case "x":
				p.X = &g
			case "y":
				p.Y = &g
			default:
				l.errs.Addf(file, "position."+key, "expected \"x\" or \"y\"")
			}
		}
	}
	for _, kn := range sortedKeys(s.Groups) {
		k := m.GroupIndex[kn]
		if k == nil {
			l.errs.Addf(file, "groups."+kn, "unknown group kind %q", kn)
			continue
		}
		g, err := ParseGenerator(s.Groups[kn], KindString)
		if err != nil {
			l.errs.Addf(file, "groups."+kn, "%v", err)
			continue
		}
		p.Groups = append(p.Groups, GroupGen{Kind: k, Gen: g})
	}
	for i, ls := range s.Links {
		r := m.RelationIndex[ls.Relation]
		if r == nil {
			l.errs.Addf(file, fmt.Sprintf("links[%d].relation", i), "unknown relation %q", ls.Relation)
			continue
		}
		p.Links = append(p.Links, Link{Relation: r, To: ls.To})
	}
	if len(p.Links) > 0 && (p.Count != 1 || p.Name == "") {
		l.errs.Addf(file, "links", "links are only supported for a single named agent; use an init rule to wire populations")
	}
	m.Populations = append(m.Populations, p)
}

func (l *loader) resolveLinks(m *Model) {
	named := map[string]string{}
	for _, p := range m.Populations {
		if p.Count != 1 || p.Name == "" {
			continue
		}
		if prev, dup := named[p.Name]; dup {
			l.errs.Addf(p.File, "name", "agent name %q is already used by %s", p.Name, prev)
		}
		named[p.Name] = p.File
	}
	for _, p := range m.Populations {
		for i, ln := range p.Links {
			if _, ok := named[ln.To]; !ok {
				l.errs.Addf(p.File, fmt.Sprintf("links[%d].to", i), "no single agent named %q", ln.To)
			}
		}
	}
}
