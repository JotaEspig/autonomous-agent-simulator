-- Posted prices are public: publish the cheapest price asked by a producer with bread,
-- so buyers know how much they need to offer.

function run(world)
  local lowest = 0
  for _, id in ipairs(world:ids("islander")) do
    local a = world:agent(id)
    if a.stock > 0 and (lowest == 0 or a.askPrice < lowest) then
      lowest = a.askPrice
    end
  end
  return fx.set("ctx.lowestAsk", lowest)
end
