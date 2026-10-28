package model

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Kind is the type of a scalar value stored by the engine.
type Kind uint8

const (
	KindInvalid Kind = iota
	KindInt
	KindFloat
	KindBool
	KindString
	kindCount
)

var kindNames = [...]string{"invalid", "int", "float", "bool", "string"}

func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return "invalid"
}

// Numeric reports whether values of this kind support arithmetic.
func (k Kind) Numeric() bool { return k == KindInt || k == KindFloat }

// ParseKind converts a type name used in schema files into a Kind.
func ParseKind(s string) (Kind, bool) {
	switch s {
	case "int":
		return KindInt, true
	case "float":
		return KindFloat, true
	case "bool":
		return KindBool, true
	case "string":
		return KindString, true
	}
	return KindInvalid, false
}

// Value is a tagged scalar. Only the field matching Kind is meaningful.
type Value struct {
	Kind Kind
	I    int64
	F    float64
	B    bool
	S    string
}

func Int(i int64) Value     { return Value{Kind: KindInt, I: i} }
func Float(f float64) Value { return Value{Kind: KindFloat, F: f} }
func Bool(b bool) Value     { return Value{Kind: KindBool, B: b} }
func String(s string) Value { return Value{Kind: KindString, S: s} }

// Zero returns the zero value of a kind.
func Zero(k Kind) Value { return Value{Kind: k} }

// AsFloat converts numeric and boolean values to float64; strings yield NaN.
func (v Value) AsFloat() float64 {
	switch v.Kind {
	case KindInt:
		return float64(v.I)
	case KindFloat:
		return v.F
	case KindBool:
		if v.B {
			return 1
		}
		return 0
	}
	return math.NaN()
}

// Any returns the value as a plain Go value, suitable for JSON encoding.
func (v Value) Any() any {
	switch v.Kind {
	case KindInt:
		return v.I
	case KindFloat:
		return v.F
	case KindBool:
		return v.B
	case KindString:
		return v.S
	}
	return nil
}

func (v Value) String() string {
	switch v.Kind {
	case KindInt:
		return strconv.FormatInt(v.I, 10)
	case KindFloat:
		return strconv.FormatFloat(v.F, 'g', -1, 64)
	case KindBool:
		return strconv.FormatBool(v.B)
	case KindString:
		return strconv.Quote(v.S)
	}
	return "<invalid>"
}

// Equal compares two values, treating int and float as the same number line.
func (v Value) Equal(o Value) bool {
	if v.Kind.Numeric() && o.Kind.Numeric() {
		return v.AsFloat() == o.AsFloat()
	}
	return v == o
}

// Coerce converts v to kind k, refusing lossy conversions.
func Coerce(v Value, k Kind) (Value, error) {
	if v.Kind == k {
		return v, nil
	}
	switch k {
	case KindFloat:
		if v.Kind == KindInt {
			return Float(float64(v.I)), nil
		}
	case KindInt:
		if v.Kind == KindFloat && v.F == math.Trunc(v.F) && math.Abs(v.F) < 1<<53 {
			return Int(int64(v.F)), nil
		}
	}
	return Value{}, fmt.Errorf("expected %s, got %s %s", k, v.Kind, v)
}

// FromJSON converts a value decoded with json.Decoder.UseNumber into a Value.
func FromJSON(raw any) (Value, error) {
	switch x := raw.(type) {
	case json.Number:
		s := x.String()
		if !strings.ContainsAny(s, ".eE") {
			if i, err := x.Int64(); err == nil {
				return Int(i), nil
			}
		}
		f, err := x.Float64()
		if err != nil {
			return Value{}, err
		}
		return Float(f), nil
	case float64:
		return Float(x), nil
	case string:
		return String(x), nil
	case bool:
		return Bool(x), nil
	case nil:
		return Value{}, fmt.Errorf("null is not a valid value")
	}
	return Value{}, fmt.Errorf("expected a scalar (number, string or bool), got %T", raw)
}
