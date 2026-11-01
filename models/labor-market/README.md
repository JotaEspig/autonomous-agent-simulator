# Labor market

A labor market without direct state regulation: no minimum wage, no labor code (no mandatory
severance, notice or contracts), no unemployment insurance, no welfare. Firms of one sector want
to grow their profit, and for that they need capable workers; workers want the highest pay
they can get and to feel valued, but must pay the cost of living every month or die. Capacity
grows with work. Firms read candidates through noisy interviews, invest in people who will
grow, make individual offers, poach each other's workers and counter outside offers. Workers
differ in how much they value the present over the future, and may quit jobs where they feel
undervalued. A minimum wage exists only as a yardstick: it classifies people into poverty,
lower, middle and upper class by what they earn, and binds nobody.

The question: left to themselves, how do competition for skilled labor, learning on the job,
time preference and the need to earn a living distribute wages and jobs? Who ends up poor, and
why?

```sh
agentsim run models/labor-market --seeds 20
```

## A month

One tick is a month; a run is 40 years (480 months).

| Phase  | Step           | What happens                                                                 |
|--------|----------------|------------------------------------------------------------------------------|
| init   | `setup`        | The 20 richest people own the 20 firms; 90% of the others start employed at a random firm, paid what that firm would offer them |
| pre    | `new_month`    | Monthly counters and last month's job seekers reset                          |
| pre    | `entrants`     | Young people enter the labor market                                          |
| pre    | `production`   | Firms produce and sell, pay wages, overhead, the owner's draw and dividends; firms that cannot pay go bankrupt. Prices, the price trend, wages by level and each firm's books are published; firms learn from their junior hires |
| pre    | `living`       | People pay the cost of living and spend; skills grow or rust; time preference and job satisfaction move; ageing, retirement, death |
| pre    | `succession`   | Firms whose owner died are sold to the best bidder, or close |
| decide | decisions      | People decide whether to look for a job, quit or found a firm; firms whether to open jobs or downsize |
| post   | `layoffs`      | Firms that decided to downsize lay people off                                |
| post   | `labor_market` | Interviews, offers in rounds, choices and counteroffers                      |
| post   | `promotions`   | Once a year, employers promote people whose capacity reached the next level  |
| post   | `census`       | Unemployment, income classes and capacity are measured                       |

Being paid, paying the bills and learning are routine, not choices: they are rules that run for
everyone every month. Decisions are kept for strategic choices (looking for a job, hiring,
downsizing).

## Agents

### Worker

| Aspect      | Content |
|-------------|---------|
| State       | `age` (months), `capacity` (output per month), `talent` (the capacity a lifetime of work approaches), `savings`, `wage` (0 when unemployed), `lastWage`, `expectedWage`, `employer` (firm id, 0 when unemployed), `monthsUnemployed`, `monthsShort` (months in a row in the squeeze), `age` and `experience` (public, on the CV), `level` (public: junior, pleno, senior), `reservation`, `bestOffer`, `satisfaction` (job satisfaction, 0 to 1), `timePreference` (0 to 1), `courage` (0 to 1), `owner`, `income` and `incomeAvg` (owners' draw and dividends) |
| Perception  | Its own state; the offers it gets; the average wage of each level (`levelWage`, as salary surveys publish). It does **not** know its exact capacity: its idea of its own worth comes from its level, its pay and the offers it gets |
| Goals       | 1) eat; 2) pay the rest of the cost of living; 3) earn more and feel valued |
| Decisions   | `search`: look for a job and set the lowest wage it accepts; `quit`: leave a job where it feels undervalued, without another one; `found`: start a firm |
| Routine     | Paid by the firm; pays for food, then the rest of the cost of living; spends part of what is left; learns or loses skills; ages |
| Constraints | Savings never go negative; one job at a time; one decision a month |

### Firm

