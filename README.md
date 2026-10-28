# autonomous-agent-simulator

A generic engine for agent-based simulation of societies, economies and games.

The engine does not implement any particular society. It interprets **models**:
declarative JSON files that define agents, their attributes, memory and decisions, the
shared context and its rules, plus small Lua scripts for the logic. Market economies,
planned economies, cooperatives or the prisoner's dilemma are all configurations of the
same engine. Outcomes emerge from agents interacting under the configured rules.

An *agent* is not an AI: it is an entity with attributes and capabilities that evaluates
its options, chooses one and acts on the context it lives in.

## Quick start

Requires Go 1.27+.

```sh
go build -o agentsim ./cmd/agentsim

./agentsim validate models/prisoners-dilemma
./agentsim run models/prisoners-dilemma
./agentsim run models/entrepreneurship --seeds 8      # aggregate.csv over 8 seeds
./agentsim run models/benchmark --workers 4           # 1,000,000 agents
```

Each run writes `runs/<model>-<timestamp>/` with `events.jsonl`, `metrics.csv`,
snapshots and, for several seeds, `aggregate.csv` and `summary.json`.

## A model

```
models/prisoners-dilemma/
  pd.schema.json            context, agent types, decisions, rules, metrics
  pd.context.json           initial context (the payoff matrix)
  agents/trickster.agent.json
  agents/honest.agent.json
  scripts/*.lua             what each decision does; how a round is settled
```

```lua
-- scripts/defect.lua
function propensity(self, world)
  local seen = self:memlen("opponentMoves")
  local trust = seen > 0 and self:count("opponentMoves", "C") / seen or 0.5
  return (1 - self.honesty) + self.greed * trust + self.grudge * (1 - trust)
end

function on_success(self, world)
  return fx.set("ctx.moves." .. self.id, "D")
end
```

## Documentation

- [Modeling guide](docs/modeling-guide.md): how to write models (start here).
- [Architecture](docs/architecture.md): how the engine works, determinism, performance.
- [Contributing](CONTRIBUTING.md): commit, pull request and changelog conventions.

## Example models

| Model                      | Shows                                                                      |
|----------------------------|----------------------------------------------------------------------------|
| `models/prisoners-dilemma` | Simultaneous moves through the context, memory, personality, matchmaking   |
| `models/entrepreneurship`  | Costly decisions with uncertain outcomes, spawning firms, ownership, a central bank, inequality |
| `models/bread-island`      | Supply and demand: bread and land auctions, scarce land, hunger, frugality, births, droughts and epidemics |
| `models/benchmark`         | One million agents, for measuring throughput                               |
