# Architecture

The project supports two independent types of apps:

1. **`apps/clock/` — Berry on the TC002:** local logic, rendering, and interaction with the device. Scripts are uploaded through the AWTRIX NG web UI.
2. **`apps/server/` — Go outside the clock:** fetching data from APIs, processing it, and publishing it to the clock over HTTP or MQTT. These programs run on a VPS or computer as native binaries or Docker containers.

```text
VPS / computer                         TC002
apps/server/                           AWTRIX NG
  Go service -- HTTP / MQTT --------->   built-in screens or notifications
                                        apps/clock/ Berry app for custom rendering
```

A feature can have both components, but they live in separate runtime directories. For example, a server-side weather integration could feed a clock-side weather display. Document the data contract in the components' READMEs. A simple notification needs only a server component.

A simple screen or notification can be generated entirely in Go. For custom animation or interaction, a Berry app receives data and handles rendering on the clock.

The current implementation provides HTTP notifications, Berry script upload, and display selection through `clockctl`. Shared and personal TOML configs select the clock and Berry apps to deploy. MQTT, authentication, and specific data sources will be added as needed by individual apps.

Each Go command lives in `apps/server/cmd/<name>`, with reusable logic in `apps/server/internal/`. A shared root-level `go.mod` and Dockerfile build the command selected by the `APP` build argument. Independent services get separate entries in Compose. The shared `internal/config` package reads one project TOML plus personal overrides: `server.apps` selects Go builds and `clock.apps` selects Berry uploads. Containers use a read-only config mount. Berry scripts and their assets are not included in the runtime images. See [configuration](configuration.md).

`worker` is a placeholder demonstrating periodic execution and graceful shutdown on SIGTERM. It does not fetch data or publish anything to the clock yet.
