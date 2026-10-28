-- The island is full: offer to buy a plot from another person. The bid is what the
-- plot seems worth (expected daily profit over a horizon, scaled by ambition),
-- limited by savings. The land_market rule matches bids with offers.

local function landValue(self, ctx)
  local daily = ctx.marketPrice * (1 - ctx.productionCost) * ctx.breadsPerLand * ctx.sellRate
  return daily * ctx.landHorizon * (0.5 + self.ambition)
end

function on_success(self, world)
  local bid = math.min(self.wealth, landValue(self, world.ctx))
  if bid <= 0 then
    return nil
  end
  return fx.set("ctx.landBids." .. self.id, bid)
end
