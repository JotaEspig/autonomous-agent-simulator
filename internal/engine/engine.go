// Package engine runs a model: it schedules agents, evaluates their decisions in
// parallel against an immutable snapshot, applies the resulting effects in a
// deterministic order, runs rules and records metrics.
package engine

import (
	"context"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/output"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/rng"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/script"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/world"
)

// Options configure one run. Zero values fall back to the model's settings.
type Options struct {
	Seed     uint64
	Workers  int
	MaxTicks int
	// Log, Sample and SnapshotEvery override the model's output settings when set.
	Log           *model.LogLevel
	Sample        *float64
	SnapshotEvery *int
}

// Engine is one simulation run. It is not reusable: create one per seed.
type Engine struct {
	m    *model.Model
	w    *world.World
	opt  Options
	sink output.Sink

	runtimes []script.Runtime
	workers  []*worker

	// per agent type and decision: optional compiled expressions, one per worker
	eligible   [][][]*script.Program // [worker][type] -> per decision
	propensity [][][]*script.Program
	idle       [][]*script.Program // [worker][type]

	hooks     []decisionHooks // indexed by Decision.Index
	ruleFns   []script.Func   // indexed by Rule.Index
	metricFns []script.Func   // indexed like Model.Metrics
	stopFn    script.Func

	metricNames []string
	metrics     []float64
	counters    [][]decisionCounters // [type][decision]

	log         model.LogLevel
	sample      float64
	snapEvery   int
	maxTicks    int
	ctxSnapshot map[string]any
	active      []actor
	proposals   []proposal
	order       []int
	err         error
	Warnings    []string
}

// decisionHooks are the resolved script functions of a decision (0 = undefined).
type decisionHooks struct {
	eligible, propensity, precondition, outcome, onSuccess, onFailure script.Func
}

type decisionCounters struct {
	chosen, success, failure, infeasible, rejected int
}

type worker struct {
	rt      script.Runtime
	ev      *script.Evaluator
	src     *rng.Source
	weights []float64
	undo    world.UndoBuffer
}

// Result describes a finished run.
type Result struct {
	Ticks      int
	StoppedBy  string // "maxTicks" or "stop"
	Elapsed    time.Duration
	AgentTicks int64 // number of agent activations processed
	Population int
}

// New builds the world, loads the scripts and checks that the model's scripts
// define what the schema says they do.
func New(m *model.Model, compiled *script.Compiled, opt Options) (*Engine, error) {
	if opt.Workers <= 0 {
		opt.Workers = runtime.GOMAXPROCS(0)
	}
	e := &Engine{m: m, opt: opt, log: m.Output.Log, sample: m.Output.Sample, snapEvery: m.Output.SnapshotEvery, maxTicks: m.MaxTicks}
	if opt.Log != nil {
		e.log = *opt.Log
	}
	if opt.Sample != nil {
		e.sample = *opt.Sample
	}
	if opt.SnapshotEvery != nil {
		e.snapEvery = *opt.SnapshotEvery
	}
	if opt.MaxTicks > 0 {
		e.maxTicks = opt.MaxTicks
	}
	e.w = world.New(m)
	for i := 0; i < opt.Workers; i++ {
		rt, err := compiled.NewRuntime(e.w)
		if err != nil {
			e.Close()
			return nil, err
		}
		e.runtimes = append(e.runtimes, rt)
		e.workers = append(e.workers, &worker{rt: rt, ev: script.NewEvaluator(e.w), src: rng.New()})
	}
	if err := e.compileExpressions(); err != nil {
		e.Close()
		return nil, err
	}
	if err := e.checkScripts(); err != nil {
		e.Close()
		return nil, err
	}
	e.resolveFunctions()
	e.buildMetricNames()
	e.counters = make([][]decisionCounters, len(m.Types))
	for _, t := range m.Types {
		e.counters[t.Index] = make([]decisionCounters, len(t.Decisions))
	}
	if err := e.w.Populate(opt.Seed); err != nil {
		e.Close()
		return nil, err
	}
	return e, nil
}

// World exposes the state, e.g. for tests and validation reports.
func (e *Engine) World() *world.World { return e.w }

// MetricNames lists the metrics columns: the model's metrics followed by
// automatic ones (population per type and decision counters).
func (e *Engine) MetricNames() []string { return e.metricNames }

