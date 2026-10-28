local function gap(self, world)
  return world:count("firm") / world:count("person") - self.target
end

function propensity(self, world)
  return math.max(0, gap(self, world)) * self.aggressiveness
end

function on_success(self, world)
  return fx.add("ctx.interestRate", 0.0025)
end
