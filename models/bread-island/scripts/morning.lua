-- Everyone is paid the same fixed wage; producers bake today's bread, scaled by the
-- weather (productionFactor).

function run(self, world)
  local ctx = world.ctx
  return {
    fx.add("self.wealth", ctx.salary),
    fx.set("self.income", ctx.salary),
    fx.set("self.spent", 0),
    fx.set("self.ateToday", 0),
    fx.set("self.stock", math.floor(self.lands * ctx.breadsPerLand * ctx.productionFactor)),
  }
end
