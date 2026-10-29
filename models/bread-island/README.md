# Bread island

Supply and demand on an island where bread is the only food. Everyone earns the same fixed
wage and tries to eat a full ration every day, then saves or invests what is left. Anyone who
saves enough can buy a plot of the island's limited farmland and become a bread producer.
Producers post their prices every morning, like price tags in a shop, and people buy through
an order book. Bread expires at the end of the day, and plots change hands in a land market.
People who could not eat enough are willing to pay more the next day. A week without any bread,
hunger above 15, or old age kill; producers' plots pass to their children. Droughts, golden ages
and epidemics hit at random.

The question: with scarce land and perishable food, how do prices, production, inequality and
population find (or fail to find) a balance?

```sh
agentsim run models/bread-island --seeds 20
```

## A day on the island

| Phase | Step               | What happens                                                                 |
|-------|--------------------|------------------------------------------------------------------------------|
| pre   | `new_day`          | Daily counters (deaths, births, people not fully fed) reset                 |
| pre   | `events`           | Droughts, golden ages and epidemics start or run their course               |
| pre   | `morning`          | Everyone receives the wage; each plot bakes `breadsPerLand × productionFactor` breads |
| pre   | `offers`           | Producers post today's price and stock                                      |
| pre   | `orders`           | People read the posted offers and place buy orders for their ration         |
| pre   | `bread_market`     | The order book is matched: bread changes hands                              |
| decide | decisions         | People may buy land from the island, bid for someone's plot, or offer a plot |
| post  | `land_market`      | Land bids and offers are matched                                            |
| post  | `metabolism`       | Ageing, hunger, deaths, daily spending, tomorrow's prices and limits, expired bread, births |
| post  | `funerals`         | The dead are removed; their plots go to the oldest living child, or back to the island |
| post  | `land_office`      | The island lowers the price of free plots that found no buyer                |

Eating and selling bread are routine, not choices: they are agent-scoped rules that run for
everyone every day. Decisions are kept for strategic choices about land, because the engine
lets an agent take one decision per tick, and nobody should have to choose between eating and
investing.

## Agents

One type, `islander`. People with no land are **consumers**, and people with at least one plot
are **producers**. Everyone starts as a consumer, and the role changes as plots are bought and
sold. Producers are people too: they earn the wage and get hungry. They eat from their own
bread first and sell the rest; if their own bread is not enough they buy at the market.

| Attribute         | Visibility | Meaning                                                      |
|-------------------|------------|--------------------------------------------------------------|
| `wealth`          | private    | Savings (never negative)                                     |
| `age`             | private    | Age in days                                                  |
| `hunger`          | public     | 0 after a full ration; grows otherwise; above 15 it kills    |
| `daysWithoutFood` | private    | Consecutive days without any bread                           |
| `ateToday`        | private    | Bread eaten today                                            |
| `income`, `spent` | private    | Money received and spent today (for the 80% rule)            |
| `lands`           | public     | Plots owned; 0 = consumer                                    |
| `stock`           | private    | Bread baked today and not yet eaten or sold                  |
| `offered`         | private    | Bread put on sale today, after the producer ate its share    |
| `askPrice`        | public     | Price the producer posts per bread                           |
| `bidLimit`        | private    | Most the person pays for a bread (starts at `basePrice`)     |
| `riskTolerance`   | private    | Willingness to put savings at risk (0 to 1)                  |
| `ambition`        | private    | Drive to own and expand a business (0 to 1)                  |
| `dead`            | private    | Died today; removed by `funerals`                            |

Relation `parent` (directed, parent → child) records family ties for inheritance.

`agents/islanders.agent.json` creates 500 people aged 15 to 70, with savings drawn from
normal(50, 40) (at least 0) and uniform personalities. Children are born at age 0, with no
savings and a random personality, and eat like adults.

## Context

Parameters (in `island.context.json`):

