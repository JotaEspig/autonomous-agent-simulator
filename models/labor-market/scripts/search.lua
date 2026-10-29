-- Looking for a job, and the lowest wage the person accepts.
--
-- The unemployed look almost every month. The employed look now and then (searchBase), and
-- more often the further their wage is below the average wage of their level, which salary
-- surveys publish (ctx.levelWage): people sense they are underpaid by comparing with peers.
--
-- Lowest acceptable wage (reservation wage):
--   employed: the current wage plus switchPremium; nobody changes jobs for the same pay;
--   unemployed: anchored on the wage the person expects (the last wage, or with no job yet
--   the average wage of the level), revised each month by offerLearning toward the best
--   offer received, since offers tell job seekers what they are worth (Conlon et al. 2018);
--   falling by reservationDecay each month without work, and lower when savings run short:
--   with less than runwayComfort months of living costs saved it drops toward half, and with
--   less than one month the person takes any job. Reservation wages fall slowly with
--   unemployment and sharply when money runs out (Krueger and Mueller 2016).

local function peerWage(self, world)
  local ctx = world.ctx
  return ctx.levelWage[self.level] or ctx.meanWage
end

function propensity(self, world)
  if self.employer == 0 then
    return 50
  end
  local gap = peerWage(self, world) / math.max(self.wage, 1) - 1
  return world.ctx.searchBase + world.ctx.searchGap * math.max(0, gap)
end

function on_success(self, world)
  local ctx = world.ctx
  local effects = {}
  local reservation
  if self.employer ~= 0 then
    reservation = self.wage * (1 + ctx.switchPremium)
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
    if runway < 1 then
      reservation = 0
    else
      local pressure = 0.5 + 0.5 * math.min(1, runway / ctx.runwayComfort)
      reservation = expected * (1 - ctx.reservationDecay) ^ self.monthsUnemployed * pressure
    end
  end
  effects[#effects + 1] = fx.set("ctx.seeking." .. self.id, reservation)
  effects[#effects + 1] = fx.set("self.reservation", reservation)
  return effects
end
