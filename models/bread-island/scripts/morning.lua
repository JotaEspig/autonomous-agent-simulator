-- Everyone is paid the same fixed wage; producers bake today's bread.

function run(self, world)
  local ctx = world.ctx
  return {
    fx.add("self.wealth", ctx.salary),
    fx.set("self.income", ctx.salary),
    fx.set("self.spent", 0),
    fx.set("self.ateToday", 0),
    fx.set("self.stock", self.lands * ctx.breadsPerLand),
  }
end
