---
name: scenario-abstraction
description: Abstracts a real-world phenomenon (or one or more aspects of it) into the individual behaviors that cause it - actors, attributes, perceptions, decision rules written as math, interactions and timing - without implementing anything. It asks "how does this actually work, and how can I abstract it into attributes and math?", and models the behavior that generates a pattern (why each buyer buys less when the price rises) instead of the pattern itself (the demand curve). Produces an abstraction document that /scenario-builder later implements. Use when the user wants to understand, decompose or abstract a situation, mechanism or phenomenon before modeling it, or asks what agents, attributes and rules a scenario would need.
---

# Scenario abstraction

You take a phenomenon from the real world and find the **generative mechanism** behind it: who
acts, what each one has, knows, wants and can do, how each one decides, and how their actions
meet. You write that down as attributes and math. You do **not** build the model: that is the
job of `/scenario-builder`, which reads your abstraction document.

Talk to the user in their language. The abstraction document is written in English
(see `CLAUDE.md`).

The question you keep asking yourself:

> **How does this actually work, and how can I abstract it into attributes and math?**

## The principle: abstract the cause, not the shape

What people notice about a phenomenon is its **shape**: a demand curve, a traffic jam, a bank
run, a segregated city, a price bubble, unemployment. The shape is the **explanandum**, the thing
the abstraction must be able to *produce*. It is never a component of the abstraction.

| The shape (do not map it) | The behavior that causes it (map this) |
|---|---|
| Demand curve slopes down | Each buyer has a budget and a reservation price for each extra unit, which falls as it already has more; buyers differ. At a higher price fewer buyers find the unit worth it. The curve is the distribution of reservation prices. |
| Supply curve slopes up | Each producer has a marginal cost that rises with output (capacity, worse inputs, overtime); it sells a unit only when the price covers that unit's cost. |
| Phantom traffic jam | Each driver keeps a safe gap that depends on speed and reacts to the car ahead after a delay; a small brake gets amplified down the line. |
| Bank run | Each depositor withdraws when it expects enough others to withdraw that the bank fails before it can; expectations come from what it sees others doing. |
| Residential segregation | Each household moves when the share of similar neighbors falls below its own mild tolerance threshold. |

The test: if an aggregate quantity (a curve, an equilibrium, a rate, a trend) appears in your
abstraction as something an agent knows, targets or is told to produce, you mapped the shape.
Go one level down and ask why each individual does what adds up to it.

## Workflow

### 1. Pin down the phenomenon and the question

- **Phenomenon**: what is observed, in plain words.
- **Question**: what the user wants to learn (why it happens, what changes it, what a policy
  would do). The question decides which aspects matter: abstract only what can change its answer.
- **Aspects in scope**: the user may want one aspect (only the price formation of a market) or
  several (price, entry of new firms, and credit). List them and keep the boundary explicit.
- **Stylized facts**: the observable regularities the real phenomenon shows (e.g. prices are
  sticky, unemployment lasts longer for older workers, jams travel backwards). They are the
  *validation targets* of the future model. Write them down; never encode them.

Look up facts and literature you can check (WebSearch) instead of asking the user.

### 2. Run the "why ladder" down to individuals

Start from each stylized fact and ask "why?" until the answer is about one actor's motive,
constraint or information, not about the aggregate:

```
Prices rise in a drought.
  why? -> there is less bread for the same number of buyers.            (still aggregate)
  why do buyers pay more? -> a hungry buyer who failed to buy yesterday
     raises the most it is willing to pay, up to what it has.             (individual: rule)
  why do bakers ask more? -> a baker who sold out early raises tomorrow's
     price; one with leftovers cuts it.                                   (individual: rule)
```

Stop at a rung when the answer is a behavior you can state for one actor with what it really
knows. If two rungs could explain the same fact, keep both as **competing hypotheses**: the
model can later tell them apart.

### 3. Identify the building blocks

| Block | Question | Examples |
|---|---|---|
| **Actors** | Who makes a choice that changes the outcome? Group them into types only when they decide the same way. | households, firms, banks, drivers, a regulator |
| **Resources / stocks** | What accumulates and depletes? | money, inventory, land, skill, health, debt, reputation |
| **Flows** | What moves between stocks and who triggers it? | wages, purchases, harvests, loans, information |
| **Arenas** | Where and how do actors meet? | posted-price shop, order book, auction, bargaining, job interviews, a network, physical space |
| **Institutions** | Fixed rules nobody chooses in the run | taxes, minimum wage, contract law, a central bank rule |
| **Environment** | Exogenous drivers | weather, technology, shocks, demography |

Every actor in the list must correspond to a real person or organization, and every arena to a
real way of meeting. An "auctioneer that finds the clearing price" is not a real actor unless
the real market has one.

### 4. Abstract each actor into attributes

For each actor type, classify every attribute; the class tells `/scenario-builder` how to
implement it:

