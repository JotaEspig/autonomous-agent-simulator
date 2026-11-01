-- Payday. Each firm produces what its staff can (output = sum of their capacities) and the
-- sector sells it all at the price the market bears: price = basePrice x (output /
-- baseDemand)^(-1/elasticity), so the more the sector produces, the lower the price. Each
-- firm then pays wages, overhead (fixedCost plus overheadShare of the wage bill: rent,
-- systems and management grow with the firm) and its owner's draw (pro-labore: what a
-- senior earns, the pay of the person running the firm, but only out of what revenue leaves
-- after wages and overhead; a founder whose firm does not cover its costs yet draws nothing
-- and lives on its own savings, until those fall below runwayComfort months of living costs:
-- then it takes at least the cost of living from the firm, since nobody starves with money in
-- the till), and pays out this month's profit as dividends to its owner, but only the part that leaves more than payoutAboveMonths of
-- costs in the bank: owners never take back the capital they put in. An owner's income is
-- its draw plus dividends, smoothed over about a year for the census.
--
-- A firm whose cash plus revenue cannot cover wages and overhead goes bankrupt: what money
-- it has is split among its staff in proportion to their wages, everyone loses the job, the
-- owner loses the capital and goes back to the labor market, and the firm closes.
--
-- Firms read demand from the price: the expected monthly price change (priceTrend) moves
-- priceAdapt of the way toward the latest change, so a falling price, a sign that the
-- sector produces more than buyers want at the old price, makes firms value hires lower.
--
-- Firms also learn whether hiring juniors pays: each junior hire promoted at the firm
-- lengthens its hiring horizon by horizonLearn, and each one who left within
-- investmentPayback months (poached, quit or laid off) shortens it.
--
-- Then the books each firm reads before deciding: headcount, output share, revenue, wage
-- bill, margin, and the staff clearly worth less than their wage: value (price x capacity
-- x (1 - share / elasticity), the revenue the person adds once the firm's own effect on the
-- price is counted) more than the firm's tolerance below the wage. Firms keep people through
-- small shortfalls (labor hoarding), each with its own patience; a firm losing money loses
-- patience, down to none when its losses reach crisisMargin of revenue. Salary surveys publish the average wage of each level.

function run(world)
  local ctx = world.ctx
  local staff, total = {}, 0
  local levelSum, levelN = {}, {}
  local wages, employed = 0, 0
  local paid, lost = {}, {}
  local effects, owners = {}, {}
  for _, id in ipairs(world:ids("worker")) do
    local w = world:agent(id)
    if w.owner then
      owners[#owners + 1] = { id = id, avg = w.incomeAvg }
    end
    if w.paidOff ~= 0 then
      paid[w.paidOff] = (paid[w.paidOff] or 0) + 1
      effects[#effects + 1] = fx.set("agent." .. id .. ".paidOff", 0)
    end
    if w.investmentLost ~= 0 then
      lost[w.investmentLost] = (lost[w.investmentLost] or 0) + 1
      effects[#effects + 1] = fx.set("agent." .. id .. ".investmentLost", 0)
    end
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

  local revenueAll, profitAll, hhi = 0, 0, 0
  local dividends = {}
  for _, fid in ipairs(world:ids("firm")) do
    local f = world:agent(fid)
    local s = staff[fid] or { cap = 0, bill = 0, people = {} }
    local share = 0
    if total > 0 then
      share = s.cap / total
    end
    local revenue = math.floor(price * s.cap)
    local owner = world:incoming("owns", fid)[1]
    local overhead = math.floor(ctx.fixedCost + ctx.overheadShare * s.bill)
    local draw = 0
    if owner then
      draw = math.min(ctx.levelWage.senior or meanWage, math.max(0, revenue - s.bill - overhead))
      if world:agent(owner).savings < ctx.runwayComfort * ctx.subsistence then
        draw = math.max(draw, ctx.subsistence)
      end
      draw = math.floor(draw)
    end
    local costs = s.bill + overhead + draw
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
      for _, owner in ipairs(world:incoming("owns", fid)) do
        effects[#effects + 1] = fx.set("agent." .. owner .. ".owner", false)
      end
      effects[#effects + 1] = fx.emit("bankrupt", { firm = fid, staff = #s.people, age = f.age, cash = f.cash, revenue = revenue, costs = costs, patience = f.patience, horizon = f.horizon, markdown = f.markdown })
      effects[#effects + 1] = fx.remove(fid)
    else
      for _, p in ipairs(s.people) do
        effects[#effects + 1] = fx.add("agent." .. p.id .. ".savings", p.wage)
      end
      local profit = revenue - costs
      local cash = available - costs
      local payout = math.max(0, math.min(profit, math.floor(cash - ctx.payoutAboveMonths * costs)))
      local margin = f.margin
      if revenue > 0 then
        margin = profit / revenue
      elseif #s.people == 0 and f.age > 0 then
        margin = -1
      end
      local tolerance = f.patience * math.max(0, math.min(1, 1 + margin / ctx.crisisMargin))
      local mr = price * (1 - share / ctx.elasticity)
      local underwater = 0
      for _, p in ipairs(s.people) do
        if mr * p.cap < p.wage * (1 - tolerance) then
          underwater = underwater + 1
        end
      end
      effects[#effects + 1] = fx.set(firm .. ".cash", cash - payout)
      if owner and payout + draw > 0 then
        effects[#effects + 1] = fx.add("agent." .. owner .. ".savings", payout + draw)
        dividends[owner] = (dividends[owner] or 0) + payout + draw
      end
      effects[#effects + 1] = fx.set(firm .. ".headcount", #s.people)
      effects[#effects + 1] = fx.set(firm .. ".share", share)
      effects[#effects + 1] = fx.set(firm .. ".capacitySum", s.cap)
      effects[#effects + 1] = fx.set(firm .. ".revenue", revenue)
      effects[#effects + 1] = fx.set(firm .. ".wageBill", s.bill)
      effects[#effects + 1] = fx.set(firm .. ".margin", margin)
      effects[#effects + 1] = fx.set(firm .. ".underwater", underwater)
      effects[#effects + 1] = fx.set(firm .. ".tolerance", tolerance)
      local up, down = paid[fid] or 0, lost[fid] or 0
      if up + down > 0 then
        local horizon = f.horizon * (1 + ctx.horizonLearn) ^ up * (1 - ctx.horizonLearn) ^ down
        effects[#effects + 1] = fx.set(firm .. ".horizon", math.max(ctx.minHorizon, math.min(ctx.maxHorizon, horizon)))
      end
      effects[#effects + 1] = fx.set(firm .. ".openings", 0)
      effects[#effects + 1] = fx.set(firm .. ".layoffsPlanned", 0)
      effects[#effects + 1] = fx.add(firm .. ".age", 1)
      revenueAll = revenueAll + revenue
      profitAll = profitAll + profit
      hhi = hhi + share * share
    end
  end

  for _, o in ipairs(owners) do
    local income = dividends[o.id] or 0
    effects[#effects + 1] = fx.set("agent." .. o.id .. ".income", income)
    effects[#effects + 1] = fx.set("agent." .. o.id .. ".incomeAvg", o.avg + (income - o.avg) / 12)
  end
  local change = 0
  if ctx.price > 0 then
    change = price / ctx.price - 1
  end
  effects[#effects + 1] = fx.set("ctx.priceTrend", ctx.priceTrend + ctx.priceAdapt * (change - ctx.priceTrend))
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
