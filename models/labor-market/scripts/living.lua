-- A month in a person's life, after payday:
--   pays the cost of living (subsistence) from savings. Food comes first, so people ration:
--     can pay it all: fine. Someone without a job pays it all only if what is left still buys
--       next month's food (survivalCost); otherwise it starts cutting now;
--     in the squeeze: buys only food (survivalCost) and leaves the rest unpaid (rent,
--       transport, bills); monthsShortToDie such months in a row kill (eviction, illness,
--       the household gives way);
--     hungry: not even survivalCost for food. The person dies of hunger, unless it has just
--       lost its job or just arrived (monthsUnemployed 0), which leaves a first month of grace
--       (the last paycheck, the family's help);
--   spends part of what the wage leaves above subsistence: people with low pay spend almost
--   all of it (mpcLow), people with high pay save a larger share (falling toward mpcHigh);
--   and spends wealthSpending of its savings, as people consume out of wealth (about 5% a
--   year), so savings do not pile up forever, but only out of savings beyond runwayComfort
--   months of living costs: the safety margin for food and rent is never spent on wants.
--   Time preference shifts both: present-minded people (timePreference near 1) spend up to
--   timePreferenceSpend more of the surplus and half again as much of their spare savings;
--   future-minded people (near 0) save more. Higher time preference means a higher valuation
--   of present over future goods (Mises), and impatience goes with lower saving in the data;
--   learns from experience: a month of hardship (unemployed, or unable to pay the full cost
--   of living) teaches that savings matter and lowers time preference by
--   timePreferenceLearn; a month at work with more than comfortRunway months saved raises it
--   by timePreferenceDrift, as savings never needed feel less necessary. People who lived
--   through hard times save more afterwards (Malmendier and Nagel 2011);
--   at work, feels more or less valued: job satisfaction moves satisfactionAdapt of the way
--   toward this month's feeling, which is 0.5 when paid like its level's average, higher
--   when paid more (0.7 at 20% above), lower when paid less, and 0 when the pay does not
--   cover the cost of living. People judge their pay against their peers' (Card et al. 2012);
--   works and learns: capacity closes learningRate of the distance to talent each month
--   (a learning curve: fast at first, slower near one's limit); unemployed, it decays by
--   skillDecay a month (skills rust);
--   ages; retires at retirementAge (owners keep their firm and do not retire); may die of
--   natural causes (Gompertz law).
-- Owners live on their dividends (income) instead of a wage, and are not job seekers.

function run(self, world)
  local ctx = world.ctx
  local savings = self.savings
  local short = 0
  local jobless = self.employer == 0 and not self.owner
  local hungry = savings < ctx.survivalCost and not (jobless and self.monthsUnemployed == 0)
  local reserve = 0
  if jobless then
    reserve = ctx.survivalCost
  end
  if savings - ctx.subsistence >= reserve then
    savings = savings - ctx.subsistence
  else
    savings = math.max(0, savings - ctx.survivalCost)
    short = self.monthsShort + 1
  end

  if hungry or short >= ctx.monthsShortToDie then
    local cause = hungry and "hunger" or "squeeze"
    local effects = {
      fx.add("ctx.deaths", 1),
      fx.emit("died", { cause = cause, age = math.floor(self.age / 12), timePreference = self.timePreference, monthsUnemployed = self.monthsUnemployed, monthsShort = short, capacity = self.capacity, lastWage = self.lastWage, wage = self.wage }),
      fx.remove(),
    }
    if hungry then
      effects[#effects + 1] = fx.add("ctx.hungerDeaths", 1)
    end
    return effects
  end
  local years = self.age / 12
  if world:chance(ctx.mortalityBase * math.exp(ctx.mortalityGrowth * years) / 12) then
    return { fx.add("ctx.naturalDeaths", 1), fx.remove() }
  end
  if self.age >= ctx.retirementAge and not self.owner then
    return { fx.add("ctx.retirements", 1), fx.remove() }
  end

  local earned = self.wage
  if self.owner then
    earned = self.income
  end
  local surplus = math.max(0, earned - ctx.subsistence)
  if surplus > 0 then
    local mpc = ctx.mpcLow - (ctx.mpcLow - ctx.mpcHigh) * surplus / (surplus + 5 * ctx.subsistence)
    mpc = math.max(0, math.min(1, mpc + 2 * ctx.timePreferenceSpend * (self.timePreference - 0.5)))
    savings = savings - math.min(savings, math.floor(mpc * surplus))
  end
  local spare = math.max(0, savings - ctx.runwayComfort * ctx.subsistence)
  savings = savings - math.floor(ctx.wealthSpending * (0.5 + self.timePreference) * spare)

  local preference = self.timePreference
  if jobless or short > 0 then
    preference = preference - ctx.timePreferenceLearn
  elseif savings > ctx.comfortRunway * ctx.subsistence then
    preference = preference + ctx.timePreferenceDrift
  end

  local effects = {
    fx.set("self.savings", savings),
    fx.set("self.monthsShort", short),
    fx.set("self.timePreference", preference),
    fx.add("self.age", 1),
  }
  if self.employer ~= 0 then
    effects[#effects + 1] = fx.set("self.capacity", self.capacity + ctx.learningRate * math.max(0, self.talent - self.capacity))
    effects[#effects + 1] = fx.add("self.experience", 1)
    effects[#effects + 1] = fx.set("self.monthsUnemployed", 0)
    local feeling = 0
    if short == 0 then
      local peer = ctx.levelWage[self.level] or ctx.meanWage
      feeling = math.max(0, math.min(1, 0.5 + (self.wage / peer - 1)))
    end
    effects[#effects + 1] = fx.set("self.satisfaction", self.satisfaction + ctx.satisfactionAdapt * (feeling - self.satisfaction))
  elseif jobless then
    effects[#effects + 1] = fx.set("self.capacity", math.max(ctx.minCapacity, self.capacity * (1 - ctx.skillDecay)))
    effects[#effects + 1] = fx.add("self.monthsUnemployed", 1)
  end
  return effects
end
