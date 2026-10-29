package engine

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/output"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/rng"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/script"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/world"
)

type actor struct {
	id  world.AgentID
	typ int32
	idx int32
}

// proposal is what one agent decided during the read phase; it is applied later.
type proposal struct {
	decision   *model.Decision // nil when idle
	dindex     int
	infeasible bool
	reason     string
	success    bool
	pre        []*world.Effect
	post       []*world.Effect
	weights    map[string]float64 // only kept for logged agents
	local      bool               // applied in the parallel pass
	outcome    string
}

// parallel runs fn(worker, i) for i in [0, n), splitting work in chunks between
// workers. Results must be written to index i, so output order never depends on
// scheduling.
func (e *Engine) parallel(n int, fn func(wk int, i int) error) error {
	if n == 0 {
		return nil
	}
	nw := min(len(e.workers), (n+255)/256)
	if nw <= 1 {
		for i := 0; i < n; i++ {
			if err := fn(0, i); err != nil {
				return err
			}
		}
		return nil
	}
	const chunk = 256
	var next atomic.Int64
	var failed atomic.Bool
	errs := make([]error, nw)
	var wg sync.WaitGroup
	for wk := 0; wk < nw; wk++ {
		wg.Add(1)
		go func(wk int) {
			defer wg.Done()
			for !failed.Load() {
				lo := int(next.Add(chunk)) - chunk
				if lo >= n {
					return
				}
				for i := lo; i < min(lo+chunk, n); i++ {
					if err := fn(wk, i); err != nil {
						errs[wk] = err
						failed.Store(true)
						return
					}
				}
			}
		}(wk)
	}
	wg.Wait()
	return errors.Join(errs...)
}

// schedule lists the agents that act this tick, in type then index order.
func (e *Engine) schedule() {
	e.active = e.active[:0]
	src := e.workers[0].src
	tick := uint64(e.w.Tick)
	for ti, st := range e.w.Stores {
		act := st.Type.Activation
		for i, id := range st.IDs {
			if act.Every > 1 && (tick+uint64(id))%uint64(act.Every) != 0 {
				continue
			}
			if act.HasProbability {
				src.Reset(e.opt.Seed, rng.StreamActivation, tick, uint64(id))
				if src.Float() >= act.Probability {
					continue
				}
			}
			e.active = append(e.active, actor{id: id, typ: int32(ti), idx: int32(i)})
		}
	}
}

// decide runs the decision phase: every active agent chooses on the same snapshot
// (in parallel), then the proposals are applied one by one in a seeded random
// order, so no agent is systematically first in line for scarce resources.
func (e *Engine) decide() (int, error) {
	e.w.BeginPhase()
	e.ctxSnapshot = script.ContextSnapshot(e.w)
	e.schedule()
	n := len(e.active)
	if cap(e.proposals) < n {
		e.proposals = make([]proposal, n)
	}
	e.proposals = e.proposals[:n]
	err := e.parallel(n, func(wk, i int) error {
		e.proposals[i] = proposal{}
		return e.decideOne(wk, e.active[i], &e.proposals[i])
	})
	if err != nil {
		return n, err
	}

	// Transactions that only touch the acting agent commute with everything else
	// in this phase, so they are applied in parallel first. The rest (context,
	// spawns, links, removals...) are applied one by one in a seeded random order,
	// so no agent is systematically first in line for shared resources.
	if e.log < model.LogEffects {
		err = e.parallel(n, func(wk, i int) error {
			p := &e.proposals[i]
			act := world.Actor{ID: e.active[i].id}
			if p.decision == nil || p.infeasible || !world.IsLocal(act, p.pre) || !world.IsLocal(act, p.post) {
				return nil
			}
			p.local = true
			return e.applyProposal(act, p, func(effs []*world.Effect) (string, error) {
				return e.w.ApplyLocal(act, effs, &e.workers[wk].undo)
			})
		})
		if err != nil {
			return n, err
		}
	}

	e.order = e.order[:0]
	for i := range e.proposals {
		if e.proposals[i].decision != nil {
			e.order = append(e.order, i)
		}
	}
	src := e.workers[0].src
	src.Reset(e.opt.Seed, rng.StreamOrder, uint64(e.w.Tick), 0)
	src.Shuffle(len(e.order), func(i, j int) { e.order[i], e.order[j] = e.order[j], e.order[i] })

	for _, i := range e.order {
		a, p := e.active[i], &e.proposals[i]
		if !p.local {
			act := world.Actor{ID: a.id}
			if err := e.applyProposal(act, p, func(effs []*world.Effect) (string, error) { return e.w.Apply(act, effs) }); err != nil {
				return n, err
			}
		}
		c := &e.counters[a.typ][p.dindex]
		c.chosen++
		switch p.outcome {
		case "infeasible":
			c.infeasible++
		case "rejected":
			c.rejected++
		case "success":
			c.success++
		default:
			c.failure++
		}
		if e.log >= model.LogDecisions && e.sampled(a.id) {
			e.emit(&output.Event{Tick: e.w.Tick, Kind: "decision", Agent: uint64(a.id), Name: e.w.Name(a.id),
				Decision: p.decision.Name, Outcome: p.outcome, Reason: p.reason, Weights: p.weights})
		}
	}
	return n, e.err
}

