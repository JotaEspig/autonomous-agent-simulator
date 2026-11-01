-- Starting owners and jobs. Every firm has an owner: the richest people own the first firms,
-- one each, and live on their dividends; the firm's hiring horizon starts from its owner's
-- time preference (from 12 months at 1 to 120 at 0).
--
-- The run starts in a working economy rather than with everyone unemployed: every other
-- person has a job with chance initialEmployment, at a firm drawn at random, paid what
-- that firm would offer after an interview (see labor_market.lua): the estimated value of
-- the person's output, read with interviewNoise and pulled toward the average of the firm's
-- candidates as recruiters do, minus the firm's markdown.

-- Recruiters distrust extreme reads (shrinkInterviews). They know from experience how much
-- interviews err (interviewNoise), so they can tell how much of the spread among this
-- month's candidates is real: reliability = (variance of the reads - variance of the error)
-- / variance of the reads. Each read is pulled toward the candidates' average in that
-- proportion: estimate = average + reliability x (read - average). This is the best guess of
-- a candidate's capacity given a noisy read (Bayesian shrinkage, as in Kelley's formula).
local function shrink(rated, ctx)
  local n = #rated
  if not ctx.shrinkInterviews or n < 2 then
    return
  end
  local sum, sumSq = 0, 0
  for _, c in ipairs(rated) do
    sum = sum + c.estimate
    sumSq = sumSq + c.estimate * c.estimate
  end
  local mean = sum / n
  local variance = sumSq / n - mean * mean
  if variance <= 0 then
    return
  end
  local s2 = ctx.interviewNoise * ctx.interviewNoise
  local noise = s2 * (sumSq / n) / (1 + s2)
  local reliability = math.max(0, math.min(1, (variance - noise) / variance))
  for _, c in ipairs(rated) do
    c.estimate = mean + reliability * (c.estimate - mean)
  end
end

function run(world)
  local ctx = world.ctx
  local firms = world:ids("firm")
  if #firms == 0 then
    return nil
  end
  local effects = {}
  local people = {}
  for _, id in ipairs(world:ids("worker")) do
    people[#people + 1] = { id = id, savings = world:agent(id).savings }
  end
  table.sort(people, function(a, b)
    if a.savings ~= b.savings then return a.savings > b.savings end
    return a.id < b.id
  end)
  local owners = {}
  for i, fid in ipairs(firms) do
    local p = people[i]
    if p then
      owners[p.id] = true
      effects[#effects + 1] = fx.link("owns", p.id, fid)
      effects[#effects + 1] = fx.set("agent." .. p.id .. ".owner", true)
      effects[#effects + 1] = fx.set("agent." .. fid .. ".horizon", 12 + 108 * (1 - world:agent(p.id).timePreference))
    end
  end

  local staff, capSum, total = {}, {}, 0
  for _, id in ipairs(world:ids("worker")) do
    if not owners[id] and world:chance(ctx.initialEmployment) then
      local firm = world:pick(firms)
      local cap = world:agent(id).capacity
      staff[#staff + 1] = { id = id, firm = firm, cap = cap }
      capSum[firm] = (capSum[firm] or 0) + cap
      total = total + cap
    end
  end
  if total == 0 then
    return effects
  end
  local price = ctx.basePrice * (total / ctx.baseDemand) ^ (-1 / ctx.elasticity)

  local reads = {}
  for _, s in ipairs(staff) do
    s.estimate = s.cap * math.max(0.1, 1 + world:normal(0, ctx.interviewNoise))
    reads[s.firm] = reads[s.firm] or {}
    table.insert(reads[s.firm], s)
  end
  for _, fid in ipairs(firms) do
    if reads[fid] then
      shrink(reads[fid], ctx)
    end
  end

  local wages = 0
  for _, s in ipairs(staff) do
    local f = world:agent(s.firm)
    local estimate = s.estimate
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
