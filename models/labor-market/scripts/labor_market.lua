-- The labor market, as recruiting works for skilled jobs.
--
-- 1. Interviews. Each firm with openings interviews interviewsPerOpening job seekers per
--    opening, drawn at random among this month's seekers (it does not meet them all), and
--    never its own staff. An interview reads the candidate's capacity with a relative error
--    of interviewNoise; the title and experience on the CV are already in that read.
--    Recruiters then pull extreme reads toward the average of the candidates of the same
--    level (see shrink).
-- 2. Ranking: hiring as an investment. The firm ranks candidates by the profit it expects
--    from them over its own horizon (the firm's `horizon`, cut short by the months the
--    candidate has left before retirement, read from the age on the CV). Capacity grows with
--    work toward what an average career reaches (careerCeiling), at the pace firms see in
--    their own staff (learningRate), while the wage stays what was agreed unless a rival
--    bids. So a cheap junior can be worth more than an experienced hire to a firm that looks
--    far ahead, and someone close to retirement is worth less:
--      average capacity over H months = C - (C - estimate) x (1 - (1 - learningRate)^H) / (H x learningRate)
--      expected profit = value per unit of capacity x H x (average capacity - wage / value per unit) - hiring cost
--    Candidates who would not pay for themselves get no offer.
-- 3. Offers. The firm offers jobs to its best-ranked candidates, one per opening, each at
--    the estimated value of the person's current output to the firm (expected price x
--    estimate x (1 - share / elasticity), where the expected price is today's plus the
--    recent trend (priceTrend): a falling price says buyers want less than the sector makes, with the share the firm would have after the hire) minus the
--    firm's markdown: pay follows what the person delivers today. When a candidate turns an
--    offer down or takes another job, the firm moves on to the next candidate on its list,
--    in rounds, until its openings are filled or its list runs out.
-- 4. Choice. Each person takes the best offer (a new job starts with a honeymoon:
--    satisfactionHire), if it is at least the lowest wage it accepts
--    (registered in ctx.seeking by the search decision: for an employed person, its wage
--    plus switchPremium, or any raise when its pay does not cover the cost of living). An
--    unemployed person remembers the best offer it turned down, which revises what it
--    expects to earn (see search.lua).
-- 5. Counteroffers. An employer who sees one of its people about to leave matches the
--    outside offer when the person is still worth more to it (it knows the real capacity),
--    keeping at least counterMargin of the value; the person stays at the new wage, and
--    feels more valued (raiseBoost). This is
--    how wages rise within a job: through outside offers (Postel-Vinay and Robin 2002).
--    Otherwise the person moves.
-- 6. Learning. A firm that filled all its openings offers a little less next time
--    (markdownRise); one left with openings it could not fill offers more, in proportion to
--    the share left unfilled (markdownCut), as firms raise pay when vacancies stay open, but
--    never so far that what it keeps no longer covers its overhead (overhead over last
--    month's revenue): no firm bids itself into losses. A start-up with no revenue yet has
--    no such floor: it must match the market to hire anyone.
--    Each hire costs hiringCost months of its wage.

-- Recruiters distrust extreme reads (shrinkInterviews). They know from experience how much
-- interviews err (interviewNoise), so they can tell how much of the spread among this
-- month's candidates is real: reliability = (variance of the reads - variance of the error)
-- / variance of the reads. Each read is pulled toward the candidates' average in that
-- proportion: estimate = average + reliability x (read - average). This is the best guess of
-- a candidate's capacity given a noisy read (Bayesian shrinkage, as in Kelley's formula).
-- Recruiters compare like with like: the title is on the CV, so each candidate is measured
-- against the candidates of its own level (see shrinkByLevel); a junior is not pulled up
-- toward the seniors' average.
local function shrink(rated, ctx)
  local n = #rated
  if not ctx.shrinkInterviews or n < 2 then
    return
  end
  local sum, sumSq = 0, 0
  for _, c in ipairs(rated) do
    sum = sum + c.estimate
    sumSq = sumSq + c.estimate * c.estimate
  end
  local mean = sum / n
  local variance = sumSq / n - mean * mean
  if variance <= 0 then
    return
  end
  local s2 = ctx.interviewNoise * ctx.interviewNoise
  local noise = s2 * (sumSq / n) / (1 + s2)
  local reliability = math.max(0, math.min(1, (variance - noise) / variance))
  for _, c in ipairs(rated) do
    c.estimate = mean + reliability * (c.estimate - mean)
  end
