# Modeling guide

This guide explains how to write a model for `agentsim`. The engine is the easy part;
the model is where the thinking happens. Start small, validate often, and let behavior
emerge from rules instead of scripting outcomes.

## 1. Files

```
my-model/
  README.md                what the model is about: agents, context, decisions, rules, metrics
  my-model.schema.json     the mold: context, agent types, decisions, rules, metrics
  my-model.context.json    optional: initial values of the context
  agents/*.agent.json      one agent, or a generated population, per file
  scripts/*.lua            decision logic, rules, metrics, stop condition
```

Every model has a `README.md` explaining the question it explores, the context fields, each
agent type and its attributes, the decisions (eligibility, propensity, effects), the rules,
the metrics, and what results to expect. `agentsim validate` warns when it is missing. The
example models under `models/` follow this structure.

Run `agentsim validate my-model` after every change: it reports every problem it finds
(unknown keys, typos in types, missing scripts, bad expressions, misnamed hooks) at once.

Run `agentsim jsonschema my-model` to generate `.jsonschema/agent.json` and
`.jsonschema/context.json`, then add `"$schema": "../.jsonschema/agent.json"` to agent
files for autocompletion and validation in your editor.

Every JSON object accepts a `"description"` key. Use it: the schema is documentation.

## 2. Think in four layers

| Layer       | Question                                   | Where                                   |
|-------------|--------------------------------------------|-----------------------------------------|
| State       | What exists and what can be measured?      | `context`, `agentTypes.*.attributes`    |
| Perception  | What does each agent know?                 | `visibility`, `memory`, topology        |
| Behavior    | How does an agent choose?                  | `decisions` (eligible, propensity)      |
| Mechanics   | What actually happens in the world?        | effects, constraints, `rules`           |

Keep them separate. A common mistake is putting mechanics in behavior (e.g. an agent
script that "pays the other player"). Agents can only change themselves and the context;
the mechanics of interaction belong to rules.

## 3. The schema

```jsonc
{
  "name": "my-model",
  "simulation": { "maxTicks": 100, "stop": "scripts/stop.lua" },
  "context": {
    "interestRate": { "type": "float", "default": 0.09, "min": 0, "max": 0.5, "clamp": true },
    "moves": { "type": "map", "of": "string", "values": ["C", "D"] },
    "target": { "type": "float", "visibility": "hidden" }
  },
  "relations": { "owns": { "directed": true } },
  "groups": { "community": {} },
  "space": { "width": 100, "height": 100, "cellSize": 5 },
  "agentTypes": {
    "person": {
      "attributes": {
        "wealth": { "type": "int", "min": 0 },
        "score": { "type": "float", "visibility": "public" }
      },
      "memory": { "partnerMoves": { "type": "string", "capacity": 10 } },
      "activation": { "every": 1 },
      "choice": { "mode": "weighted", "idle": 0.2 },
      "spatial": false,
      "decisions": ["work", "invest"]
    }
  },
  "decisions": {
    "work": { "script": "scripts/work.lua", "propensity": "1" },
    "invest": {
      "script": "scripts/invest.lua",
      "eligible": "wealth >= 1000",
      "propensity": "sigmoid(2 * riskTolerance - 10 * ctx.interestRate)"
    }
  },
  "rules": [
    { "name": "market", "phase": "post", "script": "scripts/market.lua" },
    { "name": "aging", "phase": "post", "agentType": "person", "script": "scripts/aging.lua", "every": 12 }
  ],
  "metrics": [
    { "name": "wealth.gini", "agg": "gini", "agentType": "person", "attribute": "wealth" },
    { "name": "rate", "context": "interestRate" },
    { "name": "alice.wealth", "agent": "alice", "attribute": "wealth" },
    { "name": "custom", "script": "scripts/custom_metric.lua" }
  ],
  "output": { "log": "decisions", "sample": 0.01, "snapshotEvery": 50, "snapshotFinal": true }
}
```

### Fields and attributes

