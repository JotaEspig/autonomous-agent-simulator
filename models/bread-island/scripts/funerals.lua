-- Removes the people who died today. Their plots go to their oldest living child; with no
-- living child the plots are abandoned and go back to the island, which sells them ever
-- cheaper until someone buys (see land_office.lua). Savings are not inherited.

function run(world)
  local effects = {}
  for _, id in ipairs(world:ids("islander")) do
    local person = world:agent(id)
    if person.dead then
      if person.lands > 0 then
        local heir
        for _, child in ipairs(world:neighbors("parent", id)) do
          if not world:agent(child).dead then
            heir = world:agent(child)
            break
          end
        end
        if heir then
          effects[#effects + 1] = fx.add("agent." .. heir.id .. ".lands", person.lands)
          if heir.lands == 0 then
            effects[#effects + 1] = fx.set("agent." .. heir.id .. ".askPrice", person.askPrice)
          end
          effects[#effects + 1] = fx.emit("inherited", { heir = heir.id, from = id, lands = person.lands })
        else
          effects[#effects + 1] = fx.add("ctx.landsFree", person.lands)
          effects[#effects + 1] = fx.emit("abandoned", { from = id, lands = person.lands })
        end
      end
      effects[#effects + 1] = fx.remove(id)
    end
  end
  return effects
end
