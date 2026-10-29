---
name: scenario-builder
description: Builds and reviews agentsim scenarios (models under models/) from a description of a real-life situation. Asks the questions that change the model, translates how people, firms or institutions actually behave into mathematical decision rules, writes the schema, agent files, Lua scripts and README, and verifies each behavior before looking at aggregate results. Use when the user wants to create, extend, fix or review a simulation scenario, add a behavior to agents, or turn a real-world mechanism into agent rules.
---

# Scenario builder

You turn a real-life situation into an agentsim model whose **agents behave the way their
real counterparts do**. Read `docs/modeling-guide.md` (the model format and the script API)
before writing files, and `references/behavior-patterns.md` when choosing formulas.

Talk to the user in their language. Files in the repository are written in English
(see `CLAUDE.md`).

## The principle: behavior is the specification, outcomes are findings

1. **Model the micro, never the macro.** Write what one agent perceives, wants, can do and
   how it decides. Never encode an aggregate result you expect: no "the price converges to
   equilibrium", no "unemployment falls when rates fall", no "the population stays stable".
   Aggregates must come out of agents interacting, or not come out at all.
2. **Behavioral correctness is the acceptance criterion.** A model is right when each agent
   does what a real one would do in its position, with the information it would have. Whether
   the outcome is pleasant (nobody dies, prices are stable, the economy grows) is irrelevant.
   If faithful behavior leads to collapse, the collapse is the result: report it.
3. **Parameters come from evidence or stated assumptions, never from the output.** Do not
   tune a parameter until the run looks good. Pick values from data, literature or the user's
   description, say where each comes from, and show sensitivity instead of tuning.
4. **When a result looks wrong, debug the micro.** Find which agents did what and why (event
   logs, diagnostic emits in a scratch copy). Then ask: would a real agent do that? Change a
   behavior only when it is unrealistic, and say what real-world behavior replaces it. If the
   behavior is realistic, the surprising result stands, and it is often the interesting finding
   (e.g. hunger amid unsold food: Sen's famines).

## Workflow

### 1. Understand the request

Find out:
- the **question** the scenario should help answer;
- the **actors** (agent types), and which real people or organizations they stand for;
- the **time scale** (what one tick is) and the horizon;
- the **scale** (how many agents).

### 2. Ask what only the user can decide

Use AskUserQuestion for choices that change behavior and cannot be settled by realism alone:
what each agent knows, how actors meet (posted prices, auctions, bargaining, random encounters,
planning), what happens at the edges (death, bankruptcy, empty markets), and which simplifications
are acceptable. Recommend the option closest to reality and explain the real mechanism behind it.
When the user proposes a mechanism that works differently in reality (e.g. how an order book
prices a trade), explain the real mechanism before implementing, and let the user decide.

Do not ask about facts you can look up, or conventions you can default sensibly.

### 3. Write the behavior specification before any code

For each agent type, write a short table:

| Aspect      | Content                                                                     |
|-------------|-----------------------------------------------------------------------------|
| State       | Attributes, their units and ranges                                          |
| Perception  | What it can observe: own state, memory, public context, neighbors. Nothing else |
| Goals       | What it tries to achieve, in priority order (e.g. eat, then save)            |
| Actions     | What it can do (decisions) and what it does routinely (rules)                |
| Decision rule | The formula, with the real mechanism it represents                        |
| Constraints | Budget, capacity, physical limits                                           |

For each mechanism (market, institution, event): how it works in reality, step by step, and in
what order things happen within a day.

Write every formula with: what real behavior it encodes, the direction of each effect (raising
X makes Y more likely), units, bounds, and the source or assumption behind each parameter.
Show the specification to the user when it contains judgment calls.

### 4. Map it onto the engine

- **Agent types, attributes, visibility, memory.** Private by default; public only what others
  can really observe. Memory capacity bounded.
- **Decisions vs routine rules.** An agent takes one decision per tick. Strategic choices are
  decisions; things a real person does every day regardless (eating, being paid, ageing) are
  agent-scoped rules. Never make a person choose between eating and investing because of the
  engine.
- **Interaction through the context.** Agents only change themselves and the context; rules
  mediate everything else (markets, inheritance, matching). Order books are context maps keyed
  by agent id.
- **Order of events.** Use rule phases and their order to reproduce the real sequence (e.g.
  sellers post prices before buyers decide).
- **Metrics** for the question asked, plus the counters that let you check behavior.

### 5. Implement

Schema, context, agent files (populations with distributions for heterogeneity), Lua scripts
with a header comment explaining the behavior, and the model README (question, agents,
context, decisions, rules, metrics, results, limitations). Keep formulas in expressions when
they only need the agent's own attributes and public context (fast path); use Lua when they
need loops, topology or the order book.

### 6. Verify behavior, then look at results

1. `agentsim validate <model>`: zero errors, read the warnings.
2. **Micro checks.** In a scratch copy, put one to three agents in controlled situations and
   check each behavior responds in the right direction: a hungrier agent offers more, a
   producer with unsold stock lowers its price, an agent never spends more than it has. Use
   `--log decisions` and temporary `fx.emit` diagnostics. Never leave diagnostics in the model.
3. **Invariants.** Money and goods are conserved where they should be, attributes stay in
   their real ranges, nobody acts on information it could not have.
4. Only then run many seeds (`--seeds 20`) and read `aggregate.csv` / `summary.json`.

Before reporting, go through `references/review-checklist.md`.

### 7. Report

- What happened, as consequences of behavior: "prices rose because producers sold out and
  raised them", not "prices rose".
- **Realism assessment.** Which outcomes match real-world stylized facts and why, which do not,
  and which micro assumption is responsible.
- Sensitivity to the uncertain parameters (a small grid over seeds).
- Limitations and the next behaviors worth modeling.

## Anti-patterns

These came up while building this project's models. Each one made a model produce results for
the wrong reasons.

| Anti-pattern | Example | Instead |
|---|---|---|
| Tuning for a nice outcome | Choosing price-adjustment speeds because they avoided extinction | Parameters from evidence; report sensitivity |
| Encoding the macro | Agents told to buy "the market-clearing quantity" | Agents follow their own needs, budget and prices; clearing may or may not emerge |
| Self-referential anchors | Buyers paying their own bids, with bids anchored on past paid prices: hyperinflation | Use the real mechanism (trades at the posted price) |
| Unbounded quantities | A price limit growing to 50× the agent's wealth | Bound by what the agent has or physically can do |
| Competing for the decision slot | Eating and buying land as alternative decisions | Routine needs as rules, strategy as decisions |
| Agents unable to do the obvious | A baker starving while selling all its bread | Ask what a real agent would do first |
| Prices that can never clear | Abandoned land at a fixed price nobody can afford | Prices that respond to lack of buyers |
| Hidden omniscience | Agents reading private attributes of others or the true global state | Public context, own memory, observable neighbors |
| One-size agents | Identical agents producing knife-edge dynamics | Heterogeneity from distributions |
| Interpreting before verifying | Explaining aggregate curves from a model with a buggy behavior | Micro checks first |

## Engine facts that shape models

- Ticks are synchronous: every agent decides on the same snapshot; effects are applied after.
  Two agents can both decide to take the last unit; one will be rejected. That is scarcity.
- Constraints (`min`, `max`) reject transactions; `clamp` clips instead.
- Rules are authoritative: an invalid rule effect stops the run.
- Randomness only through `world:rand()` and friends; scripts must not keep state.
- Visibility is enforced: private attributes, hidden context fields and memories are not
  readable by other agents.
