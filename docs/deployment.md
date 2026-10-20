# VPS deployment

## Requirements

- A Linux VPS with Docker Engine and Docker Compose v2; source-based updates also require Git.
- A user with permission to run Docker.
- Make and `gzip` on the build machine for Make-based builds and image exports.
- A network route from the container to the TC002. A home IP address is normally unreachable from a VPS: use a VPN, or a broker reachable by both sides once MQTT support is implemented.

## Initial deployment

On the VPS:

```sh
git clone https://github.com/LikiPiki/tc002-apps.git
cd tc002-apps
cp configs/config.local.toml.example configs/config.local.toml
# Edit clock.url, server.apps, and server.worker.interval as needed.
sh deploy/deploy.sh worker
```

The script validates Compose, builds the selected service, and starts it in the background. `restart: unless-stopped` restarts the worker after a reboot when Docker is running. With no arguments, the script selects only `worker`.

Both native Go commands and containers read the [same TOML files](configuration.md). Compose mounts `configs/` read-only; no `.env` file is needed. The shared config must exist, and personal overrides are optional. Keep actual personal settings out of Git.

## Build selection

Set the Go commands to build in your personal config:

```toml
[server]
apps = ["worker"]
```

`["*"]` or an omitted selection builds all Go commands. An empty list is an error.

```sh
make list-apps
make build         # Native binaries; requires Go
make docker-build # Selected runtime images
make docker-save  # Selected images as dist/<app>.tar.gz
```

`server.apps` selects Make build/export artifacts. It does not choose running services. Explicit deployment selects services separately:

```sh
sh deploy/deploy.sh worker
# Once other services exist: sh deploy/deploy.sh weather monitor
```

Only specified services are built and updated. Dependencies are not started and other containers are not removed. To build according to TOML on the VPS, use `make docker-build`, then `sh deploy/deploy.sh --no-build worker`. Direct Compose build commands select their services independently of TOML.

`clockctl` is a one-off CLI tool: use `compose run`, not a persistent service. For a new Go app, add `apps/server/cmd/<name>/main.go`; managed deployment also needs a Compose service using `APP: <name>` and image `tc002-apps/<name>:local`.

## Transfer prebuilt images

On the build machine, configure `server.apps`, then export only the images you need:

```sh
make docker-save
```

Prepare `/opt/tc002-apps/deploy` and `/opt/tc002-apps/configs` on the VPS. Transfer:

- The selected `dist/<app>.tar.gz` archives.
- `deploy/compose.yaml` and `deploy/deploy.sh`.
- `configs/config.toml` and your VPS-specific `configs/config.local.toml` (optional).

Then, from `/opt/tc002-apps` on the VPS:

```sh
docker load -i worker.tar.gz
sh deploy/deploy.sh --no-build worker
```

A full Git checkout, source code, and Go installation are not required. The archive contains the selected binary and runtime dependencies; shared defaults are included, and the mounted config directory supplies deployment-specific settings.

Match the image architecture to the VPS. An ARM Mac builds ARM64 images by default. For x86-64, build on an x86-64 machine or use Docker with target-architecture emulation:

```sh
DOCKER_DEFAULT_PLATFORM=linux/amd64 make docker-save
```

Existing `bin/` and `dist/` files are not removed automatically. Transfer only the selected apps' current archives. For image-only rollback, load the previous archive and redeploy with `--no-build`.

## Logs and one-off commands

Run from the repository root, or `/opt/tc002-apps` for an image-only deployment:

```sh
docker compose -f deploy/compose.yaml logs -f --tail=100 worker
docker compose -f deploy/compose.yaml ps
docker compose -f deploy/compose.yaml run --rm --build worker --once
docker compose -f deploy/compose.yaml run --rm --build clockctl --text 'Hello from VPS'
docker compose -f deploy/compose.yaml down
```

For image-only deployment, load each needed image and omit `--build`. The worker logs its effective interval but remains a placeholder; it does not yet publish data. Notification commands use `clock.url` from the mounted TOML. Logs rotate with up to three files of 10 MB each.

## Updating and rollback

For a source-based deployment:

```sh
git rev-parse HEAD  # Save this SHA for rollback.
git pull --ff-only
sh deploy/deploy.sh worker
```

Push new changes before updating the VPS. For rollback, check out the saved SHA in a clean working tree and rerun deployment; return to the working branch for later updates. For image-only deployment, export/load the new archives and rerun with `--no-build`.

Config changes are read at process startup. After editing TOML, restart the running service:

```sh
docker compose -f deploy/compose.yaml restart worker
```

See [Project configuration](configuration.md) for merge rules and [Go commands](commands.md) for flags.