| Field              | Default | Meaning                                                        |
|--------------------|---------|----------------------------------------------------------------|
| `salary`           | 10      | Wage per day, the same for everyone, forever                    |
| `landsFree`        | 15      | Plots the island still has (all 15 at the start)                |
| `islandLandPrice`  | 100     | The island's starting price for free plots                     |
| `landPriceDrop`    | 0.1     | Daily drop of the free-plot price while free plots find no buyer |
| `minLandPrice`     | 10      | Floor of the free-plot price                                   |
| `breadsPerLand`    | 40      | Bread per plot per day in normal weather                        |
| `productionCost`   | 0.5     | Share of each sale that is cost; the producer keeps the rest as profit |
| `basePrice`        | 2       | Everyone's initial price limit for bread                       |
| `bidRaise`         | 0.1     | Limit increase after a day without a full ration, × share of the ration missed |
| `bidRelax`         | 0.02    | Limit decrease after a day with a full ration                  |
| `priceRise`        | 0.1     | Price increase after selling the whole stock                   |
| `priceCut`         | 0.5     | Price cut after a day with bread left, × unsold share of what was offered |
| `undercut`         | 0.05    | After a day with bread left, the price is at most 5% below the market average |
| `minPrice`         | 0.1     | Floor for prices and limits                                    |
| `maxBreadsPerDay`  | 3       | Normal ration: eating 3 breads resets hunger to 0               |
| `hungerPerDay`     | 3       | Hunger added on a day three or more breads short of the ration |
| `starvationDays`   | 7       | Days in a row without any bread that kill                      |
| `maxHunger`        | 15      | Hunger above this kills by malnutrition                        |
| `daysPerYear`      | 365     | Days in a year (ticks are days)                                |
| `mortalityBase`, `mortalityGrowth` | 0.00005, 0.09 | Gompertz law of old-age mortality: yearly risk = a × e^(b × age in years) |
| `birthChance`      | 0.01    | Daily chance that a person with zero hunger has a child         |
| `landHorizon`      | 15      | Days of profit a plot is worth to an average trader            |
| `droughtChance`, `droughtDuration`, `droughtImpact` | 0.01, 20, 0.5 | Daily chance, length in days, and harvest lost |
| `goldenChance`, `goldenDuration`, `goldenImpact`    | 0.01, 20, 0.5 | Daily chance, length in days, and extra harvest |
| `epidemicChance`, `epidemicDuration`, `epidemicRation` | 0.005, 30, 4 | Daily chance, length in days, and breads needed to reset hunger |

State published by the rules: `freeLandPrice` (what the island asks today for a free plot),
`islandSalesToday`, `marketPrice` (average price paid in the last day with trades),
`breadSold`, `breadWasted`, `unmetDemand`, `notFullyFed`, `sellRate`, `scarcity`, `landPrice`,
`bestLandBid`, `deaths`, `births`, the event state (`productionFactor`, `ration` and the days
left of each event), and the order books `breadBids`, `breadAskQty`, `breadAskPrice`,
`landBids` and `landAsks`. Each agent writes only its own keys, so orders never conflict.

## Random events

| Event      | Chance per day | Duration | Effect                                                  |
|------------|----------------|----------|---------------------------------------------------------|
| Drought    | 1%             | 20 days  | Harvest −50% (`productionFactor` 0.5)                   |
| Golden age | 1%             | 20 days  | Harvest +50% (`productionFactor` 1.5)                   |
| Epidemic   | 0.5%           | 30 days  | A full ration is 4 breads instead of 3 (`ration` 4)     |

The weather is either normal, a drought or a golden age, and a new weather event only starts in
normal weather. Epidemics are independent of the weather and can overlap with it. Each start is
logged (`drought_started`, `golden_age_started`, `epidemic_started`).

## The bread market

The market works like a shop with price tags and a stock broker's order book.

**1. Producers post prices.** Every morning each producer posts its price and the bread it
baked. The price was set at the end of the previous day:

- sold the whole stock: raise the price by `priceRise` (10%);
- bread left over: cut it by `priceCut × unsold share` (selling 60% of the stock cuts 20%).

**2. People place orders.** Everyone tries to eat a full ration (3 breads, 4 during an
epidemic) and only then saves or invests. A person reads the posted offers from the cheapest
up and plans the day's shopping: as many breads as the ration needs, from offers priced within
its own limit, as long as savings cover the posted prices. It then places one buy order per
planned bread at its limit.

Each person learns its limit from experience. After a day without a full ration, it raises the
limit by `bidRaise × share of the ration missed`: eating 1 of 3 breads raises it by about 6.7%.
After a full day, it lowers the limit by `bidRelax` (2%), to test whether bread can be had for
less.

