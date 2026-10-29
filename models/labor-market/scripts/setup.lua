-- Starting jobs. The run starts in a working economy rather than with everyone unemployed:
-- each person has a job with chance initialEmployment, at a firm drawn at random, paid what
-- that firm would offer after an interview (see labor_market.lua): the estimated value of
-- the person's output, minus the firm's markdown.

function run(world)
  local ctx = world.ctx
  local firms = world:ids("firm")
  if #firms == 0 then
    return nil
  end
  local staff, capSum, total = {}, {}, 0
  for _, id in ipairs(world:ids("worker")) do
    if world:chance(ctx.initialEmployment) then
      local firm = world:pick(firms)
      local cap = world:agent(id).capacity
      staff[#staff + 1] = { id = id, firm = firm, cap = cap }
      capSum[firm] = (capSum[firm] or 0) + cap
      total = total + cap
    end
  end
  if total == 0 then
    return nil
  end
  local price = ctx.basePrice * (total / ctx.baseDemand) ^ (-1 / ctx.elasticity)

  local effects, wages = {}, 0
  for _, s in ipairs(staff) do
    local f = world:agent(s.firm)
    local estimate = s.cap * math.max(0.1, 1 + world:normal(0, ctx.interviewNoise))
    local wage = math.floor(price * (1 - capSum[s.firm] / total / ctx.elasticity) * estimate * (1 - f.markdown))
    local who = "agent." .. s.id
    effects[#effects + 1] = fx.set(who .. ".employer", s.firm)
    effects[#effects + 1] = fx.set(who .. ".wage", wage)
    effects[#effects + 1] = fx.set(who .. ".lastWage", wage)
    effects[#effects + 1] = fx.set(who .. ".expectedWage", wage)
    wages = wages + wage
  end
  effects[#effects + 1] = fx.set("ctx.price", price)
  effects[#effects + 1] = fx.set("ctx.meanWage", wages / #staff)
  effects[#effects + 1] = fx.set("ctx.levelWage.junior", wages / #staff)
  return effects
end
