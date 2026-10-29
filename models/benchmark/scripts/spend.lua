function propensity(self, world)
  return (1 - self.thrift) * world.ctx.price
end

function on_success(self, world)
  local amount = math.floor(self.wealth * 0.1)
  return { fx.add("self.wealth", -amount), fx.remember("spent", amount) }
end
