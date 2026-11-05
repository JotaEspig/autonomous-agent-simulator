-- Plan to lay off the staff who are worth less to the firm than their wage. The layoffs
-- rule picks who goes: the lowest value for the wage first.

function on_success(self, world)
  return fx.set("self.layoffsPlanned", self.underwater)
end
