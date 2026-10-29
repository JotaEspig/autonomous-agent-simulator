-- Nobody is left in the labor market.
function stop(world)
  return world:count("worker") == 0
end