// applyProposal applies the precondition effects and, if they succeed, the
// consequence effects, as two separate transactions, and records the outcome.
func (e *Engine) applyProposal(act world.Actor, p *proposal, apply func([]*world.Effect) (string, error)) error {
	if p.infeasible {
		p.outcome = "infeasible"
		return nil
	}
	reason, err := apply(p.pre)
	if err == nil && reason == "" {
		reason, err = apply(p.post)
		if reason != "" {
			reason = "consequence rejected: " + reason
		}
	} else if reason != "" {
		reason = "precondition rejected: " + reason
	}
	if err != nil {
		return fmt.Errorf("tick %d, decision %q: %w", e.w.Tick, p.decision.Name, err)
	}
	switch {
	case reason != "":
		p.outcome, p.reason = "rejected", reason
	case p.success:
		p.outcome = "success"
	default:
		p.outcome = "failure"
	}
	return nil
}

func (e *Engine) decideOne(wk int, a actor, p *proposal) error {
	w := e.workers[wk]
	st := e.w.Stores[a.typ]
	t := st.Type
	if len(t.Decisions) == 0 {
		return nil
	}
	w.src.Reset(e.opt.Seed, rng.StreamDecision, uint64(e.w.Tick), uint64(a.id))
	w.ev.Bind(st, int(a.idx), e.w.Tick, e.ctxSnapshot)

	fail := func(d *model.Decision, what string, err error) error {
		return fmt.Errorf("tick %d, agent %d (%s), decision %q, %s: %w", e.w.Tick, a.id, t.Name, d.Name, what, err)
	}
	weights := w.weights[:0]
	for di, d := range t.Decisions {
		h := &e.hooks[d.Index]
		ok := true
		if prog := e.eligible[wk][a.typ][di]; prog != nil {
			var err error
			if ok, err = w.ev.Bool(prog); err != nil {
				return fail(d, "eligible", err)
			}
		} else if h.eligible != 0 {
			r, err := w.rt.CallAgent(h.eligible, a.id, w.src, false)
			if err != nil {
				return fail(d, "eligible", err)
			}
			ok = r.Truthy()
		}
		weight := 0.0
		if ok {
			weight = 1
			if prog := e.propensity[wk][a.typ][di]; prog != nil {
				var err error
				if weight, err = w.ev.Number(prog); err != nil {
					return fail(d, "propensity", err)
				}
			} else if h.propensity != 0 {
				r, err := w.rt.CallAgent(h.propensity, a.id, w.src, false)
				if err != nil {
					return fail(d, "propensity", err)
				}
				if r.Kind != script.ResultNumber {
					return fail(d, "propensity", fmt.Errorf("must return a number"))
				}
				weight = r.Number
			}
			if math.IsNaN(weight) || math.IsInf(weight, 0) || weight < 0 {
				return fail(d, "propensity", fmt.Errorf("returned %v; propensities must be finite numbers >= 0", weight))
			}
		}
		weights = append(weights, weight)
	}
	idle := 0.0
	if t.Choice.HasIdle {
		idle = t.Choice.IdleConst
		if prog := e.idle[wk][a.typ]; prog != nil {
			var err error
			if idle, err = w.ev.Number(prog); err != nil {
				return fmt.Errorf("tick %d, agent %d (%s), choice.idle: %w", e.w.Tick, a.id, t.Name, err)
			}
		}
		if math.IsNaN(idle) || math.IsInf(idle, 0) || idle < 0 {
			return fmt.Errorf("tick %d, agent %d (%s): choice.idle returned %v; it must be a finite number >= 0", e.w.Tick, a.id, t.Name, idle)
		}
	}
	w.weights = weights

	if e.log >= model.LogDecisions && e.sampled(a.id) {
		p.weights = make(map[string]float64, len(weights)+1)
		for di, d := range t.Decisions {
			p.weights[d.Name] = weights[di]
		}
		if t.Choice.HasIdle {
			p.weights["(idle)"] = idle
		}
	}

	choice := choose(t.Choice, weights, idle, w.src)
	if choice < 0 {
		return nil
	}
	d := t.Decisions[choice]
	p.decision, p.dindex = d, choice
	h := &e.hooks[d.Index]

	if h.precondition != 0 {
		r, err := w.rt.CallAgent(h.precondition, a.id, w.src, false)
		if err != nil {
			return fail(d, "precondition", err)
		}
		switch r.Kind {
		case script.ResultBool:
			if !r.Bool {
				p.infeasible, p.reason = true, r.Reason
				return nil
			}
		case script.ResultEffects:
			p.pre = r.Effects
		case script.ResultNumber:
			return fail(d, "precondition", fmt.Errorf("must return nil, a boolean or effects, not a number"))
		}
	}

	p.success = true
	if h.outcome != 0 {
		r, err := w.rt.CallAgent(h.outcome, a.id, w.src, false)
		if err != nil {
			return fail(d, "outcome", err)
		}
		switch r.Kind {
		case script.ResultBool:
			p.success = r.Bool
		case script.ResultNumber:
			if r.Number < 0 || r.Number > 1 || math.IsNaN(r.Number) {
				return fail(d, "outcome", fmt.Errorf("a probability must be in [0, 1], got %v", r.Number))
			}
			p.success = w.src.Float() < r.Number
		case script.ResultEffects:
			return fail(d, "outcome", fmt.Errorf("must return a boolean or a probability, not effects"))
		}
		p.reason = r.Reason
	}

	hook, fn := "on_success", h.onSuccess
	if !p.success {
		hook, fn = "on_failure", h.onFailure
	}
	if fn != 0 {
		r, err := w.rt.CallAgent(fn, a.id, w.src, false)
		if err != nil {
			return fail(d, hook, err)
		}
		if r.Kind == script.ResultEffects {
			p.post = r.Effects
		} else if r.Kind != script.ResultNil {
			return fail(d, hook, fmt.Errorf("must return effects (fx.*) or nil"))
		}
	}
	return nil
}