**3. The order book is matched**, as in a stock broker:

- buy orders are served from the **highest limit** (ties drawn at random);
- each order buys the **cheapest bread still on offer**, if its price is within the order's limit;
- the buyer pays the **posted price**, not its limit.

Someone willing to pay 2.70 for bread posted at 2.50 goes ahead of someone willing to pay 2.60,
and pays 2.50. Scarcity reaches the price through the producer instead: a shop that sells out
raises its price the next day. The producer keeps 50% of each sale as profit, and the other 50%
is production cost and leaves the economy. Unsold bread expires at the end of the day.

## Decisions (land)

Every person may take one of these each day, or do nothing (idle weight 1):

| Decision          | Eligible                                   | Propensity grows with                                   | Effect |
|-------------------|--------------------------------------------|----------------------------------------------------------|--------|
| `buy_island_land` | island has free plots and `wealth >= freeLandPrice` | expected return (price × profit share × yield × sell rate / plot price), scarcity, risk tolerance, ambition, own hunger | Pays `freeLandPrice` and takes a plot from the island; becomes a producer |
| `bid_for_land`    | island full and `wealth >= 50`             | the same, measured against the last land price           | Posts a bid: expected daily profit × `landHorizon` × (0.5 + ambition), capped by savings |
| `sell_land`       | owns a plot                                | unsold bread, own hunger, high unmet land bids; less with ambition | Offers one plot at what the owner thinks it is worth |

When several people buy the island's last plots on the same day, the engine applies the
purchases in random order and rejects those that find no plot left (`landsFree` cannot go
below 0). The rejection is counted in `decision.islander.buy_island_land.rejected`.