| Aspect      | Content |
|-------------|---------|
| State       | `cash`, `markdown` (share of a hire's estimated value the firm keeps), `patience` (how far below their wage it lets people's value fall before considering a layoff), `tolerance` (the patience left this month, which shrinks with losses), `horizon` (months ahead it weighs when hiring), `openings`, `layoffsPlanned`, `headcount` and `share` (public), `capacitySum`, `revenue`, `wageBill`, `margin`, `underwater`, `age` |
| Perception  | The true capacity of its own staff; candidates only through an interview (capacity read with a relative error of `interviewNoise`, which recruiters know from experience) and their CV (age); how fast people learn and how far careers go, from its own staff; the market price and its own output share |
| Goals       | Grow profit by employing capable people |
| Decisions   | `expand` (open jobs), `downsize` (lay off people who cost more than they are worth), or nothing |
| Constraints | Must pay wages and overhead or go bankrupt; can recruit for only so many jobs a month |

Every firm has an owner, a person of the model (relation `owns`, owner → firm). Owners do not
work and are not job seekers: they live on the firm's draw and dividends, and keep their firm
until they die (they do not retire). An owner whose firm goes bankrupt loses the capital and goes
back to the labor market.

`agents/workers.agent.json` creates 2,000 people aged 18 to 64, all juniors, with capacity drawn
from normal(1.0, 0.2) in [0.5, 1.5], talent from normal(2.5, 0.6) in [1.2, 4] and savings from
normal(5000, 4000) (at least 0). `agents/firms.agent.json` creates 20 firms with 500,000 in cash,
a markdown drawn from uniform(0.2, 0.5), a patience from uniform(0.02, 0.3) and a hiring horizon
from uniform(12, 120) months. Firms differ in how they are run, and new firms draw theirs the
same way.

## Behavior

### Production and prices

A firm produces the sum of its staff's capacities. The sector sells everything it produces at
the price the market bears, with constant elasticity demand:

    price = basePrice × (output / baseDemand)^(-1 / elasticity)

The more the sector produces, the lower the price. A person's **value** to a firm is the revenue
it adds once the firm's own effect on the price is counted (marginal revenue product):

    value = price × capacity × (1 - share / elasticity)

That is the brake on expansion: a growing firm pushes its own price down, and a firm's offers
fall as its share grows. Demand is outside the model (consumers are not the workers). Each firm
pays overhead: `fixedCost` (5,000) a month whatever its size, plus `overheadShare` (3%) of its
wage bill, since rent, systems and management grow with the firm.

Firms read demand from the price. The expected monthly price change (`priceTrend`) moves
`priceAdapt` (30%) of the way toward the latest change, and firms value hires at the price they
expect next month: a falling price, the sign that buyers want less than the sector makes at the
old price, makes hiring less attractive.

### Looking for a job (`search`)

- **Who looks.** The unemployed almost always (propensity 50, idle 1), and so do the employed
  whose pay does not cover the cost of living (in the squeeze). Other employed people with
  propensity `searchBase + searchGap × max(0, levelWage / wage - 1)`: now and then, and more
  often the further their pay is below the average of their level. People sense they are
  underpaid by comparing with peers.
- **Job satisfaction** adds to the urge to look: `searchDissatisfaction × max(0, 0.5 - satisfaction)`.
  Dissatisfied people look around and quit more (Freeman 1978; Clark 2001).
- **Lowest acceptable wage of the employed**: the current wage plus `switchPremium` (10%).
  Changing jobs brings raises of about 10% in the data, and nobody moves for the same pay. In the
  squeeze, any raise will do.
