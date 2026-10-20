# Go command reference

Run commands from the repository root. All commands described here run outside the clock, on your computer or VPS. Sources live in `apps/server/`. Native examples require Go 1.24+; Docker examples only require Docker with Compose v2.

## clockctl

Send a single notification to an AWTRIX NG instance:

```sh
AWTRIX_URL=http://192.168.1.50 go run ./apps/server/cmd/clockctl --text 'Hello TC002'
# Or set the URL explicitly:
go run ./apps/server/cmd/clockctl --url http://192.168.1.50 --text 'Hello TC002'
```

| Option | Default | Description |
| --- | --- | --- |
| `--url` | Value of `AWTRIX_URL` | Absolute HTTP or HTTPS base URL of AWTRIX NG |
| `--text` | `Hello from Go` | Notification text |

An explicit `--url` takes precedence over `AWTRIX_URL`. The client posts to `/api/v1/notifications` with a 10-second timeout. Invalid URLs, failed requests, and non-2xx responses produce a nonzero exit status. Authentication is not implemented yet.

For Docker, create `deploy/.env` from `deploy/.env.example`, set `AWTRIX_URL`, then run:

```sh
docker compose --env-file deploy/.env -f deploy/compose.yaml run --rm --build clockctl --text 'Hello from Docker'
```

For the local virtual clock, use `make dev` followed by `make dev-notify TEXT="Hello!"`.

### Berry deployment

`clockctl deploy`, `clockctl show`, and `clockctl list` use a shared TOML config plus optional personal overrides. Run these commands from the repository root:

```sh
make clock-list                 # Show the target and selected source files; no device access
make clock-deploy               # Upload selected scripts
make clock-deploy APP=mackerel  # Upload one script regardless of config selection
make clock-show APP=mackerel    # Show the installed script and check its runtime status
```

These Make commands require Go 1.24+. The base config `configs/clock.toml` is committed:

```toml
# Local virtual clock
url = "http://localhost:8080"

# All Berry apps; or a list such as ["cigar", "mackerel"]
apps = ["*"]
```

- `url` is the target's base HTTP/HTTPS URL.
- `apps = ["*"]` discovers all `apps/clock/*/main.be` files in alphabetical order. If neither config specifies `apps`, discovery is also the default.
- Set `apps` to a nonempty list such as `["cigar", "seagull"]` to deploy a subset. An empty list, unknown field, invalid name, missing source, or empty source is an error.

For your physical clock, create the optional personal config:

```sh
cp configs/clock.local.toml.example configs/clock.local.toml
# Edit the URL in configs/clock.local.toml, then inspect the selection:
make clock-list
```

Only fields present in the personal config override the base config. For example, `url = "http://192.168.1.50"` keeps the base app selection. Arrays replace the selection rather than append to it. Set `apps = ["*"]` to restore automatic discovery. The personal file is ignored by Git and Docker builds; a missing personal file is allowed, but a missing base config or malformed TOML fails the command. TOML supports `#` comments and multiline arrays.

Precedence for deployment URLs: explicit CLI `--url` > `AWTRIX_URL` environment variable > personal config > base config. The legacy notification mode (`clockctl --text ...`) continues to use its `--url` flag or `AWTRIX_URL`, not these TOML files.

To target the local virtual clock without personal overrides:

```sh
make dev
make clock-deploy CLOCK_LOCAL_CONFIG=
make clock-show APP=mackerel CLOCK_LOCAL_CONFIG=
```

Ensure `AWTRIX_URL` is unset when relying on config URLs. Alternatively, override it explicitly: `AWTRIX_URL=http://localhost:8080 make clock-deploy`.

For separate device profiles, keep a private TOML file outside the repository and pass its path:

```sh
make clock-deploy CLOCK_LOCAL_CONFIG="$HOME/.config/tc002/office.toml"
```

`CLOCK_CONFIG` can also select a different base file. Direct CLI equivalents:

```sh
go run ./apps/server/cmd/clockctl deploy --app mackerel
go run ./apps/server/cmd/clockctl show --app mackerel
go run ./apps/server/cmd/clockctl list --local-config ''
```

Deployment validates and reads all selected local sources before writing to the device. It checks that Berry scripting is running, uploads each source, requires `error: null`, and checks the installed script's enabled/error status. It stops on the first failure; earlier successful uploads are not rolled back. Existing scripts with the same names are replaced, preserving AWTRIX's saved app data. Apps not selected are not deleted.

Deployment does not explicitly switch the active screen. Use `clock-show` to select an app, allow its `draw()` to execute, and check its error status again. This is not a full visual or hardware test; later callbacks can still fail. The normal app rotation continues after showing an app.

The source is compiled on AWTRIX; no local Berry compiler is needed. `configs/apps.list` controls Go builds only and has no effect on Berry deployment.

For a Docker-only workflow, first build the `clockctl` image (include it in your Go build selection), then mount the repository read-only so its config and scripts are available:

```sh
docker run --rm --workdir /repo \
  --mount "type=bind,source=$PWD,target=/repo,readonly" \
  tc002-apps/clockctl:local list
```

Replace `list` with `deploy` or `show --app mackerel`. Config URLs must be reachable from the container; `localhost` inside a container is not the host. Personal config is read from the bind mount, not baked into the image.

## worker

The worker currently logs `worker tick`; it does not fetch data or send anything to the clock.

```sh
go run ./apps/server/cmd/worker --once
go run ./apps/server/cmd/worker --interval 30s
```

| Option | Default | Description |
| --- | --- | --- |
| `--once` | `false` | Run once and exit |
| `--interval` | `1m` | Positive Go duration between runs, such as `30s` or `5m` |

The first run happens immediately. In periodic mode, SIGINT or SIGTERM stops the process gracefully. The interval must be positive, including when `--once` is used.

## Deployment environment

`deploy/.env` provides Compose interpolation values; it is not automatically read by native Go commands.

| Variable | Example/default | Used by |
| --- | --- | --- |
| `AWTRIX_URL` | Example: `http://192.168.1.50` | The `clockctl` container |
| `WORKER_INTERVAL` | Default: `1m` | The worker's `--interval` argument |

Inside Docker, `localhost` refers to the container itself. Use a clock address reachable from the container. The development Compose file already points `clockctl` at `http://awtrix:8080`.

Pass `--help` to either Go command to see its flags.
