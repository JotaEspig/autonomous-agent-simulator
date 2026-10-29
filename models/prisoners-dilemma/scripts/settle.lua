-- Settles the round: pays each player according to the payoff matrix, tells each
-- one what its opponent played, and clears the moves for the next round.

function run(world)
  local ctx = world.ctx
  local payoff = {
    C = { C = ctx.reward, D = ctx.sucker },
    D = { C = ctx.temptation, D = ctx.punishment },
  }
  local effects, played, cooperated = {}, 0, 0
  for _, id in ipairs(world:ids("player")) do
    local mine = ctx.moves[id]
    local opponent = world:neighbors("opponent", id)[1]
    local theirs = opponent and ctx.moves[opponent]
    if mine and theirs then
      local me = "agent." .. id
      effects[#effects + 1] = fx.add(me .. ".score", payoff[mine][theirs])
      effects[#effects + 1] = fx.remember("opponentMoves", theirs, id)
      if mine == "C" and theirs == "D" then
        effects[#effects + 1] = fx.add(me .. ".betrayed", 1)
      end
      played = played + 1
      if mine == "C" then
        cooperated = cooperated + 1
      end
    end
  end
  if played > 0 then
    effects[#effects + 1] = fx.set("ctx.cooperationRate", cooperated / played)
  end
  effects[#effects + 1] = fx.clear("ctx.moves")
  return effects
end
