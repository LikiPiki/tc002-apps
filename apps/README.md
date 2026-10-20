# Apps by runtime

Apps are separated by where they run:

| Directory | Language | Runs on | Delivery |
| --- | --- | --- | --- |
| [`clock/`](clock/) | Berry (`.be`) | TC002 inside AWTRIX NG | Upload through the Scripts tab |
| [`server/`](server/) | Go (`.go`) | VPS or development computer | Native binary or Docker image |
| [`tools/`](tools/) | Go (`.go`) | Computer or one-off container | CLI tools for build/deploy/check workflows |

Go binaries and Docker images are not uploaded to the clock. Berry scripts are not included in the server images. A feature may have both a server component that processes data and a clock component that renders it; keep each component in its runtime directory and document how they communicate.

`server/` is reserved for VPS applications; a development computer may run them for tests. Utilities belong to `tools/`. Shared Go libraries live in the root `internal/`, not under either app family.

The project TOML selects server artifacts with `server.apps`, utility artifacts with `tools.apps`, and Berry scripts by category under `clock.deploy`. App behavior is separate from build/upload selection. Cross-runtime pairing by [integration ID and data contract](../docs/integrations.md) is an alpha design, not an implemented registry. See [configuration](../docs/configuration.md) and [architecture](../docs/architecture.md).
