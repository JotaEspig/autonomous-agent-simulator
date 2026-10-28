-- Routine market behavior, not a choice: everyone tries to eat and producers try to
-- sell bread before it expires. Each agent writes only its own keys of the order maps.
--
-- Each person wants to save as much as possible once fed, and how much counts as "fed"
-- depends on frugality:
--   * Survival bread: after enough days without food, the first bread of the day is a
--     must. Frugal people wait longer (up to two days before starvation); the least
--     frugal eat every day.
--   * Comfort bread: any other bread is bought only up to a price limit per bread of
--     salary x (1 - frugality) / ration. Fully frugal people never buy it.
--
-- Willingness to pay is the island's base price plus a premium per point of hunger;
-- each further bread is worth marginalDiscount less. Each bid has two prices:
--   * the offer, what the buyer tries to pay: buyers pay what they bid, so they offer no
--     more than the cheapest posted price, marked up (by outbid) as far as bread was
--     scarce yesterday;
--   * the maximum, what the buyer accepts if the cheaper bread is gone.
-- Both are limited by the savings not yet committed, so the first bread is secured
-- before the next ones. Nobody wants more than today's ration.

function run(self, world)
  local ctx = world.ctx
  local effects = {}
  local id = self.id

  local willingness = ctx.basePrice + ctx.hungerPremium * self.hunger
  local cap = math.huge
  if ctx.lowestAsk > 0 then
    cap = ctx.lowestAsk * (1 + ctx.outbid * ctx.scarcity)
  end
  local waitDays = math.floor(self.frugality * (ctx.starvationDays - 2))
  local mustEat = self.daysWithoutFood >= waitDays
  local comfortLimit = ctx.salary * (1 - self.frugality) / ctx.ration

  local budget = self.wealth
  for unit = 1, ctx.ration do
    local max = math.min(willingness * (1 - ctx.marginalDiscount * (unit - 1)), budget)
    if not (unit == 1 and mustEat) then
      max = math.min(max, comfortLimit)
    end
    local offer = math.min(max, cap)
    if offer < ctx.minPrice then
      break
    end
    local key = id .. "." .. unit
    effects[#effects + 1] = fx.set("ctx.breadBids." .. key, offer)
    effects[#effects + 1] = fx.set("ctx.breadBidMax." .. key, max)
    budget = budget - max
  end

  if self.stock > 0 then
    effects[#effects + 1] = fx.set("ctx.breadAskQty." .. id, self.stock)
    effects[#effects + 1] = fx.set("ctx.breadAskPrice." .. id, self.askPrice)
  end
  return effects
end
