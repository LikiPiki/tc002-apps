# Apps by runtime

Apps are separated by where they run:

| Directory | Language | Runs on | Delivery |
| --- | --- | --- | --- |
| [`clock/`](clock/) | Berry (`.be`) | TC002 inside AWTRIX NG | Upload through the Scripts tab |
| [`server/`](server/) | Go (`.go`) | VPS or development computer | Native binary or Docker image |

Go binaries and Docker images are not uploaded to the clock. Berry scripts are not included in the server images. A feature may have both a server component that processes data and a clock component that renders it; keep each component in its runtime directory and document how they communicate.

The shared project TOML and personal overrides select Go commands with `server.apps` and Berry scripts by category under `clock.deploy`. Per-app behavior settings are separate from upload selection. See [configuration](../docs/configuration.md) and the [architecture guide](../docs/architecture.md).
