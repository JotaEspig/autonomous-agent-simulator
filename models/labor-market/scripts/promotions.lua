-- Yearly review. The employer, who sees what each of its people delivers, promotes those
-- whose capacity reached the next level. Titles never go down, and the unemployed keep
-- the title of their last job.

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
    return { fx.set("self.level", level), fx.emit("promoted", { level = level, capacity = self.capacity, wage = self.wage }) }
  end
  return nil
end
