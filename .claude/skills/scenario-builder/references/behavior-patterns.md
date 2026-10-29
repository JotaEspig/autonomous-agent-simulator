# Behavior patterns

Formulas for common real-world behaviors, with the evidence behind them and how to write them
in agentsim. Use a pattern only when its story matches the behavior being modeled, and state the
parameter values you choose and why.

## Choosing between options

### Random utility (logit) choice

People do not always pick the best option; they pick better options more often. McFadden's
discrete choice model: `P(i) = exp(β·U_i) / Σ_j exp(β·U_j)`, with β the sensitivity to utility
differences (β → ∞ is a perfect optimizer, β → 0 is random).

agentsim: `choice.mode: "softmax"` with `temperature = 1/β`, and each decision's propensity
returns its utility `U_i`. With `weighted` mode the propensity is a weight, so use `exp(β·U)`
explicitly if you need logit.

### Propensity from several factors

Willingness to take an action rises smoothly with motives and falls with obstacles:
`p = p_max · sigmoid(b0 + Σ b_k·x_k)`. Normalize each `x_k` to a comparable scale (0 to 1, or a
ratio such as return / cost) so the coefficients can be read and justified.

agentsim: an expression, e.g. `"0.3 * sigmoid(-4 + 25 * roi + 2 * riskTolerance)"`.

### Needs before wants (priority)

People cover necessities (food, housing) before discretionary spending and investment. Model
it as a sequence, not as weights: first allocate the budget to the need, then decide what to do
with the rest.

agentsim: the need is a routine rule run before the decision phase; investment decisions are
eligible only with what is left.

### Satisficing

Herbert Simon: people search until they find an option above their aspiration level, and adjust
the aspiration after success (up) or failure (down). It fits searching for a job, a house or a
supplier better than full optimization does.

## Prices and markets

### Posted prices with inventory feedback

Shops set prices before they see the day's demand, and adjust them from sales: sold out, raise;
stock left over, cut in proportion to what is left. A price far from what competitors charge is
pulled back toward the market (law of one price, search by customers).

```lua
if stock == 0 then price = price * (1 + rise)
else price = math.min(price * (1 - cut * stock / offered), marketAverage * (1 - undercut)) end
```

### Order book (limit orders)

A buy limit order executes against the cheapest sell offer at or below its limit, at the
**seller's posted price**. Priority goes to the best price (highest buy limit), then to arrival
time (draw at random within a tick). Buyers never pay their limit unless the offer is at it.

### Reservation price learned from experience

A buyer's maximum price adapts: after failing to get what it needed, raise it in proportion to
the shortfall; after succeeding, relax it slightly to test whether it can pay less. Bound it by
what the buyer has.

```lua
if missing > 0 then limit = limit * (1 + raise * missing / needed)
else limit = limit * (1 - relax) end
limit = math.min(limit, wealth)
```

### Budget constraint

Nobody spends more than it has, or can borrow. Check affordability against actual prices
before committing, and have the market skip buyers who can no longer pay. Use `min: 0` on money
so the engine rejects overspending.

## Expectations and learning

### Adaptive expectations

`E_t = E_{t-1} + λ·(x_t − E_{t-1})`, with λ in (0, 1]: people update beliefs partially toward
what they observe. Small λ means sticky beliefs. Store `E` as an attribute, or use memory and
`self:mean(...)` for a moving average.

### Reinforcement

Repeat what worked: increase an action's propensity after a good outcome, decrease it after a
bad one (`w ← w + α·(reward − w)`). Store the weights as attributes; record outcomes in memory.

### Imitation and social influence

Granovetter's threshold model: an agent adopts a behavior when the share of its neighbors who
adopted it passes the agent's own threshold. Draw thresholds from a distribution for
heterogeneity.

agentsim: `world:neighbors(...)` and the neighbors' public attributes.

## Risk and time

### Loss aversion (prospect theory)

Losses weigh about twice as much as equal gains (Kahneman and Tversky, λ ≈ 2.25):
`v(x) = x^0.88` for gains, `−2.25·(−x)^0.88` for losses, relative to a reference point (usually
the current situation). Use it when agents compare a risky action with the status quo.

### Risk aversion (CRRA utility)

`u(c) = c^(1−γ) / (1−γ)` (log c when γ = 1); γ between 1 and 3 fits most household data.
Expected utility over outcomes, `Σ p_k·u(c_k)`, gives realistic aversion to gambles that risk a
large share of wealth.

### Discounting

Exponential: value of `x` in `t` ticks is `x·δ^t`, with δ derived from a yearly rate converted
to ticks. Present bias (hyperbolic, β-δ): `x·β·δ^t` for `t > 0`, β ≈ 0.7 is common. Present bias
explains under-saving and procrastination.

### Consumption and saving

Poorer households consume almost all of their income (marginal propensity to consume near 1);
richer ones save a larger share. Keep a buffer: people save more when income is uncertain
(precautionary saving).

## Life events

### Mortality: Gompertz law

Adult human mortality grows exponentially with age: yearly risk `a·e^(b·age)`, with `b ≈ 0.085
to 0.09` (risk doubles about every 8 years) and `a ≈ 5e-5`. That gives about 1% a year at 60
and 7% at 80. Per tick: divide the yearly risk by ticks per year.

### Births

Fertility depends on age (mostly 15 to 45), and falls with hardship and with income in modern
societies. Use a per-tick probability conditioned on those factors, and link parent and child
with a relation if inheritance or family matters.

### Random events

Shocks (droughts, epidemics, crises) arrive as a Poisson process: a fixed chance per tick
`p = 1 − e^(−rate·Δt)`, a duration, and an effect on context fields that agents perceive. Keep
the effect on the environment (harvest, needs), not on agents' outcomes directly.

## Time and units

- State what a tick is, and convert every rate to it (yearly 5% interest is about 0.0134% a day).
- Keep money in consistent units, and use `int` when exact accounting matters.
- Bound every quantity by a real limit: hunger by a death threshold, prices by budgets, stocks by
  capacity.
