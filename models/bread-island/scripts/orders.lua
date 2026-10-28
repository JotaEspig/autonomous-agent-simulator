-- Routine market behavior, not a choice: everyone tries to eat and producers try to
-- sell bread before it expires. Each agent writes only its own keys of the order maps.
--
-- Willingness to pay for the first bread is the island's base price plus a premium per
-- point of hunger; each further bread is worth marginalDiscount less. That is the most a
-- buyer would pay, not what it offers: buyers pay what they bid, so they offer no more
-- than the cheapest posted price, marked up (by outbid) as far as bread was scarce yesterday. With bread
-- to spare they offer the posted price; under scarcity they outbid it, and the hungriest
-- (highest willingness) can outbid the most. The buyer secures the first bread before
-- thinking about the next: each bid is also limited by the savings not yet committed.

function run(self, world)
  local ctx = world.ctx
  local effects = {}
  local id = self.id

  local willingness = ctx.basePrice + ctx.hungerPremium * self.hunger
  local cap = math.huge
  if ctx.lowestAsk > 0 then
    cap = ctx.lowestAsk * (1 + ctx.outbid * ctx.scarcity)
  end
  local budget = self.wealth
  for unit = 1, ctx.maxBreadsPerDay do
    local bid = math.min(willingness * (1 - ctx.marginalDiscount * (unit - 1)), cap, budget)
    if bid < ctx.minPrice then
      break
    end
    effects[#effects + 1] = fx.set("ctx.breadBids." .. id .. "." .. unit, bid)
    budget = budget - bid
  end

  if self.stock > 0 then
    effects[#effects + 1] = fx.set("ctx.breadAskQty." .. id, self.stock)
    effects[#effects + 1] = fx.set("ctx.breadAskPrice." .. id, self.askPrice)
  end
  return effects
end
