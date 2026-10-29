# Labor market

A labor market with no minimum wage. Firms of one sector want to grow their profit, and for
that they need capable workers; workers want the highest pay they can get, but must pay the
cost of living every month or die. Capacity grows with work. Firms read candidates through
noisy interviews, make individual offers, poach each other's workers and counter outside
offers. A minimum wage exists only as a yardstick: it classifies people into poverty, lower,
middle and upper class by what they earn, and binds nobody.

The question: left to themselves, how do competition for skilled labor, learning on the job
and the need to earn a living distribute wages and jobs? Who ends up poor, and why?

```sh
agentsim run models/labor-market --seeds 20
```

## A month

One tick is a month; a run is 40 years (480 months).

| Phase  | Step           | What happens                                                                 |
|--------|----------------|------------------------------------------------------------------------------|
| init   | `setup`        | 90% of people start employed at a random firm, paid what that firm would offer them |
| pre    | `new_month`    | Monthly counters and last month's job seekers reset                          |
| pre    | `entrants`     | Young people enter the labor market                                          |
| pre    | `production`   | Firms produce and sell, pay wages, overhead and dividends; firms that cannot pay go bankrupt. Prices, wages by level and each firm's books are published |
| pre    | `living`       | People pay the cost of living and spend; skills grow or rust; ageing, retirement, death |
| decide | decisions      | People decide whether to look for a job; firms whether to open jobs or downsize |
| post   | `layoffs`      | Firms that decided to downsize lay people off                                |
| post   | `labor_market` | Interviews, offers in rounds, choices and counteroffers                      |
| post   | `promotions`   | Once a year, employers promote people whose capacity reached the next level  |
| post   | `firm_entry`   | New firms enter while the sector is profitable                               |
| post   | `census`       | Unemployment, income classes and capacity are measured                       |

Being paid, paying the bills and learning are routine, not choices: they are rules that run for
everyone every month. Decisions are kept for strategic choices (looking for a job, hiring,
downsizing).

## Agents

### Worker

| Aspect      | Content |
|-------------|---------|
| State       | `age` (months), `capacity` (output per month), `talent` (the capacity a lifetime of work approaches), `savings`, `wage` (0 when unemployed), `lastWage`, `expectedWage`, `employer` (firm id, 0 when unemployed), `monthsUnemployed`, `monthsShort` (months in a row unable to pay the cost of living), `experience` (public), `level` (public: junior, pleno, senior), `reservation`, `bestOffer` |
| Perception  | Its own state; the offers it gets; the average wage of each level (`levelWage`, as salary surveys publish). It does **not** know its exact capacity: its idea of its own worth comes from its level, its pay and the offers it gets |
| Goals       | 1) pay the cost of living; 2) earn more |
| Decision    | `search`: look for a job and set the lowest wage it accepts |
| Routine     | Paid by the firm; pays `subsistence`; spends part of the rest; learns or loses skills; ages |
| Constraints | Savings never go negative; one job at a time |

### Firm

