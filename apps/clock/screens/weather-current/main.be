# @name Weather Now
# @desc Today's weather: temperature, feels-like, wind, condition and range gauge.
# @version 2.0.2
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

  def sprite(x, y, rows, shade)
    for row : 0 .. size(rows) - 1
      for col : 0 .. size(rows[row]) - 1
        if rows[row][col] == "1"
          pixel(x + col, y + row, shade)
        end
      end
    end
  end

  def draw_icon(kind, daylight)
    if kind == "CLEAR" && daylight
      self.sprite(20, 5, ["0000001000000", "0010001000100", "0001111111000", "0011111111100", "1111111111111", "0011111111100", "0001111111000", "0010001000100"], 0xEFBB66)
      return
    end
    if kind == "CLEAR"
      circle_fill(26, 8, 3, 0x99BBFF)
      circle_fill(28, 7, 2, 0x000000)
      return
    end

    self.sprite(20, 5, ["0000111000000", "0001111100000", "0111111111000", "1111111111110", "1111111111111", "0111111111110"], 0xA3AFA9)
    if kind == "RAIN" || kind == "DRIZZLE"
      self.sprite(20, 11, ["0010010010000", "0100100100000"], 0x3E9CFA)
    elif kind == "SNOW"
      self.sprite(20, 11, ["0100100100000", "0010010010000"], 0xEEF1EA)
    elif kind == "STORM"
      line(27, 9, 25, 11, 0xEFBB66)
      line(25, 11, 28, 11, 0xEFBB66)
      pixel(27, 12, 0xEFBB66)
    elif kind == "FOG"
      line(21, 11, 31, 11, 0xA3AFA9)
      line(23, 12, 29, 12, 0xA3AFA9)
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
    var today = self.reading.find("today")
    if !isinstance(current, map) || !isinstance(today, map)
      self.draw_state("NO DATA")
      return
    end

    var temperature = num(current.find("temperature_c"))
    var feels = num(current.find("feels_like_c"))
    var wind = num(current.find("wind_speed_ms"))
    var low = num(today.find("minimum_c"))
    var high = num(today.find("maximum_c"))
    if temperature == nil || feels == nil || wind == nil || low == nil || high == nil
      self.draw_state("NO DATA")
      return
    end
    if temperature < -100 || temperature > 70 || feels < -120 || feels > 90 || wind < 0 || wind > 150 || low < -100 || high > 70 || low > high
      self.draw_state("NO DATA")
      return
    end

    var feeling = format("%d", round(feels))
    var breeze = format("%d", round(wind))
    var maximum = format("%d", round(high))
    var wind_x = 52 - (size(breeze) * 4 - 1) - 6
    var person_x = min(32, wind_x - 1 - (4 + size(feeling) * 4 - 1))
    text(0, 6, format("%d", round(temperature)), 0xEEF1EA)
    self.sprite(person_x, 0, ["010", "000", "111", "010", "101", "101"], 0xEFBB66)
    text(person_x + 4, 6, feeling, 0xEFBB66)
    self.sprite(wind_x, 0, ["00010", "00001", "11111", "00000", "11110", "00001", "00010"], 0x62C891)
    text(wind_x + 6, 6, breeze, 0x62C891)

    text(0, 15, format("%d", round(low)), 0x65B9EC)
    text(52 - (size(maximum) * 4 - 1), 15, maximum, 0xEFBB66)
    self.draw_icon(current.find("condition"), current.find("is_day"))
    for x : 17 .. 33
      var fraction = (x - 17) / 16.0
      pixel(x, 15, rgb(round(101 + 138 * fraction), round(185 + 2 * fraction), round(236 - 134 * fraction)))
    end
    var position = high == low ? 0.5 : max(0.0, min(1.0, (temperature - low) * 1.0 / (high - low)))
    var marker = 17 + round(16 * position)
    line(marker, 14, marker, 15, 0xEEF1EA)
    pixel(51, 0, 0x22CC66)
  end
end

return WeatherCurrent()
