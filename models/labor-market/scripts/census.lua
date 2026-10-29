-- Measures the labor force after this month's hires and layoffs: unemployment, the income
-- classes in minimum wages (poverty below 1, lower 1 to 3, middle 3 to 10, upper above 10;
-- the unemployed earn 0 and count as poor), mean capacity of the employed and of the
-- unemployed, and the share of seniors. The minimum wage is only this yardstick.

function run(world)
  local ctx = world.ctx
  local mw = ctx.minimumWage
  local n, unemployed, poor, lower, middle, upper, seniors = 0, 0, 0, 0, 0, 0, 0
  local capE, capU = 0, 0
  for _, id in ipairs(world:ids("worker")) do
    local w = world:agent(id)
    n = n + 1
    if w.employer == 0 then
      unemployed = unemployed + 1
      capU = capU + w.capacity
    else
      capE = capE + w.capacity
    end
    local m = w.wage / mw
    if m < 1 then
      poor = poor + 1
    elseif m < 3 then
      lower = lower + 1
    elseif m < 10 then
      middle = middle + 1
    else
      upper = upper + 1
    end
    if w.level == "senior" then
      seniors = seniors + 1
    end
  end
  local function share(x)
    if n == 0 then return 0 end
    return x / n
  end
  local employed = n - unemployed
  return {
    fx.set("ctx.unemployment", share(unemployed)),
    fx.set("ctx.poverty", share(poor)),
    fx.set("ctx.lowerClass", share(lower)),
    fx.set("ctx.middleClass", share(middle)),
    fx.set("ctx.upperClass", share(upper)),
    fx.set("ctx.seniors", share(seniors)),
    fx.set("ctx.capacityEmployed", employed > 0 and capE / employed or 0),
    fx.set("ctx.capacityUnemployed", unemployed > 0 and capU / unemployed or 0),
  }
end
