-- Starting a firm. People with courage (willingness to risk their savings) and a low time
-- preference (willing to give up consumption now for profit later) found firms, if they
-- have saved enough: wealth is the main barrier to entrepreneurship (Evans and Jovanovic
-- 1989). The urge grows with the sector's profit margin (opportunity), and with being out
-- of work or unhappy at work (necessity and push). See the propensity in the schema.
--
-- The founder puts foundingShare of its savings into the firm and keeps the rest to live on,
-- leaves its job, and from then on lives on the firm's draw and dividends. The founder knows
-- what the sector pays (salary surveys), so the firm starts offering competitive wages: it
-- aims to keep the sector's profit margin (at least 5%) of each hire's value. The firm draws
-- its patience like the others; its hiring horizon starts from the founder's time preference: a patient
-- founder invests further ahead (from 12 months at time preference 1 to 120 at 0).

function on_success(self, world)
  local ctx = world.ctx
  local capital = math.floor(ctx.foundingShare * self.savings)
  local firm = fx.spawn("firm", {
    cash = capital,
    markdown = math.max(0.05, ctx.profitMargin),
    patience = 0.02 + 0.28 * world:rand(),
    horizon = 12 + 108 * (1 - self.timePreference),
    margin = ctx.profitMargin,
  })
  return {
    firm,
    fx.link("owns", self.id, firm),
    fx.add("self.savings", -capital),
    fx.set("self.owner", true),
    fx.set("self.employer", 0),
    fx.set("self.lastWage", self.wage),
    fx.set("self.wage", 0),
    fx.set("self.incomeAvg", 0),
    fx.add("ctx.newFirms", 1),
    fx.emit("firm_founded", { capital = capital, courage = self.courage, timePreference = self.timePreference, wasEmployed = self.employer ~= 0, age = math.floor(self.age / 12) }),
  }
end
