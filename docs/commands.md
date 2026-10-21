# Go command reference

## Standalone device management

With an explicit `--url` or `AWTRIX_URL`, notification, show/start/stop, rollback, doctor and status commands work outside a checkout even if the implicit shared config is absent. Existing or explicitly selected config files are still validated. `--config '' --local-config ''` explicitly disables file loading. Catalog commands accept `--apps-dir /path/to/apps/clock` along with absolute config paths.

```sh
clockctl --url http://192.168.1.50 --text 'Hello'
clockctl doctor --url http://192.168.1.50
clockctl status --url http://192.168.1.50 --json
clockctl deploy --dry-run --json
```

`doctor` checks device/API access, scripting, installed app errors, atomic updates and frame dimensions. `status` reports device and installed app state. Both return nonzero on diagnosed problems. `--json` is also available for catalog `list`, offline `validate`, and deployment (including `--dry-run`); diagnostics go to stderr so stdout remains parseable.

## Offline validation

Run `make clock-validate` or `clockctl validate` to validate the selected clock apps without network requests. This checks lifecycle markers, declared settings, defaults, lengths, select options, numeric bounds, colors and display-policy time semantics before deployment.

## Deployment plans and recovery

`clockctl deploy --dry-run` reads current sources/settings and prints changes without writing. Normal deployment skips unchanged values, uses compare-and-swap source updates, saves a private recovery record in `dist/deploy/`, and observes enabled scripts for 1.5 seconds. Use `--backup <new-file>` to choose the recovery path, `--observe 3s` to change the observation window, and `--timeout 2m` to bound the entire command. Ctrl-C and SIGTERM cancel in-flight requests. Disabled apps remain disabled.

`clockctl rollback --backup dist/deploy/<record>.json` restores previous sources, declared settings and enabled state on the same target. Recovery stops if a source differs from both the before/after record. Newly installed apps are disabled rather than deleted, because the device has no atomic delete operation. Recovery is explicit, sequential and may itself be partial. Settings writes have no compare-and-swap API; avoid concurrent settings edits during deployment or recovery. Records capture declared configuration, not arbitrary app storage or external side effects.

Run commands from the repository root. Utility commands live in `apps/tools/`; independent VPS programs live in `apps/server/`. Both can run natively or in their separate Docker images. Native commands require Go 1.24+; Docker workflows require Docker with Compose v2.

Both commands read `configs/config.toml` and optional `configs/config.local.toml`. See [Project configuration](configuration.md) for the schema, merging, paths, and Docker mounts.

## clockctl notifications

```sh
# Use clock.url from the project config:
go run ./apps/tools/cmd/clockctl --text 'Hello TC002'
# Or explicitly override the URL:
go run ./apps/tools/cmd/clockctl --url http://192.168.1.50 --text 'Hello TC002'
```

| Option | Default | Description |
| --- | --- | --- |
| `--url` | `AWTRIX_URL`, then TOML `clock.url` | Absolute HTTP/HTTPS base URL |
| `--text` | `Hello from Go` | Notification text |
| `--config` | `configs/config.toml` | Shared config; default path can use `TC002_CONFIG` |
| `--local-config` | `configs/config.local.toml` | Optional overrides; empty disables them; default path can use `TC002_LOCAL_CONFIG` |

The client posts to `/api/v1/notifications` with a 10-second timeout. Invalid URLs, failed requests, and non-2xx responses produce a nonzero exit status. Authentication is not implemented yet.

HTTP failures include the method, API path (including the app name for script operations), status, and AWTRIX's error code/message/field when present. Plain-text or malformed responses use a bounded body excerpt. Transport and JSON-decoding failures also include the operation. For example: `PUT /api/v1/apps/script-update/demo: AWTRIX returned 422 ... (validationFailed): ...`. The same error handling is used for notifications and script operations.

For a standalone Docker tool, set `clock.url` in your personal TOML and run:

```sh
make docker-build GROUP=tools
docker run --rm --mount "type=bind,source=$PWD/configs,target=/app/configs,readonly" \
  tc002-apps/tools/clockctl:local --text 'Hello from Docker'
```

For the virtual clock, use `make dev` followed by `make dev-notify TEXT="Hello!"`.

