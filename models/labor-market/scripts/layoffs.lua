-- Firms that decided to downsize lay off the people who are worth the least to them for
-- what they cost: lowest value (capacity x the firm's marginal revenue) over wage first,
-- one at a time. Each layoff shrinks the firm's share of the market and so raises what the
-- others are worth to it; the firm stops when the next person is no longer more than
-- layoffTolerance below the wage, or when it has laid off the planned number. The firm
-- knows the real capacity of its own staff.

function run(world)
  local ctx = world.ctx
  local planned, any = {}, false
  for _, fid in ipairs(world:ids("firm")) do
    local f = world:agent(fid)
    if f.layoffsPlanned > 0 then
      planned[fid] = { n = f.layoffsPlanned, cap = f.capacitySum, people = {} }
      any = true
    end
  end
  if not any then
    return nil
  end
  for _, id in ipairs(world:ids("worker")) do
    local w = world:agent(id)
    local p = planned[w.employer]
    if p then
      p.people[#p.people + 1] = { id = id, wage = w.wage, cap = w.capacity, ratio = w.capacity / math.max(w.wage, 1) }
    end
  end
  local effects = {}
  for fid, p in pairs(planned) do
    table.sort(p.people, function(a, b)
      if a.ratio ~= b.ratio then return a.ratio < b.ratio end
      return a.id < b.id
    end)
    local fired, cap, total = 0, p.cap, ctx.output
    for i = 1, math.min(p.n, #p.people) do
      local person = p.people[i]
      local value = ctx.price * (1 - cap / math.max(total, 1e-9) / ctx.elasticity) * person.cap
      if value >= person.wage * (1 - ctx.layoffTolerance) then
        break
      end
      cap = cap - person.cap
      total = total - person.cap
      local who = "agent." .. person.id
      effects[#effects + 1] = fx.set(who .. ".employer", 0)
      effects[#effects + 1] = fx.set(who .. ".wage", 0)
      effects[#effects + 1] = fx.set(who .. ".lastWage", person.wage)
      effects[#effects + 1] = fx.set(who .. ".expectedWage", person.wage)
      fired = fired + 1
    end
    effects[#effects + 1] = fx.add("agent." .. fid .. ".headcount", -fired)
    effects[#effects + 1] = fx.add("ctx.layoffs", fired)
  end
  return effects
end
