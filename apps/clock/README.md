# Clock apps (Berry)

Everything in this directory targets the TC002 itself and runs inside the AWTRIX NG Berry interpreter.

Each app lives at `apps/clock/<name>/main.be`, alongside a `README.md` describing its purpose, supported AWTRIX version, settings, and input data. Keep app-specific assets with the app; shared icons belong in `assets/icons/` within this directory.

## Available apps

| App | Description |
| --- | --- |
| [cigar](cigar/) | An animated cigar with curling smoke, pulsing embers, and sparks |
| [seagull](seagull/) | A screenshot-derived upside-down seagull with kicking feet |
| [mackerel](mackerel/) | A photo-derived golden mackerel with a waving tail and swimming motion |

## Deploy from this repository

From the repository root:

```sh
make clock-list                 # Preview the target and selected scripts; no device access
make clock-deploy               # Upload all selected Berry apps
make clock-deploy APP=mackerel  # Upload only one app
make clock-show APP=mackerel    # Show an installed app and check for runtime errors
```

The shared `configs/config.toml` defaults to the local virtual clock and all Berry apps. Copy `configs/config.local.toml.example` to `configs/config.local.toml` and set `[clock] url` for your physical clock. The personal file is excluded from Git and Docker build context. See [configuration](../../docs/configuration.md) and [Berry deployment](../../docs/commands.md#berry-deployment).

You can also install apps through the Scripts tab or the HTTP examples in each app's README.

For local testing, run `make dev`, then `make clock-deploy LOCAL_CONFIG=` to ignore personal overrides. Keep `AWTRIX_URL` unset, open http://localhost:8080, and inspect the live preview. Hardware-dependent features still need testing on a physical TC002.

This directory does not contain Go services or a Docker runtime. See the [development guide](../../docs/development.md) and [contribution guide](../../CONTRIBUTING.md) for details.
