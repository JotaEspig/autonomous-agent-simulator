function run(world)
  local d = world.ctx.demand
  return fx.set("ctx.demand", d + 0.05 * (1 - d) + world:normal(0, 0.03))
end
