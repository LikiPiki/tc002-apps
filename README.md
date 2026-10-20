# tc002-apps

Apps for Ulanzi TC002 running AWTRIX NG, with Go services for processing data and publishing it to the clock from a VPS.

## Current status

This repository is an initial toolkit, not a collection of finished clock apps:

- `clockctl` sends notifications to AWTRIX NG over HTTP.
- `worker` demonstrates periodic execution and graceful shutdown; it does not process or publish data yet.
- `apps/` is ready for Berry apps but does not contain any yet.
- A local virtual clock is available for development without TC002 hardware.

Physical clocks must run **AWTRIX NG**. The stock Ulanzi firmware and AWTRIX 3 are not supported.

## Project structure

```text
apps/                Berry apps running on the clock
assets/icons/        App icons
cmd/worker/          Placeholder periodic worker for a VPS
cmd/clockctl/        CLI for sending notifications to the clock
internal/awtrix/     AWTRIX NG HTTP client
configs/             Example app-selection config
deploy/              Docker Compose, example environment, deployment script
dev/                 Containerized virtual AWTRIX
tools/               Build-selection helper
docs/                Architecture, setup, development, and deployment guides
```

## Quick start

### Requirements

- Git to clone the repository.
- Make and a POSIX shell for the Make commands (Linux, macOS, or WSL).
- Go 1.24+ for native builds and checks.
- Docker with Compose v2 for containers; local development uses `up --wait`.
- `gzip` for `make docker-save`.

Go is not required on the host for Docker-only workflows. An internet connection is needed for the initial image builds and dependency downloads.

Clone the repository first:

```sh
git clone https://github.com/LikiPiki/tc002-apps.git
cd tc002-apps
```

### Native Go

```sh
make check
make build
go run ./cmd/worker --once
# Replace the address with your clock's reachable IP or hostname.
AWTRIX_URL=http://192.168.1.50 go run ./cmd/clockctl --text 'Hello from Go'
```

The notification command needs a reachable AWTRIX NG instance without API authentication. See [clock setup](docs/setup.md) and the [command reference](docs/commands.md) for flags and environment variables.

### Select what to build

Select Go apps in `configs/apps.list`, with one directory name from `cmd/` per line. If the file is absent, all apps are built. Start with `cp configs/apps.list.example configs/apps.list`; the example selects only `worker`.

```sh
make list-apps
make build
make docker-build
```

Run `make docker-save` to export the selected images to `dist/<app>.tar.gz` for transfer to a VPS. See the [deployment guide](docs/deployment.md).

This selection applies to `make build`, `make docker-build`, and `make docker-save`. It does not control which Compose services are started.

## Docker

```sh
cp deploy/.env.example deploy/.env
# Edit deploy/.env before sending notifications to a physical clock.
sh deploy/deploy.sh worker
docker compose --env-file deploy/.env -f deploy/compose.yaml logs -f worker
```

The worker is a placeholder and logs a tick at the configured interval. To stop the deployment, run `docker compose --env-file deploy/.env -f deploy/compose.yaml down`. For image-only deployment and platform selection, see the [VPS deployment guide](docs/deployment.md).

## Local testing without a clock

```sh
make dev-check
# Open http://localhost:8080 for a virtual TC002 with a 52 x 16 display.
make dev-notify TEXT="Hello TC002"
# When finished:
make dev-down
```

The development environment builds the actual headless AWTRIX NG engine in an Alpine container. See the [local development guide](docs/development.md) for details and limitations. Run `make help` to list the main commands.

Further reading: [VPS deployment](docs/deployment.md), [clock setup](docs/setup.md), and [architecture](docs/architecture.md).

## Contributing

New apps, integrations, fixes, and documentation improvements are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for the app layout, checks, and pull request workflow. Keep documentation, code comments, and examples in English.

## License

The code in this repository is [MIT licensed](LICENSE). AWTRIX NG is a separate dependency with its own [PolyForm Noncommercial license](https://github.com/Blueforcer/awtrix-ng/blob/main/LICENSE.md); the local development image builds and includes that project.
