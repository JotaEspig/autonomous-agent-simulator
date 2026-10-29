-- Resets the monthly counters and last month's job seekers.
function run(world)
  local effects = { fx.clear("ctx.seeking") }
  for _, field in ipairs({ "hires", "jobChanges", "counteroffers", "layoffs", "bankruptcies", "newFirms",
    "deaths", "naturalDeaths", "retirements" }) do
    effects[#effects + 1] = fx.set("ctx." .. field, 0)
  end
  return effects
end
