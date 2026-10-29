package engine

import (
	"github.com/JotaEspig/autonomous-agent-simulator/internal/output"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/world"
)

type recorder struct {
	rows   [][]float64
	events []*output.Event
}

func (r *recorder) Event(e *output.Event) error { r.events = append(r.events, e); return nil }
func (r *recorder) Metrics(_ int, v []float64) error {
	r.rows = append(r.rows, append([]float64(nil), v...))
	return nil
}
func (r *recorder) Snapshot(*world.World) error { return nil }
func (r *recorder) Close() error                { return nil }
