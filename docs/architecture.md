# Architecture

`agentsim` is a generic engine for agent-based simulation. The engine knows nothing about
economics, games or politics: it knows **agents**, **attributes**, **memory**, a shared
**context**, a **topology** (relations, groups, space), **decisions**, **effects** and
**rules**. Everything domain-specific lives in a *model*: JSON files plus Lua scripts.

An *agent* here is not an AI: it is an entity with attributes and capabilities that
evaluates options, chooses, and acts on the context it lives in.

```
model directory ──► model.Load ──► Model (immutable, validated)
                                     │
                    script.Compile ──┤  Lua bytecode, compiled once
                                     ▼
                    engine.New ──► World (columnar state) + N script runtimes
                                     │
                    engine.Run ──► output.Sink (events.jsonl, metrics.csv, snapshots)
```

## Principles

1. **The core stays small and generic.** Market, firm, central bank or prisoner are
   configurations, never engine concepts.
2. **Scripts compute, the engine changes.** Scripts get read-only views of the world and
   return *effects*. Only the engine applies effects, after validating them, atomically.
3. **Agents act only on themselves and the context.** An agent cannot change another agent.
   Interactions happen through the shared context and are resolved by *rules*, which are
   the model's authority and may change anything.
4. **Reproducibility is not optional.** A run is a pure function of (model, seed). The
   number of worker goroutines never changes the result.
5. **Built for large populations.** Columnar state, native expressions, cached aggregates,
   parallel decision phase, bounded memory.

## Packages

| Package           | Responsibility                                                                 |
|-------------------|--------------------------------------------------------------------------------|
| `internal/model`  | File formats, loading, validation (every error at once), compiled `Model`      |
| `internal/world`  | Columnar agent stores, context, topology, effects and transactional `Apply`    |
| `internal/script` | `Runtime` interface, gopher-lua backend (sandbox, proxies, `fx`), expressions  |
| `internal/engine` | Scheduling, decision phase, ordered application, rules, metrics, stop          |
| `internal/output` | JSONL events, metrics CSV, streaming snapshots, multi-seed aggregation         |
| `internal/rng`    | Counter-based random streams keyed by (seed, stream, tick, entity)             |
| `cmd/agentsim`    | CLI: `validate`, `run`, `jsonschema`                                           |

## The tick

```
init rules (once, tick 0) ─► metrics(0)
for tick in 1..maxTicks:
  1. pre rules            each rule: run → effects → applied as one transaction
  2. schedule             which agents act (activation: every N ticks / probability)
  3. decide (parallel)    every active agent, on the SAME snapshot:
                            eligible → propensity → choose → precondition → outcome
                            → on_success | on_failure          (all reads, no writes)
  4. apply                a) self-only transactions in parallel
                          b) the rest one by one, in a seeded random order
  5. post rules
  6. metrics, snapshots, stop condition
```

Because every agent decides on the same snapshot, decisions are simultaneous (as in the
prisoner's dilemma) and the decision phase is embarrassingly parallel. Conflicts are
resolved at application time, where the current state, not the snapshot, is checked.

### Decision lifecycle

| Step           | Where                                      | Returns                                          |
|----------------|--------------------------------------------|--------------------------------------------------|
| `eligible`     | expression in the schema, or Lua function  | bool; ineligible decisions get weight 0          |
| `propensity`   | expression in the schema, or Lua function  | finite number >= 0 (default 1)                   |
| choose         | engine (`weighted`, `softmax`, `max`)      | one decision, or idle (`choice.idle` weight)     |
| `precondition` | Lua                                        | `false[, reason]` (infeasible), or cost effects  |
| `outcome`      | Lua                                        | bool, or a success probability in [0, 1]         |
| `on_success`   | Lua                                        | effects                                          |
| `on_failure`   | Lua                                        | effects                                          |

The precondition effects (the cost of trying) and the consequence effects are two
separate transactions: if the cost is paid, it stays paid even if the consequence is
later rejected. A decision is counted as `success`, `failure` (outcome said no),
`infeasible` (precondition returned false) or `rejected` (a transaction was refused by
the state at application time, e.g. not enough money any more).

### Effects and transactions

Effects: `set`, `add`, `transfer`, `clear`, `delete`, `remember`, `spawn`, `remove`,
`link`, `unlink`, `join`, `leave`, `move`, `emit`.

`World.Apply` treats a list of effects as one transaction:

1. **Check** every effect: types, permissions, referenced agents, memories. Wrong types or
   permission violations are `ModelError`s and stop the run, because they are bugs.
2. **Value effects** (`set`, `add`, `transfer`) run first, with an undo log. Constraints
   (`min`, `max`, enum) are enforced; a violation rolls the whole transaction back and
   returns a *rejection reason*, which is a normal outcome (scarcity).
3. **Structural effects** run in order and cannot fail any more.

Context write conflicts: two different actors `set`ting the same context value in one
phase follow the field's `merge` policy: `error` (default, stops the run with an
explanation), `first`, or `last`. `add` and `transfer` are commutative and never conflict.

