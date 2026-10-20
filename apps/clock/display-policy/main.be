# @name Display Policy
# @desc Background display power schedule using the clock's local time.
# @version 1.0.0
# @headless true
# @config enabled bool "Enable schedule" default=false
# @config on text "Display on (HH:MM)" default="10:00" maxlen=5
# @config off text "Display off (HH:MM)" default="20:00" maxlen=5

class DisplayPolicy
  var enabled, on_minute, off_minute

  def init()
    self.enabled = store.get("enabled", false)
    self.on_minute = self.parse_time(store.get("on", "10:00"))
    self.off_minute = self.parse_time(store.get("off", "20:00"))
    if self.on_minute == self.off_minute
      raise "value_error", "on and off must be different"
    end
  end

  def parse_time(value)
    if type(value) != "string" || size(value) != 5 || value[2] != ":"
      raise "value_error", "schedule time must use HH:MM"
    end
    for i : 0 .. 4
      if i != 2 && (value[i] < "0" || value[i] > "9")
        raise "value_error", "schedule time must use HH:MM"
      end
    end
    var h = int(value[0..1])
    var m = int(value[3..4])
    if h > 23 || m > 59
      raise "value_error", "schedule time is outside 00:00-23:59"
    end
    return h * 60 + m
  end

  def in_window(now, start, finish)
    if start < finish
      return now >= start && now < finish
    end
    return now >= start || now < finish
  end

  def setup()
    self.reconcile()
  end

  def loop()
    self.reconcile()
  end

  def reconcile()
    shared.set("enabled", self.enabled)
    if !self.enabled
      shared.set("allowed", nil)
      return
    end
    var h = hour()
    var m = minute()
    var ready = h >= 0 && m >= 0
    shared.set("time_ready", ready)
    if !ready
      shared.set("allowed", nil)
      return
    end
    var wanted = self.in_window(h * 60 + m, self.on_minute, self.off_minute)
    shared.set("allowed", wanted)
    if display.is_on() != wanted
      display.power(wanted)
      log("schedule: display " + (wanted ? "on" : "off"))
    end
  end
end

return DisplayPolicy()
