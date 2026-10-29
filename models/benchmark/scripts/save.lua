function on_success(self, world)
  return fx.add("self.wealth", math.floor(10 * self.skill) + 1)
end
