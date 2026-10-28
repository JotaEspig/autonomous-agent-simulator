-- Everyone starved.
function stop(world)
  return world:count("islander") == 0
end