- Types: `int`, `float`, `bool`, `string`; context fields may also be `map` (with `of`).
- Use `int` for money and stocks: integer arithmetic never drifts.
- `min`/`max` are **hard constraints**: an effect that would break them is rejected (the
  transaction rolls back). With `"clamp": true` the value is clamped instead.
- `values` turns a string into an enum.
- Agent attributes are `private` by default; mark `public` what others may observe.
  Context fields are `public` by default; mark `hidden` what only rules may read.
- Context `merge` (`error` | `first` | `last`) decides what happens when two actors `set`
  the same context value in one phase. Prefer designs that avoid it: a map keyed by
  agent id (`ctx.moves.<id>`), or `add`, which never conflicts.

### Memory

Memories are bounded histories (ring buffers). Capacity is required, so a long run never
grows. Write with `fx.remember`, read with `self:last`, `self:recall`, `self:count`,
`self:mean`, `self:memlen` (or the same names in expressions). Memories are private.

### Activation and choice

- `activation.every: N` makes an agent act every N ticks (staggered by id);
  `activation.probability: p` makes it act with probability p each tick.
- `choice.mode`: `weighted` (roulette over propensities, the default), `softmax` (with
  `temperature`), or `max` (highest propensity; ties broken at random).
- `choice.idle`: weight (number or expression) of doing nothing. Without it, an agent
  with any positive propensity always acts.

## 4. Agent files

```jsonc
// a single named agent
{ "type": "person", "name": "alice", "attributes": { "wealth": 5000 },
  "groups": { "community": "north" }, "links": [{ "relation": "owns", "to": "bakery" }] }

// a population
{ "type": "person", "name": "people", "count": 100000,
  "attributes": {
    "wealth": { "dist": "normal", "mean": 3000, "std": 2500, "min": 0 },
    "riskTolerance": { "dist": "uniform", "min": 0, "max": 1 },
    "employed": { "dist": "bernoulli", "p": 0.9 },
    "class": { "dist": "choice", "values": ["low", "mid", "high"], "weights": [5, 4, 1] }
  },
  "groups": { "community": { "dist": "choice", "values": ["north", "south"] } },
  "position": { "x": { "dist": "uniform", "min": 0, "max": 100 } } }
```

Populations are generated from the seed, so they are reproducible too. To wire a
population into a network, use an `init` rule.

## 5. Scripts

### Decision scripts

All functions are optional except that at least one of `precondition`, `on_success`,
`on_failure` must exist. Every function receives `(self, world)`.

```lua
function eligible(self, world) return self.wealth >= world.ctx.businessCost end
function propensity(self, world) return self.riskTolerance * 2 end   -- number >= 0

-- The cost of trying; rejected as a whole if it breaks a constraint.
function precondition(self, world)
  if world.ctx.licenses == 0 then return false, "no licenses left" end
  return fx.add("self.wealth", -world.ctx.businessCost)
end

-- Did it work? A boolean, or a probability the engine draws against.
function outcome(self, world) return 0.2 + 0.6 * self.knowledge end

function on_success(self, world)
  local firm = fx.spawn("firm", { capital = 5000 })
  return { firm, fx.link("owns", self.id, firm), fx.remember("ventures", true) }
end

function on_failure(self, world)
  return fx.emit("venture_failed", { wealth = self.wealth })
end
```

Write a propensity either as an expression in the schema or as a Lua function, never
both. Expressions are much faster; use Lua when you need the topology or loops.

### Rules

```lua
-- world rule: run(world). Runs once per tick (or every N ticks) with full access.
function run(world)
  local effects = {}
  for _, id in ipairs(world:ids("player")) do ... end
  return effects
end

-- agent-scoped rule ("agentType" set): run(self, world), once per agent, in parallel.
function run(self, world)
  return fx.add("self.age", 1)
end
```

Rules are the model's authority. They may change any agent. If a rule produces an effect
the state cannot accept, the run stops with an error: check the state before changing it.

### Metrics and stop

```lua
function metric(world) return world:count("firm") / world:count("person") end
function stop(world) return world:count("person") == 0 end
```

### The `world` API

