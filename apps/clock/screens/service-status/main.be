# @name Service Status
# @desc HTTP health from the VPS provider, with explicit data expiry.
# @version 1.0.0
# @display 52x16

class ServiceStatus
  var reading

  def init()
    self.reading = store.get("status", nil)
  end

  def duration()
    return 10000
  end

  def draw()
    clear()
    var label = "WAIT"
    var state = "NO DATA"
    var shade = 0xFFAA22

    if isinstance(self.reading, map) && self.reading.find("version") == 1
      var payload = self.reading
      var updated = num(payload.find("updated_at_ms"))
      var ttl = payload.find("ttl_seconds")
      var valid_time = updated != nil && type(ttl) == "int" && ttl > 0
      if type(payload.find("label")) == "string"
        label = payload["label"]
      end
      if valid_time && epoch_ms() >= 0
        var age = epoch_ms() - updated
        if age >= 0 && age < ttl * 1000
          if payload.find("state") == "up"
            state = "UP"
            shade = 0x22CC66
          elif payload.find("state") == "down"
            state = "DOWN"
            shade = 0xFF4444
          end
        else
          state = "STALE"
        end
      else
        state = "STALE"
      end
    end

    rect_fill(0, 0, 2, 2, shade)
    text(4, 6, label, 0xFFFFFF)
    text(4, 14, state, shade)
  end
end

return ServiceStatus()