func (e *Engine) Close() {
	for _, rt := range e.runtimes {
		rt.Close()
	}
}

func (e *Engine) compileExpressions() error {
	errs := &model.Errors{}
	nt := len(e.m.Types)
	e.eligible = make([][][]*script.Program, len(e.workers))
	e.propensity = make([][][]*script.Program, len(e.workers))
	e.idle = make([][]*script.Program, len(e.workers))
	for wi, wk := range e.workers {
		e.eligible[wi] = make([][]*script.Program, nt)
		e.propensity[wi] = make([][]*script.Program, nt)
		e.idle[wi] = make([]*script.Program, nt)
		for _, t := range e.m.Types {
			e.eligible[wi][t.Index] = make([]*script.Program, len(t.Decisions))
			e.propensity[wi][t.Index] = make([]*script.Program, len(t.Decisions))
			for di, d := range t.Decisions {
				where := fmt.Sprintf("decisions.%s (agent type %q)", d.Name, t.Name)
				if d.Eligible != "" {
					p, err := wk.ev.Compile(d.Eligible, t, true)
					if err != nil && wi == 0 {
						errs.Addf("", where+".eligible", "%v", err)
					}
					e.eligible[wi][t.Index][di] = p
				}
				if d.Propensity != "" {
					p, err := wk.ev.Compile(d.Propensity, t, false)
					if err != nil && wi == 0 {
						errs.Addf("", where+".propensity", "%v", err)
					}
					e.propensity[wi][t.Index][di] = p
				}
			}
			if t.Choice.IdleExpr != "" {
				p, err := wk.ev.Compile(t.Choice.IdleExpr, t, false)
				if err != nil && wi == 0 {
					errs.Addf("", fmt.Sprintf("agentTypes.%s.choice.idle", t.Name), "%v", err)
				}
				e.idle[wi][t.Index] = p
			}
		}
		if errs.Len() > 0 {
			return errs
		}
	}
	return nil
}

var decisionFns = []string{"eligible", "propensity", "precondition", "outcome", "on_success", "on_failure"}

func (e *Engine) resolveFunctions() {
	rt := e.runtimes[0]
	for _, d := range e.m.Decisions {
		e.hooks = append(e.hooks, decisionHooks{
			eligible:     rt.Lookup(d.Script, "eligible"),
			propensity:   rt.Lookup(d.Script, "propensity"),
			precondition: rt.Lookup(d.Script, "precondition"),
			outcome:      rt.Lookup(d.Script, "outcome"),
			onSuccess:    rt.Lookup(d.Script, "on_success"),
			onFailure:    rt.Lookup(d.Script, "on_failure"),
		})
	}
	for _, r := range e.m.Rules {
		e.ruleFns = append(e.ruleFns, rt.Lookup(r.Script, "run"))
	}
	for _, mt := range e.m.Metrics {
		var f script.Func
		if mt.Source == model.MetricScript {
			f = rt.Lookup(mt.Script, "metric")
		}
		e.metricFns = append(e.metricFns, f)
	}
	if e.m.Stop != "" {
		e.stopFn = rt.Lookup(e.m.Stop, "stop")
	}
}

func (e *Engine) checkScripts() error {
	rt := e.runtimes[0]
	has := func(script, fn string) bool { return rt.Lookup(script, fn) != 0 }
	errs := &model.Errors{}
	for _, d := range e.m.Decisions {
		fns := rt.Functions(d.Script)
		if !has(d.Script, "precondition") && !has(d.Script, "on_success") && !has(d.Script, "on_failure") {
			errs.Addf(d.Script, "", "decision %q: the script defines none of precondition, on_success or on_failure, so choosing it would change nothing", d.Name)
		}
		if d.Propensity != "" && has(d.Script, "propensity") {
			errs.Addf(d.Script, "", "decision %q: propensity is defined both as an expression in the schema and as a script function; keep one", d.Name)
		}
		if d.Eligible != "" && has(d.Script, "eligible") {
			errs.Addf(d.Script, "", "decision %q: eligible is defined both as an expression in the schema and as a script function; keep one", d.Name)
		}
		for _, f := range fns {
			if !slices.Contains(decisionFns, f) && looksLikeHook(f) {
				e.Warnings = append(e.Warnings, fmt.Sprintf("%s: function %q is not a decision hook (expected one of %s); it will only run if called by the script itself",
					d.Script, f, strings.Join(decisionFns, ", ")))
			}
		}
	}
	need := func(path, fn, what string) {
		if !has(path, fn) {
			errs.Addf(path, "", "%s scripts must define function %s", what, fn)
		}
	}
	for _, r := range e.m.Rules {
		need(r.Script, "run", "rule")
	}
	for _, mt := range e.m.Metrics {
		if mt.Source == model.MetricScript {
			need(mt.Script, "metric", "metric")
		}
	}
	if e.m.Stop != "" {
		need(e.m.Stop, "stop", "stop")
	}
	if errs.Len() > 0 {
		return errs
	}
	return nil
}

