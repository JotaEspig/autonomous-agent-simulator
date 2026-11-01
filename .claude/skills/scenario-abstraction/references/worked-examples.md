# Worked examples

Condensed abstractions showing the expected reasoning: start at the shape, climb down the why
ladder, and end with attributes and math for individuals. They are illustrations, not
specifications to copy; parameters are indicative.

## 1. Supply and demand in a bread market

**Shape (do not map)**: a downward demand curve, an upward supply curve, a price that moves
toward where they cross, shortages and rising prices when supply falls.

**Why ladder**

- Demand slopes down -> at a higher price, fewer people buy -> because each buyer buys a unit
  only if it is worth more to it than the price, and that value differs across people and falls
  with each extra unit it already has (diminishing marginal utility) -> and nobody can spend more
  than it has (budget).
- Supply slopes up -> at a higher price, more is offered -> because each baker's cost per extra
  loaf rises (an oven has capacity; more output means overtime or worse inputs) and it bakes a
  loaf only if it expects to sell it above that cost.
- The price moves toward the crossing -> no one computes it -> bakers who sell out raise
  tomorrow's price, bakers left with stock cut it; buyers who went without raise what they are
  willing to pay.

**Actors**: households (buyers), bakers (sellers).

**Attributes**

| Actor | Attribute | Class | Distribution / dynamics | Observable by |
|---|---|---|---|---|
| Household | money | state | lognormal (income inequality) | private |
| Household | baseValue `v0` | trait | lognormal; how much the first loaf is worth to it | private |
| Household | hunger | state | grows each tick, falls when it eats | private |
| Household | maxPrice | belief | adapts after success or failure | private |
| Baker | posted price | state | adapts to sales | public |
| Baker | cost parameters `c0`, `k` | trait | normal around input costs | private |
| Baker | stock | state | baked minus sold; perishes | private |

**Math**

- Value of the n-th loaf today: `v(n) = v0 · (1 + hunger) · δ^(n-1)`, `δ < 1` (diminishing
  value). A household buys the n-th loaf at posted price `p` if `v(n) ≥ p` and `money ≥ p`.
  Summing over households at each `p` gives the demand curve: it is the distribution of `v`,
  not a rule anyone follows.
- Baker's marginal cost: `mc(q) = c0 + k·q`. It bakes up to `q*` where `mc(q*) = expected price`.
  Summing over bakers gives the supply curve.
- Price adjustment (posted prices with inventory feedback): sold out -> `p ← p·(1 + r)`;
  leftovers -> `p ← p·(1 − c·leftover/offered)`.

**Arena**: posted prices. Bakers post before buyers arrive; buyers visit in random order and
buy at the posted price from the cheapest baker they know with stock.

**Generative sufficiency**: a drought raises `c0` -> bakers bake less -> more bakers sell out ->
prices rise; households with low `v0` or little money stop buying first -> the hungry poor go
without even though the market "works" (Sen's entitlement failure, a non-obvious prediction).

## 2. Bank run

**Shape**: deposits stable for a long time, then a sudden collective withdrawal that can break
a solvent bank.

**Why ladder**: the bank fails -> it cannot pay everyone at once (it lent most deposits out) ->
too many withdraw together -> each depositor withdraws when it expects enough others to
withdraw that its money is at risk -> expectations form from what it can see: news, queues at
the branch, what friends do.

**Actors**: depositors, the bank.

**Attributes**: depositor `deposit` (state), `liquidityNeed` (trait: chance of needing cash for
real reasons), `threshold` (trait: share of withdrawing contacts that makes it panic, drawn from
a distribution), `beliefFail` (belief). Bank `reserves` (state), `reserveRatio` (institution).

**Math**

- Belief update from what it sees: `beliefFail ← beliefFail + λ·(observedWithdrawShare − beliefFail)`,
  where the observed share is among its network contacts (Granovetter threshold dynamics).
- Withdraw if `needsCash` (Bernoulli `liquidityNeed`) or `beliefFail > threshold`.
- Bank pays withdrawals in arrival order until reserves run out; then it fails.

**Generative sufficiency**: with heterogeneous thresholds, a few low-threshold depositors can
cascade through the network; with uniform thresholds nothing happens until a knife edge.
Competing hypotheses: runs triggered by fundamentals (bad loans) vs by pure coordination
(Diamond-Dybvig); the model separates them by holding the bank's assets sound.

## 3. Residential segregation

**Shape**: strongly segregated neighborhoods.

**Why ladder**: neighborhoods are uniform -> households moved until they were among similar
people -> each household moves when the share of similar neighbors falls below its own threshold
-> the threshold can be mild (most people accept being a minority), but each move changes
neighbors' shares and can push them over their thresholds (Schelling).

**Actors**: households on a grid.

**Attributes**: `group` (trait), `tolerance` (trait: minimum share of similar neighbors, e.g.
uniform 0.2 to 0.5), `position` (state), `movingCost` (trait).

**Math**: unhappy if `similarNeighbors / neighbors < tolerance`; an unhappy household moves to a
random vacant cell it can afford where it would be happy, with probability decreasing in
`movingCost`.

**Generative sufficiency**: mild individual preferences produce strong aggregate segregation;
the non-obvious prediction is that segregation is not evidence of strong preferences. Falsified
if segregation stayed low with realistic tolerances and moving costs.
