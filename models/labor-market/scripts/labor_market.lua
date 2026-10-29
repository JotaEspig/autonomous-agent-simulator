-- The labor market, as recruiting works for skilled jobs.
--
-- 1. Interviews. Each firm with openings interviews interviewsPerOpening job seekers per
--    opening, drawn at random among this month's seekers (it does not meet them all), and
--    never its own staff. An interview reads the candidate's capacity with a relative error
--    of interviewNoise; the title and experience on the CV are already in that read.
-- 2. Offers. The firm offers jobs to its best-rated candidates, one per opening, each at
--    the estimated value of the person's output to the firm (price x estimate x (1 - share /
--    elasticity), with the share the firm would have after the hire) minus the firm's
--    markdown. When a candidate turns an offer down or takes
--    another job, the firm moves on to the next candidate on its list, in rounds, until its
--    openings are filled or its list runs out.
-- 3. Choice. Each person takes the best offer, if it is at least the lowest wage it accepts
--    (registered in ctx.seeking by the search decision). An employed person needs
--    switchPremium more than its wage. An unemployed person remembers the best offer it
--    turned down, which revises what it expects to earn (see search.lua).
-- 4. Counteroffers. An employer who sees one of its people about to leave matches the
--    outside offer when the person is still worth more to it (it knows the real capacity),
--    keeping at least counterMargin of the value; the person stays at the new wage. This is
--    how wages rise within a job: through outside offers (Postel-Vinay and Robin 2002).
--    Otherwise the person moves.
-- 5. Learning. A firm that filled all its openings offers a little less next time
--    (markdownRise); one left with openings it could not fill offers more, in proportion to
--    the share left unfilled (markdownCut), as firms raise pay when vacancies stay open.
--    Each hire costs hiringCost months of its wage.

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

  -- Interviews: each firm's candidates, best-rated first.
  local firms, order = {}, {}
  for _, fid in ipairs(world:ids("firm")) do
    local f = world:agent(fid)
    local firm = { id = fid, markdown = f.markdown, share = f.share,
      openings = f.openings, open = f.openings, next = 1, rated = {}, cap = f.capacitySum,
      mr = ctx.price * (1 - f.share / ctx.elasticity) }
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
          firm.rated[#firm.rated + 1] = { id = pool[i], estimate = estimate }
        end
      end
      table.sort(firm.rated, function(a, b)
        if a.estimate ~= b.estimate then return a.estimate > b.estimate end
        return a.id < b.id
      end)
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
          local after = (firm.cap + c.estimate) / (ctx.output + c.estimate)
          local offer = math.floor(ctx.price * (1 - after / ctx.elasticity) * c.estimate * (1 - firm.markdown))
          if offer > 0 then
            if not offers[c.id] then
              offers[c.id] = {}
              receivers[#receivers + 1] = c.id
            end
            offers[c.id][#offers[c.id] + 1] = { firm = fid, wage = offer, tie = world:rand() }
            slots = slots - 1
          end
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
      if w.employer ~= 0 then
        accept = accept and best.wage >= w.wage * (1 + ctx.switchPremium)
      end
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
        markdown = f.markdown - ctx.markdownCut * f.open / f.openings
      end
      effects[#effects + 1] = fx.set("agent." .. fid .. ".markdown", markdown)
    end
  end
  return effects
end
