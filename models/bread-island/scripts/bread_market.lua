-- Double auction for bread.
--
-- Each bid is for one bread. Bids are served from the highest offer; each one buys the
-- cheapest bread still available if the buyer accepts its price (ask <= the bid's
-- maximum). The buyer pays its offer, or the ask when the ask is higher. The producer
-- keeps (1 - productionCost) of each sale as profit; the rest is the cost of making it.

local function byPriceDesc(a, b)
  if a.price ~= b.price then return a.price > b.price end
  return a.tie < b.tie
end

local function byPriceAsc(a, b)
  if a.price ~= b.price then return a.price < b.price end
  return a.tie < b.tie
end

local function addTo(t, id, v)
  t[id] = (t[id] or 0) + v
end

function run(world)
  local ctx = world.ctx
  local bids, asks = {}, {}
  for _, key in ipairs(world:keys("breadBids")) do
    local id = tonumber((string.match(key, "^(%d+)%.")))
    bids[#bids + 1] = { id = id, price = ctx.breadBids[key], max = ctx.breadBidMax[key], tie = world:rand() }
  end
  local supply = 0
  for _, key in ipairs(world:keys("breadAskQty")) do
    local qty = ctx.breadAskQty[key]
    asks[#asks + 1] = { id = tonumber(key), price = ctx.breadAskPrice[key], left = qty, tie = world:rand() }
    supply = supply + qty
  end
  table.sort(bids, byPriceDesc)
  table.sort(asks, byPriceAsc)

  local spend, eaten, profit, sold = {}, {}, {}, {}
  local effects = {}
  local units, revenue, cheapest = 0, 0, 1
  for _, bid in ipairs(bids) do
    while asks[cheapest] and asks[cheapest].left == 0 do
      cheapest = cheapest + 1
    end
    local ask = asks[cheapest]
    if ask and ask.price <= bid.max then
      local price = math.max(bid.price, ask.price)
      ask.left = ask.left - 1
      addTo(spend, bid.id, price)
      addTo(eaten, bid.id, 1)
      addTo(profit, ask.id, price * (1 - ctx.productionCost))
      addTo(sold, ask.id, 1)
      effects[#effects + 1] = fx.remember("salePrices", price, ask.id)
      units = units + 1
      revenue = revenue + price
    end
  end

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
  local sellRate = 0
  if supply > 0 then
    sellRate = units / supply
  end
  local scarcity = 0
  if #bids > 0 then
    scarcity = (#bids - units) / #bids
  end
  effects[#effects + 1] = fx.set("ctx.breadSold", units)
  effects[#effects + 1] = fx.set("ctx.breadWasted", supply - units)
  effects[#effects + 1] = fx.set("ctx.unmetDemand", #bids - units)
  effects[#effects + 1] = fx.set("ctx.sellRate", sellRate)
  effects[#effects + 1] = fx.set("ctx.scarcity", scarcity)
  for _, field in ipairs({ "breadBids", "breadBidMax", "breadAskQty", "breadAskPrice" }) do
    effects[#effects + 1] = fx.clear("ctx." .. field)
  end
  return effects
end
