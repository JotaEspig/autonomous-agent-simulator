-- Measures the population after this month's hires and layoffs: unemployment (among those
-- who are not owners), the income classes in minimum wages (income is the wage, or for
-- owners their dividends smoothed over a year; poverty below 1, lower 1 to 3, middle 3 to
-- 10, upper above 10; the unemployed earn 0 and count as poor), the share of owners and
-- their mean income, mean capacity of the employed and of the unemployed, the mean job
-- satisfaction of the employed, the mean time preference, the firms' mean hiring horizon,
-- the share of seniors, and the share in the squeeze (unable to pay the full cost of living
-- this month). The minimum wage is only this yardstick.

function run(world)
  local ctx = world.ctx
  local mw = ctx.minimumWage
  local n, unemployed, poor, lower, middle, upper, seniors, squeezed = 0, 0, 0, 0, 0, 0, 0, 0
  local capE, capU, sat, tp = 0, 0, 0, 0
  local owners, ownerIncome = 0, 0
  for _, id in ipairs(world:ids("worker")) do
    local w = world:agent(id)
    n = n + 1
    tp = tp + w.timePreference
    local income = w.wage
    if w.owner then
      owners = owners + 1
      income = w.incomeAvg
      ownerIncome = ownerIncome + income
    elseif w.employer == 0 then
      unemployed = unemployed + 1
      capU = capU + w.capacity
    else
      capE = capE + w.capacity
      sat = sat + w.satisfaction
    end
    local m = income / mw
    if m < 1 then
      poor = poor + 1
    elseif m < 3 then
      lower = lower + 1
    elseif m < 10 then
      middle = middle + 1
    else
      upper = upper + 1
    end
    if w.monthsShort > 0 then
      squeezed = squeezed + 1
    end
    if w.level == "senior" then
      seniors = seniors + 1
    end
  end
  local function share(x)
    if n == 0 then return 0 end
    return x / n
  end
  local employed = n - unemployed - owners
  local horizon, firms = 0, 0
  for _, fid in ipairs(world:ids("firm")) do
    horizon = horizon + world:agent(fid).horizon
    firms = firms + 1
  end
  return {
    fx.set("ctx.horizon", firms > 0 and horizon / firms or 0),
    fx.set("ctx.unemployment", n - owners > 0 and unemployed / (n - owners) or 0),
    fx.set("ctx.owners", share(owners)),
    fx.set("ctx.ownerIncome", owners > 0 and ownerIncome / owners or 0),
    fx.set("ctx.poverty", share(poor)),
    fx.set("ctx.lowerClass", share(lower)),
    fx.set("ctx.middleClass", share(middle)),
    fx.set("ctx.upperClass", share(upper)),
    fx.set("ctx.seniors", share(seniors)),
    fx.set("ctx.squeezed", share(squeezed)),
    fx.set("ctx.satisfaction", employed > 0 and sat / employed or 0),
    fx.set("ctx.timePreference", n > 0 and tp / n or 0),
    fx.set("ctx.capacityEmployed", employed > 0 and capE / employed or 0),
    fx.set("ctx.capacityUnemployed", unemployed > 0 and capU / unemployed or 0),
  }
end
