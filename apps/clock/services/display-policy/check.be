# Executed against the production class in the sandbox with substituted clock APIs.
class CheckState
  var h, m, power_on, values
  def init()
    self.h = 0
    self.m = 0
    self.power_on = false
    self.values = {}
  end
  def set(k, v) self.values[k] = v end
  def is_on() return self.power_on end
  def power(v) self.power_on = v end
end
var check_state = CheckState()
def check_hour() return check_state.h end
def check_minute() return check_state.m end

class PolicyCheck
  def expect(policy, h, m, expected)
    check_state.h = h
    check_state.m = m
    policy.reconcile()
    if check_state.power_on != expected
      raise "assertion_error", "display policy boundary failed"
    end
  end
  def init()
    var p = DisplayPolicy()
    p.enabled = true
    p.on_minute = 600
    p.off_minute = 1200
    self.expect(p, 9, 59, false)
    self.expect(p, 10, 0, true)
    self.expect(p, 19, 59, true)
    self.expect(p, 20, 0, false)
    p.on_minute = 1200
    p.off_minute = 120
    self.expect(p, 20, 0, true)
    self.expect(p, 23, 59, true)
    self.expect(p, 0, 0, true)
    self.expect(p, 1, 59, true)
    self.expect(p, 2, 0, false)
    check_state.power_on = true
    self.expect(p, -1, -1, true)
    p.enabled = false
    self.expect(p, 12, 0, true)
  end
end
return PolicyCheck()