// looksLikeHook flags likely typos of hook names (e.g. "on_sucess", "propensty")
// without warning about every helper function.
func looksLikeHook(name string) bool {
	if strings.HasPrefix(name, "on_") {
		return true
	}
	for _, h := range decisionFns {
		if levenshtein(name, h) <= 2 {
			return true
		}
	}
	return false
}

func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// Run executes the simulation until maxTicks or the stop condition.
func (e *Engine) Run(ctx context.Context, sink output.Sink) (Result, error) {
	start := time.Now()
	e.sink = sink
	res := Result{StoppedBy: "maxTicks"}
	if e.log > model.LogNone {
		e.w.Hook = e.onEffect
	}
	e.w.Tick = 0
	if err := e.runRules(model.PhaseInit); err != nil {
		return res, err
	}
	if err := e.recordMetrics(); err != nil {
		return res, err
	}
	for tick := 1; tick <= e.maxTicks; tick++ {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		e.w.Tick = tick
		if err := e.runRules(model.PhasePre); err != nil {
			return res, err
		}
		n, err := e.decide()
		if err != nil {
			return res, err
		}
		res.AgentTicks += int64(n)
		if err := e.runRules(model.PhasePost); err != nil {
			return res, err
		}
		if err := e.recordMetrics(); err != nil {
			return res, err
		}
		res.Ticks = tick
		if e.snapEvery > 0 && tick%e.snapEvery == 0 {
			if err := sink.Snapshot(e.w); err != nil {
				return res, err
			}
		}
		if e.m.Stop != "" {
			e.w.BeginPhase()
			src := e.workers[0].src
			src.Reset(e.opt.Seed, rng.StreamStop, uint64(tick), 0)
			r, err := e.runtimes[0].CallWorld(e.stopFn, src)
			if err != nil {
				return res, err
			}
			if r.Truthy() {
				res.StoppedBy = "stop"
				break
			}
		}
	}
	if e.m.Output.SnapshotFinal && (e.snapEvery <= 0 || res.Ticks%e.snapEvery != 0) {
		if err := sink.Snapshot(e.w); err != nil {
			return res, err
		}
	}
	res.Elapsed = time.Since(start)
	res.Population = e.w.Population()
	return res, nil
}

// sampled decides, deterministically, whether an agent's decisions are logged.
func (e *Engine) sampled(id world.AgentID) bool {
	if e.sample >= 1 {
		return true
	}
	return float64(rng.Hash(e.opt.Seed, rng.StreamMetric, uint64(id))>>11)/(1<<53) < e.sample
}

func (e *Engine) emit(ev *output.Event) {
	if e.err != nil {
		return
	}
	if err := e.sink.Event(ev); err != nil {
		e.err = err
	}
}

func (e *Engine) onEffect(a world.Actor, eff *world.Effect) {
	if a.ID != 0 && !e.sampled(a.ID) {
		return
	}
	ev := &output.Event{Tick: e.w.Tick, Agent: uint64(a.ID), Name: e.w.Name(a.ID)}
	if a.Rule != nil {
		ev.Rule = a.Rule.Name
	}
	switch {
	case eff.Op == world.OpEmit:
		ev.Kind, ev.Event, ev.Data = "emit", eff.Name, eff.Data
	case eff.Op == world.OpSpawn || eff.Op == world.OpRemove:
		ev.Kind, ev.Effect = eff.Op.String(), eff.String()
	case e.log >= model.LogEffects:
		ev.Kind, ev.Effect = "effect", eff.String()
	default:
		return
	}
	e.emit(ev)
}
