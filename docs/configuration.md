# Project configuration

The [weather integration](integrations/weather.md) uses `[server.weather]`: required `latitude`/`longitude`, plus `label`, `timezone`, `screens`, `interval`, `timeout`, `ttl` and `attempts`. Coordinates have no implicit default; zero is valid. Fields merge individually, while `screens` replaces the selected consumers. `weather-provider --print` fetches one reading for terminal inspection without a clock connection.

All project settings use one TOML schema and one Go loader:

```text
configs/config.toml                Shared defaults, committed
configs/config.local.toml.example  Personal override template, committed
configs/config.local.toml          Optional personal overrides, ignored by Git
```

## One server, multiple clocks

Set `[clock] urls` to devices that share one screen configuration. For different screen sets, use the file-based profiles below. A computer and a VPS may use different personal config files and therefore different sets:

```toml
[clock]
urls = ["192.168.1.50", "192.168.1.51", "http://clock.example.net:8080"]
```

Bare IPs/hostnames use HTTP. Full HTTP(S) URLs support ports. The list replaces the shared selection; duplicate addresses are contacted once. Legacy `clock.url` remains supported as a single-device form. Set only one target form per file (`url`, `urls` or `profiles`); personal settings replace the shared form.

Deployment, notifications, show/start/stop, doctor/status and the VPS providers operate on all listed clocks. They verify the AWTRIX device/capabilities API before writing. Unknown endpoints and unreachable addresses are skipped with a warning identifying the target. Device operations run independently, with at most four concurrent targets; one failure does not prevent the others from updating. All targets failing (or an empty list) produces a nonzero exit status, while partial delivery succeeds with warnings.

The weather provider fetches one reading per cycle and publishes it to the configured screens on every reachable clock. Skipped clocks are checked again next cycle. Server delivery has a 10-second budget per clock and identification is limited to three seconds.

Explicit `--url` or `AWTRIX_URL` selects **one clock**, replacing the entire configured set. A matching configured clock retains its profile's screen settings; an unlisted address uses shared clock settings. This also keeps the virtual-clock environment confined to its sandbox target. Config parsing stays offline; `list`/`validate` show selections without probing the devices.

## Different clock configurations from the main config

The main/personal config lists any number of files:

```toml
[clock]
profiles = [
  "clocks/main.local.toml",
  "clocks/second.local.toml",
  "clocks/office.local.toml",
]
```

References are relative to the file that declares them, so `configs/config.local.toml` resolves these under `configs/clocks/`. Absolute references also work. The list is loaded dynamically; there is no fixed number of profiles or clock addresses in the program. The four-target limit controls concurrent HTTP operations, not the size of the configuration.

Each file contains **only `[clock]` settings**. For example, `main.local.toml`:

```toml
[clock]
urls = ["192.168.1.50"]

[clock.deploy]
screens = ["weather-current", "weather-forecast", "cigar"]
services = []
on_demand = []
```

And `second.local.toml`:

```toml
[clock]
urls = ["192.168.1.51"]

[clock.deploy]
screens = ["weather-current", "seagull"]
services = []
on_demand = []
```

Copy the provided `configs/clocks/*.toml.example` templates to `*.local.toml`. Local profiles are excluded from Git and image build context, but are available through the mounted `configs/` directory. A profile can contain several addresses when those clocks share its settings.

Clock settings inherit shared + personal defaults, then the individual profile overrides them. This includes deployment selections and Berry settings such as `[clock.services.display-policy]`. The profile name in reports comes from its filename (`main.local.toml` → `main`). All selected source files are validated before any clock writes. Unknown fields, missing files, duplicate names, nested profiles and an address assigned to different profiles fail before deployment.

VPS configuration stays in the main config. Profiles cannot override `[server]` or `[tools]`. Weather is fetched once per cycle and remains common to all clocks; each profile receives data only for its selected weather screens. A current-only profile does not get writes to `weather-forecast`. A profile with no consumers for a provider is left out of that provider's audience.

