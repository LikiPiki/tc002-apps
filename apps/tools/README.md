# Utility applications

Go utilities for development, deployment, and device management. They run on a computer or on demand in Docker, independently of the VPS service lifecycle.

```text
cmd/clockctl/          Notification, Berry lifecycle, and build-selection CLI
internal/build/       Select and validate Go build artifacts
internal/clockapps/   Discover Berry apps and validate lifecycle/config metadata
internal/clockcheck/  Observe script execution and pixel frames in runtime checks
internal/clockdeploy/ Report sequential source/settings/verification deployment phases
Dockerfile            Tool-only build: apps/tools + shared internal libraries
```

Device API calls are provided by the root `internal/awtrix` package. Sampling loops and assertions belong to `clockcheck`, not to the HTTP client. TOML parsing lives in root `internal/config`; source-directory discovery belongs to the utility build/catalog packages.

## Build and run

From the repository root:

```sh
make list-tools
make build-tools
make docker-build-tools
make docker-save-tools
go run ./apps/tools/cmd/clockctl --help
```

`tools.apps` selects utility artifacts. Omitted selection or `["*"]` builds all utilities; `[]` skips their build/export artifacts. Outputs are `bin/tools/<name>`, image `tc002-apps/tools/<name>:local`, and `dist/tools/<name>.tar.gz`.

For notifications, mount only the project config:

```sh
docker run --rm --mount "type=bind,source=$PWD/configs,target=/app/configs,readonly" \
  tc002-apps/tools/clockctl:local --text 'Hello from Docker'
```

For source-based clock operations, mount the repository and set the working directory:

```sh
docker run --rm --workdir /repo \
  --mount "type=bind,source=$PWD,target=/repo,readonly" \
  tc002-apps/tools/clockctl:local list
```

Replace `list` with `deploy`, `show --app <name>`, or `start`/`stop` for on-demand apps. URLs must be reachable from the container; container-local `localhost` is not your computer.

Production Compose does not include utilities. You may run this tool on a VPS when needed without deploying it as a server service. Development Compose supplies a convenient tool container, while `make clock-check` creates an isolated temporary AWTRIX and runner. Shell entry points live in root `scripts/`.

See the [command reference](../../docs/commands.md) and [runtime checks](../../docs/development.md#runtime-checks).
