-- Open jobs for this month: as many as the firm can recruit for, a share of its headcount
-- (hireRate) or at least minHires. Whether each opening is filled, and at what wage, is
-- settled in the labor market.

function on_success(self, world)
  local ctx = world.ctx
  return fx.set("self.openings", math.max(ctx.minHires, math.ceil(ctx.hireRate * self.headcount)))
end
