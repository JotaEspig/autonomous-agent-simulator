-- The bank only sees aggregates: how many firms exist per person, versus its target.
local function gap(self, world)
  return self.target - world:count("firm") / world:count("person")
end

function eligible(self, world)
  return world.ctx.interestRate >= 0.0025
end

function propensity(self, world)
  return math.max(0, gap(self, world)) * self.aggressiveness
end

function on_success(self, world)
  return fx.add("ctx.interestRate", -0.0025)
end
