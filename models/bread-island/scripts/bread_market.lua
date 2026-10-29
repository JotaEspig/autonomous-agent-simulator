-- Order book, as in a stock broker. Producers' offers were posted first; buy orders are
-- served from the highest limit (ties drawn at random). Each order buys the cheapest
-- bread still on offer if its price is within the order's limit, and pays the posted
-- price, not the limit: someone willing to pay 2.70 for bread posted at 2.50 goes first
-- in line and pays 2.50. A buyer who can no longer cover the price is skipped. The
-- producer keeps (1 - productionCost) of each sale as profit; the rest is the cost of
-- making it.

local function addTo(t, id, v)
  t[id] = (t[id] or 0) + v
end

function run(world)
  local ctx = world.ctx
  local bids, asks = {}, {}
  for _, key in ipairs(world:keys("breadBids")) do
    local id = tonumber((string.match(key, "^(%d+)%.")))
    bids[#bids + 1] = { id = id, limit = ctx.breadBids[key], tie = world:rand() }
  end
  local supply = 0
  for _, key in ipairs(world:keys("breadAskQty")) do
    local qty = ctx.breadAskQty[key]
    asks[#asks + 1] = { id = tonumber(key), price = ctx.breadAskPrice[key], left = qty, tie = world:rand() }
    supply = supply + qty
  end
  table.sort(bids, function(a, b)
    if a.limit ~= b.limit then return a.limit > b.limit end
    return a.tie < b.tie
  end)
  table.sort(asks, function(a, b)
    if a.price ~= b.price then return a.price < b.price end
    return a.tie < b.tie
  end)

  local cash, spend, eaten, profit, sold = {}, {}, {}, {}, {}
  local units, revenue, cheapest = 0, 0, 1
  for _, bid in ipairs(bids) do
    while asks[cheapest] and asks[cheapest].left == 0 do
      cheapest = cheapest + 1
    end
    local ask = asks[cheapest]
    if cash[bid.id] == nil then
      cash[bid.id] = world:agent(bid.id).wealth
    end
    if ask and ask.price <= bid.limit and ask.price <= cash[bid.id] then
      ask.left = ask.left - 1
      cash[bid.id] = cash[bid.id] - ask.price
      addTo(spend, bid.id, ask.price)
      addTo(eaten, bid.id, 1)
      addTo(profit, ask.id, ask.price * (1 - ctx.productionCost))
      addTo(sold, ask.id, 1)
      units = units + 1
      revenue = revenue + ask.price
    end
  end

  local effects = {}
  for id, amount in pairs(spend) do
    local who = "agent." .. id
    effects[#effects + 1] = fx.add(who .. ".wealth", -amount)
    effects[#effects + 1] = fx.add(who .. ".spent", amount)
    effects[#effects + 1] = fx.add(who .. ".ateToday", eaten[id])
  end
  for id, amount in pairs(profit) do
    local who = "agent." .. id
    effects[#effects + 1] = fx.add(who .. ".wealth", amount)
    effects[#effects + 1] = fx.add(who .. ".income", amount)
    effects[#effects + 1] = fx.add(who .. ".stock", -sold[id])
  end

  if units > 0 then
    effects[#effects + 1] = fx.set("ctx.marketPrice", revenue / units)
  end
  local sellRate, scarcity = 0, 0
  if supply > 0 then
    sellRate = units / supply
  end
  if #bids > 0 then
    scarcity = (#bids - units) / #bids
  end
  effects[#effects + 1] = fx.set("ctx.breadSold", units)
  effects[#effects + 1] = fx.set("ctx.breadWasted", supply - units)
  effects[#effects + 1] = fx.set("ctx.unmetDemand", #bids - units)
  effects[#effects + 1] = fx.set("ctx.sellRate", sellRate)
  effects[#effects + 1] = fx.set("ctx.scarcity", scarcity)
  for _, field in ipairs({ "breadBids", "breadAskQty", "breadAskPrice" }) do
    effects[#effects + 1] = fx.clear("ctx." .. field)
  end
  return effects
end
