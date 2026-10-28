-- Runs once per firm each tick (agent-scoped rule). Profit depends on demand and on
-- the cost of capital; half of any profit goes to the owner; insolvent firms close.

function run(self, world)
  local ctx = world.ctx
  local margin = 0.012 * ctx.demand + world:normal(0, 0.02) - ctx.interestRate / 12
  local profit = math.floor(self.capital * margin)
  local effects = { fx.add("self.age", 1) }

  if self.capital + profit <= 0 then
    effects[#effects + 1] = fx.emit("firm_closed", { age = self.age })
    effects[#effects + 1] = fx.remove(self.id)
    return effects
  end

  local owner = world:incoming("owns", self.id)[1]
  if profit > 0 and owner then
    local dividend = math.floor(profit / 2)
    effects[#effects + 1] = fx.add("self.capital", profit - dividend)
    effects[#effects + 1] = fx.add("agent." .. owner .. ".wealth", dividend)
  else
    effects[#effects + 1] = fx.add("self.capital", profit)
  end
  return effects
end
