-- Defect: dishonesty, plus greed when the opponent has been trusting (it is
-- worth exploiting), plus resentment when the opponent has been betraying.

function eligible(self, world)
  return #world:neighbors("opponent") > 0
end

function propensity(self, world)
  local seen = self:memlen("opponentMoves")
  local trust = 0.5
  if seen > 0 then
    trust = self:count("opponentMoves", "C") / seen
  end
  return (1 - self.honesty) + self.greed * trust + self.grudge * (1 - trust)
end

function on_success(self, world)
  return fx.set("ctx.moves." .. self.id, "D")
end
