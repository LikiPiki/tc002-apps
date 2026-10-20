# tc002-apps

Apps for Ulanzi TC002 running AWTRIX NG, with Go services for processing data and publishing it to the clock from a VPS.

## Current status

This repository is an initial toolkit, not a collection of finished clock apps:

- `clockctl` sends notifications, deploys Berry scripts, and selects the displayed app over HTTP.
- `worker` demonstrates periodic execution and graceful shutdown; it does not process or publish data yet.
- `apps/clock/` contains on-device Berry apps, each with its own README.
- A local virtual clock is available for development without TC002 hardware.

Physical clocks must run **AWTRIX NG**. The stock Ulanzi firmware and AWTRIX 3 are not supported.

## Project structure

```text
tc002-apps/
├── apps/
│   ├── README.md                  Which code runs where
│   ├── clock/                     ON THE TC002 — Berry scripts and assets
│   │   ├── README.md              Clock-app layout and installation
│   │   ├── <app>/                 Each on-device app follows this layout
│   │   │   ├── main.be            Berry source uploaded to AWTRIX NG
│   │   │   └── README.md          App-specific setup and usage
│   │   └── assets/icons/          Shared clock icons (placeholder)
│   └── server/                    ON A VPS / COMPUTER — Go programs
│       ├── README.md              Server-app layout and build instructions
│       ├── cmd/
│       │   ├── worker/main.go     Periodic service entry point (placeholder)
│       │   └── clockctl/          Notifications, Berry deployment, and app display control
│       └── internal/awtrix/
│           └── client.go          Shared AWTRIX NG HTTP client
├── configs/
│   ├── apps.list.example          Copy to apps.list to select Go apps to build
│   ├── clock.toml                 Shared Berry deployment defaults
│   └── clock.local.toml.example   Personal clock config template
├── deploy/                        PRODUCTION — run Go services on your VPS
│   ├── compose.yaml               Services, images, and runtime settings
│   ├── .env.example               Clock URL and worker interval template
│   └── deploy.sh                  Deploy selected services or loaded images
├── dev/                           LOCAL — virtual clock for development
│   ├── compose.yaml               AWTRIX + Go tools, isolated from production
│   └── Dockerfile.awtrix          Build the pinned headless AWTRIX engine
├── tools/
│   └── select-apps.sh             Read app selection; absent config = all Go apps
├── docs/
│   ├── setup.md                   Connect to a physical TC002
│   ├── development.md             Virtual clock, commands, troubleshooting
│   ├── commands.md                Go flags and environment variables
│   ├── deployment.md              VPS deployment, image transfer, updates
│   └── architecture.md            Runtime boundaries and data flow
├── .github/workflows/ci.yml        Go checks, builds, and production Docker builds
├── Dockerfile                     Build one Go command into an Alpine image
├── Makefile                       Main entry point for build and dev commands
├── go.mod                         Shared Go module for server-side code
├── .gitignore                     Keep local settings and build output out of Git
├── .dockerignore                  Keep local settings and output out of build context
├── CONTRIBUTING.md                How to add apps and submit changes
├── LICENSE                        MIT license for this repository
└── README.md                      Start here

Generated locally (not committed):
  bin/<app>                        Native Go binaries
  dist/<app>.tar.gz                Docker images exported for transfer to a VPS
  configs/apps.list               Your Go app selection (optional)
  configs/clock.local.toml        Your clock URL and Berry app selection (optional)
  deploy/.env                     Your deployment settings
```

### Where to start

| I want to... | Start here | First step |
| --- | --- | --- |
| Try it without a clock | [Local development](docs/development.md) | After cloning, run `make dev-check` and open http://localhost:8080 |
| Send something to my TC002 | [Clock setup](docs/setup.md), [Go commands](docs/commands.md) | Set `AWTRIX_URL` and run `clockctl` |
| Write an app that runs on the clock | [Clock apps](apps/clock/README.md) | Add `apps/clock/<name>/main.be` and upload it through Scripts |
| Process data on my VPS | [Server apps](apps/server/README.md) | Add `apps/server/cmd/<name>/main.go` |
| Build and deploy only selected Go apps | [VPS deployment](docs/deployment.md) | Create `configs/apps.list`, then run `make docker-save` |
| Contribute a change | [Contributing](CONTRIBUTING.md) | Follow the app layout and run the relevant checks |

### How the pieces connect

```text
VPS / computer                            TC002 running AWTRIX NG
┌────────────────────────┐                ┌──────────────────────────────┐
│ apps/server/ (Go)       │   HTTP / MQTT  │ Built-in screens / alerts    │
│ Fetch and process data ├───────────────>│ or apps/clock/ (Berry)       │
└────────────────────────┘                │ Custom on-device rendering   │
                                          └──────────────────────────────┘

Local development: dev/ runs headless AWTRIX in Docker instead of a TC002.
Production: deploy/ runs Go services on the VPS; Berry stays on the clock.
HTTP notifications and on-device Berry apps work today; MQTT integration is planned.
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
go run ./apps/server/cmd/worker --once
# Replace the address with your clock's reachable IP or hostname.
AWTRIX_URL=http://192.168.1.50 go run ./apps/server/cmd/clockctl --text 'Hello from Go'
```

The notification command needs a reachable AWTRIX NG instance without API authentication. See [clock setup](docs/setup.md) and the [command reference](docs/commands.md) for flags and environment variables.

### Select what to build

Select server-side Go apps in `configs/apps.list`, with one directory name from `apps/server/cmd/` per line. If the file is absent, all Go apps are built. Start with `cp configs/apps.list.example configs/apps.list`; the example selects only `worker`. This config does not select or install clock-side Berry apps.

```sh
make list-apps
make build
make docker-build
```

Run `make docker-save` to export the selected images to `dist/<app>.tar.gz` for transfer to a VPS. See the [deployment guide](docs/deployment.md).

This selection applies to `make build`, `make docker-build`, and `make docker-save`. It does not control which Compose services are started.

### Deploy Berry apps to the clock

```sh
cp configs/clock.local.toml.example configs/clock.local.toml
# Set your clock's URL in configs/clock.local.toml.
make clock-list
make clock-deploy          # All Berry apps, or the list selected in clock config
make clock-show APP=<name>
```

Use `make clock-deploy APP=<name>` to update just one app. The committed `configs/clock.toml` defaults to the local virtual clock; personal overrides are excluded from Git. To use the shared defaults without personal overrides, pass `CLOCK_LOCAL_CONFIG=` and keep `AWTRIX_URL` unset. See [Berry deployment](docs/commands.md#berry-deployment) for config precedence and error checks.

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