| Call                                                     | Returns                                        |
|----------------------------------------------------------|------------------------------------------------|
| `world.tick`, `world.ctx.<field>`, `world.ctx.<map>[key]` | values (hidden fields only for rules)          |
| `world:rand()`, `randint(a, b)`, `normal(mean, std)`, `chance(p)`, `pick(list)`, `shuffle(list)` | reproducible randomness |
| `world:agent(id)`                                        | read-only agent (public attributes only)       |
| `world:exists(id)`, `world:find(name)`                   | bool, id                                       |
| `world:neighbors(rel [, id])`, `world:incoming(rel [, id])` | ids                                         |
| `world:nearby(radius [, type [, id]])`                   | ids within radius (spatial types)              |
| `world:members(kind, name)`, `world:groups(kind [, id])` | ids, group names                               |
| `world:count/sum/mean/min/max/std/gini(type [, attr])`   | cached aggregate over a type                   |
| `world:group_count/sum/mean/min/max(kind, name [, attr])` | cached aggregate over a group                 |
| `world:sample(type, k)`                                  | k distinct random ids                          |
| `world:ids(type)`                                        | all ids (rules only)                           |
| `world:keys(mapField)`                                   | sorted keys of a context map                   |

Agents: `self.id`, `self.type`, `self.name`, `self.x`, `self.y`, `self.<attr>`,
`self:last(m)`, `self:recall(m, n)`, `self:count(m, v)`, `self:mean(m)`, `self:memlen(m)`.

### The `fx` API

| Effect                                   | Meaning                                                  |
|------------------------------------------|----------------------------------------------------------|
| `fx.set(path, value)`                    | set a value                                              |
| `fx.add(path, delta)`                    | add to a number (commutative)                            |
| `fx.transfer(from, to, amount)`          | move a quantity between two paths; total is conserved    |
| `fx.clear("ctx.<map>")`, `fx.delete("ctx.<map>.<key>")` | empty a map, remove an entry              |
| `fx.remember(memory, value [, id])`      | push to a memory                                         |
| `fx.spawn(type [, attrs [, {x, y, groups}]])` | create an agent; pass the result as the agent of later effects in the same list (link, remember, join, move, remove) |
| `fx.remove([id])`                        | remove an agent                                          |
| `fx.link(rel, a, b)`, `fx.unlink(rel, a, b)` | relations                                            |
| `fx.join(kind, name [, id])`, `fx.leave(kind, name [, id])` | groups                                |
| `fx.move(x, y [, id])`                   | move in space                                            |
| `fx.emit(name [, data])`                 | write an event to the log; changes nothing               |

Paths: `self.<attr>`, `ctx.<field>`, `ctx.<map>.<key>`, `agent.<id>.<attr>` (rules).
Agents may only target themselves, the context, and agents they spawn in the same list.

## 6. Pitfalls

- **Do not keep state in scripts.** Globals are frozen; top-level `local` variables are
  not, but values kept there between calls make runs depend on how agents are split
  between workers. Store state in attributes, memory or the context.
- **Never use `math.random`** (disabled). Use `world:rand()` and friends.
- **Decisions are simultaneous.** Two agents may both decide to buy the last unit; one
  transaction will be rejected at application time. That is scarcity working, not a bug.
  Count it with the automatic `decision.<type>.<name>.rejected` metric.
- **Propensities are weights, not probabilities.** With weights 2 and 1, the first
  decision is chosen two thirds of the time. Add `choice.idle` to allow doing nothing.
- **Attributes do not determine behavior; they shape it.** Prefer smooth propensities
  (`sigmoid`, weighted sums) to `if` thresholds, so heterogeneous agents react
  differently to the same context.
- **A single run proves little.** Use `--seeds 30` and read `aggregate.csv`, and vary
  parameters to see how sensitive conclusions are.

## 7. Workflow

1. Write the schema with two or three agents and `maxTicks` small.
2. `agentsim validate`, then `agentsim run --log decisions` and read `events.jsonl`.
3. Add metrics for the question you are asking before scaling up.
4. Scale the population with `count`, set `output.sample`, and run many seeds.
5. Move hot propensities from Lua to expressions; use sparse activation if needed.
