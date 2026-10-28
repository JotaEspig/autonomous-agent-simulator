-- Stop early if the economy collapses: nobody owns a firm and nobody can afford one.
function stop(world)
  return world.tick > 24 and world:count("firm") == 0 and world:max("person", "wealth") < world.ctx.businessCost
end
