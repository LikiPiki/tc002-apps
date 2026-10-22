# Local development

Runtime checks save animation previews and runner output under `dist/clock-check/`.
Visual apps use the same runtime and frame-validity checks, with no per-app test configuration. Static and black frames are valid; inspect the GIF to evaluate the picture and animation.
On failure, the sandbox engine log is saved too; CI uploads these diagnostics.

Artifacts live in a separate subdirectory per invocation so concurrent checks do not overwrite diagnostics. GIFs are captured during the runtime observation window, including available frames before a failure.

Use `make sprites` after editing app-owned `sprite.png` files, `make sprites-check` to verify generated Berry, and `make previews` after runtime checks to regenerate the README gallery.

Use `make clock-check CHECK_ALL=true LOCAL_CONFIG=` to exercise all catalog apps regardless of deployment selections, matching CI.

## Commands

Run these commands from the repository root:

Run `make init-config` once after cloning. Existing working configs are preserved. Deterministic checks instead use `CONFIG=testdata/ci/config.toml LOCAL_CONFIG=`.

| Command | Purpose |
| --- | --- |
| `make help` | List the main commands |
| `make init-config` | Create a local config once without overwriting existing settings |
| `make check` | Check Go formatting, run vet and tests |
| `make list-tools` | Show selected utility commands |
| `make build` | Build the complete server |
| `make build-server` | Build `bin/server/tc002-server` with all server features |
| `make docker-build-server` | Build the complete `tc002-apps/server:local` image |
| `make docker-save-server` | Export `dist/server/tc002-server.tar.gz` |
| `make clock-list` | Show the Berry deployment target and selected apps |
| `make clock-deploy [APP=<name>]` | Upload selected Berry scripts to the configured target |
| `make clock-show APP=<name>` | Show an installed script and check its runtime status |
| `make clock-start APP=<name>` | Start an installed on-demand app |
| `make clock-stop APP=<name>` | End the current on-demand session |
| `make clock-check [APP=<name>]` | Execute Berry apps in disposable local AWTRIX and check frames/errors |
| `make dev` | Build and start virtual AWTRIX |
| `make dev-build` | Build all development images without starting them |
| `make dev-server` | Start the complete server against virtual AWTRIX |
| `make dev-notify TEXT="Hello!"` | Send a notification from a Go container |
| `make dev-check` | Start AWTRIX and perform an HTTP notification smoke check |
| `make dev-logs` | Follow AWTRIX logs |
| `make dev-down` | Stop the environment without deleting data |
| `make lan` | Build/start the complete server for physical clocks |
| `make lan-logs` | Follow server logs |
| `make lan-once` | Run one complete update from a disposable container |
| `make lan-restart` | Restart the server after config changes |
| `make lan-doctor` | Diagnose the physical clock from the container network |
| `make lan-down` | Stop the local LAN server project |

Go checks require Go 1.24+. The containerized development environment only requires Docker with Compose v2 supporting `up --wait`.

Make commands also require Make and a POSIX shell; use Linux, macOS, or WSL. `make docker-save` requires `gzip`. See the [command reference](commands.md) for Go flags and environment variables, and [CONTRIBUTING.md](../CONTRIBUTING.md) for adding apps.

## Local server with physical clocks

Use this mode when your computer and TC002 share a reachable LAN and you want to debug against the real display. `dev/compose.lan.yaml` runs the complete production server with a local image tag and `restart: "no"`. All server features are included in the one process/container.

The containers read `configs/config.toml` and your `configs/config.local.toml`, including the real clock address and weather location. For example:

```toml
[clock]
urls = ["192.168.1.50", "192.168.1.51"]

[server.weather]
latitude = 59.9386
longitude = 30.3141
label = "SPB"
timezone = "Europe/Moscow"
```

For different screen sets, replace `urls` with `profiles = ["clocks/main.local.toml", "clocks/second.local.toml", ...]`. The same `make lan` process reads all referenced profiles from the mounted config directory; weather remains common while delivery follows each profile's selected screens. Profile paths are resolved relative to the config that lists them. See [clock profiles](configuration.md#different-clock-configurations-from-the-main-config).

Install the weather screens once, then start the local service:

```sh
make clock-deploy APP=weather-current
make clock-deploy APP=weather-forecast
make lan-doctor
make lan
make lan-logs
```

The weather provider sends a reading immediately and repeats at `server.weather.interval`. `make lan` starts it in the background, so stopping the log command with Ctrl-C leaves the service running. Use `make lan-down` to stop it. No inbound port needs publishing: the provider calls Open-Meteo and the clock's HTTP API through normal outbound Docker networking.

For a fast edit/test loop:

```sh
make lan-once                         # Publish now instead of waiting for the interval
make lan                             # Rebuild/recreate after changing Go code
make lan-restart                     # Reload changes to the mounted TOML files
make clock-deploy APP=weather-current # Update Berry independently
make clock-show APP=weather-current
```

`AWTRIX_URL=http://<clock-address> make lan` selects one clock instead of the configured list. Apply the same override to `lan-doctor` and `lan-once`. Changes to container environment variables need `make lan` to recreate the container; `lan-restart` reloads only mounted config. `CONFIG` and `LOCAL_CONFIG` may select other files under `configs/`, which is mounted read-only into `/app/configs`. Leave `AWTRIX_URL` unset to use `[clock] urls`. Unreachable/non-AWTRIX addresses are warned about and skipped independently on every cycle.

The `tc002-lan` project has its own containers/network and local image tags. `make lan-down` removes this project's containers and network.

For IDE breakpoints or native Go debugging, run `go run ./apps/server/cmd/server` directly from the checkout. It uses the same personal config and physical-clock targets. Stop the containerized copy with `make lan-down` first so only one server publishes data.

### LAN troubleshooting