### Parallel application of self-only transactions

Since agents cannot touch each other, a transaction that only changes the actor's own
attributes and memory commutes with every other transaction of the decision phase. The
engine applies those in parallel (`World.ApplyLocal`), then applies the rest (context
writes, spawns, links, removals) sequentially in a seeded random order. The order is
random so no agent is systematically first in line for a shared resource, and seeded so
the run stays reproducible.

## State layout

Each agent type has a `Store` with one typed slice per attribute (struct of arrays).
Agent IDs are dense and never reused; an ID-indexed slice maps them to (type, index).
Removal swaps the last agent into the hole. Memories are fixed-capacity ring buffers
stored in flat typed slices (agent *i* owns `[i*cap, (i+1)*cap)`), so a long run has a
constant footprint.

Topology: relations are sorted adjacency lists (directed or not); groups keep sorted
membership both ways; space is a uniform grid hash queried by radius.

Aggregates (`count`, `sum`, `mean`, `min`, `max`, `std`, `gini`) over a type or a group
are memoized until the next phase, so a million agents asking for the same mean cost one
scan.

## Scripts

`script.Runtime` is the only interface between the engine and the scripting backend. The
gopher-lua implementation:

- compiles each script once and shares the bytecode between runtimes (one per worker);
- runs each script in its own environment, so every decision can define `propensity`;
- freezes environments after loading (no state in globals between calls);
- removes `require`, `load`, `dofile`, `io`, `os`, `debug`; disables `math.random`;
- replaces `pairs` with a sorted iteration (Go maps iterate randomly);
- exposes agents, the world and the context as read-only proxies that enforce
  visibility (private attributes, hidden context fields, private memories).

Propensity and eligibility can also be expressions (expr-lang), evaluated natively in Go
without entering Lua: the fast path for large populations.

## Determinism

Every random draw comes from a stream keyed by `(seed, stream, tick, entity)`: population
generation, activation, each agent's decision, the application order, each rule, metrics
and the stop condition have separate streams. Output order never depends on goroutine
scheduling: parallel phases write results to fixed indices. Tests check that 1, 3 and 4
workers produce identical metrics.

What a model can still break: keeping state in local upvalues at script top level, or
relying on the iteration order of `next` (removed) — see the modeling guide.

## Performance

Measured on a 4-thread machine with `models/benchmark` (1,000,000 agents, two decisions
per agent, one propensity as an expression and one in Lua, both consequences in Lua):

| Workers | Agent activations per second | Time per tick |
|---------|------------------------------|---------------|
| 1       | ~420,000                     | ~2.4 s        |
| 4       | ~740,000                     | ~1.35 s       |

Profiling shows the Lua interpreter dominates (gopher-lua is a pure-Go interpreter).
Levers, in order of cost:

1. Write propensities and eligibility as expressions; keep Lua for effects.
2. Use sparse activation (`activation.every`, `activation.probability`).
3. Prefer engine aggregates (`world:mean`, `world:count`) over Lua loops.
4. Swap the backend: `script.Runtime` is the seam for a LuaJIT (cgo) implementation.

## Outputs

```
runs/<model>-<timestamp>/
  run.json                 options of the run
  seed-<n>/events.jsonl    decisions, emits, spawns, removals, effects (by log level, sampled)
  seed-<n>/metrics.csv     one row per tick: model metrics, population, decision counters
  seed-<n>/snapshots/      full state as JSON (periodic and/or final)
  aggregate.csv            with several seeds: mean/std/min/max of every metric per tick
  summary.json             with several seeds: final-tick statistics
```
