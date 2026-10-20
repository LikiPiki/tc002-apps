# Project configuration

All project settings use one TOML schema and one Go loader:

```text
configs/config.toml                Shared defaults, committed
configs/config.local.toml.example  Personal override template, committed
configs/config.local.toml          Optional personal overrides, ignored by Git
```

The shared file is required. Personal settings may be absent. Unknown project fields, malformed TOML, invalid durations, and incorrect selection types fail with an error.

## Shared layout

```toml
[clock]
url = "http://localhost:8080"

[clock.deploy]
screens = ["*"]
services = ["display-policy"]
on_demand = ["*"]

[clock.services.display-policy]
schedule_enabled = false
on = "10:00"
off = "20:00"

[server]
apps = ["*"]

[server.worker]
interval = "1m"
```

## Selection versus settings

`clock.deploy` selects **what to upload** from `apps/clock/screens/`, `services/`, and `on-demand/`. Each list accepts names, `["*"]` for all discovered apps, or `[]` for none. If no config specifies a category, screens default to all, services and on-demand default to none. Shared defaults explicitly select all current screens and the display service. The on-demand category is initially empty, so selecting all is a valid no-op.

`clock.screens.<name>`, `clock.services.<name>`, and `clock.on_demand.<name>` define **how an app behaves**. Keys must match that app's `@config` declarations. Settings are scalar strings, booleans, or numbers. For example:

```toml
[clock.services.display-policy]
schedule_enabled = true
on = "10:00"
off = "20:00"
```

The CLI validates declared names/basic types, then patches settings through AWTRIX's API after uploading source. AWTRIX and the app validate constraints and semantics. Invalid settings can leave the installed app reporting an error; source and settings are separate updates. Settings edited through the web UI are overwritten by configured TOML values on the next deployment. See [Display Policy](../apps/clock/services/display-policy/README.md).

`server.apps` selects Go commands under `apps/server/cmd/` for Make build/export commands. Omission or `["*"]` selects all; an empty server list remains an error. This does not select running Compose services. `server.worker.interval` sets a positive Go duration, such as `30s` or `5m`.

## Personal overrides

```sh
cp configs/config.local.toml.example configs/config.local.toml
```

Override only the values you need:

```toml
[clock]
url = "http://192.168.1.50"

[clock.deploy]
screens = ["cigar", "mackerel"]
services = ["display-policy"]
on_demand = []

[clock.services.display-policy]
schedule_enabled = true

[server]
apps = ["worker"]
```

Selections replace rather than append. App settings merge by field: changing `schedule_enabled` preserves the shared `on`/`off` values. The personal file is excluded from Git and Docker build context.

## Commands and paths

```sh
make clock-list
make clock-list KIND=services
make clock-deploy APP=mackerel
make clock-deploy KIND=on-demand
make clock-deploy LOCAL_CONFIG=  # Shared local defaults only
make build LOCAL_CONFIG="$HOME/.config/tc002/vps.toml"
```

Make accepts `CONFIG` and `LOCAL_CONFIG`. Native Go commands accept `--config` and `--local-config`; environment variables `TC002_CONFIG` and `TC002_LOCAL_CONFIG` change their default paths. An explicitly empty personal path disables overrides. `APP` overrides the app selection; `KIND` filters or validates its category.

Clock URL precedence is explicit `--url` > `AWTRIX_URL` > personal TOML > shared TOML. An explicit worker `--interval` overrides TOML. Leave `AWTRIX_URL` unset when relying on config URLs.

## Docker and checks

Production Compose mounts `configs/` read-only. Images include only shared defaults, not personal settings. No `.env` file is needed. Dev containers ignore personal overrides and use the internal virtual-clock address `http://awtrix:8080`; the host browser uses http://localhost:8080.

`clock-check` always uses a disposable local target, but respects the configured category selections and app settings. Its config files must be inside the mounted repository. Native commands can read external profiles. Docker-only builds use a containerized config reader if host Go is absent.

For runtime config changes, restart the Go service. For Berry config changes, deploy the relevant app again. Deployment does not remove apps excluded by a new selection; deactivate or delete them explicitly on AWTRIX.

## Migration from the flat catalog

Move `clock.apps` into the relevant `clock.deploy` lists. Move `[clock.schedule]` into `[clock.services.display-policy]` and rename `enabled` to `schedule_enabled`. Old fields are rejected rather than silently ignored. Existing AWTRIX install names remain `cigar`, `seagull`, `mackerel`, and `display-policy`.
