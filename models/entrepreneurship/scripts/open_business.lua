-- Opening a business costs money whether or not it works out. The precondition
-- pays the cost; the engine rejects it if the person can no longer afford it.

function precondition(self, world)
  return fx.add("self.wealth", -world.ctx.businessCost)
end

-- Probability that the venture gets off the ground.
function outcome(self, world)
  local p = 0.2 + 0.6 * self.knowledge - world.ctx.interestRate
  return math.max(0, math.min(1, p))
end

function on_success(self, world)
  local firm = fx.spawn("firm", { capital = world.ctx.businessCost })
  return {
    firm,
    fx.link("owns", self.id, firm),
    fx.remember("ventures", true),
  }
end

function on_failure(self, world)
  return {
    fx.remember("ventures", false),
    fx.emit("venture_failed", { wealth = self.wealth }),
  }
end