// choose returns the chosen decision index, or -1 for doing nothing.
func choose(c model.Choice, weights []float64, idle float64, src *rng.Source) int {
	switch c.Mode {
	case model.ChoiceMax:
		// Doing nothing wins ties against decisions; ties between decisions are
		// broken uniformly (reservoir sampling).
		best, bestW, ties := -1, 0.0, 0
		if c.HasIdle {
			bestW = idle
		}
		for i, w := range weights {
			switch {
			case w <= 0 || w < bestW || (w == bestW && best == -1):
			case w > bestW:
				best, bestW, ties = i, w, 1
			default:
				ties++
				if src.IntN(ties) == 0 {
					best = i
				}
			}
		}
		return best
	case model.ChoiceSoftmax:
		top := idle
		for _, w := range weights {
			top = math.Max(top, w)
		}
		total := 0.0
		ex := make([]float64, len(weights))
		for i, w := range weights {
			if w > 0 {
				ex[i] = math.Exp((w - top) / c.Temperature)
				total += ex[i]
			}
		}
		idleW := 0.0
		if c.HasIdle {
			idleW = math.Exp((idle - top) / c.Temperature)
		}
		return roulette(ex, total+idleW, src)
	}
	total := idle
	for _, w := range weights {
		total += w
	}
	return roulette(weights, total, src)
}

// roulette picks index i with probability weights[i]/total; the remaining mass
// (the idle weight) means doing nothing.
func roulette(weights []float64, total float64, src *rng.Source) int {
	if total <= 0 {
		return -1
	}
	u := src.Float() * total
	for i, w := range weights {
		if u < w {
			return i
		}
		u -= w
	}
	return -1
}
