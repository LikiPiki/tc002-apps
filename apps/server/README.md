# Server

One process and one image contain the complete server. All server features are compiled into `cmd/server`; there is no feature-specific binary, image, build argument or build selection.

```text
cmd/server/                 Complete server entry point
internal/weatherprovider/   Weather fetching, validation and delivery
Dockerfile                  One tc002-apps/server image
```

Add server-specific processing code under `internal/` here when needed. Shared device API/config packages live in the repository-root `internal/`. Server code must not depend on `apps/tools`; Go's internal-package rules protect utility implementation packages.

Run from the repository root:

```sh
make build-server
make docker-build-server
make docker-save-server
go run ./apps/server/cmd/server --print
go run ./apps/server/cmd/server --once
sh deploy/deploy.sh
```

The binary is `bin/server/tc002-server`, the image is `tc002-apps/server:local`, and the exported image is `dist/server/tc002-server.tar.gz`. Default `make build`, `make docker-build` and `make docker-save` produce this complete server only. Server configuration controls runtime values such as weather coordinates and intervals, not which features are compiled.

Production Compose has one service, `server`. Its image contains the complete binary, runtime dependencies and shared defaults; Compose mounts your config directory read-only. `make lan` starts that same server against physical clocks, and `make dev-server` runs it against virtual AWTRIX. `make server-check` runs isolated data/consumer integration tests.

The [weather integration](../../docs/integrations/weather.md) supplies two independent screens. Configure coordinates in `[server.weather]`; `tc002-server --print` inspects one reading without contacting clocks, `--once` publishes once, and the default mode updates continuously. Future server features belong under `internal/` and are wired into the same entry point and container.

See [deployment](../../docs/deployment.md), [configuration](../../docs/configuration.md), and [utility apps](../tools/README.md).