Use `make clock-list` to preview the profile → address → screen mapping, then `make clock-deploy` to apply all profiles. `APP=...` explicitly overrides the per-profile app selection. To operate on one existing device while retaining its profile settings, pass its address with `--url`/`AWTRIX_URL`. Profiles choose what to upload; existing apps omitted from a new selection are not automatically deleted or disabled.

After changing profile files or adding references, restart long-running providers (`make lan-restart` locally, or restart the VPS Compose service) so they reload the set. Deployment commands read the current files on each invocation.

The shared file is required. Personal settings may be absent. The loader validates TOML syntax, known project fields, and field types, then merges overrides. Semantic checks run only in the relevant consumer: a malformed unused clock URL or worker interval does not stop build selection. The effective clock URL is validated after URL overrides; the effective interval is validated by the worker after an explicit interval override.

## Shared layout

```toml
[clock]
urls = ["http://localhost:8080"]

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

[tools]
apps = ["*"]
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

The Berry catalog validates declared setting names/basic types for selected apps, then the CLI patches settings through AWTRIX's API after uploading source. Unused app settings are not semantically validated by unrelated Go build/runtime consumers. AWTRIX and the app validate constraints and semantics. Invalid settings can leave the installed app reporting an error; source and settings are separate updates, with per-app phase results reported by deployment. Settings edited through the web UI are overwritten by configured TOML values on the next deployment. See [Display Policy](../apps/clock/services/display-policy/README.md).

`server.apps` selects VPS commands under `apps/server/cmd/`. `tools.apps` selects utilities under `apps/tools/cmd/`. Omission or `["*"]` selects all in that group. An empty server list remains an error; an empty tools list skips utility artifacts. This does not select running Compose services. `server.worker.interval` sets a positive Go duration, such as `30s` or `5m`.

Go Make commands accept `GROUP=all|server|tools` (default `all`), separate from the Berry `KIND` filter. `make list-apps` prints qualified targets such as `server/worker` and `tools/clockctl`. Binaries and image archives retain that group under `bin/` and `dist/`; image tags use `tc002-apps/<group>/<name>:local`.

## Personal overrides

```sh
cp configs/config.local.toml.example configs/config.local.toml
```

Override only the values you need:

```toml
[clock]
urls = ["192.168.1.50", "192.168.1.51"]

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
make docker-save GROUP=server
make build GROUP=tools
```

Make accepts `CONFIG` and `LOCAL_CONFIG`. Native Go commands accept `--config` and `--local-config`; environment variables `TC002_CONFIG` and `TC002_LOCAL_CONFIG` change their default paths. An explicitly empty personal path disables overrides. `APP` overrides the app selection; `KIND` filters or validates its category.

Clock target precedence is explicit single `--url` > single `AWTRIX_URL` > personal TOML list/single URL > shared TOML. An explicit worker `--interval` overrides TOML. Leave `AWTRIX_URL` unset to use the configured set.

## Docker and checks

Production Compose contains VPS services only and mounts `configs/` read-only. Images include only shared defaults, not personal settings. Server and tool images use separate Dockerfiles and source copies. No `.env` file is needed. Dev clockctl/worker containers ignore personal overrides; the weather provider reads personal location settings and overrides only the clock URL with `http://awtrix:8080`. The host browser uses http://localhost:8080.

`clock-check` always uses a disposable local target, but respects the configured category selections and app settings. Its config files must be inside the mounted repository. Native commands can read external profiles. Docker-only builds use a containerized config reader if host Go is absent.

For runtime config changes, restart the Go service. For Berry config changes, deploy the relevant app again. Deployment does not remove apps excluded by a new selection; deactivate or delete them explicitly on AWTRIX.

## Migration from the flat catalog

Move `clock.apps` into the relevant `clock.deploy` lists. Move `[clock.schedule]` into `[clock.services.display-policy]` and rename `enabled` to `schedule_enabled`. Old fields are rejected rather than silently ignored. Existing AWTRIX install names remain `cigar`, `seagull`, `mackerel`, and `display-policy`.

If an older config includes `clockctl` in `server.apps`, move it to `tools.apps`. Server selection now contains only actual VPS commands. Old flat `bin/<name>` and `dist/<name>.tar.gz` outputs and flat Docker image tags are no longer produced; existing local artifacts are not removed automatically.
