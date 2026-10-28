-- Put one plot up for sale, asking what the owner thinks it is worth. The land_market
-- rule decides whether someone pays it.

local function landValue(self, ctx)
  local daily = ctx.marketPrice * (1 - ctx.productionCost) * ctx.breadsPerLand * ctx.sellRate
  return daily * ctx.landHorizon * (0.5 + self.ambition)
end

function on_success(self, world)
  local ctx = world.ctx
  local ask = math.max(landValue(self, ctx), ctx.minPrice)
  return fx.set("ctx.landAsks." .. self.id, ask)
end
