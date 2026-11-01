# Labor market model: full report

This report documents everything in the `labor-market` model: what it is for, every agent,
attribute, parameter, decision and rule, how a month unfolds, what the model produces, and
where it falls short of reality. The [README](README.md) is the shorter guide; this file is the
complete reference.

## 1. Purpose

The model asks what a labor market looks like **without direct state regulation**: no minimum
wage, no labor code (no mandatory severance, notice or contracts), no unemployment insurance and
no welfare. Firms of one sector compete for workers whose capacity grows with work; workers want
higher pay and to feel valued, but must eat every month. Anyone with enough savings, courage and
patience can start a firm, and every firm has an owner who is a person of the model.

A minimum wage exists only as a **yardstick**: it classifies people into poverty, lower, middle
and upper class by income, and binds nobody.

The modeling principle: each agent behaves the way its real counterpart does, with the
information it would have. Aggregates (unemployment, wages, classes, concentration) are results
of those behaviors, never inputs.

## 2. How the model works

### 2.1 Time and scale

- One tick is **one month**; a run is **40 years** (480 ticks).
- **2,000 people** at the start, aged 18 to 64; **20 firms**, owned by the 20 richest people.
- About 3.5 young people enter each month; people retire at 65 (except owners) and die of
  natural causes (Gompertz law), of hunger, or after months unable to pay their costs.

### 2.2 A month, step by step

| # | Phase  | Step           | What happens |
|---|--------|----------------|--------------|
| 0 | init   | `setup`        | (Once.) The richest people own the firms; 90% of the others start employed, paid what their firm would offer after an interview |
| 1 | pre    | `new_month`    | Monthly counters and last month's job seekers reset |
| 2 | pre    | `entrants`     | Young people enter with family support |
| 3 | pre    | `production`   | Firms produce, sell, pay wages, overhead, the owner's draw and dividends; bankruptcies; firms learn from their junior hires; the price, price trend, wages by level and firms' books are published |
| 4 | pre    | `living`       | People pay for food, then the rest of the cost of living; spend; learn or lose skills; update time preference and job satisfaction; age, retire, die |
| 5 | pre    | `succession`   | Firms whose owner died are sold to the best bidder, or close |
| 6 | decide | decisions      | Each person may look for a job, quit or found a firm; each firm may open jobs or downsize (one decision per agent per month) |
| 7 | post   | `layoffs`      | Firms that decided to downsize lay people off, worst value for the wage first |
| 8 | post   | `labor_market` | Interviews, offers in rounds, choices, counteroffers; firms adjust their offers |
| 9 | post   | `promotions`   | Once a year, employers promote people whose capacity reached the next level |
| 10 | post  | `census`       | Unemployment, income classes, owners, satisfaction, time preference, horizons |

Routine needs (being paid, eating, paying bills, learning, ageing) are rules that run for
everyone every month. Decisions are kept for strategic choices, because the engine allows one
decision per agent per month and nobody should have to choose between eating and job hunting.

### 2.3 Where the money goes

```
                 external buyers (demand curve)
                              │ revenue
                              ▼
   ┌──────────────────────── FIRM ─────────────────────────┐
   │ cash                                                  │
   └─┬──────────┬─────────────┬──────────────┬─────────────┘
     │ wages    │ overhead,   │ owner's draw │ dividends
     │          │ hiring cost │ (pro-labore) │ (profit beyond 6 months of costs)
     ▼          ▼             ▼              ▼
  WORKERS   (leaves the     OWNER (a person of the model)
     │       model)            │
     │ cost of living, consumption, spending out of wealth
     ▼
  (leaves the model)
```

Money enters only as revenue from buyers outside the model, and leaves through the cost of
living, consumption, overhead, hiring costs, firm sales after an owner's death (no heirs) and the
cash of firms that close. Wages and owners' income stay inside.

### 2.4 Production and demand

A firm produces the sum of its staff's capacities. The sector sells everything at the price the
market bears (constant elasticity demand):

    price = basePrice × (output / baseDemand)^(-1 / elasticity)

A person's **value** to a firm is the revenue it adds once the firm's own effect on the price is
counted (marginal revenue product):

    value = price × capacity × (1 - share / elasticity)

Firms read demand from the price: the expected monthly change (`priceTrend`) moves 30% of the
way toward the latest change each month, and hires are valued at the price expected next month.
A falling price, the sign that buyers want less than the sector makes, makes hiring less
attractive. A firm that is not profitable does not open jobs (start-ups and firms with nobody
yet excepted).

### 2.5 The labor market

1. **Job search.** The unemployed almost always look; the employed look now and then, more often
   when paid below their level's average, when dissatisfied, or when their pay does not cover
   the cost of living.
2. **Reservation wage.** The employed want 10% more than their wage (any raise when in the
   squeeze). The unemployed ask what they expect to earn (last wage, revised toward the offers
   they get), less each month without work, faster for present-minded people, and much less as
   savings run out; with under a month of food money, any job.
