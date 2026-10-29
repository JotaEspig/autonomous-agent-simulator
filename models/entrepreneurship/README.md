# Entrepreneurship

A small economy of 10,000 people who either work for a wage or risk their savings to open a
firm. Firms earn or lose money depending on demand and the cost of capital, pay dividends to
their owners, and close when insolvent. A central bank moves the interest rate to keep the
number of firms per person near a target that only the bank knows.

The model shows that a monetary policy effect does not have to be hard-coded: lower rates make
opening a business more attractive and more likely to work, but each person still decides based
on their own wealth, knowledge and risk tolerance.

```sh
agentsim run models/entrepreneurship --seeds 8
```

## Context

| Field          | Type  | Default | Meaning                                                 |
|----------------|-------|---------|---------------------------------------------------------|
| `interestRate` | float | 0.09    | Yearly base rate set by the central bank (clamped to [0, 0.5]) |
| `demand`       | float | 1.0     | Aggregate demand index (clamped to [0.2, 2]); 1 is normal |
| `businessCost` | int   | 5000    | Capital needed to open a firm                           |
| `wage`         | int   | 100     | Income per tick of working                              |

## Agents

### `person` (10,000, `agents/people.agent.json`)

| Attribute       | Initial distribution                  | Meaning                                       |
|-----------------|---------------------------------------|-----------------------------------------------|
| `wealth`        | normal(3000, 2500), at least 0        | Cash; can never go negative (`min: 0`)        |
| `knowledge`     | uniform(0, 1)                         | Business knowledge; raises the chance of success |
| `riskTolerance` | normal(0.4, 0.2), clamped to [0, 1]   | Willingness to take risks                     |

Memory: `ventures`, the outcomes (true/false) of the person's last 5 ventures.

### `firm` (created during the run)

| Attribute | Meaning                                  |
|-----------|------------------------------------------|
| `capital` | Money invested in the firm (public)      |
| `age`     | Ticks since the firm was opened          |

Firms do not decide anything; the `operate` rule runs them. The owner is linked through the
directed relation `owns` (person → firm).

### `centralBank` (1, `agents/central-bank.agent.json`)

| Attribute        | Value | Meaning                                                  |
|------------------|-------|----------------------------------------------------------|
| `aggressiveness` | 50    | How strongly it reacts to deviations from the target    |
| `target`         | 0.05  | Firms per person it aims for; private to the bank       |

The bank acts every 6 ticks and has an idle weight of 1, so it often holds the rate.

## Decisions

| Decision        | Who         | Eligible                    | Propensity                                    | Effects |
|-----------------|-------------|-----------------------------|-----------------------------------------------|---------|
| `work`          | person      | always                      | 1                                             | `wealth += wage` |
| `open_business` | person      | `wealth >= businessCost`    | `0.3 × sigmoid(−4 + 3·knowledge + 3·riskTolerance − 20·interestRate + 2·(demand − 1) − past failures)` | see below |
| `cut_rate`      | centralBank | rate >= 0.0025              | `max(0, target − firms/person) × aggressiveness` | rate −0.25 pp |
| `raise_rate`    | centralBank | always                      | `max(0, firms/person − target) × aggressiveness` | rate +0.25 pp |

`open_business` goes through every step of a decision:

1. **Precondition**: pays `businessCost` up front. The engine rejects it if the person can no
   longer afford it.
2. **Outcome**: succeeds with probability `0.2 + 0.6·knowledge − interestRate`.
3. **On success**: spawns a `firm` with `capital = businessCost`, links `owns` from the person,
   and remembers a success.
4. **On failure**: the money is lost; remembers a failure and emits a `venture_failed` event.

Past failures lower the person's propensity to try again.

## Rules

| Rule      | Phase | Scope    | What it does                                                        |
|-----------|-------|----------|---------------------------------------------------------------------|
| `economy` | pre   | world    | Mean-reverting random walk of `demand`                              |
| `operate` | post  | per firm | Profit = capital × (1.2% × demand + noise − rate/12). Half of a profit goes to the owner. A firm whose capital would drop to 0 or below closes (removed, `firm_closed` event). |

The simulation stops early (`scripts/stop.lua`) if, after tick 24, there are no firms and nobody
can afford to open one.

## Metrics

`firms`, `wealth.mean`, `wealth.gini`, `capital.mean`, `interestRate`, `demand`, plus population
per type and decision counters such as `decision.person.open_business.failure`.

## Results

Over 240 ticks the number of firms grows steadily (about 6,600 at the end). The central bank
reacts by raising the rate from 9% to about 17%. Mean wealth grows, and the Gini coefficient of
wealth falls from 0.40 to about 0.13, because wages and dividends compound for everyone.

## Things to try

- Change the bank's `target` or `aggressiveness` and compare `aggregate.csv` across seeds.
- Remove the bank (delete its agent file) to see the economy with a fixed rate.
- Raise `businessCost` and watch inequality and the firm count change.
