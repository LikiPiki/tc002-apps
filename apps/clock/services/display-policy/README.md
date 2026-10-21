# Display Policy

A headless Berry controller that switches the TC002 display on and off using the clock's local time. It has no display page and keeps running while the matrix is dark. No VPS, MQTT, external assets, or network requests are needed.

## Configuration

In your personal project TOML:

```toml
[clock.services.display-policy]
schedule_enabled = true
on = "10:00"
off = "20:00"
```

Shared defaults leave scheduling disabled. Times use strict `HH:MM`, with an inclusive on-time and exclusive off-time. Overnight windows such as `20:00` to `02:00` are supported. Equal times are rejected; disable scheduling instead of using an ambiguous full-day window.

```sh
make clock-deploy APP=display-policy
make clock-check APP=display-policy
```

Deployment uploads the script, then applies the TOML values through AWTRIX's script-settings API. Settings are persisted by AWTRIX and can also be edited in its Apps tab. A later repository deployment reapplies the TOML values. There is no source generation step.

## Behavior

- Reconciles during setup and approximately once per second in `loop()`.
- Uses the timezone configured in AWTRIX under System / Time, independent of the VPS timezone.
- Leaves the current display state unchanged while the clock reports unknown time (`hour()` or `minute()` is `-1`).
- Has priority over manual power changes: the next loop restores the scheduled state.
- Calls `display.power()` only when the current state differs from the desired state.
- Disabling the schedule stops managing power and leaves the current state unchanged.
- Switching off the display keeps AWTRIX, Wi-Fi, and scripts running. Deactivating this script stops the controller, including automatic morning wake-up.
- Notifications with `wakeup: true` may temporarily appear even while scheduled off. Avoid that flag for ordinary nighttime notifications.

The volatile `shared` values `display-policy.enabled`, `display-policy.time_ready`, and `display-policy.allowed` expose policy state. `allowed` is absent while scheduling is disabled or time is unknown. No state is written to flash on each loop.

## Local checks

The sandbox also executes the production policy class against a deterministic clock and display, checking daytime/overnight boundaries, unknown time, and disabled scheduling. `check.be` contains the behavioral scenarios.

`clock-check` observes the controller without showing it. It disables the controller after observation inside the disposable sandbox and turns the display on before checking later visual apps, so a nighttime schedule does not invalidate their checks. The physical clock is not changed by this command.
