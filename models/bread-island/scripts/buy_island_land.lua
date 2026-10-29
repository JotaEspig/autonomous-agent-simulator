-- Buying a free plot from the island at today's price (freeLandPrice). The precondition
-- pays for it and takes it from the island's stock; if other people bought the last
-- plots first, the engine rejects it (landsFree cannot go below 0) and nothing is paid.

function precondition(self, world)
  local price = world.ctx.freeLandPrice
  return {
    fx.add("ctx.landsFree", -1),
    fx.add("ctx.islandSalesToday", 1),
    fx.add("self.wealth", -price),
    fx.add("self.spent", price),
  }
end

function on_success(self, world)
  local effects = { fx.add("self.lands", 1) }
  if self.lands == 0 then
    effects[#effects + 1] = fx.set("self.askPrice", world.ctx.marketPrice)
    effects[#effects + 1] = fx.emit("became_producer", { wealth = self.wealth })
  end
  return effects
end
