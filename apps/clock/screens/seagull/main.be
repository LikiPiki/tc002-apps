# @name Kicking Seagull
# @desc An upside-down seagull waving its feet.
# @version 1.0.0
# The body sprite is generated from sprite.png with make sprites.

import math

class Seagull
  var colors, rows

  def init()
    # sprite:begin
    # Generated from sprite.png by make sprites; edit the PNG instead.
    self.colors = [0x000000, 0xF4F4EF, 0xB4B7B4, 0x767D7E, 0xC64B25, 0xEAB044]
    self.rows = [
      [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
      [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
      [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
      [0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 3, 2, 2, 2, 1, 1, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0],
      [0, 0, 0, 0, 0, 3, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 1, 1, 1, 1, 2, 2, 2, 2, 2, 0, 0, 0, 0],
      [0, 0, 0, 0, 0, 3, 4, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 1, 1, 1, 2, 2, 2, 2, 2, 2, 0, 0, 0],
      [0, 0, 2, 5, 2, 2, 5, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 0, 0],
      [0, 0, 0, 2, 2, 1, 1, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 0, 0, 0, 0, 0, 0],
      [0, 0, 0, 0, 0, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
      [0, 0, 0, 0, 0, 0, 2, 1, 1, 1, 1, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
      [0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 2, 1, 1, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
      [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]
    ]
    # sprite:end
  end

  def duration()
    return 15000
  end

  def draw()
    clear()
    var t = (now_ms() % 60000) / 1000.0

    # Draw the feet behind the body, alternating their kicks.
    for leg : 0 .. 1
      var phase = t * 5 + leg * 3.14
      var kick = int(math.sin(phase) * 2)
      var lift = int((math.sin(phase + 1) + 1) * 0.8)
      var knee_x = 29 + leg * 2 + kick
      var foot_x = 25 + leg * 2 + kick
      var foot_y = 2 - lift
      var shade = leg == 0 ? 0xD49B36 : 0xFFD06A
      line(26 + leg * 2, 7, knee_x, 4, shade)
      line(knee_x, 4, foot_x, foot_y, shade)
      line(foot_x, foot_y, foot_x - 3, foot_y, shade)
      line(foot_x, foot_y, foot_x - 2, max(0, foot_y - 1), shade)
      line(foot_x, foot_y, foot_x - 2, foot_y + 1, shade)
    end

    # The screenshot-derived body stays still while the legs move.
    for y : 0 .. 11
      for x : 0 .. 47
        var ink = self.rows[y][x]
        if ink != 0
          pixel(x + 2, y + 4, self.colors[ink])
        end
      end
    end
  end
end

return Seagull()
