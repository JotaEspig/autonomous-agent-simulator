# Bread island

Supply and demand on an island where bread is the only food. Everyone earns the same fixed
wage and must eat. Anyone who saves enough can buy a plot of the island's limited farmland and
become a bread producer. Bread is sold in a daily auction and expires at the end of the day, and
plots change hands in a land market. Hungry people offer more for bread, and people who go
three days without any bread die.

The question: with scarce land and perishable food, how do prices, production, inequality and
population find (or fail to find) a balance?

```sh
agentsim run models/bread-island --seeds 20
```

## A day on the island

| Phase | Step               | What happens                                                                 |
|-------|--------------------|------------------------------------------------------------------------------|
| pre   | `new_day`          | Daily counters (deaths, births) reset                                       |
| pre   | `morning`          | Everyone receives the wage; each plot bakes `breadsPerLand` breads           |
| pre   | `quotes`           | The cheapest price asked by a producer is published                          |
| pre   | `orders`           | Everyone bids for up to 3 breads; producers offer their stock                |
| pre   | `bread_market`     | Double auction: bread changes hands                                          |
| decide | decisions         | People may buy land from the island, bid for someone's plot, or offer a plot |
| post  | `land_market`      | Land bids and offers are matched                                            |
| post  | `metabolism`       | Hunger, starvation, daily spending, repricing, expired bread, births         |

Eating and selling bread are routine, not choices: they are agent-scoped rules that run for
everyone every day. Decisions are kept for strategic choices about land, because the engine
lets an agent take one decision per tick, and nobody should have to choose between eating and
investing.

## Agents

One type, `islander`. People with no land are **consumers**, and people with at least one plot
are **producers**. Everyone starts as a consumer, and the role changes as plots are bought and
sold. Producers are people too: they earn the wage, get hungry and buy bread at the market,
even their own.

| Attribute         | Visibility | Meaning                                                      |
|-------------------|------------|--------------------------------------------------------------|
| `wealth`          | private    | Savings (never negative)                                     |
| `hunger`          | public     | 0 after a full day of bread; grows otherwise                 |
| `daysWithoutFood` | private    | Consecutive days without any bread                           |
| `ateToday`        | private    | Bread eaten today                                            |
| `income`, `spent` | private    | Money received and spent today (for the 80% rule)            |
| `lands`           | public     | Plots owned; 0 = consumer                                    |
| `stock`           | private    | Bread baked today and not yet sold                           |
| `askPrice`        | public     | Price the producer asks per bread                            |
| `riskTolerance`   | private    | Willingness to put savings at risk (0 to 1)                  |
| `ambition`        | private    | Drive to own and expand a business (0 to 1)                  |

Memory: `salePrices`, the prices paid by the producer's last 5 customers.

`agents/islanders.agent.json` creates 100 people with savings drawn from normal(50, 40)
(at least 0) and uniform personalities.

## Context

Parameters (in `island.context.json`):

| Field              | Default | Meaning                                                        |
|--------------------|---------|----------------------------------------------------------------|
| `salary`           | 10      | Wage per day, the same for everyone, forever                    |
| `landsFree`        | 10      | Plots the island still has (all 10 at the start)                |
| `islandLandPrice`  | 100     | Price of a plot bought from the island                         |
| `breadsPerLand`    | 10      | Bread per plot per day                                          |
| `productionCost`   | 0.5     | Share of each sale that is cost; the producer keeps the rest as profit |
| `basePrice`        | 2       | The island's reference price for bread                         |
| `hungerPremium`    | 1       | Extra a person is willing to pay per point of hunger           |
| `marginalDiscount` | 0.1     | Each extra bread of the day is worth 10% less than the first    |
| `outbid`           | 0.1     | How far above the cheapest posted price buyers go under scarcity |
| `priceDrop`        | 0.2     | How much a producer cuts its price after a day with unsold bread |
| `minPrice`         | 0.1     | Price floor                                                    |
| `maxBreadsPerDay`  | 3       | Most bread a person eats a day; eating 3 resets hunger to 0     |
| `hungerPerDay`     | 3       | Hunger added on a day without a full ration; each bread removes 1 |
| `starvationDays`   | 3       | Days in a row without any bread that kill                      |
| `birthChance`      | 0.01    | Daily chance that a person with zero hunger has a child         |
| `landHorizon`      | 15      | Days of profit a plot is worth to an average trader            |

State published by the rules: `marketPrice` (average price paid yesterday), `lowestAsk`,
`breadSold`, `breadWasted`, `unmetDemand`, `sellRate`, `scarcity`, `landPrice`, `bestLandBid`,
`deaths`, `births`, and the order books `breadBids`, `breadAskQty`, `breadAskPrice`, `landBids`
and `landAsks`. Each agent writes only its own keys, so orders never conflict.

