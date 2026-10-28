-- Land market between people. Bids are served from the highest; each buys the cheapest
-- plot on offer whose ask it covers (never one's own), at the midpoint of bid and ask.
-- Buyers who can no longer afford the price are skipped.

local function byPriceDesc(a, b)
  if a.price ~= b.price then return a.price > b.price end
  return a.tie < b.tie
end

local function byPriceAsc(a, b)
  if a.price ~= b.price then return a.price < b.price end
  return a.tie < b.tie
end

function run(world)
  local ctx = world.ctx
  local bids, asks = {}, {}
  for _, key in ipairs(world:keys("landBids")) do
    bids[#bids + 1] = { id = tonumber(key), price = ctx.landBids[key], tie = world:rand() }
  end
  for _, key in ipairs(world:keys("landAsks")) do
    asks[#asks + 1] = { id = tonumber(key), price = ctx.landAsks[key], tie = world:rand() }
  end
  table.sort(bids, byPriceDesc)
  table.sort(asks, byPriceAsc)

  local effects, bestUnmatched = {}, 0
  for _, bid in ipairs(bids) do
    local buyer = world:agent(bid.id)
    local matched = false
    for _, ask in ipairs(asks) do
      if not ask.done and ask.id ~= bid.id and ask.price <= bid.price then
        local price = (ask.price + bid.price) / 2
        local seller = world:agent(ask.id)
        if buyer.wealth >= price and seller.lands > 0 then
          local b, s = "agent." .. bid.id, "agent." .. ask.id
          effects[#effects + 1] = fx.transfer(b .. ".wealth", s .. ".wealth", price)
          effects[#effects + 1] = fx.add(b .. ".spent", price)
          effects[#effects + 1] = fx.add(b .. ".lands", 1)
          effects[#effects + 1] = fx.add(s .. ".lands", -1)
          if buyer.lands == 0 then
            effects[#effects + 1] = fx.set(b .. ".askPrice", ctx.marketPrice)
          end
          effects[#effects + 1] = fx.set("ctx.landPrice", price)
          effects[#effects + 1] = fx.emit("land_sold", { buyer = bid.id, seller = ask.id, price = price })
          ask.done, matched = true, true
          break
        end
      end
    end
    if not matched and bid.price > bestUnmatched then
      bestUnmatched = bid.price
    end
  end
  effects[#effects + 1] = fx.set("ctx.bestLandBid", bestUnmatched)
  effects[#effects + 1] = fx.clear("ctx.landBids")
  effects[#effects + 1] = fx.clear("ctx.landAsks")
  return effects
end
