-- Cooperate: the propensity is an expression in pd.schema.json; this script only
-- says when the move is possible and what playing it does.

function eligible(self, world)
  return #world:neighbors("opponent") > 0
end

function on_success(self, world)
  return fx.set("ctx.moves." .. self.id, "C")
end
