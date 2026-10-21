# @name Weather Now
# @desc Current weather from the VPS Open-Meteo provider.
# @version 1.0.0
# @display 52x16

class WeatherCurrent
  var reading

  def init()
    self.reading = store.get("weather", nil)
  end

  def duration()
    return 10000
  end

  def fresh()
    var updated = num(self.reading.find("updated_at_ms"))
    var ttl = num(self.reading.find("ttl_seconds"))
    var now = epoch_ms()

    if updated == nil || ttl == nil || ttl <= 0 || now < 0
      return false
    end

    var age = now - updated
    return age >= 0 && age < ttl * 1000
  end

  def draw_state(message)
    text(0, 6, "WEATHER", 0xFFFFFF)
    text(0, 14, message, 0xFFAA22)
    pixel(51, 0, 0xFFAA22)
  end

  def draw_icon(kind, daylight)
    if kind == "CLEAR"
      if daylight
        circle_fill(4, 3, 2, 0xFFCC33)
        line(4, 0, 4, 6, 0xFFCC33)
        line(1, 3, 7, 3, 0xFFCC33)
      else
        circle_fill(4, 3, 3, 0x99BBFF)
        circle_fill(6, 2, 2, 0x000000)
      end
    else
      circle_fill(3, 3, 2, 0xAAAAAA)
      circle_fill(5, 3, 2, 0xCCCCCC)
      rect_fill(1, 3, 7, 3, 0xAAAAAA)

      if kind == "RAIN" || kind == "DRIZZLE"
        pixel(2, 7, 0x3399FF)
        pixel(5, 7, 0x3399FF)
      elif kind == "SNOW"
        pixel(2, 7, 0xFFFFFF)
        pixel(4, 6, 0xFFFFFF)
        pixel(6, 7, 0xFFFFFF)
      elif kind == "STORM"
        line(4, 4, 3, 6, 0xFFCC33)
        line(3, 6, 5, 6, 0xFFCC33)
        pixel(4, 7, 0xFFCC33)
      elif kind == "FOG"
        line(0, 6, 8, 6, 0x888888)
      end
    end
  end

  def draw()
    clear()

    if !isinstance(self.reading, map) || self.reading.find("version") != 1
      self.draw_state("NO DATA")
      return
    end

    if !self.fresh()
      self.draw_state("STALE")
      return
    end

    var current = self.reading.find("current")
    if !isinstance(current, map)
      self.draw_state("NO DATA")
      return
    end

    var temperature = num(current.find("temperature_c"))
    var wind = num(current.find("wind_speed_ms"))
    var label = self.reading.find("label")
    if temperature == nil || wind == nil || type(label) != "string"
      self.draw_state("NO DATA")
      return
    end

    self.draw_icon(current.find("condition"), current.find("is_day"))
    text(10, 6, label, 0xFFFFFF)
    text(0, 14, format("%dC %dm/s", round(temperature), round(wind)), 0x66CCFF)
    pixel(51, 0, 0x22CC66)
  end
end

return WeatherCurrent()
