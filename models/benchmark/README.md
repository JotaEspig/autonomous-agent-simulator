# Benchmark

A synthetic model for measuring engine throughput, not a model of anything real. One million
agents choose between two decisions every tick for 10 ticks. Logging and snapshots are off, so
the run measures the engine itself.

```sh
agentsim run models/benchmark --workers 4
agentsim run models/benchmark --workers 4 --cpuprofile cpu.prof   # then: go tool pprof
```

## Context

| Field   | Type  | Default | Meaning                              |
|---------|-------|---------|--------------------------------------|
| `price` | float | 1       | Scales the propensity to spend       |

## Agents

`worker` × 1,000,000 (`agents/workers.agent.json`):

| Attribute | Initial distribution | Meaning                         |
|-----------|----------------------|---------------------------------|
| `wealth`  | uniform int(0, 1000) | Cash, never negative (`min: 0`) |
| `skill`   | uniform(0, 1)        | Raises what saving earns        |
| `thrift`  | uniform(0, 1)        | Inclination to save             |

Memory: `spent`, the last 4 amounts spent.

## Decisions

The decisions deliberately mix both propensity paths, so the benchmark exercises native
expressions and Lua.

| Decision | Propensity                                   | Effects (Lua)                                   |
|----------|----------------------------------------------|-------------------------------------------------|
| `save`   | expression: `thrift + 0.1 × skill`           | `wealth += floor(10 × skill) + 1`               |
| `spend`  | Lua: `(1 − thrift) × ctx.price`              | `wealth −= floor(10% of wealth)`; remembers the amount |

Each agent costs two or three Lua calls per tick. Both consequences only touch the agent itself,
so their transactions are applied in parallel.

## Rules

None.

## Metrics

`wealth.mean` and `wealth.gini`. The Gini coefficient sorts a million values every tick, which
is a realistic cost for large models.

## Reference numbers

On a 4-thread machine: about 740,000 agent activations per second with 4 workers (about 1.35 s
per tick), and about 420,000 with 1 worker. The Lua interpreter dominates the profile. See
"Performance" in [the architecture document](../../docs/architecture.md).
