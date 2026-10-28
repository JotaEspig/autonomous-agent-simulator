-- End of the day for each person:
--   a full ration (3 breads, 4 during an epidemic) resets hunger to 0; otherwise hunger grows by
--   hungerPerDay and falls by 1 per bread eaten;
--   starvationDays in a row without any bread kill (plots go back to the island);
--   80% of what is left of today's income is spent elsewhere, 20% is saved;
--   producers' unsold bread expires and they reprice: the average of their last 5
--   sales, lowered by priceDrop if bread was left over;
--   a person with zero hunger may have a child.

function run(self, world)
  local ctx = world.ctx

  local starving = 0
  if self.ateToday == 0 then
    starving = self.daysWithoutFood + 1
  end
  if starving >= ctx.starvationDays then
    local effects = {
      fx.add("ctx.deaths", 1),
      fx.emit("died", { hunger = self.hunger, wealth = self.wealth, lands = self.lands }),
    }
    if self.lands > 0 then
      effects[#effects + 1] = fx.add("ctx.landsFree", self.lands)
    end
    effects[#effects + 1] = fx.remove(self.id)
    return effects
  end

  local hunger = 0
  if self.ateToday < ctx.ration then
    hunger = math.max(0, self.hunger + ctx.hungerPerDay - self.ateToday)
  end
  local effects = {
    fx.set("self.hunger", hunger),
    fx.set("self.daysWithoutFood", starving),
    fx.set("self.stock", 0),
  }

  local leftover = self.income - self.spent
  if leftover > 0 then
    effects[#effects + 1] = fx.add("self.wealth", -0.8 * leftover)
  end

  if self.lands > 0 then
    local price = self.askPrice
    if self:memlen("salePrices") > 0 then
      price = self:mean("salePrices")
    end
    if self.stock > 0 then
      price = math.min(price, self.askPrice) * (1 - ctx.priceDrop)
    end
    effects[#effects + 1] = fx.set("self.askPrice", math.max(price, ctx.minPrice))
  end

  if hunger == 0 and world:chance(ctx.birthChance) then
    effects[#effects + 1] = fx.spawn("islander", {
      riskTolerance = world:rand(),
      ambition = world:rand(),
      frugality = world:rand(),
    })
    effects[#effects + 1] = fx.add("ctx.births", 1)
  end
  return effects
end
