-- Payday. Each firm produces what its staff can (output = sum of their capacities) and the
-- sector sells it all at the price the market bears: price = basePrice x (output /
-- baseDemand)^(-1/elasticity), so the more the sector produces, the lower the price. Each
-- firm then pays wages and overhead (fixedCost) and pays out this month's profit as
-- dividends, but only the part that leaves more than payoutAboveMonths of costs in the bank;
-- owners are outside the model and never take back the capital they put in.
--
-- A firm whose cash plus revenue cannot cover wages and overhead goes bankrupt: what money
-- it has is split among its staff in proportion to their wages, everyone loses the job, and
-- the firm closes.
--
-- Then the books each firm reads before deciding: headcount, output share, revenue, wage
-- bill, margin, and the staff clearly worth less than their wage: value (price x capacity
-- x (1 - share / elasticity), the revenue the person adds once the firm's own effect on the
-- price is counted) more than layoffTolerance below the wage. Firms keep people through
-- small shortfalls (labor hoarding). Salary surveys publish the average wage of each level.

function run(world)
  local ctx = world.ctx
  local staff, total = {}, 0
  local levelSum, levelN = {}, {}
  local wages, employed = 0, 0
  for _, id in ipairs(world:ids("worker")) do
    local w = world:agent(id)
    if w.employer ~= 0 then
      local s = staff[w.employer]
      if not s then
        s = { cap = 0, bill = 0, people = {} }
        staff[w.employer] = s
      end
      s.cap = s.cap + w.capacity
      s.bill = s.bill + w.wage
      s.people[#s.people + 1] = { id = id, cap = w.capacity, wage = w.wage }
      total = total + w.capacity
      wages = wages + w.wage
      employed = employed + 1
      levelSum[w.level] = (levelSum[w.level] or 0) + w.wage
      levelN[w.level] = (levelN[w.level] or 0) + 1
    end
  end

  local price = ctx.price
  if total > 0 then
    price = ctx.basePrice * (total / ctx.baseDemand) ^ (-1 / ctx.elasticity)
  end
  local meanWage = ctx.meanWage
  if employed > 0 then
    meanWage = wages / employed
  end

  local effects = {}
  local revenueAll, profitAll, hhi = 0, 0, 0
  for _, fid in ipairs(world:ids("firm")) do
    local f = world:agent(fid)
    local s = staff[fid] or { cap = 0, bill = 0, people = {} }
    local share = 0
    if total > 0 then
      share = s.cap / total
    end
    local revenue = math.floor(price * s.cap)
    local costs = s.bill + ctx.fixedCost
    local available = f.cash + revenue
    local firm = "agent." .. fid

    if available < costs then
      local pot = math.max(0, available)
      for _, p in ipairs(s.people) do
        local who = "agent." .. p.id
        if s.bill > 0 then
          effects[#effects + 1] = fx.add(who .. ".savings", math.floor(p.wage * pot / s.bill))
        end
        effects[#effects + 1] = fx.set(who .. ".employer", 0)
        effects[#effects + 1] = fx.set(who .. ".wage", 0)
        effects[#effects + 1] = fx.set(who .. ".lastWage", p.wage)
        effects[#effects + 1] = fx.set(who .. ".expectedWage", p.wage)
      end
      effects[#effects + 1] = fx.add("ctx.layoffs", #s.people)
      effects[#effects + 1] = fx.add("ctx.bankruptcies", 1)
      effects[#effects + 1] = fx.emit("bankrupt", { firm = fid, staff = #s.people, age = f.age, cash = f.cash, revenue = revenue, costs = costs })
      effects[#effects + 1] = fx.remove(fid)
    else
      for _, p in ipairs(s.people) do
        effects[#effects + 1] = fx.add("agent." .. p.id .. ".savings", p.wage)
      end
      local profit = revenue - costs
      local cash = available - costs
      local payout = math.max(0, math.min(profit, math.floor(cash - ctx.payoutAboveMonths * costs)))
      local mr = price * (1 - share / ctx.elasticity)
      local underwater = 0
      for _, p in ipairs(s.people) do
        if mr * p.cap < p.wage * (1 - ctx.layoffTolerance) then
          underwater = underwater + 1
        end
      end
      local margin = f.margin
      if revenue > 0 then
        margin = profit / revenue
      elseif #s.people == 0 and f.age > 0 then
        margin = -1
      end
      effects[#effects + 1] = fx.set(firm .. ".cash", cash - payout)
      effects[#effects + 1] = fx.set(firm .. ".headcount", #s.people)
      effects[#effects + 1] = fx.set(firm .. ".share", share)
      effects[#effects + 1] = fx.set(firm .. ".capacitySum", s.cap)
      effects[#effects + 1] = fx.set(firm .. ".revenue", revenue)
      effects[#effects + 1] = fx.set(firm .. ".wageBill", s.bill)
      effects[#effects + 1] = fx.set(firm .. ".margin", margin)
      effects[#effects + 1] = fx.set(firm .. ".underwater", underwater)
      effects[#effects + 1] = fx.set(firm .. ".openings", 0)
      effects[#effects + 1] = fx.set(firm .. ".layoffsPlanned", 0)
      effects[#effects + 1] = fx.add(firm .. ".age", 1)
      revenueAll = revenueAll + revenue
      profitAll = profitAll + profit
      hhi = hhi + share * share
    end
  end

  effects[#effects + 1] = fx.set("ctx.price", price)
  effects[#effects + 1] = fx.set("ctx.output", total)
  effects[#effects + 1] = fx.set("ctx.meanWage", meanWage)
  effects[#effects + 1] = fx.set("ctx.hhi", hhi)
  if revenueAll > 0 then
    effects[#effects + 1] = fx.set("ctx.profitMargin", profitAll / revenueAll)
  end
  for level, sum in pairs(levelSum) do
    effects[#effects + 1] = fx.set("ctx.levelWage." .. level, sum / levelN[level])
  end
  return effects
end
