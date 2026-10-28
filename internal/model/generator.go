package model

import (
	"fmt"
	"math"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/rng"
)

// Generator produces initial values for a population: a constant or a distribution.
type Generator struct {
	Kind    Kind
	Dist    string // "" for constants
	Const   Value
	Min     float64
	Max     float64
	HasMin  bool
	HasMax  bool
	Mean    float64
	Std     float64
	P       float64
	Values  []Value
	Weights []float64
}

// Sample draws one value. Integer kinds round to the nearest integer.
func (g *Generator) Sample(r *rng.Source) Value {
	var x float64
	switch g.Dist {
	case "":
		return g.Const
	case "uniform":
		if g.Kind == KindInt {
			lo, hi := int64(math.Ceil(g.Min)), int64(math.Floor(g.Max))
			return Int(lo + int64(r.IntN(int(hi-lo+1))))
		}
		x = g.Min + r.Float()*(g.Max-g.Min)
	case "normal":
		x = g.Mean + r.Norm()*g.Std
		if g.HasMin && x < g.Min {
			x = g.Min
		}
		if g.HasMax && x > g.Max {
			x = g.Max
		}
	case "bernoulli":
		return Bool(r.Float() < g.P)
	case "choice":
		if len(g.Weights) == 0 {
			return g.Values[r.IntN(len(g.Values))]
		}
		total := 0.0
		for _, w := range g.Weights {
			total += w
		}
		u := r.Float() * total
		for i, w := range g.Weights {
			if u < w {
				return g.Values[i]
			}
			u -= w
		}
		return g.Values[len(g.Values)-1]
	}
	if g.Kind == KindInt {
		return Int(int64(math.Round(x)))
	}
	return Float(x)
}

func number(m map[string]any, key string) (float64, bool, error) {
	raw, ok := m[key]
	if !ok {
		return 0, false, nil
	}
	v, err := FromJSON(raw)
	if err != nil || !v.Kind.Numeric() {
		return 0, true, fmt.Errorf("%q must be a number", key)
	}
	return v.AsFloat(), true, nil
}

// ParseGenerator reads a generator for a value of kind k. Accepted forms:
//
//	42 | "text" | true                                   constant
//	{"dist": "uniform", "min": 0, "max": 1}              numeric
//	{"dist": "normal", "mean": 0, "std": 1, "min"?, "max"?}  numeric, clamped
//	{"dist": "bernoulli", "p": 0.5}                      bool
//	{"dist": "choice", "values": [...], "weights"?: [...]}  any kind
func ParseGenerator(raw any, k Kind) (Generator, error) {
	g := Generator{Kind: k}
	obj, isObj := raw.(map[string]any)
	if !isObj {
		v, err := FromJSON(raw)
		if err != nil {
			return g, err
		}
		if g.Const, err = Coerce(v, k); err != nil {
			return g, err
		}
		return g, nil
	}
	dist, _ := obj["dist"].(string)
	allowed := map[string]bool{"dist": true, "description": true}
	g.Dist = dist
	var err error
	switch dist {
	case "uniform":
		allowed["min"], allowed["max"] = true, true
		if !k.Numeric() {
			return g, fmt.Errorf("uniform requires a numeric attribute, not %s", k)
		}
		var okMin, okMax bool
		if g.Min, okMin, err = number(obj, "min"); err != nil {
			return g, err
		}
		if g.Max, okMax, err = number(obj, "max"); err != nil {
			return g, err
		}
		if !okMin || !okMax || g.Max < g.Min {
			return g, fmt.Errorf("uniform requires \"min\" <= \"max\"")
		}
	case "normal":
		allowed["mean"], allowed["std"], allowed["min"], allowed["max"] = true, true, true, true
		if !k.Numeric() {
			return g, fmt.Errorf("normal requires a numeric attribute, not %s", k)
		}
		var ok bool
		if g.Mean, ok, err = number(obj, "mean"); err != nil || !ok {
			return g, fmt.Errorf("normal requires \"mean\"")
		}
		if g.Std, ok, err = number(obj, "std"); err != nil || !ok || g.Std < 0 {
			return g, fmt.Errorf("normal requires \"std\" >= 0")
		}
		if g.Min, g.HasMin, err = number(obj, "min"); err != nil {
			return g, err
		}
		if g.Max, g.HasMax, err = number(obj, "max"); err != nil {
			return g, err
		}
	case "bernoulli":
		allowed["p"] = true
		if k != KindBool {
			return g, fmt.Errorf("bernoulli requires a bool attribute, not %s", k)
		}
		var ok bool
		if g.P, ok, err = number(obj, "p"); err != nil || !ok || g.P < 0 || g.P > 1 {
			return g, fmt.Errorf("bernoulli requires \"p\" in [0, 1]")
		}
	case "choice":
		allowed["values"], allowed["weights"] = true, true
		vals, _ := obj["values"].([]any)
		if len(vals) == 0 {
			return g, fmt.Errorf("choice requires a non-empty \"values\" list")
		}
		for i, rv := range vals {
			v, err := FromJSON(rv)
			if err == nil {
				v, err = Coerce(v, k)
			}
			if err != nil {
				return g, fmt.Errorf("values[%d]: %v", i, err)
			}
			g.Values = append(g.Values, v)
		}
		if ws, ok := obj["weights"].([]any); ok {
			if len(ws) != len(vals) {
				return g, fmt.Errorf("\"weights\" must have the same length as \"values\"")
			}
			for i, rw := range ws {
				v, err := FromJSON(rw)
				if err != nil || !v.Kind.Numeric() || v.AsFloat() < 0 {
					return g, fmt.Errorf("weights[%d] must be a number >= 0", i)
				}
				g.Weights = append(g.Weights, v.AsFloat())
			}
		}
	default:
		return g, fmt.Errorf("unknown \"dist\" %q (expected uniform, normal, bernoulli or choice)", dist)
	}
	for key := range obj {
		if !allowed[key] {
			return g, fmt.Errorf("unknown key %q for %s distribution", key, dist)
		}
	}
	return g, nil
}