end

local function shrinkByLevel(rated, ctx)
  local groups = { junior = {}, pleno = {}, senior = {} }
  for _, c in ipairs(rated) do
    local g = groups[c.level]
    g[#g + 1] = c
  end
  for _, level in ipairs({ "junior", "pleno", "senior" }) do
    shrink(groups[level], ctx)
  end
end

function run(world)
  local ctx = world.ctx
  local seekers = {}
  for _, key in ipairs(world:keys("seeking")) do
    local id = tonumber(key)
    if world:exists(id) then
      seekers[#seekers + 1] = id
    end
  end
  if #seekers == 0 then
    return nil
  end

  -- Firms value people at the price they expect next month: today's, plus the recent trend.
  local expected = ctx.price * (1 + ctx.priceTrend)

  -- Interviews: each firm's candidates, ranked by the profit it expects from them.
  local firms, order = {}, {}
  for _, fid in ipairs(world:ids("firm")) do
    local f = world:agent(fid)
    local firm = { id = fid, markdown = f.markdown, share = f.share, horizon = f.horizon, revenue = f.revenue, wageBill = f.wageBill,
      openings = f.openings, open = f.openings, next = 1, rated = {}, cap = f.capacitySum,
      mr = expected * (1 - f.share / ctx.elasticity) }
    firms[fid] = firm
    order[#order + 1] = fid
    if f.openings > 0 then
      local pool = {}
      for i = 1, #seekers do
        pool[i] = seekers[i]
      end
      local want = math.min(#pool, f.openings * ctx.interviewsPerOpening)
      for i = 1, want do
        local j = world:randint(i, #pool)
        pool[i], pool[j] = pool[j], pool[i]
        local w = world:agent(pool[i])
        if w.employer ~= fid then
          local estimate = w.capacity * math.max(0.1, 1 + world:normal(0, ctx.interviewNoise))
          firm.rated[#firm.rated + 1] = { id = pool[i], estimate = estimate, age = w.age, level = w.level }
        end
      end
      shrinkByLevel(firm.rated, ctx)
      local rate = ctx.learningRate
      local ranked = {}
      for _, c in ipairs(firm.rated) do
        local unit = expected * (1 - (firm.cap + c.estimate) / (ctx.output + c.estimate) / ctx.elasticity)
        c.offer = math.floor(unit * c.estimate * (1 - firm.markdown))
        local months = math.max(1, math.min(firm.horizon, ctx.retirementAge - c.age))
        local ceiling = math.max(ctx.careerCeiling, c.estimate)
        local average = ceiling - (ceiling - c.estimate) * (1 - (1 - rate) ^ months) / (months * rate)
        c.profit = months * (unit * average - c.offer) - ctx.hiringCost * c.offer
        if c.offer > 0 and c.profit > 0 then
          ranked[#ranked + 1] = c
        end
      end
      table.sort(ranked, function(a, b)
        if a.profit ~= b.profit then return a.profit > b.profit end
        return a.id < b.id
      end)
      firm.rated = ranked
    end
  end

  -- Rounds of offers. A person who took a job, or stayed after a counteroffer, is settled
  -- for the month; one who turned offers down may still get better ones in a later round.
  local settled, bestOffer = {}, {}
  local effects = {}
  while true do
    local offers, receivers = {}, {}
    for _, fid in ipairs(order) do
      local firm = firms[fid]
      local slots = firm.open
      while slots > 0 and firm.next <= #firm.rated do
        local c = firm.rated[firm.next]
        firm.next = firm.next + 1
        if not settled[c.id] then
          if not offers[c.id] then
            offers[c.id] = {}
            receivers[#receivers + 1] = c.id
          end
          offers[c.id][#offers[c.id] + 1] = { firm = fid, wage = c.offer, tie = world:rand() }
          slots = slots - 1
        end
      end
    end
    if #receivers == 0 then
      break
    end
    table.sort(receivers)
    for _, id in ipairs(receivers) do
      local list = offers[id]
      table.sort(list, function(a, b)
        if a.wage ~= b.wage then return a.wage > b.wage end
        return a.tie < b.tie
      end)
      local best = list[1]
      local w = world:agent(id)
      local who = "agent." .. id
      local accept = best.wage >= ctx.seeking[tostring(id)]
      if not accept and w.employer == 0 and best.wage > (bestOffer[id] or 0) then
        bestOffer[id] = best.wage
      end
      if accept then
        settled[id] = true
        bestOffer[id] = nil
        local current = firms[w.employer]
        local matched = false
        if current then
          local value = current.mr * w.capacity
          if best.wage <= value * (1 - ctx.counterMargin) then
            matched = true
            effects[#effects + 1] = fx.set(who .. ".wage", best.wage)
            effects[#effects + 1] = fx.set(who .. ".expectedWage", best.wage)
            effects[#effects + 1] = fx.add(who .. ".satisfaction", ctx.raiseBoost)
            effects[#effects + 1] = fx.add("ctx.counteroffers", 1)
            effects[#effects + 1] = fx.emit("counteroffer", { worker = id, employer = w.employer, rival = best.firm, from = w.wage, to = best.wage })
          end
        end
        if not matched then
          if current then
            effects[#effects + 1] = fx.add("agent." .. w.employer .. ".headcount", -1)
            effects[#effects + 1] = fx.add("ctx.jobChanges", 1)
            effects[#effects + 1] = fx.emit("poached", { worker = id, from = w.employer, to = best.firm, oldWage = w.wage, wage = best.wage })
          else
            effects[#effects + 1] = fx.add("ctx.hires", 1)
          end
          local firm = firms[best.firm]
          effects[#effects + 1] = fx.set(who .. ".employer", best.firm)
          effects[#effects + 1] = fx.set(who .. ".wage", best.wage)
          effects[#effects + 1] = fx.set(who .. ".lastWage", best.wage)
          effects[#effects + 1] = fx.set(who .. ".expectedWage", best.wage)
          effects[#effects + 1] = fx.set(who .. ".satisfaction", ctx.satisfactionHire)
          effects[#effects + 1] = fx.set(who .. ".hiredAt", world.tick)
          effects[#effects + 1] = fx.set(who .. ".hiredJunior", w.level == "junior")
          if current and w.hiredJunior and world.tick - w.hiredAt < ctx.investmentPayback then
            effects[#effects + 1] = fx.set(who .. ".investmentLost", w.employer)
          end
          effects[#effects + 1] = fx.add("agent." .. best.firm .. ".headcount", 1)
          effects[#effects + 1] = fx.add("agent." .. best.firm .. ".cash", -math.floor(ctx.hiringCost * best.wage))
          firm.open = firm.open - 1
        end
      end
    end
  end

  for _, id in ipairs(seekers) do
    if bestOffer[id] then
      effects[#effects + 1] = fx.set("agent." .. id .. ".bestOffer", bestOffer[id])
    end
  end

  for _, fid in ipairs(order) do
    local f = firms[fid]
    if f.openings > 0 then
      local markdown
      if f.open == 0 then
        markdown = f.markdown + ctx.markdownRise
      else
        local floor = 0
        if f.revenue > 0 then
          floor = (ctx.fixedCost + ctx.overheadShare * f.wageBill) / math.max(f.revenue, ctx.fixedCost)
        end
        markdown = math.max(math.min(f.markdown, floor), f.markdown - ctx.markdownCut * f.open / f.openings)
      end
      effects[#effects + 1] = fx.set("agent." .. fid .. ".markdown", markdown)
    end
  end
  return effects
end
