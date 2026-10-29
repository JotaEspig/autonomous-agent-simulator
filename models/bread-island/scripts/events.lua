-- Random events. The weather is either normal, a drought (less harvest) or a golden age
-- (more harvest); a new weather event only starts when the weather is normal. An
-- epidemic, independent of the weather, raises the bread needed to reset hunger.

function run(world)
  local ctx = world.ctx
  local effects = {}
  local drought, golden, epidemic = ctx.droughtDaysLeft, ctx.goldenDaysLeft, ctx.epidemicDaysLeft

  if drought > 0 then
    drought = drought - 1
  elseif golden > 0 then
    golden = golden - 1
  elseif world:chance(ctx.droughtChance) then
    drought = ctx.droughtDuration
    effects[#effects + 1] = fx.emit("drought_started", { days = drought })
  elseif world:chance(ctx.goldenChance) then
    golden = ctx.goldenDuration
    effects[#effects + 1] = fx.emit("golden_age_started", { days = golden })
  end

  if epidemic > 0 then
    epidemic = epidemic - 1
  elseif world:chance(ctx.epidemicChance) then
    epidemic = ctx.epidemicDuration
    effects[#effects + 1] = fx.emit("epidemic_started", { days = epidemic })
  end

  local factor = 1
  if drought > 0 then
    factor = 1 - ctx.droughtImpact
  elseif golden > 0 then
    factor = 1 + ctx.goldenImpact
  end
  local ration = ctx.maxBreadsPerDay
  if epidemic > 0 then
    ration = ctx.epidemicRation
  end

  effects[#effects + 1] = fx.set("ctx.droughtDaysLeft", drought)
  effects[#effects + 1] = fx.set("ctx.goldenDaysLeft", golden)
  effects[#effects + 1] = fx.set("ctx.epidemicDaysLeft", epidemic)
  effects[#effects + 1] = fx.set("ctx.productionFactor", factor)
  effects[#effects + 1] = fx.set("ctx.ration", ration)
  return effects
end