3. **Interviews.** Each firm with openings interviews 3 seekers per opening, drawn at random, and
   reads capacity with a 25% error. Recruiters discount extreme reads, comparing each candidate
   with the others of the same level (Bayesian shrinkage).
4. **Ranking as an investment.** Candidates are ranked by the profit expected over the firm's
   horizon (cut short by retirement): juniors are cheap and will grow while their wage stays put.
5. **Offers in rounds.** Offers are the estimated current value minus the firm's markdown; when a
   candidate refuses or goes elsewhere, the firm moves to the next on its list.
6. **Choice and counteroffers.** Each person takes the best offer that meets its reservation
   wage; the current employer matches it when the person is still worth more to it, and the
   person stays at the new wage. Wages rise within a job only this way.
7. **Learning the going wage.** Firms that fill their openings offer slightly less next month;
   those left with vacancies offer more, but never so much that they stop covering overhead
   (start-ups excepted).

### 2.6 Layoffs, bankruptcy, founding and succession

- **Layoffs.** A firm counts the staff worth more than its tolerance below their wage. Tolerance
  is the firm's own patience (2% to 30%), shrinking to nothing as losses reach 10% of revenue.
  It lays them off one at a time, re-checking after each.
- **Owner's draw.** Each month the owner takes a pro-labore (a senior's wage) out of what
  revenue leaves after wages and overhead, and dividends from profit beyond six months of costs.
  A founder whose firm does not cover its costs draws nothing until its own savings fall below
  six months of living costs; then it takes at least the cost of living from the firm.
- **Bankruptcy.** A firm that cannot pay wages and overhead splits its cash among its staff and
  closes; the owner loses the capital and goes back to the labor market.
- **Founding.** People with savings for 150,000 of capital (with 70% of savings), courage and a
  low time preference found firms, more so when the sector is profitable and when they are
  unemployed or unhappy at work. The new firm offers market wages from the start and its hiring
  horizon reflects the founder's patience.
- **Succession.** When an owner dies, the firm goes to the bidder with the most courage × savings
  among those who can put up the minimum capital, at its worth or what the buyer can pay; with no
  such bidder it closes.

### 2.7 Life

