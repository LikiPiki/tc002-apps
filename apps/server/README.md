# VPS applications

This directory contains processes intended to run on a VPS. It does not contain the clock-management CLI, source catalogs, or local display checks.

```text
cmd/worker/     Placeholder periodic service
cmd/status-provider/  HTTP health producer for the service-status screen
cmd/weather-provider/ Open-Meteo current conditions and three-day forecast
Dockerfile      Server-only build: apps/server + shared internal libraries
```

Add server-specific processing code under `internal/` here when needed. Shared device API/config packages live in the repository-root `internal/`. Server code must not depend on `apps/tools`; Go's internal-package rules protect utility implementation packages.

Run from the repository root:

```sh
make list-apps GROUP=server
make build GROUP=server
make docker-build GROUP=server
make docker-save GROUP=server
go run ./apps/server/cmd/worker --once
```

`server.apps` in TOML selects commands within this group. Binaries are written to `bin/server/<name>`, images are tagged `tc002-apps/server/<name>:local`, and exported archives go to `dist/server/<name>.tar.gz`.

Production `deploy/compose.yaml` contains VPS services only. Runtime images include the selected binary, runtime dependencies, and shared config defaults; Compose mounts your config directory read-only. Neither Berry source nor CLI/checking code is copied into the server build.

The [service-status integration](../../docs/integrations/service-status.md) provides a working server/screen example and `make status-check` end-to-end verification. Automatic pairing and config wiring are not implemented yet.

The [weather integration](../../docs/integrations/weather.md) supplies two independent screens. Configure coordinates in `[server.weather]`, run `go run ./apps/server/cmd/weather-provider --print` to inspect weather directly on the VPS, or deploy `weather-provider` to publish continuously. `make weather-check` runs its isolated end-to-end scenarios.

See [deployment](../../docs/deployment.md), [configuration](../../docs/configuration.md), and [utility apps](../tools/README.md).