| Class | Meaning | Example |
|---|---|---|
| **Trait** | Fixed heterogeneity, drawn once from a distribution | risk aversion, patience, tolerance threshold |
| **State** | Changes through actions, interactions and time | money, stock, hunger, age, employed |
| **Belief** | What the actor expects; updated from experience | expected price, expected wage, trust in a bank |
| **Memory** | Bounded record of past observations | last 10 prices paid, who hired it |

For each attribute give: unit, range, initial distribution (and why that shape), **who can
observe it** (private, public, neighbors only), and **what changes it**.

Heterogeneity is usually what produces the shape (a demand curve is the spread of reservation
prices). Ask which traits really differ between people and draw them from a distribution;
identical agents tend to produce knife-edge, all-or-nothing dynamics.

### 5. Abstract behavior into math

For each actor, write what it does as a function of **only what it perceives**:

- **Perception**: own state, own memory, public information, observable neighbors. Nothing it
  could not know in reality (other people's private reservation prices, the true global state).
- **Goals in priority order**: needs before wants (eat, then save, then invest).
- **Actions**: strategic choices (decisions) versus routines that happen anyway (eating, ageing,
  being paid).
- **Decision rule**: a formula with
  - the real mechanism it encodes, in one sentence;
  - the **direction of each effect** (more hunger -> higher bid);
  - units and bounds (never spend more than it has);
  - each parameter with its source (data, literature, user, or a stated assumption) and a
    plausible range for sensitivity.
- **Learning**: how beliefs and propensities change with outcomes.

Choose functional forms from the real story, not from convenience. The pattern library in
`.claude/skills/scenario-builder/references/behavior-patterns.md` (logit choice, satisficing,
adaptive expectations, loss aversion, inventory-driven pricing, thresholds, Gompertz mortality...)
lists forms with their evidence; reuse them when the story matches and say why.

### 6. Abstract interactions and timing

- **Meeting protocol**: who meets whom, with what information, and how conflicts resolve (two
  buyers want the last unit: who gets it, and why that is how it works in reality).
- **What is conserved**: money, goods, people. Where things are created or destroyed.
- **Order of events within one time step**, matching the real sequence (prices are posted before
  customers arrive; wages are paid before rent is due).
- **Time step and horizon**: what one tick is, and every rate converted to it.

### 7. Check generative sufficiency (thought experiment)

Before handing off, trace each stylized fact through the abstraction by reasoning, step by step,
about what the agents would do:

- Which micro rules could make this fact emerge? If none can, a mechanism is missing (go back
  to step 2). If a fact can only appear because you wrote it in, you mapped the shape.
- What would the abstraction predict that is **not** in the stylized facts? Is it plausible?
  This is how you catch rules that are wrong but happen to fit.
- What result would **falsify** the abstraction? Write it down.
- Edge cases: what does each actor do when it is broke, the market is empty, nobody hires,
  everyone wants the same thing? Real actors do something; so must the abstraction.

### 8. Choose the level of abstraction

Keep a mechanism only if removing it could change the answer to the question. For every
simplification, state what is left out and how it would bias the result. Prefer the smallest
set of behaviors that can generate the stylized facts, and list the next ones to add.

### 9. Ask what only the user can decide

Use AskUserQuestion for choices that change the mechanism and cannot be settled by realism or
literature: the aspects in scope, which competing hypothesis to model first, what actors know,
which institutions to hold fixed. Recommend the option closest to reality and explain the real
mechanism behind it. Do not ask about facts you can look up.

### 10. Write the abstraction document and hand off

Write the document following `references/abstraction-template.md` to
`docs/abstractions/<slug>.md` (create the folder if needed), and show the user the judgment
calls. Do not write schema, agent files, scripts or models. When the user approves, suggest
running `/scenario-builder` with that document as its specification.

`references/worked-examples.md` has complete abstractions (supply and demand, a bank run,
segregation) showing the level of detail expected.

## Anti-patterns

| Anti-pattern | Example | Instead |
|---|---|---|
| Mapping the shape | "Demand: `q = a - b·p`" as the buyers' rule | Each buyer's reservation price and budget; the curve is their distribution |
| Targeting the equilibrium | Firms "set the market-clearing price" | Firms adjust from what they observe (unsold stock, lost customers) |
| Aggregate as an agent attribute | A global "market sentiment" knob that drives everyone | Each agent's beliefs, formed from what it sees, possibly from others' visible actions |
| Representative agent | One "average household" standing for all | A population with heterogeneous traits |
| Hidden omniscience | Workers knowing every firm's wage offer | What they can learn: ads, their network, their own search |
| Fictional actors | A Walrasian auctioneer in a street market | The real meeting protocol |
| Rules justified by the output | "Buyers raise bids by 5% because then prices stabilize" | Rules justified by how people behave; the output is a finding |
| Detail with no bearing on the question | Modeling commuting routes to study wage bargaining | Keep what can change the answer; list the rest as limitations |
| Behavior without a constraint | Consumers who want more and can always buy | Budgets, capacities, time, physical limits |
| Single explanation | Assuming one cause when several are plausible | Competing hypotheses the model can test |
