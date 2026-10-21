# VPS deployment

The VPS runs one `server` container containing the complete `tc002-server` binary and all server features. There are no per-feature images, server app lists or generated build sets.

## Initial deployment

Use a Linux VPS with Docker Engine, Compose v2 and a route to the clock addresses (for home LAN addresses, typically through a VPN). Source-based updates also require Git.

```sh
git clone https://github.com/LikiPiki/tc002-apps.git
cd tc002-apps
cp configs/config.local.toml.example configs/config.local.toml
# Configure clock addresses/profiles and server.weather coordinates.
sh deploy/deploy.sh
```

The script validates Compose, builds the complete image and starts `server` in the background. `restart: unless-stopped` starts it after a reboot when Docker is running. The mounted `configs/` directory supplies your settings and clock profiles; no `.env` file is needed.

## Build the complete server

```sh
make build-server        # bin/server/tc002-server; requires Go
make docker-build-server # tc002-apps/server:local; requires Docker
make docker-save-server  # dist/server/tc002-server.tar.gz
```

Default `make build`, `make docker-build` and `make docker-save` produce the complete server only. Utilities use explicit `*-tools` targets and are not deployed as server services. The server Dockerfile always builds `apps/server/cmd/server`; runtime settings do not change its feature set.

## Transfer one prebuilt image

On the build machine:

```sh
make docker-save-server
# For an x86-64 VPS when building on ARM:
DOCKER_DEFAULT_PLATFORM=linux/amd64 make docker-save-server
```

Prepare `/opt/tc002-apps/deploy` and `/opt/tc002-apps/configs` on the VPS. Transfer the one archive, `deploy/compose.yaml`, `deploy/deploy.sh`, and your config files including referenced clock profiles. Then run:

```sh
docker load -i tc002-server.tar.gz
sh deploy/deploy.sh --no-build
```

A full source checkout or host Go installation is not required for image-only deployment. Existing local build artifacts are not deleted automatically.

## Logs, one-off updates and config reload

```sh
docker compose -f deploy/compose.yaml logs -f --tail=100 server
docker compose -f deploy/compose.yaml ps
docker compose -f deploy/compose.yaml run --rm server --once
docker compose -f deploy/compose.yaml run --rm server --print
docker compose -f deploy/compose.yaml restart server
docker compose -f deploy/compose.yaml down
```

The default mode updates continuously; `--once` runs one update, and `--print` inspects a reading without connecting to clocks. Config and profile changes take effect after restarting the process. Logs rotate with up to three files of 10 MB each.

## Updates and migration

```sh
git rev-parse HEAD  # Save the deployed SHA for rollback.
git pull --ff-only
sh deploy/deploy.sh
```

The deployment script removes orphaned per-feature containers from the same Compose project when replacing the old layout with the single `server`. Remove obsolete server build-selection and placeholder-worker fields from personal config; only runtime feature settings such as `[server.weather]` remain. Older service-name arguments to `deploy.sh` are no longer accepted.

For source rollback, check out the saved SHA in a clean working tree and deploy it. For image-only rollback, load the previous complete server archive and run `sh deploy/deploy.sh --no-build`.

See [configuration](configuration.md) and [local LAN debugging](development.md#local-server-with-physical-clocks).
