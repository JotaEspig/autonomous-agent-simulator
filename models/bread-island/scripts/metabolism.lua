-- End of the day for each person:
--   hunger depends on how many breads of the ration (3, or 4 during an epidemic) were
--   missed: none resets it to 0, one leaves it unchanged, two add 1, three or more add
--   hungerPerDay;
--   starvationDays in a row without any bread, or hunger above maxHunger, kill;
--   old age kills at random, with a risk that grows exponentially with age (Gompertz
--   law: yearly risk a x e^(b x years), the pattern of human adult mortality). The
--   funerals rule removes the dead and passes their plots on;
--   80% of what is left of today's income is spent elsewhere, 20% is saved;
--   the price limit rises after a day without a full ration, in proportion to the share
--   of the ration missed, and eases a little after a full one; it never exceeds what the
--   person has (savings, or at least a day's wage);
--   producers reprice: sold everything, raise the price; bread left over, cut it in
--   proportion to the unsold share, and to no more than undercut below the market
--   average. Unsold bread expires;
--   a person with zero hunger may have a child.

function run(self, world)
  local ctx = world.ctx

  local starving = 0
  if self.ateToday == 0 then
    starving = self.daysWithoutFood + 1
  end
  local missing = math.max(0, ctx.ration - self.ateToday)
  local hunger = self.hunger
  if missing == 0 then
    hunger = 0
  elseif missing == 2 then
    hunger = hunger + 1
  elseif missing >= 3 then
    hunger = hunger + ctx.hungerPerDay
  end

  local years = self.age / ctx.daysPerYear
  local dailyRisk = ctx.mortalityBase * math.exp(ctx.mortalityGrowth * years) / ctx.daysPerYear
  local cause
  if starving >= ctx.starvationDays then
    cause = "starvation"
  elseif hunger > ctx.maxHunger then
    cause = "malnutrition"
  elseif world:chance(dailyRisk) then
    cause = "old_age"
  end

  if cause then
    return {
      fx.set("self.dead", true),
      fx.add("ctx.deaths", 1),
      fx.emit("died", { cause = cause, age = math.floor(years), hunger = hunger, wealth = self.wealth, lands = self.lands }),
    }
  end

  local effects = {
    fx.add("self.age", 1),
    fx.set("self.hunger", hunger),
    fx.set("self.daysWithoutFood", starving),
    fx.set("self.stock", 0),
  }

  local limit
  if missing > 0 then
    limit = self.bidLimit * (1 + ctx.bidRaise * missing / ctx.ration)
    effects[#effects + 1] = fx.add("ctx.notFullyFed", 1)
  else
    limit = self.bidLimit * (1 - ctx.bidRelax)
  end
  -- Nobody is willing to pay more for a bread than it has.
  limit = math.min(limit, math.max(self.wealth, ctx.salary))
  effects[#effects + 1] = fx.set("self.bidLimit", math.max(limit, ctx.minPrice))

  local leftover = self.income - self.spent
  if leftover > 0 then
    effects[#effects + 1] = fx.add("self.wealth", -0.8 * leftover)
  end

  if self.offered > 0 then
    local price
    if self.stock == 0 then
      price = self.askPrice * (1 + ctx.priceRise)
    else
      price = math.min(
        self.askPrice * (1 - ctx.priceCut * self.stock / self.offered),
        ctx.marketPrice * (1 - ctx.undercut)
      )
    end
    effects[#effects + 1] = fx.set("self.askPrice", math.max(price, ctx.minPrice))
  end

  if hunger == 0 and world:chance(ctx.birthChance) then
    local child = fx.spawn("islander", {
      riskTolerance = world:rand(),
      ambition = world:rand(),
      bidLimit = ctx.basePrice,
    })
    effects[#effects + 1] = child
    effects[#effects + 1] = fx.link("parent", self.id, child)
    effects[#effects + 1] = fx.add("ctx.births", 1)
  end
  return effects
end
