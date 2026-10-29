package engine

import (
	"fmt"
	"math"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/rng"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/script"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/world"
)

// runRules runs every rule of a phase, in declaration order. Each rule sees the
// effects of the rules before it. Rules are the model's authority: a rule whose
// effects are rejected is a modeling error and stops the run.
func (e *Engine) runRules(phase model.Phase) error {
	for _, r := range e.m.Rules {
		if r.Phase != phase || (phase != model.PhaseInit && e.w.Tick%r.Every != 0) {
			continue
		}
		e.w.BeginPhase()
		if r.Type == nil {
			src := e.workers[0].src
			src.Reset(e.opt.Seed, rng.StreamRule, uint64(e.w.Tick), uint64(r.Index))
			res, err := e.runtimes[0].CallWorld(e.ruleFns[r.Index], src)
			if err != nil {
				return fmt.Errorf("tick %d, rule %q: %w", e.w.Tick, r.Name, err)
			}
			if err := e.applyRule(r, 0, res); err != nil {
				return err
			}
			continue
		}
		if err := e.runAgentRule(r); err != nil {
			return err
		}
	}
	return e.err
}

// runAgentRule calls run(self, world) for every agent of the rule's type in
// parallel, then applies the results in ID order.
func (e *Engine) runAgentRule(r *model.Rule) error {
	st := e.w.Stores[r.Type.Index]
	ids := append([]world.AgentID(nil), st.IDs...)
	results := make([]script.Result, len(ids))
	err := e.parallel(len(ids), func(wk, i int) error {
		w := e.workers[wk]
		w.src.Reset(e.opt.Seed, rng.StreamRule, uint64(e.w.Tick), uint64(ids[i])^uint64(r.Index)<<48)
		res, err := w.rt.CallAgent(e.ruleFns[r.Index], ids[i], w.src, true)
		if err != nil {
			return fmt.Errorf("tick %d, rule %q, agent %d: %w", e.w.Tick, r.Name, ids[i], err)
		}
		results[i] = res
		return nil
	})
	if err != nil {
		return err
	}
	for i, res := range results {
		if err := e.applyRule(r, ids[i], res); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) applyRule(r *model.Rule, id world.AgentID, res script.Result) error {
	if res.Kind == script.ResultNil {
		return nil
	}
	if res.Kind != script.ResultEffects {
		return fmt.Errorf("tick %d, rule %q: run() must return effects (fx.*) or nil", e.w.Tick, r.Name)
	}
	reason, err := e.w.Apply(world.Actor{ID: id, Rule: r}, res.Effects)
	if err != nil {
		return fmt.Errorf("tick %d, rule %q: %w", e.w.Tick, r.Name, err)
	}
	if reason != "" {
		return fmt.Errorf("tick %d, rule %q: effects rejected: %s (rules must only produce valid effects; check the state before changing it)", e.w.Tick, r.Name, reason)
	}
	return nil
}

func (e *Engine) buildMetricNames() {
	for _, mt := range e.m.Metrics {
		e.metricNames = append(e.metricNames, mt.Name)
	}
	for _, t := range e.m.Types {
		e.metricNames = append(e.metricNames, "population."+t.Name)
	}
	for _, t := range e.m.Types {
		for _, d := range t.Decisions {
			prefix := "decision." + t.Name + "." + d.Name + "."
			e.metricNames = append(e.metricNames, prefix+"chosen", prefix+"success", prefix+"failure", prefix+"infeasible", prefix+"rejected")
		}
	}
	e.metrics = make([]float64, len(e.metricNames))
}

func (e *Engine) recordMetrics() error {
	e.w.BeginPhase()
	vals := e.metrics[:0]
	for mi, mt := range e.m.Metrics {
		v := math.NaN()
		switch mt.Source {
		case model.MetricAggregate:
			v = e.w.Aggregate(mt.Agg, mt.Type, mt.Attr)
		case model.MetricContext:
			v = e.w.Context.Get(mt.Context).AsFloat()
		case model.MetricAgent:
			if id, ok := e.w.Find(mt.Agent); ok {
				if f := e.w.TypeOf(id).AttrIndex[mt.Attr.Name]; f != nil {
					val, _ := e.w.Get(id, f)
					v = val.AsFloat()
				}
			}
		case model.MetricScript:
			src := e.workers[0].src
			src.Reset(e.opt.Seed, rng.StreamMetric, uint64(e.w.Tick), 0)
			res, err := e.runtimes[0].CallWorld(e.metricFns[mi], src)
			if err != nil {
				return fmt.Errorf("tick %d, metric %q: %w", e.w.Tick, mt.Name, err)
			}
			switch res.Kind {
			case script.ResultNumber:
				v = res.Number
			case script.ResultBool:
				v = 0
				if res.Bool {
					v = 1
				}
			case script.ResultEffects:
				return fmt.Errorf("tick %d, metric %q: metric() must return a number", e.w.Tick, mt.Name)
			}
		}
		vals = append(vals, v)
	}
	for _, st := range e.w.Stores {
		vals = append(vals, float64(st.Len()))
	}
	for ti, cs := range e.counters {
		for di := range cs {
			c := &e.counters[ti][di]
			vals = append(vals, float64(c.chosen), float64(c.success), float64(c.failure), float64(c.infeasible), float64(c.rejected))
			*c = decisionCounters{}
		}
	}
	return e.sink.Metrics(e.w.Tick, vals)
}
