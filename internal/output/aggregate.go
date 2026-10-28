package output

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

// Stats summarizes one metric across seeds.
type Stats struct {
	Mean float64 `json:"mean"`
	Std  float64 `json:"std"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	N    int     `json:"n"`
}

func summarize(vals []float64) Stats {
	s := Stats{Min: math.Inf(1), Max: math.Inf(-1)}
	for _, v := range vals {
		if math.IsNaN(v) {
			continue
		}
		s.N++
		d := v - s.Mean
		s.Mean += d / float64(s.N)
		s.Std += d * (v - s.Mean)
		s.Min = math.Min(s.Min, v)
		s.Max = math.Max(s.Max, v)
	}
	if s.N == 0 {
		return Stats{Mean: math.NaN(), Std: math.NaN(), Min: math.NaN(), Max: math.NaN()}
	}
	s.Std = math.Sqrt(s.Std / float64(s.N))
	return s
}

// jsonStats replaces NaN, which JSON cannot represent, with null.
type jsonStats map[string]*Stats

// WriteAggregate summarizes a batch of runs. series[s][t][m] is metric m at the
// t-th recorded row (tick t) of seed s. Runs may stop at different ticks; each tick
// is aggregated over the seeds that reached it.
//
//	aggregate.csv  tick, then <metric>_mean, _std, _min, _max for every metric
//	summary.json   seeds and the final-tick statistics of every metric
func WriteAggregate(dir string, names []string, seeds []uint64, series [][][]float64) error {
	f, err := os.Create(filepath.Join(dir, "aggregate.csv"))
	if err != nil {
		return err
	}
	cw := csv.NewWriter(f)
	header := []string{"tick"}
	for _, n := range names {
		header = append(header, n+"_mean", n+"_std", n+"_min", n+"_max")
	}
	if err := cw.Write(header); err != nil {
		f.Close()
		return err
	}
	maxLen := 0
	for _, s := range series {
		maxLen = max(maxLen, len(s))
	}
	vals := make([]float64, 0, len(series))
	for t := 0; t < maxLen; t++ {
		row := []string{strconv.Itoa(t)}
		for m := range names {
			vals = vals[:0]
			for _, s := range series {
				if t < len(s) {
					vals = append(vals, s[t][m])
				}
			}
			st := summarize(vals)
			row = append(row, formatFloat(st.Mean), formatFloat(st.Std), formatFloat(st.Min), formatFloat(st.Max))
		}
		if err := cw.Write(row); err != nil {
			f.Close()
			return err
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	final := jsonStats{}
	for m, n := range names {
		vals = vals[:0]
		for _, s := range series {
			if len(s) > 0 {
				vals = append(vals, s[len(s)-1][m])
			}
		}
		st := summarize(vals)
		if st.N == 0 {
			final[n] = nil
			continue
		}
		final[n] = &st
	}
	ticks := make([]int, len(series))
	for i, s := range series {
		ticks[i] = len(s) - 1
	}
	data, err := json.MarshalIndent(map[string]any{"seeds": seeds, "ticks": ticks, "final": final}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "summary.json"), append(data, '\n'), 0o644)
}