The **land market** matches the highest bids with the cheapest offers (never one's own plot) at
the midpoint price, if the buyer can still pay.

**Free plots.** The island starts selling its 15 plots at 100. Plots also come back to the
island when an owner dies without a living child. While free plots find no buyer, the island
lowers their price by 10% a day, down to 10 (a day's wage). The price therefore stops at the
first level someone can afford, which gives consumers a reason to save. Once all plots are taken
again, the price goes back to 100.

## End of the day

- **Hunger** depends on how many breads of the ration were missed:

  | Breads eaten (ration 3) | Hunger                  |
  |-------------------------|-------------------------|
  | 3                       | reset to 0              |
  | 2                       | unchanged               |
  | 1                       | +1                      |
  | 0                       | +3 (`hungerPerDay`)     |

  During an epidemic (ration 4) the same rule applies to the breads missed: missing one leaves
  hunger unchanged, missing two adds 1, missing three or more adds 3.
- **Death** comes after `starvationDays` days in a row without any bread, when hunger passes
  `maxHunger` (malnutrition), or from old age. Old-age mortality follows the Gompertz law, the
  pattern observed in human populations: the yearly risk is about 1% at 60, 7% at 80 and 16%
  at 90. The `died` event records the cause and the age.
- **Inheritance.** The `funerals` rule removes the dead. Their plots go to their oldest living
  child; with no living child they are abandoned to the island. Savings are not inherited.
- **Spending.** 80% of what is left of the day's income is spent elsewhere: income is wage plus
  profit, minus spending on bread and land. The remaining 20% is saved.
- **Limits and prices for tomorrow.**
  - A person's limit rises after a day without a full ration and eases after a full one, but it
    never exceeds what the person has (savings, or at least a day's wage).
  - A producer that sold everything raises its price by 10%. A producer with bread left over
    cuts its price by `priceCut × unsold share`, and to at most 5% below the market average: a
    price far above the market is plainly out of line.
  - Unsold bread expires.
- **Births.** A person with zero hunger has a 1% chance a day of having a child.

## Metrics

`producers`, `landsFree`, `marketPrice`, `landPrice`, `breadSold`, `breadWasted`,
`freeLandPrice`, `unmetDemand`, `notFullyFed`, `deaths`, `births`, `productionFactor`, `ration`,
`hunger.mean`, `wealth.mean`, `wealth.gini`, plus the population and the counters of every
decision.

## Results

The island can bake 600 breads a day in normal weather (15 plots × 40). Since eating 2 breads a
day keeps hunger stable, it can feed about 300 people for good, but 500 people start on it.
Over 20 seeds of 365 days:

| Day | Population | Producers | Bread price | Not fully fed | Land price |
|-----|------------|-----------|-------------|---------------|------------|
| 1   | 500        | 3         | 2.00        | 100%          | 100        |
| 3   | 501        | 15        | 2.04        | 88%           | 100        |
| 30  | 460        | 15        | 7.66        | 67%           | 181        |
| 60  | 236        | 13        | 3.74        | 73%           | 902        |
| 365 | 240        | 10        | 2.80        | 44%           | 1,783      |

- **Famine, then balance.** Nobody bakes on day 1, and from day 2 on there is far less bread
  than 500 people need. Producers sell out every day and raise their prices, until bread costs
  three quarters of a day's wage. The poorest go hungry and, about a month later, die of
  malnutrition. The population settles around 240 (between 150 and 370 depending on the seed),
  and no island dies out.
- **Hunger amid unsold bread.** On a normal day about 74 breads expire while half the people
  miss part of their ration. They are not short of bread but of money. A full ration costs
  nearly the whole wage, the cheapest bread goes to those willing to pay most, and what is left
  is priced above what the poor can afford. Amartya Sen described famines the same way: food
  exists, but the poor cannot buy it.
- **Land and inequality.** Plots are the scarce asset: their price rises from 100 to about
  1,800, and they concentrate in about 10 producers. The Gini coefficient of wealth reaches
  0.83, because producers accumulate profits they have nothing to spend on.
- **Producers survive.** Bakers eat their own bread, so no producer starves. No plot was
  inherited or abandoned in a year.
- **Old age** kills about one person per seed per year (median age 62). Use longer runs
  (`--ticks 18250` for 50 years) to see ageing and inheritance at work.

The random events move the market as expected (averages over all days of all seeds):

| Condition  | Days | Bread price | Bread sold/day | Not fully fed | Deaths/day |
|------------|------|-------------|----------------|---------------|------------|
| Normal     | 4808 | 3.08        | 486            | 50%           | 1.7        |
| Drought    | 1024 | 4.70        | 246            | 72%           | 5.0        |
| Golden age | 859  | 2.01        | 688            | 20%           | 0.8        |
| Epidemic   | 736  | 3.12        | 479            | 53%           | 4.3        |

### What is not realistic yet

- Producers' profits leave circulation: there is nothing to spend them on, no jobs and no
  lending, so money piles up with the landowners.
- Wages are fixed forever, even when bread takes almost all of them.
- The island starts with more people than it can feed, so the first-month famine is a product
  of the initial conditions rather than of the market.

### Sensitivity

How fast buyers raise their limits (`bidRaise`) and producers cut prices (`priceCut`), over 20
seeds:

| bidRaise | priceCut | Extinct runs | Final population | Bread price | Not fully fed (avg) | Bread wasted/day |
|----------|----------|--------------|------------------|-------------|---------------------|------------------|
| 0.1      | 0.5      | 0/20         | ~240             | ~2.80       | 42%                 | ~77              |
| 0.05     | 0.5      | 0/20         | ~320             | ~2.61       | 44%                 | ~78              |
| 0.2      | 0.5      | 0/20         | ~254             | ~3.75       | 46%                 | ~79              |
| 0.1      | 0.25     | 0/20         | ~203             | ~2.06       | 37%                 | ~169             |
| 0.1      | 1.0      | 0/20         | ~254             | ~2.07       | 31%                 | ~28              |

Producers who cut prices faster waste less bread and feed more people. Buyers who raise their
limits faster push prices up.

## Things to try

- Set `starvationDays` back to 3 and see how much harder the first famine hits.
- Run 50 years (`--ticks 18250`) to see ageing, inheritance and abandoned plots.
- Make droughts longer or more frequent (`droughtDuration`, `droughtChance`).
- Double `breadsPerLand` or `landsFree` and see whether hunger disappears.
- Raise `salary` (inflation of wages against a fixed supply).
- Start with fewer people (`count` in `agents/islanders.agent.json`) and see whether the famine disappears.
- Make buyers react faster to hunger (`bidRaise`) or producers to unsold bread (`priceCut`).
