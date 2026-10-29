// Command agentsim validates and runs agent-based simulation models.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/engine"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/output"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/script"
)

const usage = `agentsim - generic agent-based simulation engine

Usage:
  agentsim validate <model-dir>             check a model and report every problem found
  agentsim run <model-dir> [flags]          run a model (see "agentsim run -h")
  agentsim jsonschema <model-dir> [-out d]  generate JSON Schemas for the model's agent and context files

A model directory holds one <name>.schema.json, an optional <name>.context.json,
any number of *.agent.json files and the Lua scripts they reference.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "validate":
		err = cmdValidate(os.Args[2:])
	case "run":
		err = cmdRun(os.Args[2:])
	case "jsonschema":
		err = cmdJSONSchema(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// splitArgs lets the model directory come before or after the flags.
func splitArgs(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	if len(args) > 0 && !strings.HasPrefix(args[len(args)-1], "-") {
		return args[len(args)-1], args[:len(args)-1]
	}
	return "", args
}

func load(dir string) (*model.Model, *script.Compiled, error) {
	if dir == "" {
		return nil, nil, errors.New("missing model directory")
	}
	m, err := model.Load(dir)
	if err != nil {
		return nil, nil, err
	}
	c, err := script.Compile(m)
	if err != nil {
		return nil, nil, err
	}
	return m, c, nil
}

func cmdValidate(args []string) error {
	dir, _ := splitArgs(args)
	m, c, err := load(dir)
	if err != nil {
		return err
	}
	e, err := engine.New(m, c, engine.Options{Seed: 1, Workers: 1})
	if err != nil {
		return err
	}
	defer e.Close()
	warnings := e.Warnings
	if _, err := os.Stat(filepath.Join(m.Dir, "README.md")); err != nil {
		warnings = append(warnings, "README.md not found: document the model's agents, context, decisions and rules (see docs/modeling-guide.md)")
	}
	for _, w := range warnings {
		fmt.Println("warning:", w)
	}
	fmt.Printf("model %q is valid\n", m.Name)
	fmt.Printf("  max ticks: %d\n", m.MaxTicks)
	for _, t := range m.Types {
		var ds []string
		for _, d := range t.Decisions {
			ds = append(ds, d.Name)
		}
		fmt.Printf("  agent type %-16s %8d agents, %d attributes, %d memories, decisions: %s\n",
			t.Name, e.World().Stores[t.Index].Len(), len(t.Attrs), len(t.Memories), strings.Join(ds, ", "))
	}
	for _, r := range m.Rules {
		scope := "world"
		if r.Type != nil {
			scope = "per " + r.Type.Name
		}
		fmt.Printf("  rule %-22s phase %-4s every %d tick(s), %s\n", r.Name, r.Phase, r.Every, scope)
	}
	fmt.Printf("  metrics: %s\n", strings.Join(e.MetricNames(), ", "))
	return nil
}

func cmdRun(args []string) error {
	dir, rest := splitArgs(args)
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	seed := fs.Uint64("seed", 1, "first seed")
	seeds := fs.Int("seeds", 1, "number of seeds to run (seed, seed+1, ...)")
	parallel := fs.Int("parallel", 0, "seeds run at the same time (default: min(seeds, CPUs))")
	workers := fs.Int("workers", 0, "worker goroutines per run (default: CPUs / parallel)")
	ticks := fs.Int("ticks", 0, "override simulation.maxTicks")
	out := fs.String("out", "", "output directory (default: runs/<model>-<timestamp>)")
	logLevel := fs.String("log", "", "override output.log: none, events, decisions or effects")
	sample := fs.Float64("sample", -1, "override output.sample: fraction of agents whose events are logged")
	snapEvery := fs.Int("snapshot-every", -1, "override output.snapshotEvery (0 disables periodic snapshots)")
	cpuProfile := fs.String("cpuprofile", "", "write a Go CPU profile to this file")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
	}
	if dir == "" && fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	m, c, err := load(dir)
	if err != nil {
		return err
	}
	if *seeds < 1 {
		return errors.New("-seeds must be >= 1")
	}
	cpus := runtime.GOMAXPROCS(0)
	if *parallel <= 0 {
		*parallel = min(*seeds, cpus)
	}
	if *workers <= 0 {
		*workers = max(1, cpus / *parallel)
	}
	opt := engine.Options{Workers: *workers, MaxTicks: *ticks}
	if *logLevel != "" {
		lvl, ok := model.ParseLogLevel(*logLevel)
		if !ok {
			return fmt.Errorf("unknown log level %q", *logLevel)
		}
		opt.Log = &lvl
	}
	if *sample >= 0 {
		opt.Sample = sample
	}
	if *snapEvery >= 0 {
		opt.SnapshotEvery = snapEvery
	}
	if *out == "" {
		*out = filepath.Join("runs", fmt.Sprintf("%s-%s", m.Name, time.Now().Format("20060102-150405")))
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	if err := writeRunInfo(*out, m, *seed, *seeds, opt); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Printf("running %q: %d seed(s), %d in parallel, %d worker(s) each -> %s\n", m.Name, *seeds, *parallel, *workers, *out)
	seedList := make([]uint64, *seeds)
	series := make([][][]float64, *seeds)
	errs := make([]error, *seeds)
	var names []string
	var mu sync.Mutex
	sem := make(chan struct{}, *parallel)
	var wg sync.WaitGroup
	for i := range seedList {
		seedList[i] = *seed + uint64(i)
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			o := opt
			o.Seed = seedList[i]
			res, ns, s, err := runSeed(ctx, m, c, o, filepath.Join(*out, fmt.Sprintf("seed-%d", o.Seed)), *seeds > 1)
			if err != nil {
				errs[i] = fmt.Errorf("seed %d: %w", o.Seed, err)
				return
			}
			mu.Lock()
			names, series[i] = ns, s
			rate := float64(res.AgentTicks) / res.Elapsed.Seconds()
			fmt.Printf("  seed %-6d %5d ticks (%s) in %-10s %12.0f agent-activations/s, final population %d\n",
				o.Seed, res.Ticks, res.StoppedBy, res.Elapsed.Round(time.Millisecond), rate, res.Population)
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return err
	}
	if *seeds > 1 {
		if err := output.WriteAggregate(*out, names, seedList, series); err != nil {
			return err
		}
		fmt.Printf("aggregate over %d seeds: %s\n", *seeds, filepath.Join(*out, "aggregate.csv"))
	}
	return nil
}

func runSeed(ctx context.Context, m *model.Model, c *script.Compiled, opt engine.Options, dir string, keep bool) (engine.Result, []string, [][]float64, error) {
	e, err := engine.New(m, c, opt)
	if err != nil {
		return engine.Result{}, nil, nil, err
	}
	defer e.Close()
	sink, err := output.NewDirSink(dir, e.MetricNames(), keep)
	if err != nil {
		return engine.Result{}, nil, nil, err
	}
	res, runErr := e.Run(ctx, sink)
	if err := sink.Close(); runErr == nil {
		runErr = err
	}
	return res, e.MetricNames(), sink.Series, runErr
}

func writeRunInfo(dir string, m *model.Model, seed uint64, seeds int, opt engine.Options) error {
	info := map[string]any{
		"model":     m.Name,
		"modelDir":  m.Dir,
		"firstSeed": seed,
		"seeds":     seeds,
		"workers":   opt.Workers,
		"maxTicks":  max(opt.MaxTicks, 0),
		"startedAt": time.Now().Format(time.RFC3339),
		"go":        runtime.Version(),
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "run.json"), append(data, '\n'), 0o644)
}

func cmdJSONSchema(args []string) error {
	dir, rest := splitArgs(args)
	fs := flag.NewFlagSet("jsonschema", flag.ContinueOnError)
	out := fs.String("out", "", "output directory (default: <model-dir>/.jsonschema)")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if dir == "" {
		return errors.New("missing model directory")
	}
	m, err := model.Load(dir)
	if err != nil {
		return err
	}
	if *out == "" {
		*out = filepath.Join(m.Dir, ".jsonschema")
	}
	files, err := model.JSONSchemas(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	for name, data := range files {
		path := filepath.Join(*out, name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		fmt.Println("wrote", path)
	}
	return nil
}
