# Server apps (Go)

Everything in this directory runs outside the clock, on a VPS or a development computer. Go programs call AWTRIX NG over the network; they are not installed on the TC002.

```text
cmd/
  worker/       Placeholder periodic service for a VPS
  clockctl/     Notifications, Berry deployment, and app display control
internal/
  awtrix/       Shared AWTRIX NG HTTP client
```

The repository root contains the shared `go.mod`, Makefile, and Dockerfile. Run commands from that root:

```sh
make list-apps
make build
make docker-build
go run ./apps/server/cmd/worker --once
```

`server.apps` in the project TOML selects commands by name (`worker`, `clockctl`); `["*"]` or an omitted selection builds all commands. Native binaries go to `bin/`, Docker image exports go to `dist/`. Runtime images include the selected binary, runtime dependencies, and shared TOML defaults, not Berry apps. Compose mounts your config directory read-only. See [configuration](../../docs/configuration.md).

Deploy persistent services on the VPS and run CLI tools on demand. See the [command reference](../../docs/commands.md) and [deployment guide](../../docs/deployment.md).
