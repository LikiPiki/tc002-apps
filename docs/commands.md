# Go command reference

Run commands from the repository root. All commands here run outside the clock, on a computer or VPS. Sources live in `apps/server/`. Native commands require Go 1.24+; Docker workflows require Docker with Compose v2.

Both commands read `configs/config.toml` and optional `configs/config.local.toml`. See [Project configuration](configuration.md) for the schema, merging, paths, and Docker mounts.

## clockctl notifications

```sh
# Use clock.url from the project config:
go run ./apps/server/cmd/clockctl --text 'Hello TC002'
# Or explicitly override the URL:
go run ./apps/server/cmd/clockctl --url http://192.168.1.50 --text 'Hello TC002'
```

| Option | Default | Description |
| --- | --- | --- |
| `--url` | `AWTRIX_URL`, then TOML `clock.url` | Absolute HTTP/HTTPS base URL |
| `--text` | `Hello from Go` | Notification text |
| `--config` | `configs/config.toml` | Shared config; default path can use `TC002_CONFIG` |
| `--local-config` | `configs/config.local.toml` | Optional overrides; empty disables them; default path can use `TC002_LOCAL_CONFIG` |

The client posts to `/api/v1/notifications` with a 10-second timeout. Invalid URLs, failed requests, and non-2xx responses produce a nonzero exit status. Authentication is not implemented yet.

For production Docker, set `clock.url` in your personal TOML and run:

```sh
docker compose -f deploy/compose.yaml run --rm --build clockctl --text 'Hello from Docker'
```

For the virtual clock, use `make dev` followed by `make dev-notify TEXT="Hello!"`.

## Berry deployment

```sh
make clock-list                 # Target and selected files; no device access
make clock-deploy               # Upload clock.apps from TOML
make clock-deploy APP=mackerel  # Upload one app instead of the config selection
make clock-show APP=mackerel    # Show an installed app and check its error status
make clock-check APP=mackerel   # Test in disposable local AWTRIX, never the configured clock
```

The list/deploy/show Make commands require Go; `clock-check` requires Docker with Compose v2 and builds its Go runner in a container. For the regular local virtual clock without personal settings:

```sh
make dev
make clock-deploy LOCAL_CONFIG=
make clock-show APP=mackerel LOCAL_CONFIG=
```

Keep `AWTRIX_URL` unset when relying on config URLs. To explicitly override the target, use `AWTRIX_URL=http://localhost:8080 make clock-deploy`.

Direct CLI equivalents:

```sh
go run ./apps/server/cmd/clockctl deploy --app mackerel
go run ./apps/server/cmd/clockctl show --app mackerel
go run ./apps/server/cmd/clockctl list --local-config ''
go run ./apps/server/cmd/clockctl server-list # Go build selection, names only
```

Deployment reads all selected sources before writing to the device. It requires running Berry scripting and the `scriptUpdates` capability. It reads the exact previous source, then uses `/api/v1/apps/script-update/{name}` with `expected_source` (or `null` for a new app). AWTRIX preserves the previous script if the new source fails compilation or setup, and rejects concurrent changes with HTTP 409. There is no unsafe replacement fallback. Update AWTRIX NG if atomic updates are unsupported.

After each upload the command checks the installed script's enabled/error status. It stops on the first failure; earlier successful apps in the batch are not rolled back. Same-name scripts preserve AWTRIX's saved app data. Unselected apps are not deleted. Errors in later `draw()` or callbacks still require runtime checks.

Deployment does not explicitly switch the current screen. `clock-show` selects an app, allows `draw()` to execute, and checks its error status again. Later callbacks can still fail; this is not a full visual or hardware test. Normal rotation continues afterwards. AWTRIX compiles the source; no local Berry compiler is needed.

To deploy scripts from Docker, mount the repository read-only so the config and source are available:

```sh
# Include clockctl in server.apps before building its image.
make docker-build
docker run --rm --workdir /repo \
  --mount "type=bind,source=$PWD,target=/repo,readonly" \
  tc002-apps/clockctl:local list
```

Replace `list` with `deploy` or `show --app mackerel`. Inside Docker, `localhost` means the container itself; use a reachable physical-clock address or the appropriate container network address. Production Compose mounts config only, not Berry source files.

## worker

The worker currently logs `worker tick`; it does not fetch data or publish to the clock.

```sh
go run ./apps/server/cmd/worker --once
go run ./apps/server/cmd/worker --interval 30s
```

| Option | Default | Description |
| --- | --- | --- |
| `--once` | `false` | Run once and exit |
| `--interval` | TOML `server.worker.interval` (`1m`) | Explicit positive Go duration override |
| `--config` | `configs/config.toml` | Shared project config |
| `--local-config` | `configs/config.local.toml` | Optional personal config; empty disables it |

The first run happens immediately. SIGINT or SIGTERM stops periodic mode gracefully. The interval must be positive, including when `--once` is used. The effective interval is included in each log entry.

Pass `--help` to either command, or to a `clockctl` subcommand, for its flags.
