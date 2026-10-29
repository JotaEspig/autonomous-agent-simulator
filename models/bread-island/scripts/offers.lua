-- A producer first eats from its own bread (up to the day's ration), then posts its price
-- and the rest of the stock before anyone buys, like price tags in a shop. The price was
-- set at the end of yesterday from how much bread was sold.

function run(self, world)
  if self.stock == 0 then
    return nil
  end
  local own = math.min(world.ctx.ration, self.stock)
  local rest = self.stock - own
  local effects = {
    fx.add("self.ateToday", own),
    fx.set("self.stock", rest),
    fx.set("self.offered", rest),
  }
  if rest > 0 then
    effects[#effects + 1] = fx.set("ctx.breadAskQty." .. self.id, rest)
    effects[#effects + 1] = fx.set("ctx.breadAskPrice." .. self.id, self.askPrice)
  end
  return effects
end