- **Lowest acceptable wage of the unemployed** (reservation wage). The poorer, the more desperate:

      reservation = expectedWage × (1 - reservationDecay × (0.5 + timePreference))^monthsUnemployed × pressure
      pressure    = min(1, runway / runwayComfort)      runway = savings / subsistence

  Present-minded people (high `timePreference`) lower their demands faster: they discount the
  better job they might find later (DellaVigna and Paserman 2005). **Scarcity overrides
  character**: with little saved, the present weighs more whatever the person's usual time
  preference, so the one that drives the search is at least `1 - runway / runwayComfort` (1 with
  nothing saved), for as long as money is short (Mullainathan and Shafir 2013). Food always comes
  first: time preference never makes anyone spend the money for food (see Life).

  With three months saved (half of `runwayComfort`) a person asks half of what it expects; with
  less than one month, or already in the squeeze, it takes any job to eat. Liquidity constraints
  make the poor accept worse jobs sooner (Chetty 2008). `expectedWage`
  starts at the last wage (with no job yet, the average wage of the level) and moves each month
  by `offerLearning` toward the best offer received. Reservation wages are anchored on the last
  wage and fall slowly with unemployment, but sharply when money runs out (Krueger and Mueller
  2016); offers teach job seekers what they are worth (Conlon et al. 2018).

### Quitting (`quit`)

Someone whose job satisfaction has fallen below `quitSatisfaction` (0.25) and who has saved at
least `quitRunway` (6) months of living costs may walk out without another job, with propensity
`quitRate × (1 - satisfaction / quitSatisfaction)`: the lower the satisfaction, the likelier. It
expects to earn at least what its peers earn, since it left because it felt underpaid. Without
savings, the dissatisfied stay and look for another job while they work.

### Hiring (`expand`) and the labor market

- **Opening jobs.** A firm opens jobs only if it is **profitable** (positive margin last month),
  or is still starting up (`startupMonths`, 6) or has nobody yet, and has at least
  `cashBufferMonths` of costs in cash. A firm losing money does not grow. The propensity is
  `5 × sigmoid(-1 + markdown × expectedTenure - hiringCost × (1 - markdown) + 5 × margin)`
  (without the margin term for a firm with no staff, whose only way to earn is to hire): more
  eagerly the more a hire earns it over its expected tenure against the cost of recruiting. It opens `max(minHires, hireRate × headcount)` jobs,
  its recruiting capacity for the month.
- **Interviews.** Each firm interviews `interviewsPerOpening` job seekers per opening, drawn at
  random (it does not meet everyone), and reads each one's capacity with a relative error of
  `interviewNoise` (25%).