## The bread market

**Buyers.** A person is willing to pay `basePrice + hungerPremium × hunger` for the first bread
of the day, and 10% less for each further one. Since buyers pay what they bid, nobody offers
more than needed. A bid is capped at the cheapest posted price, marked up by
`outbid × scarcity` when bread was short yesterday, so the hungriest can outbid others only
when bread is scarce. A person secures the first bread before the next ones: each bid is also
limited by the savings not committed to earlier breads.

**Matching.** Bids are served from the highest. Each buys the cheapest bread still on offer,
if its price covers the ask. The buyer pays its bid. The producer keeps 50% as profit, and the
other 50% is production cost and leaves the economy.

**Producers' prices.** After the day, a producer asks the average price paid by its last 5
customers. If bread was left over, it takes the lower of that average and its current price,
and cuts it by `priceDrop`. Unsold bread expires.

## Decisions (land)

Every person may take one of these each day, or do nothing (idle weight 1):

| Decision          | Eligible                                   | Propensity grows with                                   | Effect |
|-------------------|--------------------------------------------|----------------------------------------------------------|--------|
| `buy_island_land` | island has free plots and `wealth >= 100`  | expected return (price × profit share × yield × sell rate / plot price), scarcity, risk tolerance, ambition, own hunger | Pays 100 and takes a plot from the island; becomes a producer |
| `bid_for_land`    | island full and `wealth >= 50`             | the same, measured against the last land price           | Posts a bid: expected daily profit × `landHorizon` × (0.5 + ambition), capped by savings |
| `sell_land`       | owns a plot                                | unsold bread, own hunger, high unmet land bids; less with ambition | Offers one plot at what the owner thinks it is worth |

When several people buy the island's last plots on the same day, the engine applies the
purchases in random order and rejects those that find no plot left (`landsFree` cannot go
below 0). The rejection is counted in `decision.islander.buy_island_land.rejected`.

The **land market** matches the highest bids with the cheapest offers (never one's own plot) at
the midpoint price, if the buyer can still pay.

## End of the day

- Eating `maxBreadsPerDay` breads resets hunger to 0. Otherwise hunger grows by `hungerPerDay`
  minus the bread eaten.
- `starvationDays` days in a row without any bread kill: the person is removed and their plots
  go back to the island.
- **80% of what is left of the day's income is spent elsewhere**: income is wage plus profit,
  minus spending on bread and land. The remaining 20% is saved.
- A person with zero hunger has a 1% chance of having a child, who starts with no savings, no
  hunger and a random personality.

## Metrics

`producers`, `landsFree`, `marketPrice`, `landPrice`, `breadSold`, `breadWasted`,
`unmetDemand`, `deaths`, `births`, `hunger.mean`, `wealth.mean`, `wealth.gini`, plus the
population and the counters of every decision.

## Results

With the defaults, the island can bake at most 100 breads a day, while 100 people need 300.

- **Famine.** Production only starts on day 2, after the first plots are bought, so about a
  third of the population starves in the first days.
- **Balance.** The population then settles around 50 to 60, fed by 5 to 8 producers.
- **Land.** All plots are taken and change hands often. Land becomes the scarce asset: its
  price rises from 100 to around 230 on average, and above 450 in some runs.
- **Inequality.** The Gini coefficient of wealth rises while producers earn rents, then falls
  as plots change hands and producers die or sell.
- **Survival.** 1 run in 20 ends with everyone dead.

### Sensitivity

The balance depends on the race between buyers raising bids (`outbid`) and sellers cutting
prices (`priceDrop`). If prices rise faster than they fall, bread outgrows the wage and the
island starves. Final population and extinctions over 20 seeds:

| outbid | priceDrop | Extinct runs | Final population | Bread price |
|--------|-----------|--------------|------------------|-------------|
| 0.1    | 0.2       | 1/20         | ~52              | ~2.4        |
| 0.2    | 0.2       | 1/20         | ~49              | ~3.3        |
| 0.2    | 0.1       | 13/20        | ~17              | ~7.9        |
| 0.5    | 0.1       | 19/20        | ~4               | ~12.1       |

## Things to try

- Double `breadsPerLand` or `landsFree` and see whether the famine disappears.
- Raise `salary` (inflation of wages against a fixed supply).
- Make land cheaper (`islandLandPrice`) and watch the number of producers and the land price.
- Set `outbid` to 0.5: buyers who panic under scarcity make bread unaffordable.