- **Cannot reach the clock:** run `make lan-doctor`. If the host can reach its IP but the container cannot, check Docker/VM routing and the host VPN. Use the LAN address of the clock rather than `localhost`.
- **Weather does not change after an edit:** use `make lan-once` for an immediate request, `make lan` after Go changes, or `make lan-restart` after config changes.
- **Clock reports a missing app:** install both weather screens or select the installed consumers in `server.weather.screens`.
- **Data arrives but the display is dark:** the headless display-policy service still controls power according to the configured schedule.

## Virtual clock

```sh
make dev-check
```

The first build downloads a pinned revision of AWTRIX NG and its dependencies, then compiles the C++ engine. This takes significantly longer than building the Go commands. Subsequent runs use the Docker cache.

Open **http://localhost:8080**. The live preview shows a 52 x 16 display matching the TC002. The smoke-check notification may have disappeared by the time you open the page; send another one:

```sh
make dev-notify TEXT="Hello TC002"
```

You can also run the Go client directly on your computer:

```sh
AWTRIX_URL=http://localhost:8080 go run ./apps/tools/cmd/clockctl --text 'Native Go works'
```

For a template-only virtual preview, use `make clock-deploy CONFIG=examples/configs/config.toml LOCAL_CONFIG=` and the equivalent `clock-show`. These explicitly select host-side setup defaults; leave `AWTRIX_URL` unset. Application containers receive only the mounted runtime config directory. See [configuration](configuration.md).

## Runtime checks

### Berry runtime smoke check

```sh
make clock-check
make clock-check APP=mackerel CHECK_DURATION=3s
```

This Docker-only command starts a separate, temporary AWTRIX with a 52 x 16 display, no published ports, and no persistent app data. It uses `clock.deploy` selections (or `APP`), with optional `KIND=screens|services|on-demand`, but always targets that sandbox, ignoring the physical-clock URL and `AWTRIX_URL` environment variable. Config paths must be inside the mounted repository.

Each selected script is installed, shown, and sampled every 100ms for two seconds by default. The command checks script errors, the active app, and valid pixel frames. Static or black output is valid; this is not an image comparison or an animation-quality test. Apps must stay visible for the observation window. Delayed callbacks and hardware-dependent behavior still need separate checks.

Headless scripts are observed for errors without switching the active app or requiring pixel frames. Their observation window must be at least 1500ms so `loop()` can execute. After observation, each headless script is disabled inside the sandbox, and the display is turned on before checking subsequent visual apps. This prevents power controllers from interfering with drawing checks.

On-demand apps are installed dormant, started explicitly for the check, observed as visual apps, then stopped so their instance is unloaded. The repository category must match the Berry header; duplicate names, unknown configured apps/settings, and incorrect setting types fail before upload. The on-demand directory initially has no bundled tools; an empty selection is valid.

Temporary containers, networks, and data are removed on success or failure. The regular virtual-clock instance and physical TC002 are not modified by the command. Image downloads/build caches remain.

The CLI `check` subcommand is an internal runner: it requires the Compose sandbox marker, accepts only the fixed `http://awtrix-check:8080` endpoint, and verifies `boardType=linux` before any script writes. A direct `clockctl check --url <device>` invocation is rejected before contacting the device. Use `make clock-check`, not a custom check URL.

CI runs the same check using the dedicated fixture `CONFIG=testdata/ci/config.toml LOCAL_CONFIG=`. The temporary check runner reads it through the repository mount. Buildx caches AWTRIX and runner images separately; fixtures and setup templates are not included in application images. Normal local runs build images by default; `CHECK_BUILD=0` is for prebuilt images.

### Go integration smoke check

- The actual headless AWTRIX engine starts and responds to HTTP requests.
- The Go client sends a notification to the actual HTTP API. A request error or a non-2xx response fails the check.

`make server-check` separately verifies server data processing and the real Berry consumers with a local fixture, without requiring external weather access or personal coordinates.

This is an integration smoke check, not an automated image comparison. Inspect the rendered output in the browser. Add data-processing unit tests and Berry-specific scenarios as individual apps are implemented.

Sensors, physical buttons, Bluetooth, audio output, and TC002 performance are not emulated.

## Data and networking

Settings, Berry scripts, and assets are stored in the `tc002-dev_awtrix-data` Docker volume. `make dev-down` preserves it. To completely reset the local clock, including its scripts and settings:

```sh
docker compose -p tc002-dev -f dev/compose.yaml down -v
```

The container uses `--lan` so other services can reach its API. The host port is published only on `127.0.0.1`. This environment is separate from production Compose and does not connect to a physical clock or VPS.

Keep authentication disabled in this development instance: the current `clockctl` expects an unauthenticated API. If you enable it accidentally, disable it in the web UI or reset the development volume.

## AWTRIX version

The upstream commit is pinned through `AWTRIX_REV` in `dev/Dockerfile.awtrix`. When updating, change the SHA and rerun `make dev-check`. Before uploading apps to a device, check compatibility with the AWTRIX version installed on the clock.

## Troubleshooting

- **Port 8080 is already in use:** stop the conflicting process or change the host side of the port mapping in `dev/compose.yaml`. With a different host port, use that port in your browser and native client URL. Container-to-container traffic still uses port 8080.
- **The first build is slow:** it compiles the C++ engine and downloads dependencies. Check Docker's build output; cached rebuilds are much faster.
- **AWTRIX fails its health check:** run `make dev-logs` to inspect startup errors. The service must become healthy before `make dev-check` sends a notification.
- **A notification is not visible:** open the web UI before running `make dev-notify` again; notifications are temporary.
- **A utility is missing from the build:** run `make list-tools` and check `tools.apps`. The server is always built as one complete `bin/server/tc002-server` binary; add server features to its existing entry point.
