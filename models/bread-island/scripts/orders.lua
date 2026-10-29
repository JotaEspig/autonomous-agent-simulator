-- Everyone tries to eat a full ration every day; savings and investment come after.
-- Producers have already eaten from their own bread and only buy what is missing.
--
-- A person reads the posted offers from the cheapest up and plans the day's shopping:
-- as many breads as the ration needs, from offers priced within its own price limit,
-- as long as the savings cover the posted prices. It then places one buy order per
-- planned bread at its limit, the most it would pay. The limit is learned from
-- experience (see metabolism.lua): it rises after hungry days and eases after full ones.

function run(self, world)
  local ctx = world.ctx
  local offers = {}
  for _, key in ipairs(world:keys("breadAskPrice")) do
    offers[#offers + 1] = { price = ctx.breadAskPrice[key], qty = ctx.breadAskQty[key] }
  end
  table.sort(offers, function(a, b) return a.price < b.price end)

  local wanted, budget, planned = ctx.ration - self.ateToday, self.wealth, 0
  for _, offer in ipairs(offers) do
    if wanted == 0 or offer.price > self.bidLimit then
      break
    end
    local n = math.min(offer.qty, wanted, math.floor(budget / offer.price))
    if n == 0 then
      break
    end
    planned, wanted, budget = planned + n, wanted - n, budget - n * offer.price
  end

  local effects = {}
  for unit = 1, planned do
    effects[#effects + 1] = fx.set("ctx.breadBids." .. self.id .. "." .. unit, self.bidLimit)
  end
  return effects
end