| Aspect      | Content |
|-------------|---------|
| State       | `cash`, `markdown` (share of a hire's estimated value the firm keeps), `openings`, `layoffsPlanned`, `headcount` and `share` (public), `capacitySum`, `revenue`, `wageBill`, `margin`, `underwater`, `age` |
| Perception  | The true capacity of its own staff; candidates only through an interview (capacity read with a relative error of `interviewNoise`); the market price and its own output share |
| Goals       | Grow profit by employing capable people |
| Decisions   | `expand` (open jobs), `downsize` (lay off people who cost more than they are worth), or nothing |
| Constraints | Must pay wages and overhead or go bankrupt; can recruit for only so many jobs a month |

Owners are outside the model: they put in the starting capital and receive dividends.

`agents/workers.agent.json` creates 2,000 people aged 18 to 64, all juniors, with capacity drawn
from normal(1.0, 0.2) in [0.5, 1.5], talent from normal(2.5, 0.6) in [1.2, 4] and savings from
normal(5000, 4000) (at least 0). `agents/firms.agent.json` creates 20 firms with 500,000 in cash
and a markdown drawn from uniform(0.2, 0.5).

## Behavior

### Production and prices

A firm produces the sum of its staff's capacities. The sector sells everything it produces at
the price the market bears, with constant elasticity demand:

    price = basePrice × (output / baseDemand)^(-1 / elasticity)

The more the sector produces, the lower the price. A person's **value** to a firm is the revenue
it adds once the firm's own effect on the price is counted (marginal revenue product):

    value = price × capacity × (1 - share / elasticity)

That is the brake on expansion: a growing firm pushes its own price down, and a firm's offers
fall as its share grows. Demand is outside the model (consumers are not the workers), and each
firm pays `fixedCost` a month in overhead.

### Looking for a job (`search`)

- **Who looks.** The unemployed almost always (propensity 50, idle 1). The employed with
  propensity `searchBase + searchGap × max(0, levelWage / wage - 1)`: now and then, and more
  often the further their pay is below the average of their level. People sense they are
  underpaid by comparing with peers.
- **Lowest acceptable wage of the employed**: the current wage plus `switchPremium` (10%).
  Changing jobs brings raises of about 10% in the data, and nobody moves for the same pay.
- **Lowest acceptable wage of the unemployed** (reservation wage):

      reservation = expectedWage × (1 - reservationDecay)^monthsUnemployed × pressure
      pressure    = 0.5 + 0.5 × min(1, runway / runwayComfort)      runway = savings / subsistence

  It is 0 (any job will do) with less than one month of living costs saved. `expectedWage`
  starts at the last wage (with no job yet, the average wage of the level) and moves each month
  by `offerLearning` toward the best offer received. Reservation wages are anchored on the last
  wage and fall slowly with unemployment, but sharply when money runs out (Krueger and Mueller
  2016); offers teach job seekers what they are worth (Conlon et al. 2018).

### Hiring (`expand`) and the labor market

- **Opening jobs.** A firm with at least `cashBufferMonths` of costs in cash opens jobs with
  propensity `5 × sigmoid(-1 + markdown × expectedTenure - hiringCost × (1 - markdown) + 5 × margin)`:
  more eagerly the more a hire earns it over its expected tenure against the cost of recruiting,
  less eagerly while it is losing money. It opens `max(minHires, hireRate × headcount)` jobs,
  its recruiting capacity for the month.
- **Interviews.** Each firm interviews `interviewsPerOpening` job seekers per opening, drawn at
  random (it does not meet everyone), and reads each one's capacity with a relative error of
  `interviewNoise` (25%).
- **Offers in rounds.** The firm offers jobs to its best-rated candidates at
  `value(estimate, share after the hire) × (1 - markdown)`. When a candidate turns it down or
  takes another job, the firm moves on to the next candidate on its list in the next round,
  until its openings are filled or its list runs out.
- **Choice.** Each person takes its best offer if it meets its reservation wage.
- **Counteroffers.** When one of its people is about to leave, the employer, who knows the
  person's real capacity, matches the outside offer if it still keeps `counterMargin` of the
  person's value. The person stays at the new wage. Wages rise within a job only this way, through
  outside offers (sequential auctions, Postel-Vinay and Robin 2002); they never fall.
- **Learning the going wage.** A firm that filled its openings offers a little less next month
  (`markdownRise`); one left with vacancies offers more, in proportion to the share unfilled
  (`markdownCut`). Each hire costs `hiringCost` months of its wage.

### Layoffs (`downsize`)

The firm counts the staff whose value is more than `layoffTolerance` (10%) below their wage. It
keeps people through smaller shortfalls (labor hoarding). With such people on staff, it may
decide to downsize (propensity rises with their share and with losses). It then lays them off
one at a time, lowest value for the wage first. Each layoff shrinks its share and so raises what
the others are worth, and it stops when the next person is no longer clearly underwater.

### Payday and bankruptcy

Wages and overhead come out of revenue and cash. Profit is paid out as dividends as long as
`payoutAboveMonths` of costs stay in the bank, so owners never take back the capital they put in.
A firm that cannot pay its wages and overhead goes bankrupt: its cash is split among its staff in
proportion to their wages, and everyone is out of a job.

### Life

- **Cost of living.** `subsistence` (900) a month, paid from savings after payday. After
  `monthsShortToDie` (3) months in a row without enough money, the person dies.
- **Spending.** People spend most of what the wage leaves above subsistence: `mpcLow` (95%)
  when pay is low, falling toward `mpcHigh` (70%) as it rises. They also spend `wealthSpending`
  of their savings each month (about 5% a year, consumption out of wealth).
- **Learning by doing.** Working closes `learningRate` (1.5%) of the distance between capacity
  and talent each month: a learning curve, fast at first and slower near one's limit. Out of
  work, capacity rusts by `skillDecay` (0.5% a month).
- **Levels.** Once a year, employers promote juniors whose capacity reached `plenoCapacity`
  (1.5) to pleno, and plenos reaching `seniorCapacity` (2.2) to senior. Titles never go down.
- **Ageing.** People enter at 18 with up to `entrantSavingsMonths` (3) of living costs from
  their family, at `entrantsPerMonth` (3.5, which replaces a working life of 47 years for 2,000
  people). They retire at 65 and may die of natural causes (Gompertz law).
- **New firms.** Each month a firm is founded with chance
  `min(maxEntryChance, entrySensitivity × sector profit margin)`, with `startingCapital`.

### Income classes

The census measures the labor force by wage, in minimum wages (`minimumWage`, 1,600): poverty
below 1 (the unemployed included), lower class 1 to 3, middle class 3 to 10, upper class above 10.
The minimum wage is only this yardstick: no wage is bound by it.

## Parameters

| Field | Default | Source or assumption |
|-------|---------|----------------------|
| `minimumWage` | 1,600 | Order of magnitude of Brazil's minimum wage; used only to classify income |
| `subsistence` | 900 | Assumption: a basic food basket (DIEESE, about 700 to 850) plus minimal housing |
| `monthsShortToDie` | 3 | User's T |
| `basePrice`, `baseDemand` | 4,800, 2,000 | Calibration at the start: an average worker (capacity 1) adds about 3 minimum wages of revenue when the whole initial labor force works. The most uncertain parameter (see sensitivity) |
| `elasticity` | 1.5 | Assumption: demand for one sector's product, with substitutes, is elastic |
| `fixedCost`, `startingCapital` | 20,000, 500,000 | Assumption |
| `interviewNoise` | 0.25 | Assumption: interviews predict job performance only moderately (Schmidt and Hunter 1998) |
| `hiringCost` | 0.5 months | Assumption: recruiting and onboarding |
| `hireRate`, `minHires` | 10%, 5 | Assumption: recruiting capacity per month |
| `switchPremium` | 10% | Raises at job changes are around 10% |
| `reservationDecay` | 2% a month | Krueger and Mueller (2016): slow decline with unemployment duration |
| `offerLearning` | 0.3 | Assumption, adaptive expectations |
| `learningRate`, `skillDecay` | 1.5%, 0.5% a month | Assumption: about 7 years from junior to senior; skills rust a few percent a year out of work |
| `mpcLow`, `mpcHigh`, `wealthSpending` | 0.95, 0.7, 0.4% | Poorer households spend almost all income; richer ones save more; consumption out of wealth of 3 to 6% a year |
| `layoffTolerance` | 10% | Assumption, labor hoarding |
| `entrySensitivity`, `maxEntryChance` | 0.3, 0.2 | Assumption |

## Metrics

`workers`, `firms`, `unemployment`, `poverty`, `lowerClass`, `middleClass`, `upperClass`,
`meanWage`, `wage.gini`, `price`, `output`, `profitMargin`, `hhi` (concentration of output),
`capacityEmployed`, `capacityUnemployed`, `seniors`, `hires`, `jobChanges`, `counteroffers`,
`layoffs`, `bankruptcies`, `newFirms`, `deaths` (lack of income), `naturalDeaths`,
`retirements`, plus the population and the counters of every decision.

Events: `died` (cause, age, months unemployed, capacity, last wage), `bankrupt`, `poached`,
`counteroffer`, `promoted`, `firm_founded`.

## Results

Over 20 seeds of 40 years (means across seeds; seeds differ by less than 1 point):

| Year | People | Firms | Unemployment | Poverty | Lower | Middle | Upper | Mean wage | Capacity (employed) | Seniors |
|------|--------|-------|--------------|---------|-------|--------|-------|-----------|---------------------|---------|
| 0    | 2,000  | 20    | 0%           | 0%      | 0%    | 0%     | 0%    | 3,251     | 1.00                | 0%      |
| 1    | 1,986  | 21    | 2%           | 2%      | 60%   | 38%    | 0%    | 4,480     | 1.24                | 0%      |
| 5    | 1,916  | 20    | 16%          | 16%     | 13%   | 71%    | 0%    | 6,262     | 1.80                | 13%     |
| 10   | 1,809  | 20    | 16%          | 16%     | 12%   | 73%    | 0%    | 6,914     | 2.08                | 40%     |
| 20   | 1,616  | 19    | 15%          | 15%     | 10%   | 75%    | 0.1%  | 7,525     | 2.23                | 54%     |
| 40   | 1,334  | 12    | 16%          | 16%     | 8%    | 76%    | 0.6%  | 8,518     | 2.22                | 54%     |

(At year 0 everyone hired by `setup` is still unclassified: the census first runs at the end of
month 1.)

- **Wages rise because people learn and firms compete for them.** Capacity doubles in ten years
  and wages follow. They rise through outside offers: about 23 job changes and 4 matched
  counteroffers a month. Wages stay close to what people are worth (Gini 0.30), and almost
  nobody employed earns less than a minimum wage, even without a legal floor: firms bid for
  workers.
- **Poverty is unemployment.** About 15% of the labor force is out of work at any time, and
  they make up almost all of the poverty class. Most are experienced people between jobs
  (their mean capacity, 2.09, is close to the employed's 2.23), for 3 to 4 months, living off
  savings.
- **Unemployment comes from a winner's curse.** Firms compete for the same candidates and raise
  their offers until vacancies fill, up to nearly what the interview says a candidate is worth.
  The candidate a firm wins is often the one it overrated most, so part of each firm's staff
  ends up costing more than it is worth. Firms hoard them for a while, then lay off in waves
  (a fifth to half of a firm's staff at once). The people laid off expect their old,
  inflated wage and take months to revise it. With more accurate interviews (`interviewNoise`
  0.1) unemployment falls from 15% to 2.6% (see below).
- **The young die before their first job.** About 600 people die of poverty in 40 years, 81% of
  them having never worked and 92% under 20. Firms hire the best-rated candidates, and an
  entrant (capacity about 1) loses to an experienced job seeker (about 2). With at most three
  months of savings, a third of entrants never get a job in time. This is "no job without
  experience" taken to the extreme of a world with no families or welfare. As a result the
  labor force shrinks by a third in 40 years.
- **The sector concentrates.** Bankruptcies (13) outnumber new firms (5): profit margins are thin
  (about 4%) because firms bid wages up, so few entrepreneurs come in. 20 firms become 12 and
  concentration (HHI) doubles, from 0.05 to 0.10.

### Sensitivity

Eight seeds each, one parameter changed at a time; levels are means of years 31 to 40, deaths
are totals over 40 years.

| Scenario                         | Unemployment | Lower | Middle | Upper | Mean wage | Firms | Poverty deaths |
|----------------------------------|--------------|-------|--------|-------|-----------|-------|----------------|
| Base                             | 14.9%        | 8.5%  | 76%    | 0.4%  | 8,308     | 13.7  | 606            |
| `interviewNoise` 0.1             | 2.6%         | 20%   | 78%    | 0%    | 6,406     | 10.1  | 57             |
| `entrantSavingsMonths` 12        | 15.3%        | 11%   | 73%    | 0.1%  | 7,606     | 14.0  | 368            |
| `monthsShortToDie` 6             | 14.7%        | 14%   | 71%    | 0.07% | 7,115     | 12.8  | 175            |
| `baseDemand` 1,500               | 14.8%        | 15%   | 71%    | 0.04% | 6,932     | 12.5  | 658            |
| `baseDemand` 3,000               | 15.0%        | 2.4%  | 78%    | 4.7%  | 10,730    | 17.0  | 560            |
| `elasticity` 3                   | 13.9%        | 5.5%  | 79%    | 1.1%  | 9,240     | 12.1  | 599            |

- **Unemployment depends on how well firms read candidates, not on demand.** It barely moves
  with demand, elasticity, savings or the death threshold, but falls to 2.6% when interviews are
  accurate. With accurate reads there is little winner's curse: wages are lower and more equal
  (Gini 0.19), layoffs fall by two thirds, and poverty deaths by 90%.
- **Where the classes fall depends on demand.** `baseDemand` sets how much a unit of capacity is
  worth, and so the wage level in minimum wages: the upper class goes from almost nobody to 4.7%.
  It was calibrated, not measured, so the class shares are conditional on it. The share in
  poverty is not: it is unemployment.
- **Deaths depend on the safety margin of the young.** Twelve months of family support, or six
  months before dying, cut poverty deaths by 40% and 70%.

## Realism

- **Matches stylized facts:** wages rising with experience and through job changes (about 10%
  raises), counteroffers, reservation wages anchored on the last wage, high youth unemployment,
  layoffs in waves, thin margins in a competitive sector with concentration over time.
- **Too high:** unemployment around 15% is high for a modern economy (Brazil's has ranged from 6%
  to 14%). It comes from the 25% interview noise combined with firms that do not discount noisy
  reads. Real recruiters partly correct for the winner's curse (they shrink extreme reads toward
  the average, and use probation and references), which the model leaves out.
- **Too harsh:** young people dying before their first job. Real entrants live with their
  families, get help, or take informal work; here they have only their savings.

## Limitations and next steps

- **No households, no welfare, no informal sector.** Adding families that share income, or
  informal work at low pay, would change who survives unemployment.
- **Firms take interviews at face value.** Bayesian shrinkage of interview scores, probation
  periods, or references would reduce the winner's curse.
- **Wages never fall within a job.** Wages are rigid downward, and firms adjust only through
  layoffs. Pay cuts or variable pay would change the layoff waves.
- **Demand is outside the model.** The workers' wages do not buy the sector's product, so there
  is no feedback from unemployment to demand.
- **One sector, one kind of skill.** No education choice, no specialization, no other sectors.
- **The start is artificial.** Everyone starts as a junior with base capacity, so the first ten
  years are a transition while capacity builds up.
