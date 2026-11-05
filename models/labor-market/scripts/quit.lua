-- Quitting without another job. Someone who has felt undervalued for long (satisfaction
-- below quitSatisfaction) and has saved enough to look for work for quitRunway months may
-- walk out: the lower the satisfaction, the likelier. Dissatisfaction predicts quits
-- (Freeman 1978; Clark 2001), and people quit into unemployment when they can afford it.
-- The person expects to earn at least what its peers earn, since it left because it felt
-- underpaid. The firm learns of the departure on payday, and if the person had joined as a
-- junior less than investmentPayback months ago, that its investment was lost.

function on_success(self, world)
  local ctx = world.ctx
  local peer = ctx.levelWage[self.level] or ctx.meanWage
  local effects = {
    fx.set("self.employer", 0),
    fx.set("self.lastWage", self.wage),
    fx.set("self.expectedWage", math.max(self.wage, peer)),
    fx.set("self.wage", 0),
    fx.set("self.monthsUnemployed", 0),
    fx.add("ctx.quits", 1),
    fx.emit("quit", { wage = self.wage, peer = peer, satisfaction = self.satisfaction, savings = self.savings }),
  }
  if self.hiredJunior and world.tick - self.hiredAt < ctx.investmentPayback then
    effects[#effects + 1] = fx.set("self.investmentLost", self.employer)
  end
  return effects
end
