# @name Weather Forecast
# @desc Three daily forecasts supplied by the VPS Open-Meteo provider.
# @version 1.0.0
# @display 52x16

class WeatherForecast
  var reading, started

  def init()
    self.reading = store.get("weather", nil)
    self.started = now_ms()
  end

  def on_show()
    self.started = now_ms()
  end

  def duration()
    return 9000
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
    text(0, 6, "FORECAST", 0xFFFFFF)
    text(0, 14, message, 0xFFAA22)
    pixel(51, 0, 0xFFAA22)
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

    var days = self.reading.find("days")
    if !isinstance(days, list) || size(days) == 0
      self.draw_state("NO DATA")
      return
    end

    var index = ((now_ms() - self.started) / 3000) % size(days)
    var day = days[index]
    if !isinstance(day, map)
      self.draw_state("NO DATA")
      return
    end

    var low = num(day.find("minimum_c"))
    var high = num(day.find("maximum_c"))
    var rain = num(day.find("precipitation_probability_percent"))
    var weekday = day.find("weekday")
    if low == nil || high == nil || rain == nil || type(weekday) != "string"
      self.draw_state("NO DATA")
      return
    end

    var shade = rain >= 50 ? 0x3399FF : 0xFFCC33
    circle_fill(3, 3, 2, shade)
    text(9, 6, format("%s %d%%", weekday, round(rain)), 0xFFFFFF)
    text(0, 14, format("%d/%dC", round(low), round(high)), 0x66CCFF)
    pixel(51, 0, 0x22CC66)
  end
end

return WeatherForecast()
