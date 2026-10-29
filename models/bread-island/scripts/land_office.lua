-- The island sells free plots (never bought, or abandoned by owners who died without
-- children). While free plots find no buyer, their price falls by landPriceDrop a day,
-- down to minLandPrice, so it stops at the first price someone can afford. Once all
-- plots are taken, the price resets for the next plot that comes back.

function run(world)
  local ctx = world.ctx
  if ctx.landsFree == 0 then
    return fx.set("ctx.freeLandPrice", ctx.islandLandPrice)
  end
  if ctx.islandSalesToday == 0 then
    return fx.set("ctx.freeLandPrice", math.max(ctx.minLandPrice, ctx.freeLandPrice * (1 - ctx.landPriceDrop)))
  end
  return nil
end
