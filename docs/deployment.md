# VPS deployment

## Requirements

- A Linux VPS with Docker Engine and the Docker Compose v2 plugin. Source-based deployment also requires Git.
- A user with permission to run Docker.
- Make and `gzip` on the build machine if you use the Make-based build and image-export commands.
- A network route from the container to the TC002 for publishing to the clock. Home IP addresses and `.local` names are usually unreachable from a VPS. Use a VPN route to your home network or, once MQTT support is implemented, a broker reachable by both sides.

## Initial deployment

On the VPS:

```sh
git clone https://github.com/LikiPiki/tc002-apps.git
cd tc002-apps
cp deploy/.env.example deploy/.env
# Edit deploy/.env: set the clock address and worker interval.
sh deploy/deploy.sh
```

The script validates the Compose configuration, builds the image, and starts the worker in the background. With the Docker daemon running, `restart: unless-stopped` restarts it after a reboot.

## Selecting apps

Go commands can be built independently:

```sh
make list-apps
make build
make docker-build
```

Build selection is controlled by `configs/apps.list`: directory names from `apps/server/cmd/`, one per line. This selects Go programs running outside the clock; Berry apps in `apps/clock/` are installed separately through AWTRIX NG. Blank lines and `#` comments are allowed. For example:

```text
worker
# clockctl
```

If the config is absent, **all** Go apps are built. If it exists, only the listed apps and their dependencies are built. An empty list or an unknown name is an error. Start with `cp configs/apps.list.example configs/apps.list`. This local config is excluded from Git. To use another path, run `make build CONFIG=configs/vps.list`; an absent file at that path also means all apps will be built.

`make check` and the local development environment check the project independently of production build selection. Existing files in `bin/` and `dist/` are not removed automatically. Transfer only the archives for the apps you selected.

To build a new Docker app, add an `apps/server/cmd/<name>` directory. To manage it through Compose, also add a service with the same name and image `tc002-apps/<name>:local`.

On the VPS, specify which services to deploy:

```sh
sh deploy/deploy.sh worker
# Once other services exist: sh deploy/deploy.sh weather monitor
```

The script builds and updates only the specified services. It does not start dependencies or remove other containers. With no arguments, it starts only `worker`. `clockctl` is a one-off CLI tool: use `compose run` rather than running it as a persistent service.

The app-selection config controls the Make build commands only. `deploy.sh` selects services through its arguments and builds them unless `--no-build` is set. For a config-selected build on the VPS, run `make docker-build`, then `sh deploy/deploy.sh --no-build worker` for the service you selected. Direct Compose build commands also do not read `configs/apps.list`.

## Transferring prebuilt images

To deploy without building Go on the VPS or transferring source code:

```sh
# On the build machine:
cp configs/apps.list.example configs/apps.list # The example selects worker.
make docker-save
scp dist/worker.tar.gz user@vps:/opt/tc002-apps/
# Also transfer deploy/compose.yaml and deploy/deploy.sh to /opt/tc002-apps/deploy/.
```

Create `/opt/tc002-apps/deploy` on the VPS before transferring files. Create `deploy/.env` using `deploy/.env.example` as a template. Then, from `/opt/tc002-apps`:

```sh
docker load -i worker.tar.gz
sh deploy/deploy.sh --no-build worker
```

The archive contains only the selected runtime image, without source code or a compiler. Deployment requires the Compose file, script, and `.env`; a full Git checkout is not required. The image architecture must match the VPS. An ARM Mac builds ARM64 images by default. For an x86-64 VPS, build on an x86-64 machine or explicitly select the platform:

```sh
DOCKER_DEFAULT_PLATFORM=linux/amd64 make docker-save
```

Cross-platform builds require Docker support for emulating the target architecture. To roll back a prebuilt-image deployment, load the previous image archive and redeploy with `--no-build`.

The worker currently only logs `worker tick`. Test connectivity to the clock separately:

```sh
docker compose --env-file deploy/.env -f deploy/compose.yaml run --rm --build clockctl --text 'Hello from VPS'
```

In an image-only deployment, load the `clockctl` image separately if you need it, and omit `--build` from this command.

`deploy/.env` is excluded from Git and the Docker build context. Keep your actual settings on the VPS.

## Logs and shutdown

Run these commands from the repository root, or from `/opt/tc002-apps` for an image-only deployment:

```sh
docker compose --env-file deploy/.env -f deploy/compose.yaml logs -f --tail=100 worker
docker compose --env-file deploy/.env -f deploy/compose.yaml ps
docker compose --env-file deploy/.env -f deploy/compose.yaml down
```

Logs are rotated, with up to three files of 10 MB each.

## Updating a source-based deployment

```sh
git rev-parse HEAD  # Save the current SHA for rollback.
git pull --ff-only
sh deploy/deploy.sh
```

This workflow builds source code directly on the VPS; no image registry or GitHub token for images is required. Push new changes to the repository before updating the VPS. For image-only deployments, export and load the new images, then rerun the deployment script with `--no-build`.

## Rolling back a source-based deployment

With a clean working tree on the VPS, check out the saved SHA and redeploy:

```sh
git checkout <previous-sha>
sh deploy/deploy.sh
```

For subsequent updates, return to your working branch and follow the normal update workflow.

## Running the worker once

```sh
docker compose --env-file deploy/.env -f deploy/compose.yaml run --rm --build worker --once
```

For an image-only deployment, omit `--build`. Go does not need to be installed on the VPS: builds run in Docker. Each new command can use the same Dockerfile with a different `APP` build argument.

## Configuration reference

See the [Go command reference](commands.md) for `AWTRIX_URL`, `WORKER_INTERVAL`, and command-line options. Compose reads values from `deploy/.env`; native Go commands do not load that file automatically.
