# Architecture

## Runtime boundaries

```text
VPS applications                            TC002 / AWTRIX NG
apps/server/ -- data over HTTP ---------->   apps/clock/
Computer / one-off utility container           screens/    normal rotation
apps/tools/ -- management HTTP ---------->      services/   autonomous background work
                                               on-demand/  explicitly started sessions

Development: the same AWTRIX engine runs in Docker with a virtual 52 x 16 display.
Future MQTT: a transport for events and data; not a second owner of device policy.
```

VPS processes live in `apps/server`; CLI, build discovery, catalogs, and check scenarios live in `apps/tools`. Berry runs inside AWTRIX on the TC002. Each Go family has its own Dockerfile, which copies only that family's sources plus shared libraries. Production Compose contains server services only; utilities can be run separately on a computer or on demand on a VPS.

### Go dependency rules

```text
apps/server ──> internal/awtrix, internal/config
apps/tools  ──> internal/awtrix, internal/config
apps/server ─X─> apps/tools/internal
shared internal ─X─> application/tool internals
```

Root `internal/awtrix` implements API requests/responses, while `apps/tools/internal/clockcheck` owns sampling loops and assertions. Root `internal/config` parses/merges TOML; it does not discover source files. `apps/tools/internal/build` selects Go artifacts, and `clockapps` selects Berry apps. Go's `internal` import rules enforce the utility implementation boundary. Keep server-only business logic in `apps/server/internal` when needed.

`internal/config` enforces TOML syntax and project field types, but semantic validation belongs to consumers: build selection validates only selected Go artifacts, clock clients validate the effective URL after overrides, the Berry catalog validates selected app metadata/settings, and worker runtime validates its effective interval. Broken unused clock/runtime values do not block unrelated build operations.

The root Go module is shared, but runtime dependencies and source copies are separated. Build artifacts include their group (`server/worker`, `tools/clockctl`) so a utility and a server can even have the same short name without overwriting each other. Make is orchestration; root `scripts/`, `dev/`, and `deploy/` contain shell/infrastructure files, not application packages.

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
- `server.apps` selects VPS artifacts; `tools.apps` selects utility artifacts. `GROUP=server|tools|all` filters Go builds/exports. `server.worker` contains worker runtime settings.

App selection is not app behavior. A deployed service can have a feature disabled in its own settings. An empty clock-category list means no uploads; `["*"]` discovers all apps in that category. Unselected remote apps are not removed or disabled automatically.

The catalog validates unique names, category/header agreement, declared setting names, and basic setting types before deployment. AWTRIX validates setting constraints, and the Berry app validates semantic rules such as time windows. `clockctl` does not contain an app-name-specific settings branch. Per-field personal overrides preserve shared app defaults.

Source updates use AWTRIX's atomic compare-and-swap endpoint. Settings are then patched through the script-config API as a separate operation, restarting the app. Code and settings are not one transaction. `apps/tools/internal/clockdeploy` records confirmed per-app phases, stops the batch on failure, and reports completed, unconfirmed, failed, and unattempted work without claiming a batch rollback. Confirmed settings saves with a failed restart are distinguished from uncertain writes. Editing settings in the web UI is supported, but the next deployment reapplies the configured TOML values.

## Commands and checks

- `clock-deploy`: install selected apps without starting on-demand sessions or forcing a screen switch.
- `clock-show`: show a rotating screen.
- `clock-start` / `clock-stop`: begin/end an on-demand session.
- `clock-check`: disposable local runtime checks; observes services without showing them, and starts/observes/stops on-demand sessions.

Inside the check sandbox, services are disabled after observation and display power is restored before visual checks. This prevents a night policy from masking drawing errors in later apps. The internal runner requires the sandbox marker and fixed URL, then checks the endpoint's Linux board identity before writes. CI runs the same Berry runtime checks with separately cached AWTRIX and runner images. Hardware-dependent behavior, delayed events, and actual appearance still need dedicated checks.

## Additional AWTRIX concepts

Berry modules share code but are not running apps. They are not supported by our app catalog yet: add a dependency-aware module installation phase when a real shared-code use case appears, rather than treating modules as services. Built-in clock screens and pushed JSON apps are native AWTRIX concepts, not `.be` files in this catalog. Native device settings (timezone, network, brightness) also remain distinct from individual app settings.

## Integrations across runtimes

**Alpha / design-stage:** a shared integration ID is intended to group server producers, optional clock-side services, and visual consumers by feature without merging their source trees. Their relationship is an explicit data contract (channel, payload version, freshness, and failure behavior), not an assumed match of directory names. The current apps are autonomous and the worker is a placeholder; no pairing registry or `[integrations]` TOML schema is implemented yet. See [integration architecture](integrations.md) for the experimental convention and first-implementation requirements.

See [configuration](configuration.md), [commands](commands.md), and [contributing](../CONTRIBUTING.md).
