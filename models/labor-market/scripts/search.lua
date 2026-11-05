-- Looking for a job, and the lowest wage the person accepts.
--
-- The unemployed look almost every month, and so do the employed whose pay does not cover
-- the cost of living (in the squeeze). Other employed people look now and then (searchBase),
-- more often the further their wage is below the average wage of their level, which salary
-- surveys publish (ctx.levelWage), and the less satisfied they are with their job
-- (searchDissatisfaction per unit of satisfaction below 0.5): dissatisfied people look
-- around, and quit more (Freeman 1978; Clark 2001).
--
-- Lowest acceptable wage (reservation wage). The poorer, the more desperate:
--   employed: the current wage plus switchPremium; nobody changes jobs for the same pay. In
--   the squeeze, any raise will do;
--   unemployed: anchored on the wage the person expects (the last wage, or with no job yet
--   the average wage of the level), revised each month by offerLearning toward the best
--   offer received, since offers tell job seekers what they are worth (Conlon et al. 2018);
--   falling each month without work by reservationDecay, faster for present-minded people
--   (reservationDecay x (0.5 + time preference)), who discount the better job they might
--   find later (DellaVigna and Paserman 2005). Scarcity overrides character: with little
--   saved, the present weighs more whatever the person's usual time preference, so the time
--   preference that drives the search is at least 1 - runway / runwayComfort (1 with nothing
--   saved), for as long as money is short (Mullainathan and Shafir 2013). It also falls in
--   proportion to the months of
--   living costs saved below runwayComfort: with half of runwayComfort saved the person asks
--   half of what it expects, and with less than one month saved, or already in the squeeze,
--   it takes any job to eat. Reservation wages fall slowly with unemployment and sharply when
--   money runs out (Krueger and Mueller 2016; Chetty 2008 on liquidity constraints).

local function peerWage(self, world)
  local ctx = world.ctx
  return ctx.levelWage[self.level] or ctx.meanWage
end

function propensity(self, world)
  if self.employer == 0 or self.monthsShort > 0 then
    return 50
  end
  local gap = peerWage(self, world) / math.max(self.wage, 1) - 1
  local ctx = world.ctx
  return ctx.searchBase + ctx.searchGap * math.max(0, gap) + ctx.searchDissatisfaction * math.max(0, 0.5 - self.satisfaction)
end

function on_success(self, world)
  local ctx = world.ctx
  local effects = {}
  local reservation
  if self.employer ~= 0 then
    if self.monthsShort > 0 then
      reservation = self.wage + 1
    else
      reservation = self.wage * (1 + ctx.switchPremium)
    end
  else
    local expected = self.expectedWage
    if expected == 0 then
      expected = self.lastWage > 0 and self.lastWage or peerWage(self, world)
    end
    if self.bestOffer > 0 then
      expected = expected + ctx.offerLearning * (self.bestOffer - expected)
    end
    effects[#effects + 1] = fx.set("self.expectedWage", expected)
    effects[#effects + 1] = fx.set("self.bestOffer", 0)
    local runway = self.savings / ctx.subsistence
    if runway < 1 or self.monthsShort > 0 then
      reservation = 0
    else
      local pressure = math.min(1, runway / ctx.runwayComfort)
      local present = math.max(self.timePreference, 1 - runway / ctx.runwayComfort)
      local decay = ctx.reservationDecay * (0.5 + present)
      reservation = expected * (1 - decay) ^ self.monthsUnemployed * pressure
    end
  end
  effects[#effects + 1] = fx.set("ctx.seeking." .. self.id, reservation)
  effects[#effects + 1] = fx.set("self.reservation", reservation)
  return effects
end
