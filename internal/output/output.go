// Package output writes what a run produces: the JSONL event log, the per-tick
// metrics CSV, state snapshots and the aggregate over a batch of seeds.
package output

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/world"
)

// Event is one line of events.jsonl.
type Event struct {
	Tick     int                `json:"tick"`
	Kind     string             `json:"kind"`
	Agent    uint64             `json:"agent,omitempty"`
	Name     string             `json:"name,omitempty"`
	Rule     string             `json:"rule,omitempty"`
	Decision string             `json:"decision,omitempty"`
	Outcome  string             `json:"outcome,omitempty"`
	Reason   string             `json:"reason,omitempty"`
	Weights  map[string]float64 `json:"weights,omitempty"`
	Event    string             `json:"event,omitempty"`
	Effect   string             `json:"effect,omitempty"`
	Data     map[string]any     `json:"data,omitempty"`
}

// Sink receives everything a run produces.
type Sink interface {
	Event(*Event) error
	Metrics(tick int, values []float64) error
	Snapshot(w *world.World) error
	Close() error
}

// DirSink writes a run into a directory:
//
//	events.jsonl, metrics.csv, snapshots/tick-<n>.json
type DirSink struct {
	dir     string
	events  *os.File
	ebuf    *bufio.Writer
	enc     *json.Encoder
	metrics *os.File
	csv     *csv.Writer
	row     []string

	// Series keeps every metrics row in memory when batch aggregation needs it.
	Series [][]float64
	keep   bool
}

func NewDirSink(dir string, metricNames []string, keepSeries bool) (*DirSink, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &DirSink{dir: dir, keep: keepSeries}
	var err error
	if s.events, err = os.Create(filepath.Join(dir, "events.jsonl")); err != nil {
		return nil, err
	}
	s.ebuf = bufio.NewWriterSize(s.events, 1<<20)
	s.enc = json.NewEncoder(s.ebuf)
	if s.metrics, err = os.Create(filepath.Join(dir, "metrics.csv")); err != nil {
		return nil, err
	}
	s.csv = csv.NewWriter(s.metrics)
	if err := s.csv.Write(append([]string{"tick"}, metricNames...)); err != nil {
		return nil, err
	}
	s.row = make([]string, len(metricNames)+1)
	return s, nil
}

func (s *DirSink) Event(e *Event) error { return s.enc.Encode(e) }

func formatFloat(v float64) string {
	if math.IsNaN(v) {
		return ""
	}
	return strconv.FormatFloat(v, 'g', 10, 64)
}

func (s *DirSink) Metrics(tick int, values []float64) error {
	s.row[0] = strconv.Itoa(tick)
	for i, v := range values {
		s.row[i+1] = formatFloat(v)
	}
	if s.keep {
		s.Series = append(s.Series, append([]float64(nil), values...))
	}
	return s.csv.Write(s.row)
}

func (s *DirSink) Snapshot(w *world.World) error {
	dir := filepath.Join(s.dir, "snapshots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("tick-%06d.json", w.Tick)))
	if err != nil {
		return err
	}
	bw := bufio.NewWriterSize(f, 1<<20)
	if err := WriteSnapshot(bw, w); err != nil {
		f.Close()
		return err
	}
	if err := bw.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (s *DirSink) Close() error {
	s.csv.Flush()
	errs := []error{s.csv.Error(), s.ebuf.Flush(), s.events.Close(), s.metrics.Close()}
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
