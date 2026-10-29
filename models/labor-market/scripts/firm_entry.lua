-- Entrepreneurs outside the model open a new firm when the sector is profitable: the chance
-- grows with the sector's profit margin, up to maxEntryChance a month. The new firm starts
-- with startingCapital, no staff, and expects the sector's margin.

function run(world)
  local ctx = world.ctx
  local p = math.min(ctx.maxEntryChance, ctx.entrySensitivity * math.max(0, ctx.profitMargin))
  if not world:chance(p) then
    return nil
  end
  return {
    fx.spawn("firm", {
      cash = ctx.startingCapital,
      markdown = 0.2 + 0.3 * world:rand(),
      margin = ctx.profitMargin,
    }),
    fx.add("ctx.newFirms", 1),
    fx.emit("firm_founded", { profitMargin = ctx.profitMargin }),
  }
end
