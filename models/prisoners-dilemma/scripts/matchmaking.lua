-- Pairs players at random every round. With an odd number of players, one sits out.

function run(world)
  local players = world:shuffle(world:ids("player"))
  local effects = {}
  for _, id in ipairs(players) do
    for _, other in ipairs(world:neighbors("opponent", id)) do
      if id < other then
        effects[#effects + 1] = fx.unlink("opponent", id, other)
      end
    end
  end
  for i = 1, #players - 1, 2 do
    effects[#effects + 1] = fx.link("opponent", players[i], players[i + 1])
  end
  return effects
end