- **Cost of living (food first).** 900 a month. People who can pay it, pay it (the unemployed only
  if next month's food money remains). **In the squeeze** they buy only food (300) and leave rent
  and bills unpaid; 3 such months in a row kill. **Hungry** (not even 300): death, except in a
  first month without a job.
- **Spending.** Most of the wage above subsistence is spent (95% at low pay, toward 70% at high
  pay); 5% a year of savings beyond 6 months of costs. Present-minded people spend more.
- **Time preference** falls by 0.02 in each month of hardship (savings matter) and rises by 0.001
  in each month at work with more than 24 months saved (comfort erodes thrift). With little
  saved, the time preference that drives job search is at least 1 − runway / 6 months.
- **Job satisfaction** moves 10% a month toward how valued the person feels (0.5 at its level's
  average pay, 0 when the pay does not cover the cost of living); new jobs start at 0.7,
  promotions add 0.2 and matched offers 0.1. Below 0.25, with 6 months saved, people may quit.
- **Skills.** Working closes 1.5% a month of the distance between capacity and talent; out of
  work, capacity falls 0.5% a month. Employers promote at capacity 1.5 (pleno) and 2.2 (senior).

## 3. Agents

### 3.1 Worker

One type for everyone: employee, unemployed, or owner (`owner` true). People see only their own
state, the offers they get, public context (prices, wages by level, the sector's margin) and the
public attributes of others (age, experience, level).

| Attribute | Type | Range | Visibility | Meaning |
|---|---|---|---|---|
| `age` | int | 0 to ∞ | public | Age in months (on the CV) |
| `capacity` | float | 0 to ∞ | private | Output per month. Grows with work toward talent, decays while unemployed. Only the employer observes it |
| `talent` | float | 0 to ∞ | private | Capacity a lifetime of work approaches |
| `savings` | int | 0 to ∞ | private | Money saved |
| `wage` | int | 0 to ∞ | private | Monthly wage; 0 when unemployed |
| `lastWage` | int | 0 to ∞ | private | Wage of the last job |
| `expectedWage` | float | 0 to ∞ | private | What the person believes it can earn: the wage of its job, or for the unemployed the last wage revised toward the offers received |
| `bestOffer` | int | 0 to ∞ | private | Best offer received last month while unemployed; 0 when none |
| `employer` | int | 0 to ∞ | private | Id of the employing firm; 0 when unemployed |
| `monthsUnemployed` | int | 0 to ∞ | private | Months in a row without a job |
| `monthsShort` | int | 0 to ∞ | private | Months in a row unable to pay the cost of living |
| `experience` | int | 0 to ∞ | public | Months worked in total (shown on the CV) |
| `level` | string | junior, pleno, senior | public | Title given by employers; never goes down |
| `reservation` | float | 0 to ∞ | private | Lowest wage accepted this month |
| `satisfaction` | float | 0 to 1 (clamped) | private | Job satisfaction (0 to 1): how valued the person feels at work, from its pay against its peers', raises and promotions. Low satisfaction makes it look for another job, and at worst quit |
| `timePreference` | float | 0 to 1 (clamped) | private | Time preference (0 to 1): how much the person values the present over the future (Mises). Present-minded people save less, spend their savings faster and settle for a job sooner. It changes with experience: hardship lowers it, long comfort raises it. Hunger raises it for the moment (see search.lua) |
| `hiredAt` | int | 0 to ∞ | private | Tick at which the person joined its current firm |
| `hiredJunior` | bool |  | private | Joined its current firm as a junior: the firm's investment |
| `investmentLost` | int | 0 to ∞ | private | Firm the person left (poached, quit or laid off) within investmentPayback months of joining it as a junior: an investment that did not pay off, for the firm to learn from; 0 when none |
| `courage` | float | 0 to 1 | private | Willingness to put savings at risk: drives founding and buying firms |
| `owner` | bool |  | private | Owns a firm: lives on its draw and dividends, does not work, and does not retire |
| `income` | int | 0 to ∞ | private | Draw (pro-labore) and dividends received this month (owners) |
| `incomeAvg` | float | 0 to ∞ | private | Draw and dividends per month, smoothed over about a year: an owner's income for the census |
| `paidOff` | int | 0 to ∞ | private | Firm whose junior investment just paid off (the person was promoted there), for the firm to learn from; 0 when none |

### 3.2 Firm

Knows the true capacity of its own staff, and candidates only through interviews and their CV.
Every firm has one owner through the relation `owns` (owner → firm).

| Attribute | Type | Range | Visibility | Meaning |
|---|---|---|---|---|
| `cash` | int |  | private | Money in the bank; below what the payroll needs, the firm goes bankrupt |
| `markdown` | float | 0.02 to 0.9 (clamped) | private | Share of a candidate's estimated value the firm tries to keep when making an offer |
| `openings` | int | 0 to ∞ | private | Jobs the firm is recruiting for this month |
| `layoffsPlanned` | int | 0 to ∞ | private | People the firm decided to lay off this month |
| `headcount` | int | 0 to ∞ | public | People employed |
| `share` | float |  | public | Share of the sector's output |
| `capacitySum` | float |  | private | Total capacity of the staff: output per month |
| `revenue` | int |  | private | Revenue last month |
| `wageBill` | int | 0 to ∞ | private | Wages paid last month |
| `margin` | float |  | private | Profit over revenue last month |
| `underwater` | int | 0 to ∞ | private | Staff whose value to the firm is more than tolerance below their wage |
| `tolerance` | float | 0 to ∞ | private | Patience left this month: the firm's patience, shrinking with losses |
| `patience` | float | 0 to 0.9 | private | How far below their wage a person's value may fall before the firm considers a layoff. Each firm has its own |
| `horizon` | float | 1 to ∞ | private | Months ahead the firm weighs when choosing whom to hire: long horizons invest in people who will grow. It changes with experience: juniors promoted at the firm lengthen it, juniors who leave early shorten it |
| `age` | int | 0 to ∞ | private | Months in business |

### 3.3 Relations

| Relation | Direction | Meaning |
|---|---|---|
| `owns` | owner → firm | Every firm has one owner, a person of the model |

## 4. Context

### 4.1 Parameters

| Field | Type | Default | Visibility | Meaning |
|---|---|---|---|---|
| `minimumWage` | int | 1,600 | public | Reference minimum wage per month. Used only to classify income; it does not bind any wage |
| `subsistence` | int | 900 | public | Cost of living per month: basic food and housing |
| `survivalCost` | int | 300 | public | Least money a month for food. An unemployed person who starts a month without it, after a first month of grace, dies of hunger |
| `monthsShortToDie` | int | 3 | public | Consecutive months unable to pay the full cost of living (in the squeeze, eating but cutting everything else) that kill a person |
| `basePrice` | float | 4,800 | public | Price of one unit of output when the sector produces baseDemand units a month |
| `baseDemand` | float | 2,000 | public | Units a month the market buys at basePrice |
| `elasticity` | float | 1.5 | public | Price elasticity of demand: price = basePrice x (output / baseDemand)^(-1/elasticity) |
| `fixedCost` | int | 5,000 | public | Monthly overhead every firm pays whatever its size, to suppliers outside the model |
| `overheadShare` | float | 0.03 | public | Overhead that grows with the firm (rent, systems, management), as a share of its wage bill |
| `cashBufferMonths` | float | 2 | public | Months of payroll and overhead a firm wants in cash before it hires |
| `payoutAboveMonths` | float | 6 | public | Profit is paid out to the owners as dividends as long as this many months of costs stay in the bank |
| `hireRate` | float | 0.1 | public | Most a firm can grow in a month, as a share of its headcount (recruiting capacity) |
| `minHires` | int | 5 | public | Openings a firm can recruit for in a month, however small it is |
| `interviewsPerOpening` | int | 3 | public | Job seekers a firm interviews per opening |
| `interviewNoise` | float | 0.25 | public | Standard deviation of the relative error of an interview's read of a candidate's capacity |
| `shrinkInterviews` | bool | true | public | Recruiters distrust extreme interview reads and pull them toward the average of the candidates, in proportion to how unreliable interviews are |
| `hiringCost` | float | 0.5 | public | Cost of a hire (recruiting, onboarding), in months of the wage offered |
| `markdownRise` | float | 0.02 | public | How much a firm lowers its offers (raises the markdown) after a month in which it filled all its openings |
| `markdownCut` | float | 0.1 | public | How much a firm raises its offers after a month with openings left unfilled, x the share unfilled |
| `expectedTenure` | float | 24 | public | Months a firm expects a new hire to stay, used to weigh what a hire earns against what it costs |
| `crisisMargin` | float | 0.1 | public | Losses, as a share of revenue, at which a firm has no patience left with staff who cost more than they are worth; patience shrinks in proportion to losses below this |
| `careerCeiling` | float | 2.5 | public | Capacity firms expect an average career to approach, from their experience with their staff |
| `counterMargin` | float | 0.05 | public | Least share of a worker's value the current employer keeps when matching an outside offer |
| `switchPremium` | float | 0.1 | public | Raise an employed worker requires to move to another firm |
| `reservationDecay` | float | 0.02 | public | Monthly drop of an unemployed person's lowest acceptable wage |
| `offerLearning` | float | 0.3 | public | Share of the distance between the wage a job seeker expects and the best offer it got that it revises each month |
| `runwayComfort` | float | 6 | public | Months of savings above which an unemployed person is not pressed to lower the wage it asks |
| `searchBase` | float | 0.05 | public | Monthly weight of looking for another job for an employed person paid like its peers |
| `searchGap` | float | 0.5 | public | Extra weight of looking for another job per unit of pay shortfall against the peers' average wage |
| `mpcLow` | float | 0.95 | public | Share of the wage above subsistence that people with low pay spend |
| `mpcHigh` | float | 0.7 | public | Share of the wage above subsistence that people with very high pay spend |
| `wealthSpending` | float | 0.004 | public | Share of savings spent each month (about 5% a year): consumption out of wealth |
| `satisfactionAdapt` | float | 0.1 | public | Share of the distance to how valued it feels this month that a person's job satisfaction moves each month |
| `satisfactionHire` | float | 0.7 | public | Job satisfaction at the start of a new job (the honeymoon) |
| `promotionBoost` | float | 0.2 | public | Rise in job satisfaction after a promotion |
| `raiseBoost` | float | 0.1 | public | Rise in job satisfaction after the employer matches an outside offer |
| `searchDissatisfaction` | float | 1 | public | Extra weight of looking for another job per unit of job satisfaction below 0.5 |
| `quitSatisfaction` | float | 0.25 | public | Job satisfaction below which a person may quit without another job |
| `quitRunway` | float | 6 | public | Months of living costs a person wants saved before quitting without another job |
| `quitRate` | float | 0.3 | public | Weight of quitting for the most dissatisfied (satisfaction 0); falls to 0 at quitSatisfaction |
| `timePreferenceSpend` | float | 0.2 | public | How much more of the wage above subsistence the most present-minded spend than the average person, and the least spend less |
| `timePreferenceLearn` | float | 0.02 | public | How much a month of hardship (unemployed, or unable to pay the full cost of living) lowers a person's time preference: it learns that savings matter |
| `timePreferenceDrift` | float | 0.001 | public | How much a month at work with savings beyond comfortRunway raises a person's time preference: savings it never needs feel less necessary |
| `comfortRunway` | float | 24 | public | Months of living costs saved beyond which a person at work feels its savings are more than it needs |
| `horizonLearn` | float | 0.05 | public | Relative change of a firm's hiring horizon after one of its junior hires pays off (promoted at the firm) or fails (leaves within investmentPayback months) |
| `investmentPayback` | int | 24 | public | Months a junior hire must stay for the firm to see its investment paid back |
| `minHorizon` | float | 6 | public | Shortest hiring horizon a firm can come to |
| `maxHorizon` | float | 240 | public | Longest hiring horizon a firm can come to |
| `learningRate` | float | 0.015 | public | Monthly share of the distance to one's talent closed by working |
| `skillDecay` | float | 0.005 | public | Monthly share of capacity lost while unemployed |
| `minCapacity` | float | 0.3 | public | Capacity never falls below this |
| `plenoCapacity` | float | 1.5 | public | Capacity at which an employer promotes a junior to pleno |
| `seniorCapacity` | float | 2.2 | public | Capacity at which an employer promotes a pleno to senior |
| `entrantsPerMonth` | float | 3.5 | public | Young people entering the labor market each month, on average |
| `entrantSavingsMonths` | float | 12 | public | Months of living costs entrants bring on average (help from parents, student grants); each draws between half and one and a half times this |
| `entryAge` | int | 216 | public | Age at which people enter the labor market, in months (18 years) |
| `retirementAge` | int | 780 | public | Age at which people retire and leave the labor market, in months (65 years) |
| `mortalityBase` | float | 5e-05 | public | Gompertz a: yearly chance of dying of natural causes at age 0 |
| `mortalityGrowth` | float | 0.09 | public | Gompertz b: growth of the yearly risk per year of age |
| `minFoundingCapital` | int | 150,000 | public | Least capital a new firm needs (about two years of an average wage) |
| `foundingShare` | float | 0.7 | public | Share of its savings a founder puts into the new firm; the rest it keeps to live on |
| `foundRate` | float | 0.003 | public | Monthly weight of founding a firm for an average eligible person in a sector with no profit (about 0.3% a month) |
| `saleProfitMonths` | float | 24 | public | Months of profit a firm is worth, on top of its cash, when it is sold after its owner dies |
| `startupMonths` | int | 6 | public | Months a new firm may hire before it has to be profitable to keep hiring |
| `priceAdapt` | float | 0.3 | public | Share of the latest monthly price change that firms add to their expectation of the price trend |
| `initialEmployment` | float | 0.9 | public | Share of people employed at the start |

### 4.2 State published by the rules

| Field | Type | Default | Visibility | Meaning |
|---|---|---|---|---|
| `priceTrend` | float |  | public | Expected monthly change of the price, from recent changes; firms value hires at the price they expect next month |
| `owners` | float |  | public | Share of the population that owns a firm |
| `ownerIncome` | float |  | public | Mean monthly income of owners (draw and dividends, smoothed over a year) |
| `price` | float | 4,800 | public | Price of a unit of output this month |
| `output` | float |  | public | Units produced by the sector this month |
| `meanWage` | float | 3,000 | public | Average wage of the employed this month |
| `levelWage` | map of float |  | public | Average wage by level (junior, pleno, senior): what salary surveys publish |
| `profitMargin` | float |  | public | Profit of the sector over its revenue last month |
| `hhi` | float |  | public | Herfindahl-Hirschman index of output shares (0 to 1) |
| `seeking` | map of float |  | hidden | This month's job seekers, by agent id: the lowest wage each accepts. Only the market reads it |
| `hires` | int |  | public | Hires from unemployment this month |
| `jobChanges` | int |  | public | Employed people who moved to another firm this month |
| `counteroffers` | int |  | public | Outside offers matched by the current employer this month |
| `layoffs` | int |  | public | People laid off this month, including by bankruptcy |
| `bankruptcies` | int |  | public | Firms that went bankrupt this month |
| `newFirms` | int |  | public | Firms founded this month |
| `closures` | int |  | public | Firms closed this month because their owner died and nobody could buy them |
| `quits` | int |  | public | People who quit their job this month without another one |
| `satisfaction` | float |  | public | Mean job satisfaction of the employed |
| `timePreference` | float |  | public | Mean time preference of the labor force |
| `horizon` | float |  | public | Mean hiring horizon of the firms, in months |
| `deaths` | int |  | public | People who died this month for lack of income: of hunger or after months in the squeeze |
| `hungerDeaths` | int |  | public | People who died of hunger this month (not even survivalCost for food) |
| `squeezed` | float |  | public | Share of the labor force in the squeeze: unable to pay the full cost of living this month |
| `naturalDeaths` | int |  | public | People who died this month of natural causes |
| `retirements` | int |  | public | People who retired this month |
| `unemployment` | float |  | public | Share of the people who are not owners without a job |
| `poverty` | float |  | public | Share of the population with an income under 1 minimum wage (unemployed included) |
| `lowerClass` | float |  | public | Share with an income of 1 to 3 minimum wages |
| `middleClass` | float |  | public | Share with an income of 3 to 10 minimum wages |
| `upperClass` | float |  | public | Share with an income above 10 minimum wages |
| `capacityEmployed` | float |  | public | Mean capacity of the employed |
| `capacityUnemployed` | float |  | public | Mean capacity of the unemployed |
| `seniors` | float |  | public | Share of the labor force at the senior level |

## 5. Decisions

| Decision | Agent | Eligible | Propensity | Effect |
|---|---|---|---|---|
| `search` | worker | not an owner | 50 when unemployed or in the squeeze; else `searchBase + searchGap × max(0, levelWage / wage - 1) + searchDissatisfaction × max(0, 0.5 - satisfaction)` | Registers the lowest wage it accepts in `ctx.seeking` |
| `quit` | worker | employed, not an owner, satisfaction < 0.25, savings ≥ 6 months of costs | `quitRate × (1 - satisfaction / quitSatisfaction)` | Leaves its job, expects at least its peers' wage |
| `found` | worker | not an owner, under 60, 70% of savings ≥ 150,000 | `foundRate × exp(4 (courage - 0.5) - 4 (timePreference - 0.5) + 10 × sector margin + push)`, push 1 if unemployed, else 2 × max(0, 0.5 - satisfaction) | Puts 70% of its savings into a new firm and becomes its owner |
| `expand` | firm | 2 months of costs in cash, and profitable (or in its first 6 months, or with nobody yet) | `5 × sigmoid(-1 + markdown × expectedTenure - hiringCost × (1 - markdown) + 5 × margin)` (no margin term with nobody on staff) | Opens `max(minHires, hireRate × headcount)` jobs |
| `downsize` | firm | someone on staff is more than its tolerance below their wage | `2 × sigmoid(-2 + 4 × underwater / headcount - 5 × margin)` | Plans to lay off those people |

Each agent may also do nothing (idle weight 1).

## 6. Rules

| # | Rule | Phase | Scope | Every | What it does | Script |
|---|---|---|---|---|---|---|
| 0 | `setup` | init | world | once | Starting owners and jobs: the richest people own the first firms, one each; a share of the others already work at a random firm, paid what that firm would offer them | `scripts/setup.lua` |
| 1 | `new_month` | pre | world | 1 month(s) | Resets monthly counters and last month's job seekers | `scripts/new_month.lua` |
| 2 | `entrants` | pre | world | 1 month(s) | Young people enter the labor market | `scripts/entrants.lua` |
| 3 | `production` | pre | world | 1 month(s) | Firms produce and sell at the market price, pay wages, overhead, the owner's draw and dividends; firms that cannot pay go bankrupt. Firms learn from their junior hires. Publishes the price, the price trend, wages by level and each firm's books | `scripts/production.lua` |
| 4 | `living` | pre | worker | 1 month(s) | People pay the cost of living (food first) and spend part of the rest; skills grow or decay; time preference and job satisfaction move; ageing, retirement and death | `scripts/living.lua` |
| 5 | `succession` | pre | world | 1 month(s) | Firms whose owner died are sold to the person with the most courage and savings who can pay; with no buyer, they close | `scripts/succession.lua` |
| 6 | `layoffs` | post | world | 1 month(s) | Firms that decided to downsize lay off the staff with the lowest value for their wage | `scripts/layoffs.lua` |
| 7 | `labor_market` | post | world | 1 month(s) | Firms interview job seekers and make offers; people take the best one; employers may match offers made to their staff | `scripts/labor_market.lua` |
| 8 | `promotions` | post | worker | 12 month(s) | Yearly review: employers promote staff whose capacity reached the next level | `scripts/promotions.lua` |
| 9 | `census` | post | world | 1 month(s) | Unemployment, income classes measured in minimum wages, owners, satisfaction, time preference and the firms' horizons | `scripts/census.lua` |

## 7. Metrics and events

Metrics (one value per month, in `metrics.csv`; `aggregate.csv` over seeds): `workers`, `firms`, `unemployment`, `poverty`, `lowerClass`, `middleClass`, `upperClass`, `meanWage`, `wage.gini`, `price`, `output`, `profitMargin`, `hhi`, `capacityEmployed`, `capacityUnemployed`, `seniors`, `owners`, `ownerIncome`, `priceTrend`, `hires`, `jobChanges`, `counteroffers`, `layoffs`, `bankruptcies`, `newFirms`, `closures`, `quits`, `satisfaction`, `timePreference`, `horizon`, `deaths`, `hungerDeaths`, `squeezed`, `naturalDeaths`, `retirements`, plus
the population of each type and the counters of every decision.

Events (in `events.jsonl`): `died` (cause, age, time preference, months unemployed, months in
the squeeze, capacity, last wage), `bankrupt` (with the firm's patience, horizon and markdown),
`firm_founded`, `firm_sold`, `firm_closed`, `poached`, `counteroffer`, `promoted`, `quit`.

## 8. How it was verified

Each behavior was checked in controlled scenarios before reading aggregate results:

- A person with no savings accepts any offer; one with little capacity revises its expectation
  down after refused offers and accepts when its money runs out.
- A firm with unfilled openings raises its offers; a firm valuing its own price effect lays off
  one person at a time instead of its whole staff.
- A firm with a ten-year horizon ranks a 20-year-old junior first; one with a one-year horizon
  ranks seniors first; a 63-year-old ranks last for both.
- Job satisfaction rises with pay relative to peers (0.46 at 5 to 20% below, 0.78 at 20% above);
  savings fall with time preference (367 months at 0 to 0.3, 66 at 0.7 to 1, ages 35 to 50).
- Recruiters discounting reads against the whole pool overrated juniors, who were hired and laid
  off within months; comparing within levels halved those layoffs.
- Owners with no draw starved while their firms held millions; founders of start-ups that did
  not yet cover their costs starved with cash in their firm; start-ups that could not raise their
  offers or did not try to hire died with no staff. Each was traced to the agents involved and
  replaced by the real behavior (pro-labore, a minimum draw for living, market wages from the
  start, hiring when empty).
- Firms losing money kept hoarding overpaid staff until bankrupt; patience now shrinks with
  losses. Firms bid wages up until they lost money; offers now stop where they no longer cover
  overhead.

## 9. Results

### 9.1 Base run

Over 20 seeds of 40 years (means across seeds):

| Year | People | Firms | Owners | Unemployment | Lower | Middle | Upper | Mean wage | Owner income | Capacity (employed) | Satisfaction | Time preference | Horizon (months) | HHI | Sector margin |
|------|--------|-------|--------|--------------|-------|--------|-------|-----------|--------------|---------------------|--------------|-----------------|------------------|-----|---------------|
| 1    | 1,981  | 20    | 1.0%   | 0.2%         | 81%   | 19%    | 0.2%  | 4,063     | 6,944        | 1.24                | 0.62         | 0.50            | 54               | 0.05 | 18%          |
| 5    | 1,962  | 20    | 1.0%   | 3.3%         | 17%   | 79%    | 0.1%  | 5,778     | 7,968        | 1.85                | 0.59         | 0.50            | 99               | 0.06 | 1%           |
| 10   | 1,937  | 20    | 1.0%   | 2.6%         | 15%   | 82%    | 0%    | 6,138     | 4,865        | 2.15                | 0.55         | 0.51            | 121              | 0.07 | 1%           |
| 20   | 1,913  | 42    | 2.2%   | 3.2%         | 17%   | 78%    | 0.2%  | 6,263     | 4,341        | 2.28                | 0.54         | 0.53            | 98               | 0.10 | 2%           |
| 30   | 1,910  | 63    | 3.3%   | 4.3%         | 17%   | 76%    | 0.3%  | 6,252     | 6,325        | 2.28                | 0.54         | 0.52            | 93               | 0.12 | 3%           |
| 40   | 1,907  | 80    | 4.2%   | 4.6%         | 17%   | 75%    | 0.3%  | 6,210     | 7,128        | 2.27                | 0.54         | 0.48            | 93               | 0.12 | 4%           |

Poverty equals unemployment throughout: nobody employed earns under one minimum wage. Over a run
there are about 20 job changes and 6 matched counteroffers a month, 362 firms founded, 301
bankruptcies, 17 firms sold after an owner's death and fewer than 1 closed for lack of a buyer,
6 quits into unemployment and 18 deaths from lack of income.

### 9.2 What happens, and why

- **Wages rise because people learn and firms compete for them.** Capacity more than doubles in
  ten years and wages follow, through outside offers and counteroffers. Without a minimum wage,
  nobody employed earns less than one: firms bid for workers. Wages are fairly equal (Gini 0.21).
- **Founders are the patient, the brave and the well-off employed.** Founders have a mean courage
  of 0.57 and a time preference of 0.25 (population: 0.5 and 0.5), are 44 years old on average
  and put in about 178,000. All of them were employed when they founded: the unemployed never
  have the savings, so there is no necessity entrepreneurship, only opportunity entrepreneurship.
- **Most start-ups fail within four years.** 83% of new firms go bankrupt, most in their second
  or third year. A start-up has to hire in a tight labor market, against incumbents that already
  pay almost all of what people are worth, and its margin is thin. The sector still grows from 20
  to 80 firms, as founding outpaces failure, and owners grow from 1% to 4% of the population.
- **Entrepreneurship raises unemployment.** Without founding, unemployment is 1.8%; with it,
  4.4%. Failing start-ups lay off their staff, and layoffs rise by 70%. Founding also spreads the
  sector over more firms and lowers its margin.
- **Owners are not rich.** An owner's mean income is about 7,000, close to a senior's wage: most
  owners run small firms that pay little beyond the draw. The upper class stays near 0.3%,
  almost all of it owners of large firms.
- **Poverty deaths come from the initial conditions.** 18 deaths per run, 14 of them in the first
  year: people who start unemployed with almost no savings. Afterwards about 4 in 39 years.
- **Firms learn to invest in juniors, then partly unlearn it.** Hiring horizons lengthen to about
  120 months by year 10, as juniors grow and are promoted, then settle around 93 months, as
  trainees are poached or laid off before paying back (Becker's general training).
- **Time preference moves with experience.** It drifts up during long stretches of comfortable
  work and back down as hardship (start-up failures, layoffs) spreads later in the run.

### 9.3 Sensitivity

Eight seeds each, one change at a time; levels are means of years 31 to 40, counts are totals
over 40 years.

| Scenario | Unemployment | Mean wage | Gini | Owners | Owner income | Firms | HHI | Founded | Bankruptcies | Deaths |
|---|---|---|---|---|---|---|---|---|---|---|
| Base (20 seeds) | 4.4% | 6,230 | 0.21 | 3.7% | 7,083 | 70 | 0.12 | 362 | 301 | 18 |
| No founding (`foundRate` 0) | 1.8% | 6,043 | 0.17 | 1.0% | 27,660 | 19 | 0.14 | 0 | 0.5 | 16 |
| `minFoundingCapital` 50,000 | 12.9% | 6,722 | 0.32 | 8.4% | 2,569 | 161 | 0.10 | 2,379 | 2,212 | 19 |
| `minFoundingCapital` 300,000 | 1.8% | 6,019 | 0.17 | 1.0% | 29,150 | 20 | 0.14 | 11 | 8 | 17 |
| Flat overhead (20,000, no share) | 3.1% | 6,156 | 0.18 | 1.1% | 29,000 | 22 | 0.15 | 349 | 346 | 16 |
| `interviewNoise` 0.1 | 1.8% | 6,274 | 0.19 | 2.4% | 3,101 | 45 | 0.09 | 207 | 177 | 13 |
| `entrantSavingsMonths` 3 | 3.1% | 6,146 | 0.20 | 3.0% | 9,965 | 56 | 0.12 | 268 | 222 | 57 |

- **The capital barrier shapes the economy.** At 300,000 almost nobody founds: 20 firms, 1%
  owners earning about 29,000 a month, low unemployment. At 50,000 thousands of small firms are
  founded and fail: 13% unemployment, a Gini of 0.32, and a sector that loses money on average.
- **With flat overhead, start-ups cannot survive at all** (349 founded, 346 bankrupt): a fixed
  20,000 a month is a barrier to entry that only large incumbents can carry.
- **Accurate interviews** cut layoffs by 60% and unemployment to 1.8%, and fewer firms are founded
  and fail.
- **Family support** still decides who survives the start of working life: with three months
  instead of twelve, poverty deaths triple.

## 10. Problems and limitations

### 10.1 Unrealistic structure

- **Open economy.** Demand comes from buyers outside the model, and the cost of living,
  consumption, overhead and hiring costs leave it. Unemployment does not reduce demand, so there
  are no recessions, and money is not conserved.
- **Fixed cost of living.** Food and housing cost the same every month whatever happens; there
  is no market for them. A second sector (essentials) would make them prices.
- **One sector, one skill.** No education choice, no specialization, no moving between sectors,
  no structural unemployment.
- **No credit and no interest.** Time preference shapes individual saving but has no market in
  which to set an interest rate, and firms cannot borrow to grow; Mises's central mechanism is
  missing.

### 10.2 Unrealistic behavior

- **No families, heirs or mutual aid.** People live alone; sale proceeds after an owner's death
  vanish; nobody helps the unemployed. Death by hunger a month after the money runs out is far
  harsher than reality, where family, charity and informal work exist even without a state.
- **No private contracts.** Without a labor code, firms and workers would negotiate severance,
  notice and fixed terms; here nobody does.
- **No reputation.** Firms that lay off often or underpay hire as easily as any other.
- **Wages never fall within a job.** Firms adjust only by laying people off.
- **Firms do not correct for competition.** They discount noisy reads but not the winner's curse
  of outbidding rivals (no bid shading, no probation, no references).
- **Owners do not work and never retire,** and own at most one firm.
- **Job satisfaction comes only from pay,** promotions and raises: no working conditions, hours,
  managers or colleagues.

### 10.3 Information that is too good

- Salary surveys publish the exact average wage of each level every month.
- Firms know the exact noise of their interviews, how fast people learn, and how far an average
  career goes.
- People know the sector's exact profit margin when deciding to found a firm.

### 10.4 Calibration and initial conditions

- `basePrice` and `baseDemand` were calibrated so that an average new worker adds about 3
  minimum wages of revenue. They set the wage level in minimum wages, so the income class shares
  are conditional on them.
- Most behavioral parameters are stated assumptions, not estimates; see the sensitivity tables.
- Everyone starts as a junior with base capacity: the first ten years are a transition.
- The richest people own the first firms, but each firm starts with 500,000 in cash that came
  from nowhere, and 10% of people start unemployed with whatever savings they drew; most poverty
  deaths happen in the first year for that reason.

### 10.5 Results that do not match reality

- **Start-up failure is too high.** 83% of new firms fail within about four years; in reality
  about half survive five years. Start-ups here have no niche, no product of their own and no
  credit: they compete for the same workers with the same product.
- **No necessity entrepreneurship.** Every founder was employed; the unemployed never have the
  capital. Real economies have many small businesses founded out of unemployment, with little
  capital.
- **Unemployment is low** (2 to 5%): there are no recessions, no geography, no mismatched skills,
  no discrimination, no health shocks.
- **The top is compressed.** The upper class is about 0.3% and owners earn about a senior's wage
  on average. Real top incomes come from large firms, several businesses, capital gains and
  rare skills.
- **Sector margins of 1 to 4%** are thin, because firms bid wages up to almost what people are
  worth and there is no differentiation between firms.

### 10.6 Practical

- A 40-year run takes about 25 seconds; 20 seeds about 8 minutes on 4 cores.
- One decision per agent per month: a person cannot look for a job and plan a firm in the same
  month.

## 11. Next steps

1. **Two sectors (essentials and the current one)**, so that the cost of living becomes prices
   and wages feed demand (a closed economy).
2. **Families and mutual aid**, so that survival without a state rests on the institutions that
   historically provided it.
3. **Credit with interest set by savers' time preference**, so that time preference shapes the
   economy and firms can borrow to grow.
4. **Private labor contracts and reputation**, which a market without a labor code would develop.
