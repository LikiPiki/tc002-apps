# @name Smoking Cigar
# @desc A pixel cigar with glowing embers and animated smoke.
# @version 1.0.0

import math

class Cigar
  def duration()
    return 15000
  end

  def draw()
    clear()
    var t = (now_ms() % 60000) / 1000.0

    # Thin wisps curl away from the ember and fade as they rise.
    for strand : 0 .. 2
      for level : 0 .. 9
        var bend = math.sin(t * 2.2 - level * 0.65 + strand * 1.7)
        var x = 40 + level / 4 + int(bend * (1 + level / 3))
        var y = 9 - level
        var glow = int((95 + 35 * math.sin(t * 3 - level * 0.8)) * (12 - level) / (12 * (strand + 1)))
        pixel(x, y, rgb(glow, glow, min(255, glow + 12)))
        if level > 4
          pixel(x + 1, y, rgb(glow / 3, glow / 3, glow / 3 + 5))
        end
      end
    end

    # Rounded tobacco wrapper, highlight, shadow, and diagonal seams.
    rect_fill(8, 10, 30, 5, 0x6E351B)
    line(9, 10, 36, 10, 0xB8753D)
    line(8, 11, 37, 11, 0x955026)
    line(9, 14, 37, 14, 0x3A1C12)
    line(7, 11, 7, 13, 0x633019)
    for seam : 0 .. 3
      var sx = 12 + seam * 7
      line(sx, 12, sx + 2, 14, 0x4B2417)
    end

    # Gold-and-red cigar band.
    rect_fill(17, 10, 6, 5, 0xD8A645)
    rect_fill(18, 11, 4, 3, 0x8C2630)
    pixel(19, 12, 0xFFE4A0)
    line(17, 14, 22, 14, 0x806029)

    # Breathing ember, bright hot flecks, and a small ash cap.
    var heat = int(125 + 100 * (math.sin(t * 3.5) + 1) / 2)
    rect_fill(38, 10, 2, 5, rgb(heat, heat / 4, 5))
    pixel(38, 11, rgb(255, heat / 2, 20))
    pixel(39, 13, rgb(255, heat / 3, 10))
    line(40, 11, 40, 13, 0x8A8176)
    pixel(40, 12, 0xD4C8B7)

    # An occasional tiny spark lifts off the burning tip.
    var tick = now_ms() / 140
    var spark = tick % 24
    if spark < 6
      pixel(40 + spark / 3, 9 - spark, rgb(210 - spark * 25, 65 - spark * 9, 0))
    end
  end
end

return Cigar()
