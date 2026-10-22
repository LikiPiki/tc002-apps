# VPS deployment

The VPS runs one `server` container containing the complete `tc002-server` binary and all server features. There are no per-feature images, server app lists or generated build sets.

## Initial deployment

Use a Linux VPS with Docker Engine, Compose v2 and a route to the clock addresses (for home LAN addresses, typically through a VPN). Source-based updates also require Git.

```sh
git clone https://github.com/LikiPiki/tc002-apps.git
cd tc002-apps
make init-config
# Edit configs/config.toml; optional config.local.toml can override fields.
# Configure clock addresses/profiles and server.weather coordinates.
sh deploy/deploy.sh
```

The script validates Compose, builds the complete image and starts `server` in the background. `restart: unless-stopped` starts it after a reboot when Docker is running. The mounted `configs/` directory supplies your settings and clock profiles; no `.env` file is needed.

Runtime `configs/` is ignored by Git and Docker build context. Images contain no working configs or examples; they receive settings only through the read-only mount. Setup templates stay on the host under `examples/configs/`. Initialization preserves existing files. Deployment does not initialize or rewrite configs, and stops with setup instructions if the working file is missing.

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

Prepare `/opt/tc002-apps/deploy` and `/opt/tc002-apps/configs` on the VPS. Transfer the one archive, `deploy/compose.yaml`, `deploy/deploy.sh`, and your working config including referenced clock profiles. Preserve existing VPS-specific settings when updating images. Then run:

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

Before the one-time update from a revision that tracked `configs/config.toml`, back it up outside the checkout and restore it after Git applies the deletion. Afterwards it is ignored and subsequent pulls no longer manage it. See [configuration migration](configuration.md#migration-from-the-flat-catalog).

```sh
git rev-parse HEAD  # Save the deployed SHA for rollback.
git pull --ff-only
sh deploy/deploy.sh
```

The deployment script removes orphaned per-feature containers from the same Compose project when replacing the old layout with the single `server`. Remove obsolete server build-selection and placeholder-worker fields from personal config; only runtime feature settings such as `[server.weather]` remain. Older service-name arguments to `deploy.sh` are no longer accepted.

For source rollback, check out the saved SHA in a clean working tree and deploy it. For image-only rollback, load the previous complete server archive and run `sh deploy/deploy.sh --no-build`.

See [configuration](configuration.md) and [local LAN debugging](development.md#local-server-with-physical-clocks).
