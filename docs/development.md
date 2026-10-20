# Local development

## Commands

Run these commands from the repository root:

| Command | Purpose |
| --- | --- |
| `make help` | List the main commands |
| `make check` | Check Go formatting, run vet and tests |
| `make list-apps` | Show apps selected by the config, or all apps if it is absent |
| `make build` | Build selected Go binaries into `bin/` |
| `make docker-build` | Build selected production Go images |
| `make docker-save` | Export selected images to `dist/<app>.tar.gz` |
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
AWTRIX_URL=http://localhost:8080 go run ./apps/server/cmd/clockctl --text 'Native Go works'
```

Use the Scripts tab to manually upload Berry apps from `apps/clock/`, inspect rendering, and check runtime errors. Go programs in `apps/server/` run outside AWTRIX and publish over its API. Automated script upload is not implemented yet.

## What dev-check verifies

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
- **Your new app is missing from the build:** run `make list-apps` and check `configs/apps.list`. An absent config selects all apps; an existing config selects only its listed names.
