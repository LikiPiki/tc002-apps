# @name Swimming Mackerel
# @desc A golden pixel mackerel with a waving tail and a gentle swimming motion.
# @version 1.0.0

import math

class Mackerel
  var colors, rows

  def init()
    # sprite:begin
    # Generated from sprite.png by make sprites; edit the PNG instead.
    self.colors = [0x000000, 0xAEB2A0, 0xF4DFA4, 0xB99955, 0x8D7545, 0xDEC07B, 0x605638, 0xDFE2CD, 0x302B20]
    self.rows = [
      [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0],
      [0, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 1, 4, 4, 4, 4, 4, 3, 3, 1, 1, 1, 1, 1, 1, 1, 1, 1, 3, 3, 3, 4, 0, 0, 0],
      [0, 0, 2, 5, 3, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 4, 4, 6, 6, 6, 6, 6, 6, 1, 1, 7, 1, 1, 1, 1, 7, 7, 7, 7, 7, 7, 7, 1, 1, 1, 1, 1, 0, 0],
      [0, 0, 0, 5, 3, 3, 4, 0, 0, 0, 0, 0, 4, 6, 6, 8, 8, 8, 8, 8, 8, 8, 6, 6, 4, 4, 4, 6, 6, 6, 6, 6, 6, 6, 4, 4, 3, 1, 1, 1, 0, 1, 1, 0],
      [0, 0, 0, 0, 3, 6, 6, 8, 8, 6, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 8, 6, 6, 6, 8, 8, 6, 4, 1, 0],
      [0, 0, 0, 0, 0, 6, 6, 6, 4, 3, 3, 4, 3, 3, 3, 3, 4, 4, 4, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 4, 4, 4, 6, 6, 6, 6, 6, 6, 6, 6, 3, 4, 0, 0],
      [0, 0, 0, 0, 1, 4, 6, 4, 0, 1, 1, 1, 1, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 6, 4, 3, 4, 0, 0],
      [0, 0, 0, 0, 4, 8, 4, 0, 0, 0, 0, 0, 0, 1, 1, 3, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 2, 5, 5, 5, 5, 5, 3, 3, 3, 3, 4, 4, 3, 3, 3, 3, 0, 0],
      [0, 0, 0, 3, 6, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 3, 3, 3, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 3, 1, 0, 0],
      [0, 0, 0, 3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 2, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 0, 0, 0],
      [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 5, 5, 0, 0, 0, 0, 0, 0, 0]
    ]
    # sprite:end
  end

  def duration()
    return 15000
  end

  def draw()
    clear()
    var t = (now_ms() % 60000) / 1000.0
    var bob = int(math.sin(t * 2) * 1.1)
    var drift = int(math.sin(t * 0.9) * 1.2)

    # Bend columns with a traveling wave, strongest at the tail on the left.
    for x : 0 .. 43
      var sway = int(math.sin(t * 5 + x * 0.18) * (2.4 * (43 - x) / 43.0))
      for y : 0 .. 10
        var ink = self.rows[y][x]
        if ink != 0
          pixel(4 + drift + x, 3 + bob + sway + y, self.colors[ink])
        end
      end
    end

    # A small bubble rises beside the fish without covering it.
    var bubble = (now_ms() / 180) % 20
    if bubble < 13
      pixel(50, 14 - bubble, 0x34606D)
      if bubble % 4 == 0
        pixel(51, 14 - bubble, 0x183C48)
      end
    end
  end
end

return Mackerel()