## Berry deployment

```sh
make clock-list                 # Target and selected files; no device access
make clock-deploy               # Upload the category lists under clock.deploy
make clock-deploy KIND=services # Filter by screens, services, or on-demand
make clock-deploy APP=mackerel  # Upload one app instead of the config selection
make clock-show APP=mackerel    # Show an installed app and check its error status
make clock-start APP=<tool>     # Start an on-demand session
make clock-stop APP=<tool>      # End that session, keeping it installed
make clock-check APP=mackerel   # Test in disposable local AWTRIX, never the configured clock
```

The list/deploy/show/start/stop Make commands require Go; `clock-check` requires Docker with Compose v2 and builds its Go runner in a container. APP names are unique across categories, KIND filters list/deploy/check, and empty clock selections are valid no-ops. For the regular local virtual clock without personal settings:

```sh
make dev
make clock-deploy LOCAL_CONFIG=
make clock-show APP=mackerel LOCAL_CONFIG=
```

Keep `AWTRIX_URL` unset when relying on config URLs. To explicitly override the target, use `AWTRIX_URL=http://localhost:8080 make clock-deploy`.

Direct CLI equivalents:

```sh
go run ./apps/tools/cmd/clockctl deploy --app mackerel
go run ./apps/tools/cmd/clockctl show --app mackerel
go run ./apps/tools/cmd/clockctl list --local-config ''
go run ./apps/tools/cmd/clockctl build-list --group all # Qualified Go artifact selection
go run ./apps/tools/cmd/clockctl server-list # Compatibility alias: server command names only
```

Deployment reads all selected sources before writing to the device. It requires running Berry scripting and the `scriptUpdates` capability. It reads the exact previous source, then uses `/api/v1/apps/script-update/{name}` with `expected_source` (or `null` for a new app). AWTRIX preserves the previous script if the new source fails compilation or setup, and rejects concurrent changes with HTTP 409. There is no unsafe replacement fallback. Update AWTRIX NG if atomic updates are unsupported.

After each upload the command checks the installed script's enabled/error status. It stops on the first failure; earlier successful apps in the batch are not rolled back. Same-name scripts preserve AWTRIX's saved app data. Unselected apps are not deleted. Errors in later `draw()` or callbacks still require runtime checks.

Every deployment prints a per-app report for source, settings, and verification. A partial failure distinguishes completed apps, the failed phase, and apps not attempted yet. For example, `source=updated settings=unconfirmed verification=not-run` means the new source was confirmed but settings were not. If AWTRIX confirms that settings were saved but app restart failed, the report uses `settings=applied verification=failed FAILED at restart`. A timeout is marked unconfirmed rather than implying that no change occurred. No batch rollback is attempted; inspect unconfirmed device state before retrying.

For any app with configured settings, deployment patches its matching `clock.screens.<name>`, `clock.services.<name>`, or `clock.on_demand.<name>` table through the script-settings API and checks the result. This is separate from the atomic source update. The catalog checks lifecycle metadata and basic setting types before device writes. AWTRIX and the app validate domain constraints.

`clock-show` accepts rotating screens. `clock-start` accepts on-demand apps, creates a fresh instance, and takes over the display until exited; it does not override a power policy. `clock-stop` only stops the named currently running on-demand session, returning to normal rotation. Services have no display page and cannot be shown. Deployment does not automatically start on-demand apps.

Deployment does not explicitly switch the current screen. `clock-show` selects an app, allows `draw()` to execute, and checks its error status again. Later callbacks can still fail; this is not a full visual or hardware test. Normal rotation continues afterwards. AWTRIX compiles the source; no local Berry compiler is needed.

To deploy scripts from Docker, mount the repository read-only so the config and source are available:

```sh
# Include clockctl in tools.apps before building its image.
make docker-build GROUP=tools
docker run --rm --workdir /repo \
  --mount "type=bind,source=$PWD,target=/repo,readonly" \
  tc002-apps/tools/clockctl:local list
```

Replace `list` with `deploy` or `show --app mackerel`. Inside Docker, `localhost` means the container itself; use a reachable physical-clock address or the appropriate container network address. The utility is not a production Compose service; see [utility apps](../apps/tools/README.md).

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
