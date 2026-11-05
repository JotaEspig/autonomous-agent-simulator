-- Young people enter the labor market: entrantsPerMonth on average (the whole part every
-- month, the fraction as a chance). They start as juniors with a capacity drawn like the
-- initial population's, their own talent and time preference, and money to live on while
-- they look for a first job (help from parents, student grants): entrantSavingsMonths of
-- living costs on average, from half to one and a half times that, since families differ.

local function clamp(x, lo, hi)
  return math.max(lo, math.min(hi, x))
end

function run(world)
  local ctx = world.ctx
  local n = math.floor(ctx.entrantsPerMonth)
  if world:chance(ctx.entrantsPerMonth - n) then
    n = n + 1
  end
  local effects = {}
  for _ = 1, n do
    effects[#effects + 1] = fx.spawn("worker", {
      age = ctx.entryAge,
      capacity = clamp(world:normal(1, 0.2), 0.5, 1.5),
      talent = clamp(world:normal(2.5, 0.6), 1.2, 4),
      savings = math.floor(ctx.entrantSavingsMonths * ctx.subsistence * (0.5 + world:rand())),
      timePreference = clamp(world:normal(0.5, 0.2), 0, 1),
      courage = clamp(world:normal(0.5, 0.2), 0, 1),
      level = "junior",
    })
  end
  return effects
end
