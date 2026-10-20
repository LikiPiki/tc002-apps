# Architecture

## Runtime boundaries

```text
Computer / VPS                              TC002 / AWTRIX NG
apps/server/                                apps/clock/
  Go CLI and data services -- HTTP ------->    screens/    normal rotation
                                             services/   autonomous background work
                                             on-demand/  explicitly started sessions

Development: the same AWTRIX engine runs in Docker with a virtual 52 x 16 display.
Future MQTT: a transport for events and data; not a second owner of device policy.
```

Go programs run outside the clock, as native binaries or Docker containers. Berry source runs inside AWTRIX on the TC002. A simple pushed notification can be entirely server-side; custom local rendering uses Berry. A feature may have both components, with a documented data contract.

## Three Berry lifecycle kinds

| Kind | Loaded/running | Visible | Start / stop |
| --- | --- | --- | --- |
| Screen | While enabled | Takes turns in rotation | Normal app rotation |
| Service | While enabled, including with the display off | No page | AWTRIX app enable/disable |
| On-demand | Instance exists only while started | Owns the display for the session | Menu/API start; select-hold/API stop |

`screens/` must contain ordinary apps, `services/` requires `@headless true`, and `on-demand/` requires `@ondemand`. All app sources end with an instance, such as `return MyApp()`. Top-of-file metadata must match the repository category. Install names are unique across categories because AWTRIX has one script namespace. Moving an app preserves its install name.

Deployment of an on-demand app only compiles and stores it. Starting creates a fresh instance and runs initialization; stopping unloads it. Saved settings/data persist across sessions. Services must remain enabled to run: a disabled scheduler cannot wake the display in the morning.

## Scheduling and ownership

AWTRIX uses a shared Berry VM and heap. Apps have separate instances and logical error state, not separate operating-system threads or isolated VMs. The common script host executes hooks sequentially; `loop()` runs about once per second and timers/callbacks are dispatched by the same host. A slow hook affects the common frame budget. A failing hook retires that app's later hooks.

Use small, quick callbacks rather than private `while true` loops. An event-driven MQTT service may need only setup and callbacks, not a loop. A screen can fetch its own data if it is the sole consumer. Use a background provider plus `shared` when multiple screens need the same live data; check freshness when reading shared values.

Keep independent failure domains separate. Group related rules under the owner of a managed resource. `display-policy` owns scheduled display power; future brightness and manual-override rules should be coordinated there. A separate weather service should not be able to break the schedule by throwing an error. MQTT or Go can supply inputs to policy, rather than independently fighting it over power state.

## Configuration model

`configs/config.toml` contains shared defaults. Optional `config.local.toml` overrides only specified fields. Both native commands and containers use `internal/config` to read the same schema.

- `clock.deploy.screens/services/on_demand` select upload artifacts.
- `clock.screens.<name>`, `clock.services.<name>`, and `clock.on_demand.<name>` contain app settings.
- `server.apps` selects Go build/export artifacts; `server.worker` contains worker settings.

App selection is not app behavior. A deployed service can have a feature disabled in its own settings. An empty clock-category list means no uploads; `["*"]` discovers all apps in that category. Unselected remote apps are not removed or disabled automatically.

The catalog validates unique names, category/header agreement, declared setting names, and basic setting types before deployment. AWTRIX validates setting constraints, and the Berry app validates semantic rules such as time windows. `clockctl` does not contain an app-name-specific settings branch. Per-field personal overrides preserve shared app defaults.

Source updates use AWTRIX's atomic compare-and-swap endpoint. Settings are then patched through the script-config API as a separate operation, restarting the app. Code and settings are not one transaction; an error is reported and the batch stops. Editing settings in the web UI is supported, but the next deployment reapplies the configured TOML values.

## Commands and checks

- `clock-deploy`: install selected apps without starting on-demand sessions or forcing a screen switch.
- `clock-show`: show a rotating screen.
- `clock-start` / `clock-stop`: begin/end an on-demand session.
- `clock-check`: disposable local runtime checks; observes services without showing them, and starts/observes/stops on-demand sessions.

Inside the check sandbox, services are disabled after observation and display power is restored before visual checks. This prevents a night policy from masking drawing errors in later apps. Hardware-dependent behavior, delayed events, and actual appearance still need dedicated checks. Berry runtime checks remain local, not part of CI.

## Additional AWTRIX concepts

Berry modules share code but are not running apps. They are not supported by our app catalog yet: add a dependency-aware module installation phase when a real shared-code use case appears, rather than treating modules as services. Built-in clock screens and pushed JSON apps are native AWTRIX concepts, not `.be` files in this catalog. Native device settings (timezone, network, brightness) also remain distinct from individual app settings.

See [configuration](configuration.md), [commands](commands.md), and [contributing](../CONTRIBUTING.md).
