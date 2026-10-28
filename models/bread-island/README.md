# Bread island

Supply and demand on an island where bread is the only food. Everyone earns the same fixed
wage and must eat, and wants to save as much as possible once fed. Anyone who saves enough can
buy a plot of the island's limited farmland and become a bread producer. Bread is sold in a
daily auction and expires at the end of the day, and plots change hands in a land market.
Hungry people offer more for bread, and people who go a week without any bread die. Droughts,
golden ages and epidemics hit at random.

The question: with scarce land and perishable food, how do prices, production, inequality and
population find (or fail to find) a balance?

```sh
agentsim run models/bread-island --seeds 20
```

## A day on the island

| Phase | Step               | What happens                                                                 |
|-------|--------------------|------------------------------------------------------------------------------|
| pre   | `new_day`          | Daily counters (deaths, births) reset                                       |
| pre   | `events`           | Droughts, golden ages and epidemics start or run their course               |
| pre   | `morning`          | Everyone receives the wage; each plot bakes `breadsPerLand × productionFactor` breads |
| pre   | `quotes`           | The cheapest price asked by a producer is published                          |
| pre   | `orders`           | Everyone bids for the bread they need; producers offer their stock          |
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
| `frugality`       | private    | Preference for saving over eating well (0 = eats well, 1 = miser) |

Memory: `salePrices`, the prices paid by the producer's last 5 customers.

`agents/islanders.agent.json` creates 500 people with savings drawn from normal(50, 40)
(at least 0) and uniform personalities. Children get random personalities.

## Context

Parameters (in `island.context.json`):

| Field              | Default | Meaning                                                        |
|--------------------|---------|----------------------------------------------------------------|
| `salary`           | 10      | Wage per day, the same for everyone, forever                    |
| `landsFree`        | 15      | Plots the island still has (all 15 at the start)                |
| `islandLandPrice`  | 100     | Price of a plot bought from the island                         |
| `breadsPerLand`    | 40      | Bread per plot per day in normal weather                        |
| `productionCost`   | 0.5     | Share of each sale that is cost; the producer keeps the rest as profit |
| `basePrice`        | 2       | The island's reference price for bread                         |
| `hungerPremium`    | 1       | Extra a person is willing to pay per point of hunger           |
| `marginalDiscount` | 0.1     | Each extra bread of the day is worth 10% less than the first    |
| `outbid`           | 0.1     | How far above the cheapest posted price buyers go under scarcity |
| `priceDrop`        | 0.2     | How much a producer cuts its price after a day with unsold bread |
| `minPrice`         | 0.1     | Price floor                                                    |
| `maxBreadsPerDay`  | 3       | Normal ration: eating 3 breads resets hunger to 0               |
| `hungerPerDay`     | 3       | Hunger added on a day without a full ration; each bread removes 1 |
| `starvationDays`   | 7       | Days in a row without any bread that kill                      |
| `birthChance`      | 0.01    | Daily chance that a person with zero hunger has a child         |
| `landHorizon`      | 15      | Days of profit a plot is worth to an average trader            |
| `droughtChance`, `droughtDuration`, `droughtImpact` | 0.01, 20, 0.5 | Daily chance, length in days, and harvest lost |
| `goldenChance`, `goldenDuration`, `goldenImpact`    | 0.01, 20, 0.5 | Daily chance, length in days, and extra harvest |
| `epidemicChance`, `epidemicDuration`, `epidemicRation` | 0.005, 30, 4 | Daily chance, length in days, and breads needed to reset hunger |

State published by the rules: `marketPrice` (average price paid yesterday), `lowestAsk`,
`breadSold`, `breadWasted`, `unmetDemand`, `sellRate`, `scarcity`, `landPrice`, `bestLandBid`,
`deaths`, `births`, the event state (`productionFactor`, `ration` and the days left of each
event), and the order books `breadBids`, `breadBidMax`, `breadAskQty`, `breadAskPrice`,
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

**What a person buys.** Everyone wants to save as much as possible once fed, and frugality
decides what "fed" means:

- **Survival bread.** After enough days without food, the first bread of the day becomes a
  must. A miser waits until two days before starvation, and the least frugal eat every day
  (the wait is `frugality × (starvationDays − 2)` days). A must-have bread is bid for with
  whatever it takes, up to the person's savings.
- **Comfort bread.** Any other bread, and the first one while it is not yet a must, is bought
  only up to `salary × (1 − frugality) / ration` per bread. A full miser never buys comfort
  bread, and the least frugal will spend the whole wage on a full ration.
