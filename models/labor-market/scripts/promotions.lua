-- Yearly review. The employer, who sees what each of its people delivers, promotes those
-- whose capacity reached the next level. Titles never go down, and the unemployed keep
-- the title of their last job. A promotion makes people feel valued (promotionBoost), and
-- when the person joined as a junior, it tells the firm its investment paid off.

function run(self, world)
  if self.employer == 0 then
    return nil
  end
  local ctx = world.ctx
  local level = self.level
  if self.capacity >= ctx.seniorCapacity then
    level = "senior"
  elseif self.capacity >= ctx.plenoCapacity and level == "junior" then
    level = "pleno"
  end
  if level ~= self.level then
    local effects = { fx.set("self.level", level), fx.add("self.satisfaction", ctx.promotionBoost) }
    if self.hiredJunior then
      effects[#effects + 1] = fx.set("self.paidOff", self.employer)
      effects[#effects + 1] = fx.set("self.hiredJunior", false)
    end
    effects[#effects + 1] = fx.emit("promoted", { level = level, capacity = self.capacity, wage = self.wage })
    return effects
  end
  return nil
end
