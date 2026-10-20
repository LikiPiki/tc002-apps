# Project configuration

All project settings use one TOML schema and one Go loader, for native commands, Make builds, and containers:

```text
configs/config.toml                Shared defaults, committed
configs/config.local.toml.example  Personal config template, committed
configs/config.local.toml          Optional personal overrides, ignored by Git
```

The shared file is required. The personal file may be absent. Unknown fields, malformed TOML, empty app lists, and invalid durations fail with an error.

## Shared defaults

```toml
[clock]
url = "http://localhost:8080"
apps = ["*"]

[server]
apps = ["*"]

[server.worker]
interval = "1m"
```

| Setting | Used by |
| --- | --- |
| `clock.url` | Notifications, Berry deployment, and display selection |
| `clock.apps` | Berry scripts selected by `clock-list` and `clock-deploy` |
| `server.apps` | Go commands selected by `list-apps`, `build`, `docker-build`, and `docker-save` |
| `server.worker.interval` | Period between worker runs; a positive Go duration such as `30s` or `5m` |

An omitted app selection or `["*"]` means all apps. Named lists replace the selection: `clock.apps` names directories under `apps/clock/`, and `server.apps` names commands under `apps/server/cmd/`. A wildcard must be used alone. An empty list is an error. Missing source files fail before upload or build.

## Personal overrides

```sh
cp configs/config.local.toml.example configs/config.local.toml
```

For example:

```toml
[clock]
url = "http://192.168.1.50"
apps = ["cigar", "mackerel"]

[server]
apps = ["worker"]

[server.worker]
interval = "30s"
```

Only specified fields override shared defaults, including within nested tables. Arrays replace rather than append. To reset a selection to all apps, use `apps = ["*"]` in the relevant table.

The personal file is excluded from Git and Docker build context. Your settings are never baked into the image.

## Paths and explicit overrides

Make commands accept `CONFIG` and `LOCAL_CONFIG`:

```sh
make clock-list
make list-apps
make clock-deploy APP=mackerel
make clock-deploy LOCAL_CONFIG=  # Shared local-clock defaults only
make build LOCAL_CONFIG="$HOME/.config/tc002/vps.toml"
```

Go commands accept the same paths via `--config` and `--local-config`. Their default paths can also be changed with `TC002_CONFIG` and `TC002_LOCAL_CONFIG`. An explicitly empty local-config path disables personal overrides.

For clock operations, URL precedence is explicit `--url` > `AWTRIX_URL` > personal TOML > shared TOML. For the worker, explicit `--interval` overrides TOML. These overrides are optional conveniences, not a second config file. Keep `AWTRIX_URL` unset when using the URL from TOML.

## Docker

Production Compose mounts `configs/` at `/app/configs` read-only. Both `worker` and `clockctl` read the same files directly. No `.env` file is needed. The image includes only the shared defaults, so it can also run without the mount.

The dev Compose environment ignores personal overrides to avoid sending development traffic to your physical clock. It uses the shared TOML and sets `AWTRIX_URL=http://awtrix:8080` for the container-only network address of the virtual clock. On your computer, the same service is available at http://localhost:8080.

Native builds need Go. Docker-only builds use a containerized `clockctl server-list` to read TOML when Go is unavailable; only selected app images are exported by `make docker-save`. This fallback mounts the repository, so config files must be inside it. Native commands can also read profiles outside the repository.

`server.apps` selects build/export artifacts, not running Compose services. Choose services with `sh deploy/deploy.sh worker`; direct Compose build commands also select their services independently of TOML. Runtime settings still come from the same mounted TOML.
