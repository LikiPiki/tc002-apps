# Local development

Runtime checks save animation previews and runner output under `dist/clock-check/`.
Bundled animations declare nonempty/changing-frame expectations in `check.json`.
On failure, the sandbox engine log is saved too; CI uploads these diagnostics.

Animation `check.json` files also contain raw-frame SHA-256 goldens at fixed `now_ms()` values. Checks install a sandbox-only copy with frozen time, leaving production sources untouched. On an intentional visual change, inspect the generated `*-snapshots.gif` and update the corresponding hashes from `*-snapshots.json`. Artifacts live in a separate subdirectory per invocation so concurrent checks do not overwrite diagnostics.

Use `make sprites` after editing app-owned `sprite.png` files, `make sprites-check` to verify generated Berry, and `make previews` after runtime checks to regenerate the README gallery.

Use `make clock-check CHECK_ALL=true LOCAL_CONFIG=` to exercise all catalog apps regardless of deployment selections, matching CI.

## Commands

Run these commands from the repository root:

| Command | Purpose |
| --- | --- |
| `make help` | List the main commands |
| `make check` | Check Go formatting, run vet and tests |
| `make list-apps [GROUP=server\|tools\|all]` | Show qualified Go artifact selection |
| `make build [GROUP=...]` | Build selected binaries into `bin/<group>/` |
| `make docker-build [GROUP=...]` | Build server/tool images from separate Dockerfiles |
| `make docker-save [GROUP=...]` | Export selected images to `dist/<group>/<app>.tar.gz` |
| `make clock-list` | Show the Berry deployment target and selected apps |
| `make clock-deploy [APP=<name>]` | Upload selected Berry scripts to the configured target |
| `make clock-show APP=<name>` | Show an installed script and check its runtime status |
| `make clock-start APP=<name>` | Start an installed on-demand app |
| `make clock-stop APP=<name>` | End the current on-demand session |
| `make clock-check [APP=<name>]` | Execute Berry apps in disposable local AWTRIX and check frames/errors |
| `make dev` | Build and start virtual AWTRIX |
| `make dev-build` | Build all development images without starting them |
| `make dev-notify TEXT="Hello!"` | Send a notification from a Go container |
| `make dev-check` | Start AWTRIX, run the worker, and perform an HTTP smoke check |
| `make dev-logs` | Follow AWTRIX logs |
| `make dev-down` | Stop the environment without deleting data |

Go checks require Go 1.24+. The containerized development environment only requires Docker with Compose v2 supporting `up --wait`.

Make commands also require Make and a POSIX shell; use Linux, macOS, or WSL. `make docker-save` requires `gzip`. See the [command reference](commands.md) for Go flags and environment variables, and [CONTRIBUTING.md](../CONTRIBUTING.md) for adding apps.

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

Upload Berry apps from `apps/clock/` with `make clock-deploy LOCAL_CONFIG=` and show one with `make clock-show APP=mackerel LOCAL_CONFIG=`. These commands use the shared local URL without personal overrides; leave `AWTRIX_URL` unset. You can also use the Scripts tab. Dev containers ignore personal overrides and use the shared project TOML; their virtual-clock network address is set by Compose. See [configuration](configuration.md) and [Berry deployment](commands.md#berry-deployment).

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

CI runs the same check in a dedicated `berry` job. Buildx caches the pinned AWTRIX build and the Go runner separately using GitHub Actions cache. The job loads both images, then runs `make clock-check CHECK_BUILD=0 LOCAL_CONFIG=`. Normal local runs build images by default; `CHECK_BUILD=0` is for prebuilt images. This checks runtime errors and frames, not animation quality or hardware behavior.

### Go integration smoke check

- The actual headless AWTRIX engine starts and responds to HTTP requests.
- The worker builds and completes one run; it is currently a placeholder.
- The Go client sends a notification to the actual HTTP API. A request error or a non-2xx response fails the check.

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
- **Your new app is missing from the build:** run `make list-apps` and check `server.apps` or `tools.apps` plus the `GROUP` filter. Use `["*"]` to select all commands in that family. Binaries are under `bin/server/` or `bin/tools/`, not the old flat output directory.
