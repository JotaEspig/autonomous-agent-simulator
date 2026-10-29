package script

import (
	"fmt"
	"math"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/world"
)

// Evaluator runs propensity/eligibility expressions natively in Go, which is much
// cheaper than a Lua call. It is bound to one agent at a time and is not safe for
// concurrent use: keep one per worker.
//
// Inside an expression:
//   - the agent's own attributes are plain identifiers (numbers are float64);
//   - ctx.<field> reads public scalar context fields; tick is the current tick;
//   - last(m), recall(m, n), memlen(m), count(m, v), mean(m) read the agent's memory;
//   - sigmoid, clamp, exp, log, sqrt, pow complement expr's builtins (abs, min, max...).
type Evaluator struct {
	w   *world.World
	env map[string]any
	st  *world.Store
	idx int
	vm  vm.VM
}

// Program is an expression compiled for one agent type and bound to one Evaluator.
type Program struct {
	src  string
	prog *vm.Program
	t    *model.AgentType
	bool bool
}

func NewEvaluator(w *world.World) *Evaluator {
	return &Evaluator{w: w, env: map[string]any{}}
}

func exprValue(v model.Value) any {
	switch v.Kind {
	case model.KindInt:
		return float64(v.I)
	case model.KindFloat:
		return v.F
	case model.KindBool:
		return v.B
	case model.KindString:
		return v.S
	}
	return nil
}

func toFloat(x any) (float64, error) {
	switch v := x.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	}
	return 0, fmt.Errorf("expected a number, got %T (%v)", x, x)
}

// ContextSnapshot builds the read-only map exposed as ctx in expressions. Build it
// once per phase and share it between evaluators.
func ContextSnapshot(w *world.World) map[string]any {
	m := make(map[string]any, len(w.Model.Context))
	for _, f := range w.Model.Context {
		if f.Public && !f.IsMap {
			m[f.Name] = exprValue(w.Context.Get(f))
		}
	}
	return m
}

func (ev *Evaluator) memory(name string) (*model.Memory, error) {
	mem := ev.st.Type.MemoryIndex[name]
	if mem == nil {
		return nil, fmt.Errorf("agent type %q has no memory %q", ev.st.Type.Name, name)
	}
	return mem, nil
}

func (ev *Evaluator) functions() []expr.Option {
	f1 := func(fn func(float64) float64) func(...any) (any, error) {
		return func(p ...any) (any, error) {
			if len(p) != 1 {
				return nil, fmt.Errorf("expected 1 argument")
			}
			x, err := toFloat(p[0])
			if err != nil {
				return nil, err
			}
			return fn(x), nil
		}
	}
	memName := func(p []any, n int) (*model.Memory, error) {
		if len(p) != n {
			return nil, fmt.Errorf("expected %d argument(s)", n)
		}
		name, ok := p[0].(string)
		if !ok {
			return nil, fmt.Errorf("the memory name must be a string")
		}
		return ev.memory(name)
	}
	return []expr.Option{
		expr.Function("sigmoid", f1(func(x float64) float64 { return 1 / (1 + math.Exp(-x)) })),
		expr.Function("exp", f1(math.Exp)),
		expr.Function("log", f1(math.Log)),
		expr.Function("sqrt", f1(math.Sqrt)),
		expr.Function("pow", func(p ...any) (any, error) {
			if len(p) != 2 {
				return nil, fmt.Errorf("pow expects 2 arguments")
			}
			a, err1 := toFloat(p[0])
			b, err2 := toFloat(p[1])
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("pow expects numbers")
			}
			return math.Pow(a, b), nil
		}),
		expr.Function("clamp", func(p ...any) (any, error) {
			if len(p) != 3 {
				return nil, fmt.Errorf("clamp expects (x, lo, hi)")
			}
			x, err1 := toFloat(p[0])
			lo, err2 := toFloat(p[1])
			hi, err3 := toFloat(p[2])
			if err1 != nil || err2 != nil || err3 != nil {
				return nil, fmt.Errorf("clamp expects numbers")
			}
			return math.Min(math.Max(x, lo), hi), nil
		}),
		expr.Function("last", func(p ...any) (any, error) {
			mem, err := memName(p, 1)
			if err != nil {
				return nil, err
			}
			v, ok := ev.st.Recall(ev.idx, mem, 0)
			if !ok {
				return nil, nil
			}
			return exprValue(v), nil
		}),
		expr.Function("recall", func(p ...any) (any, error) {
			mem, err := memName(p, 2)
			if err != nil {
				return nil, err
			}
			n, err := toFloat(p[1])
			if err != nil {
				return nil, err
			}
			v, ok := ev.st.Recall(ev.idx, mem, int(n)-1)
			if !ok {
				return nil, nil
			}
			return exprValue(v), nil
		}),
		expr.Function("memlen", func(p ...any) (any, error) {
			mem, err := memName(p, 1)
			if err != nil {
				return nil, err
			}
			return float64(ev.st.MemLen(ev.idx, mem)), nil
		}),
		expr.Function("count", func(p ...any) (any, error) {
			mem, err := memName(p, 2)
			if err != nil {
				return nil, err
			}
			n := 0
			for i := 0; i < ev.st.MemLen(ev.idx, mem); i++ {
				v, _ := ev.st.Recall(ev.idx, mem, i)
				if exprValue(v) == p[1] {
					n++
				}
			}
			return float64(n), nil
		}),
		expr.Function("mean", func(p ...any) (any, error) {
			mem, err := memName(p, 1)
			if err != nil {
				return nil, err
			}
			n := ev.st.MemLen(ev.idx, mem)
			if n == 0 {
				return 0.0, nil
			}
			s := 0.0
			for i := 0; i < n; i++ {
				v, _ := ev.st.Recall(ev.idx, mem, i)
				s += v.AsFloat()
			}
			return s / float64(n), nil
		}),
	}
}

// Compile type-checks src against agent type t. wantBool selects an eligibility
// expression (bool result) instead of a propensity (number result).
func (ev *Evaluator) Compile(src string, t *model.AgentType, wantBool bool) (*Program, error) {
	sample := map[string]any{"ctx": map[string]any{}, "tick": 0.0}
	for _, f := range t.Attrs {
		sample[f.Name] = exprValue(f.Default)
	}
	opts := append(ev.functions(), expr.Env(sample))
	if wantBool {
		opts = append(opts, expr.AsBool())
	}
	prog, err := expr.Compile(src, opts...)
	if err != nil {
		return nil, err
	}
	return &Program{src: src, prog: prog, t: t, bool: wantBool}, nil
}

// Bind points the evaluator at an agent and the current context snapshot.
func (ev *Evaluator) Bind(st *world.Store, idx int, tick int, ctx map[string]any) {
	ev.st, ev.idx = st, idx
	for _, f := range st.Type.Attrs {
		ev.env[f.Name] = exprValue(st.Get(idx, f))
	}
	ev.env["ctx"] = ctx
	ev.env["tick"] = float64(tick)
}

// Number evaluates a propensity expression.
func (ev *Evaluator) Number(p *Program) (float64, error) {
	out, err := ev.vm.Run(p.prog, ev.env)
	if err != nil {
		return 0, err
	}
	return toFloat(out)
}

// Bool evaluates an eligibility expression.
func (ev *Evaluator) Bool(p *Program) (bool, error) {
	out, err := ev.vm.Run(p.prog, ev.env)
	if err != nil {
		return false, err
	}
	b, ok := out.(bool)
	if !ok {
		return false, fmt.Errorf("expected a bool, got %T", out)
	}
	return b, nil
}
