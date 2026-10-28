function run(world) return nil end
function uses_random() return math.random() end
function writes_global() counter = 1 end
function reads_hidden(self, world) return world.ctx.secret end
function reads_private(self, world) return world:agent(2).hidden end
function reads_public(self, world) return world:agent(2).shown end
function sorted_pairs()
  local keys = ""
  for k in pairs({ c = 1, a = 2, b = 3, [2] = 0, [1] = 0 }) do keys = keys .. k end
  return keys == "12abc"
end
function writes_self(self, world) self.shown = 3 end
