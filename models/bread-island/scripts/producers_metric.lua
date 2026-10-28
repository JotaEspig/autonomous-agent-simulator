function metric(world)
  local n = 0
  for _, id in ipairs(world:ids("islander")) do
    if world:agent(id).lands > 0 then
      n = n + 1
    end
  end
  return n
end
