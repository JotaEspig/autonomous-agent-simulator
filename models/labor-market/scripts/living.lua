-- A month in a person's life, after payday:
--   pays the cost of living (subsistence) from savings; a month without enough money counts
--   toward monthsShortToDie, and the person dies when it gets there;
--   spends part of what the wage leaves above subsistence: people with low pay spend almost
--   all of it (mpcLow), people with high pay save a larger share (falling toward mpcHigh);
--   and spends wealthSpending of its savings, as people consume out of wealth (about 5% a
--   year), so savings do not pile up forever;
--   works and learns: capacity closes learningRate of the distance to talent each month
--   (a learning curve: fast at first, slower near one's limit); unemployed, it decays by
--   skillDecay a month (skills rust);
--   ages; retires at retirementAge; may die of natural causes (Gompertz law).

function run(self, world)
  local ctx = world.ctx
  local savings = self.savings
  local short = 0
  if savings >= ctx.subsistence then
    savings = savings - ctx.subsistence
  else
    savings = 0
    short = self.monthsShort + 1
  end

  if short >= ctx.monthsShortToDie then
    return {
      fx.add("ctx.deaths", 1),
      fx.emit("died", { cause = "poverty", age = math.floor(self.age / 12), monthsUnemployed = self.monthsUnemployed, capacity = self.capacity, lastWage = self.lastWage }),
      fx.remove(),
    }
  end
  local years = self.age / 12
  if world:chance(ctx.mortalityBase * math.exp(ctx.mortalityGrowth * years) / 12) then
    return { fx.add("ctx.naturalDeaths", 1), fx.remove() }
  end
  if self.age >= ctx.retirementAge then
    return { fx.add("ctx.retirements", 1), fx.remove() }
  end

  local surplus = math.max(0, self.wage - ctx.subsistence)
  if surplus > 0 then
    local mpc = ctx.mpcLow - (ctx.mpcLow - ctx.mpcHigh) * surplus / (surplus + 5 * ctx.subsistence)
    savings = savings - math.min(savings, math.floor(mpc * surplus))
  end
  savings = savings - math.floor(ctx.wealthSpending * savings)

  local effects = {
    fx.set("self.savings", savings),
    fx.set("self.monthsShort", short),
    fx.add("self.age", 1),
  }
  if self.employer ~= 0 then
    effects[#effects + 1] = fx.set("self.capacity", self.capacity + ctx.learningRate * math.max(0, self.talent - self.capacity))
    effects[#effects + 1] = fx.add("self.experience", 1)
    effects[#effects + 1] = fx.set("self.monthsUnemployed", 0)
  else
    effects[#effects + 1] = fx.set("self.capacity", math.max(ctx.minCapacity, self.capacity * (1 - ctx.skillDecay)))
    effects[#effects + 1] = fx.add("self.monthsUnemployed", 1)
  end
  return effects
end
