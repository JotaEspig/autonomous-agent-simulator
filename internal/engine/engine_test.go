package engine

import (
	"context"
	"math"
	"testing"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/script"
)

func run(t *testing.T, dir string, seed uint64, workers, ticks int) [][]float64 {
	t.Helper()
	m, err := model.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	c, err := script.Compile(m)
	if err != nil {
		t.Fatal(err)
	}
	none := model.LogNone
	e, err := New(m, c, Options{Seed: seed, Workers: workers, MaxTicks: ticks, Log: &none})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	sink := &recorder{}
	if _, err := e.Run(context.Background(), sink); err != nil {
		t.Fatal(err)
	}
	return sink.rows
}

func equalRows(a, b [][]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		for j := range a[i] {
			x, y := a[i][j], b[i][j]
			if x != y && !(math.IsNaN(x) && math.IsNaN(y)) {
				return false
			}
		}
	}
	return true
}

func TestDeterministicAcrossWorkerCounts(t *testing.T) {
	for _, dir := range []string{"../../models/prisoners-dilemma", "../../models/entrepreneurship"} {
		t.Run(dir, func(t *testing.T) {
			one := run(t, dir, 7, 1, 30)
			four := run(t, dir, 7, 4, 30)
			if !equalRows(one, four) {
				t.Fatalf("1 worker and 4 workers produced different metrics")
			}
			again := run(t, dir, 7, 3, 30)
			if !equalRows(one, again) {
				t.Fatalf("repeated run produced different metrics")
			}
			other := run(t, dir, 8, 4, 30)
			if equalRows(one, other) {
				t.Fatalf("different seeds produced identical metrics")
			}
		})
	}
}
