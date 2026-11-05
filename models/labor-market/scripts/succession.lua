-- When an owner dies, its estate sells the firm to the best bid it can get. There are no
-- heirs, so the price leaves the economy. The firm is worth its cash plus saleProfitMonths of
-- its recent profit. The buyer is the person, not already an owner, with the most courage x
-- savings among those who could start a firm (foundingShare of savings at least
-- minFoundingCapital); it pays the firm's worth, or what it can put in (foundingShare of its
-- savings) if that is less, since an estate that finds no buyer at its price lowers it. The
-- buyer leaves its job and becomes the owner. With nobody able to put up minFoundingCapital,
-- the firm closes: its staff lose their jobs and its cash is lost. Owners do not retire: they
-- keep their firm until they die.

function run(world)
  local ctx = world.ctx
  local orphans = {}
  for _, fid in ipairs(world:ids("firm")) do
    if #world:incoming("owns", fid) == 0 then
      orphans[#orphans + 1] = fid
    end
  end
  if #orphans == 0 then
    return nil
  end

  local candidates = {}
  for _, id in ipairs(world:ids("worker")) do
    local w = world:agent(id)
    if not w.owner and w.savings * ctx.foundingShare >= ctx.minFoundingCapital then
      candidates[#candidates + 1] = { id = id, savings = w.savings, score = w.courage * w.savings, employer = w.employer, wage = w.wage }
    end
  end
  table.sort(candidates, function(a, b)
    if a.score ~= b.score then return a.score > b.score end
    return a.id < b.id
  end)

  local effects, taken = {}, {}
  for _, fid in ipairs(orphans) do
    local f = world:agent(fid)
    local profit = math.max(0, f.margin * f.revenue)
    local worth = math.floor(math.max(0, f.cash) + ctx.saleProfitMonths * profit)
    local buyer
    for _, c in ipairs(candidates) do
      if not taken[c.id] then
        buyer = c
        break
      end
    end
    local price = worth
    if buyer then
      price = math.min(worth, math.floor(ctx.foundingShare * buyer.savings))
      taken[buyer.id] = true
      local who = "agent." .. buyer.id
      effects[#effects + 1] = fx.link("owns", buyer.id, fid)
      effects[#effects + 1] = fx.add(who .. ".savings", -price)
      effects[#effects + 1] = fx.set(who .. ".owner", true)
      effects[#effects + 1] = fx.set(who .. ".incomeAvg", 0)
      if buyer.employer ~= 0 then
        effects[#effects + 1] = fx.set(who .. ".employer", 0)
        effects[#effects + 1] = fx.set(who .. ".lastWage", buyer.wage)
        effects[#effects + 1] = fx.set(who .. ".wage", 0)
      end
      effects[#effects + 1] = fx.emit("firm_sold", { firm = fid, buyer = buyer.id, price = price, worth = worth })
    else
      local staff = 0
      for _, id in ipairs(world:ids("worker")) do
        local w = world:agent(id)
        if w.employer == fid then
          local who = "agent." .. id
          effects[#effects + 1] = fx.set(who .. ".employer", 0)
          effects[#effects + 1] = fx.set(who .. ".lastWage", w.wage)
          effects[#effects + 1] = fx.set(who .. ".expectedWage", w.wage)
          effects[#effects + 1] = fx.set(who .. ".wage", 0)
          staff = staff + 1
        end
      end
      effects[#effects + 1] = fx.add("ctx.layoffs", staff)
      effects[#effects + 1] = fx.add("ctx.closures", 1)
      effects[#effects + 1] = fx.emit("firm_closed", { firm = fid, staff = staff, worth = worth })
      effects[#effects + 1] = fx.remove(fid)
    end
  end
  return effects
end
