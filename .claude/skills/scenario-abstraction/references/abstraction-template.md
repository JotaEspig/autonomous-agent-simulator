# Abstraction document template

Copy this structure to `docs/abstractions/<slug>.md`. Every section is required; write "none"
when a section does not apply, and say why. `/scenario-builder` implements from this document,
so anything left implicit here will be guessed there.

```markdown
# <Phenomenon>: abstraction

## Question

What the model should help answer, in one or two sentences.

## Phenomenon and scope

- **Observed phenomenon**: plain description.
- **Aspects in scope**: ...
- **Out of scope**: ... (and why it should not change the answer)

## Stylized facts (validation targets, never encoded)

| # | Fact | Source |
|---|------|--------|
| F1 | ... | data / paper / user |

## Why ladder

For each fact, the chain from the aggregate down to individual behavior.

- **F1**: aggregate -> why -> ... -> individual rule (actor, motive, constraint, information).

## Competing hypotheses

| # | Mechanism | Which facts it explains | How the model could tell them apart |
|---|-----------|-------------------------|-------------------------------------|

## Actors

| Actor type | Real counterpart | Count / scale | Why it is a separate type |
|------------|------------------|---------------|---------------------------|

## Attributes (one table per actor type)

### <Actor type>

| Attribute | Class (trait/state/belief/memory) | Unit | Range | Initial distribution (and why) | Observable by | Changed by |
|-----------|-----------------------------------|------|-------|--------------------------------|---------------|------------|

## Shared state and environment

| Field | Meaning | Unit | Who can observe it | Changed by (actors, institutions, exogenous) |
|-------|---------|------|--------------------|---------------------------------------------|

## Behavior (one section per actor type)

### <Actor type>

- **Perception**: ...
- **Goals (priority order)**: ...
- **Routines** (happen regardless of choice): ...
- **Decisions** (strategic choices): ...

For each rule:

| Rule | Formula | Real mechanism | Direction of effects | Bounds | Parameters (value, source, sensitivity range) |
|------|---------|----------------|----------------------|--------|------------------------------------------------|

- **Learning / belief update**: ...
- **Edge cases**: broke, empty market, no partner, ...

## Interactions

- **Arenas and meeting protocols**: who meets whom, information available, how conflicts resolve.
- **Institutions**: fixed rules and what they do.
- **Conservation**: what is conserved; where things are created or destroyed.

## Timing

- **One tick** = ...; horizon = ...
- **Order of events within a tick**: 1. ... 2. ...
- **Rate conversions**: ...

## Generative sufficiency

| Fact | Micro mechanism expected to produce it | Confidence |
|------|----------------------------------------|------------|

- **Unexpected predictions** of the abstraction, and whether they are plausible: ...
- **Falsification**: results that would show the abstraction is wrong: ...

## Simplifications and their bias

| Simplification | What is left out | Expected bias on the answer |
|----------------|------------------|-----------------------------|

## Open decisions and next behaviors

- Decisions still open for the user.
- Behaviors worth adding next.
```
