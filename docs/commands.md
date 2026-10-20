# Go command reference

Run commands from the repository root. Native examples require Go 1.24+; Docker examples only require Docker with Compose v2.

## clockctl

Send a single notification to an AWTRIX NG instance:

```sh
AWTRIX_URL=http://192.168.1.50 go run ./cmd/clockctl --text 'Hello TC002'
# Or set the URL explicitly:
go run ./cmd/clockctl --url http://192.168.1.50 --text 'Hello TC002'
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

## worker

The worker currently logs `worker tick`; it does not fetch data or send anything to the clock.

```sh
go run ./cmd/worker --once
go run ./cmd/worker --interval 30s
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