- Nobody buys more than today's ration.

**How much they offer.** A person is willing to pay `basePrice + hungerPremium × hunger` for
the first bread of the day, and 10% less for each further one. Each bid carries two prices:

- the **offer**, what the buyer tries to pay. Since buyers pay what they bid, nobody offers more
  than needed: the offer is capped at the cheapest posted price, marked up by
  `outbid × scarcity` when bread was short yesterday. Under scarcity the hungriest outbid the rest.
- the **maximum**, what the buyer accepts if the cheaper bread is gone.

Each bread's prices are limited by the savings not committed to earlier breads, so a person
secures the first bread before the next ones.

**Matching.** Bids are served from the highest offer. Each buys the cheapest bread still on
offer, if its ask is within the bid's maximum. The buyer pays its offer, or the ask if that is
higher. The producer keeps 50% as profit, and the other 50% is production cost and leaves the
economy.

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

- Eating a full ration (3 breads, 4 during an epidemic) resets hunger to 0. Otherwise hunger
  grows by `hungerPerDay` minus the bread eaten. Frugal people let hunger build up while saving,
  so the average hunger is dominated by them.
- `starvationDays` days in a row without any bread kill: the person is removed and their plots
  go back to the island.
- **80% of what is left of the day's income is spent elsewhere**: income is wage plus profit,
  minus spending on bread and land. The remaining 20% is saved.
- A person with zero hunger has a 1% chance of having a child, who starts with no savings, no
  hunger and a random personality.

## Metrics

`producers`, `landsFree`, `marketPrice`, `landPrice`, `breadSold`, `breadWasted`,
`unmetDemand`, `deaths`, `births`, `productionFactor`, `ration`, `hunger.mean`, `wealth.mean`,
`wealth.gini`, plus the population and the counters of every decision.

## Results

The island can bake 600 breads a day in normal weather (15 plots × 40), which is enough for
everyone to survive but far from a full ration for everyone (1,500). Over 20 seeds of 365 days:

- **No island dies out.** The week of margin before starvation gives prices time to adjust. The
  population grows from 500 to about 750, because the least frugal eat well and have children.
- **Land concentrates.** All plots are bought in the first days, and they end up in the hands of
  2 to 5 producers. A plot pays for itself in a few days, so its price climbs from 100 to around
  960 on average.
- **Moderate inequality.** The Gini coefficient of wealth stays around 0.34: everyone earns the
  same wage, and misers save almost all of it.

The random events move the market as expected (averages over all days of all seeds):

| Condition  | Days | Bread price | Bread sold/day | Deaths/day |
|------------|------|-------------|----------------|------------|
| Normal     | 4808 | 2.14        | 553            | 0.05       |
| Drought    | 1024 | 3.52        | 295            | 1.82       |
| Golden age | 859  | 1.72        | 846            | 0.01       |
| Epidemic   | 736  | 1.89        | 552            | 0.12       |

Droughts are what kill: half the harvest raises the price by about 65%, and deaths jump from
0.05 to 1.8 a day.

### Sensitivity

The balance depends on the race between buyers raising bids (`outbid`) and sellers cutting
prices (`priceDrop`). Over 20 seeds:

| outbid | priceDrop | Extinct runs | Deaths per run | Final population | Bread price |
|--------|-----------|--------------|----------------|------------------|-------------|
| 0.1    | 0.2       | 0/20         | ~107           | ~753             | ~2.25       |
| 0.2    | 0.2       | 0/20         | ~29            | ~789             | ~2.92       |
| 0.2    | 0.1       | 0/20         | ~5             | ~678             | ~2.95       |
| 0.5    | 0.1       | 0/20         | ~88            | ~518             | ~2.80       |

With 3 days to starvation instead of 7, 100 people and 10 plots of 10 breads, the same
combinations ranged from 1 to 19 extinct runs out of 20: the time the market has to correct
matters as much as how it corrects.

## Things to try

- Set `starvationDays` back to 3 and watch extinctions come back.
- Make droughts longer or more frequent (`droughtDuration`, `droughtChance`).
- Double `breadsPerLand` or `landsFree` and see whether hunger disappears.
- Raise `salary` (inflation of wages against a fixed supply).
- Make land cheaper (`islandLandPrice`) and watch the number of producers and the land price.
- Set `outbid` to 0.5: buyers who panic under scarcity make bread unaffordable.