- **Distrust of extreme reads** (`shrinkInterviews`). Recruiters know how much interviews err,
  so they can tell how much of the spread among the month's candidates is real, and pull each
  read toward the average of the candidates **of the same level** (the title is on the CV, so
  a junior is compared with juniors) in proportion to how unreliable reads are:

      reliability = (variance of the reads - variance of the error) / variance of the reads
      estimate    = average + reliability × (read - average)

  This is the best guess of a capacity from a noisy read (Bayesian shrinkage, Kelley's formula).
  A candidate who looks brilliant is probably good *and* lucky in the interview; one who looks
  weak, probably weak *and* unlucky.
- **Hiring as an investment.** The firm ranks candidates by the profit it expects from them over
  its own `horizon`, cut short by the months the candidate has before retirement (age is on the
  CV). Firms know from their staff that capacity grows with work toward what an average career
  reaches (`careerCeiling`, 2.5) at `learningRate` a month, while the wage stays what was agreed
  unless a rival bids:

      months   = min(horizon, months to retirement)
      average  = C - (C - estimate) × (1 - (1 - learningRate)^months) / (months × learningRate)
      profit   = months × (unit value × average - offer) - hiringCost × offer

  with `C = max(careerCeiling, estimate)`. A firm that looks ten years ahead prefers a cheap
  20-year-old junior to a senior; one that looks one year ahead prefers the senior; and someone
  close to retirement ranks low for both. Candidates who would not pay for themselves get no
  offer.
- **Learning whether juniors pay.** The horizon is not fixed: each junior hire later promoted at
  the firm lengthens it by `horizonLearn` (5%), and each one who leaves within
  `investmentPayback` (24) months, poached, quitting or laid off, shortens it by as much, within
  `minHorizon` and `maxHorizon` (6 to 240 months). Firms that lose their trainees to rivals learn
  to stop training, as Becker's theory of general training predicts.
- **Offers in rounds.** The firm offers jobs to its best-ranked candidates at
  `value(estimate, share after the hire) × (1 - markdown)`: pay follows what the person delivers
  today. When a candidate turns it down or
  takes another job, the firm moves on to the next candidate on its list in the next round,
  until its openings are filled or its list runs out.
- **Choice.** Each person takes its best offer if it meets its reservation wage.
- **Counteroffers.** When one of its people is about to leave, the employer, who knows the
  person's real capacity, matches the outside offer if it still keeps `counterMargin` of the
  person's value. The person stays at the new wage. Wages rise within a job only this way, through
  outside offers (sequential auctions, Postel-Vinay and Robin 2002); they never fall.
- **Learning the going wage.** A firm that filled its openings offers a little less next month
  (`markdownRise`); one left with vacancies offers more, in proportion to the share unfilled
  (`markdownCut`), but never so much that what it keeps stops covering its overhead (overhead
  over its revenue): no firm bids itself into losses. A start-up with no revenue yet has no such
  floor: it must match the market to hire anyone. Each hire costs `hiringCost` months of its
  wage.

### Layoffs (`downsize`)

The firm counts the staff whose value is more than its `tolerance` below their wage. It keeps
people through smaller shortfalls (labor hoarding). Each firm has its own `patience`, from 2% to
30%; a firm losing money loses patience in proportion to its losses, down to none when they reach
`crisisMargin` (10%) of revenue:

    tolerance = patience × clamp(1 + margin / crisisMargin, 0, 1)
 With such people on staff, it may
decide to downsize (propensity rises with their share and with losses). It then lays them off
one at a time, lowest value for the wage first. Each layoff shrinks its share and so raises what
the others are worth, and it stops when the next person is no longer clearly underwater.

### Payday and bankruptcy

Wages and overhead come out of revenue and cash. Then the owner takes a **draw** (pro-labore):
what a senior earns, the pay of the person running the firm, but only out of what revenue leaves
after wages and overhead. A founder whose firm does not cover its costs yet draws nothing and
lives on its own savings, until they fall below `runwayComfort` months of living costs: then it
takes at least the cost of living from the firm, since nobody starves with money in the till. Profit is paid out to the owner as dividends as long as
`payoutAboveMonths` of costs stay in the bank, so owners never take back the capital they put in.
A firm that cannot pay its wages and overhead goes bankrupt: its cash is split among its staff in
proportion to their wages, everyone is out of a job, and the owner loses its capital.

### Founding a firm (`found`)

A person who is not an owner, under 60, and whose savings let it put up `minFoundingCapital`
(150,000, about two years of an average wage) with `foundingShare` (70%) of them, may found a
firm. The propensity is

    foundRate × exp(4 × (courage - 0.5) - 4 × (timePreference - 0.5) + 10 × sector margin + push)
    push = 1 if unemployed, else 2 × max(0, 0.5 - satisfaction)

Courage (willingness to put savings at risk) and a low time preference (willing to give up
consumption now for profit later) make people found firms; so do a profitable sector
(opportunity) and being out of work or unhappy at work (necessity and push). Wealth is the main
barrier to entrepreneurship (Evans and Jovanovic 1989). The founder puts in 70% of its savings,
leaves its job and becomes the owner. The founder knows what the sector pays, so the firm starts
offering competitive wages (a markdown of the sector's margin, at least 5%); its hiring horizon
starts from the founder's time preference (12 months at 1, 120 at 0): a patient founder invests
further ahead.

### Succession

When an owner dies, the estate sells the firm to the best bid it can get. The firm is worth its
cash plus `saleProfitMonths` (24) of its recent profit. The buyer is the person, not already an
owner, with the most courage × savings among those who could start a firm; it pays the firm's
worth, or what it can put in (70% of its savings) if that is less, since an estate that finds no
buyer at its price lowers it. There are no heirs, so the price leaves the economy. With nobody
able to put up `minFoundingCapital`, the firm closes and its staff lose their jobs.

### Life

- **Cost of living.** `subsistence` (900) a month, paid from savings after payday, food first.
  Three situations:

  | Situation      | When                                                   | What happens |
  |----------------|--------------------------------------------------------|--------------|
  | Fine           | Can pay the whole cost (the unemployed only if what is left still buys next month's food) | Pays 900 |
  | In the squeeze | Has at least `survivalCost` (300) but not the rest      | Buys only food (300) and leaves rent and bills unpaid; `monthsShortToDie` (3) such months in a row kill |
  | Hungry         | Not even 300 for food                                   | Dies of hunger, except in a first month without a job (the last paycheck, the family's help) |

  People ration: someone without a job starts cutting rent and bills before the money for food
  runs out, rather than paying everything and starving next month.
- **Spending and time preference.** People spend most of what the wage leaves above
  subsistence: `mpcLow` (95%) when pay is low, falling toward `mpcHigh` (70%) as it rises. They
  also spend `wealthSpending` of their savings each month (about 5% a year, consumption out of
  wealth), but only out of savings beyond `runwayComfort` (6) months of living costs: the safety
  margin for food and rent is never spent on wants. Each person has a `timePreference`, drawn
  from normal(0.5, 0.2) in [0, 1]: how much it values the present over the future (Mises's time
  preference). Present-minded people spend up to `timePreferenceSpend` (20 points) more of the
  surplus and up to half again as much of their spare savings; future-minded people save more:

      spent share of surplus = mpc + 2 × timePreferenceSpend × (timePreference - 0.5)
      spent share of spare savings = wealthSpending × (0.5 + timePreference)

  **Time preference changes with experience.** A month of hardship (unemployed, or unable to pay
  the full cost of living) teaches that savings matter and lowers it by `timePreferenceLearn`
  (0.02); a month at work with more than `comfortRunway` (24) months saved raises it by
  `timePreferenceDrift` (0.001), as savings never needed feel less necessary. People who lived
  through hard times save more afterwards (Malmendier and Nagel 2011).
- **Job satisfaction.** Each month at work, satisfaction moves `satisfactionAdapt` (10%) of the
  way toward how valued the person feels: 0.5 when paid like the average of its level, higher
  when paid more (0.7 at 20% above), lower when paid less, and 0 when the pay does not cover the
  cost of living. People judge their pay against their peers' (Card et al. 2012). A new job
  starts at `satisfactionHire` (0.7, the honeymoon); a promotion adds `promotionBoost` (0.2), and
  an employer matching an outside offer adds `raiseBoost` (0.1).
- **Learning by doing.** Working closes `learningRate` (1.5%) of the distance between capacity
  and talent each month: a learning curve, fast at first and slower near one's limit. Out of
  work, capacity rusts by `skillDecay` (0.5% a month).
- **Levels.** Once a year, employers promote juniors whose capacity reached `plenoCapacity`
  (1.5) to pleno, and plenos reaching `seniorCapacity` (2.2) to senior. Titles never go down.
- **Ageing.** People enter at 18 with `entrantSavingsMonths` (12) of living costs on average,
  between half and one and a half times that (help from parents, student grants; families
  differ), at `entrantsPerMonth` (3.5, which replaces a working life of 47 years for 2,000
  people). They retire at 65 and may die of natural causes (Gompertz law).

### Income classes

The census measures the population by income, in minimum wages (`minimumWage`, 1,600): the wage,
or for owners their draw and dividends smoothed over about a year. Poverty below 1 (the
unemployed included), lower class 1 to 3, middle class 3 to 10, upper class above 10.
Unemployment is measured among those who are not owners.
The minimum wage is only this yardstick: no wage is bound by it.

## Parameters

| Field | Default | Source or assumption |
|-------|---------|----------------------|
| `minimumWage` | 1,600 | Order of magnitude of Brazil's minimum wage; used only to classify income |
| `subsistence` | 900 | Assumption: a basic food basket (DIEESE, about 700 to 850) plus minimal housing |
| `monthsShortToDie` | 3 | User's T |
| `basePrice`, `baseDemand` | 4,800, 2,000 | Calibration at the start: an average worker (capacity 1) adds about 3 minimum wages of revenue when the whole initial labor force works. The most uncertain parameter (see sensitivity) |
| `elasticity` | 1.5 | Assumption: demand for one sector's product, with substitutes, is elastic |
| `entrantSavingsMonths` | 12 | User's choice: help from parents and student grants |
| `timePreference` (per person) | normal(0.5, 0.2) in [0, 1] | Assumption: people differ in patience |
| `timePreferenceSpend` | 0.2 | Assumption: from saving 30% more to 30% less of the surplus than average |
| `satisfactionAdapt`, `satisfactionHire` | 0.1, 0.7 | Assumption: feelings about a job change over months; new jobs start with a honeymoon (Boswell et al. 2005) |
| `promotionBoost`, `raiseBoost` | 0.2, 0.1 | Assumption |
| `timePreferenceLearn`, `timePreferenceDrift`, `comfortRunway` | 0.02, 0.001, 24 months | Assumption: hardship teaches fast, comfort erodes slowly |
| `horizonLearn`, `investmentPayback` | 5%, 24 months | Assumption: about the time a junior takes to reach pleno |
| `minHorizon`, `maxHorizon` | 6, 240 months | Assumption |
| `searchDissatisfaction` | 1 | Assumption: a person at satisfaction 0 looks half as often as an unemployed one would against idle |
| `quitSatisfaction`, `quitRunway`, `quitRate` | 0.25, 6 months, 0.3 | Assumption |
| `survivalCost` | 300 | User's floor for food: below a basic food basket, a bare minimum of calories |
| `fixedCost`, `overheadShare` | 5,000, 3% | Assumption: a firm of 100 pays about what it did with a flat 20,000; small firms pay much less |
| `minFoundingCapital`, `foundingShare` | 150,000, 70% | Assumption: about two years of an average wage; founders keep some savings to live on |
| `foundRate` | 0.3% a month | Order of magnitude of monthly business start rates among adults |
| `courage` (per person) | normal(0.5, 0.2) in [0, 1] | Assumption: people differ in risk tolerance |
| `saleProfitMonths` | 24 | Assumption: small firms sell for a few years of profit |
| `startupMonths` | 6 | Assumption |
| `priceAdapt` | 0.3 | Assumption, adaptive expectations |
| `interviewNoise` | 0.25 | Assumption: interviews predict job performance only moderately (Schmidt and Hunter 1998) |
| `shrinkInterviews` | true | Recruiters discount extreme reads; set false to see firms that take reads at face value |
| `hiringCost` | 0.5 months | Assumption: recruiting and onboarding |
| `hireRate`, `minHires` | 10%, 5 | Assumption: recruiting capacity per month |
| `switchPremium` | 10% | Raises at job changes are around 10% |
| `reservationDecay` | 2% a month | Krueger and Mueller (2016): slow decline with unemployment duration |
| `offerLearning` | 0.3 | Assumption, adaptive expectations |
| `learningRate`, `skillDecay` | 1.5%, 0.5% a month | Assumption: about 7 years from junior to senior; skills rust a few percent a year out of work |
| `mpcLow`, `mpcHigh`, `wealthSpending` | 0.95, 0.7, 0.4% | Poorer households spend almost all income; richer ones save more; consumption out of wealth of 3 to 6% a year |
| `patience` (per firm) | uniform(2%, 30%) | Assumption: firms differ in how long they hoard labor |
| `crisisMargin` | 10% | Assumption: losses at which a firm keeps nobody who costs more than they are worth |
| `horizon` (per firm) | uniform(12, 120) months at the start | Assumption: from firms that plan a year ahead to firms that invest for a decade; then learned |
| `careerCeiling` | 2.5 | The mean of talent: what firms see an average career reach |

## Metrics

`workers`, `firms`, `unemployment`, `poverty`, `lowerClass`, `middleClass`, `upperClass`,
`meanWage`, `wage.gini`, `price`, `output`, `profitMargin`, `hhi` (concentration of output),
`capacityEmployed`, `capacityUnemployed`, `seniors`, `hires`, `jobChanges`, `counteroffers`,
`layoffs`, `quits` (into unemployment), `satisfaction` (mean of the employed), `bankruptcies`,
`newFirms`, `closures` (owner died, no buyer), `owners` (share of the population),
`ownerIncome` (mean, smoothed), `priceTrend`, `timePreference` (mean of the labor force),
`horizon` (mean of the firms), `deaths` (lack of income), `hungerDeaths`, `squeezed` (share of
the labor force in the squeeze), `naturalDeaths`, `retirements`, plus the population and the
counters of every decision.

Events: `died` (cause: hunger or squeeze; age, time preference, months unemployed, months in the squeeze,
capacity, last wage), `bankrupt` (with the firm's patience, horizon and markdown), `poached`,
`counteroffer`, `promoted`, `quit`, `firm_founded` (capital, courage, time preference, age),
`firm_sold`, `firm_closed`.

## Results

The full report, with every attribute, parameter, decision and rule, the results over 20 seeds,
the sensitivity analysis and the model's problems, is in [REPORT.md](REPORT.md). In short, over
20 seeds of 40 years:

- **Wages rise because people learn and firms compete for them**, and nobody employed earns less
  than a minimum wage, even without one. Wages are fairly equal (Gini 0.21).
- **Poverty is unemployment**, 2.6% to 4.6%, rising over the run as failing start-ups lay off
  their staff (1.8% without entrepreneurship).
- **Founders are the patient, the brave and the well-off employed** (courage 0.57, time
  preference 0.25, aged 44 on average); nobody founds out of unemployment, for lack of capital.
- **Most start-ups fail** (83% within about four years), yet the sector grows from 20 to 80
  firms and owners from 1% to 4% of the population. Owners earn about a senior's wage on average;
  the upper class stays near 0.3%.
- **Poverty deaths come from the initial conditions**: 18 per run, 14 in the first year.
- **Firms learn to invest in juniors**, then partly unlearn it as trainees are poached.
- **The capital barrier shapes the economy**: at 50,000 thousands of small firms fail and
  unemployment reaches 13%; at 300,000 almost nobody founds.

## Realism

- **Matches stylized facts:** wages rising with experience and through job changes, counteroffers,
  reservation wages anchored on the last wage and collapsing when money runs out, job
  satisfaction tied to relative pay, more saving after hardship, underinvestment in general
  training, high start-up failure, entrepreneurs who are more risk tolerant and patient.
- **Does not match:** start-up failure is too high, there is no necessity entrepreneurship,
  unemployment is low (no recessions), and the top is compressed. See REPORT.md, section 10.

## Limitations and next steps

- **No households, mutual aid or informal sector.** Families that share income, mutual aid
  societies (common before welfare states), or informal work at low pay.
- **No small-scale or necessity entrepreneurship.** Only people with 150,000 of capital found
  firms, all in the same sector, with no niche of their own and no credit.
- **No private contracts.** Without a labor code, firms and workers would negotiate terms:
  severance, notice, fixed-term contracts, non-compete clauses. Firms could offer them to attract
  workers; workers could demand them.
- **No reputation.** Firms that lay off often, or underpay, would find hiring harder; workers who
  quit often, likewise.
- **No credit or interest.** Time preference shapes saving, but there is no loan market where it
  would set an interest rate, and no capital for firms to invest.
- **Demand and the cost of living are outside the model.** The workers' wages do not buy the
  sector's product, the cost of living is fixed, and overhead leaves the economy, so there is no
  feedback from unemployment to demand and no recessions.
- **Wages never fall within a job.** Firms adjust only through layoffs.
- **One sector, one kind of skill.** No education choice, no specialization, no other sectors.
- **The start is artificial.** Everyone starts as a junior with base capacity, the first firms'
  cash comes from nowhere, and some people start unemployed with nearly no savings.
