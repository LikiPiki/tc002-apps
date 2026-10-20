# Clock apps (Berry)

Everything in this directory targets the TC002 itself and runs inside the AWTRIX NG Berry interpreter.

Each app lives at `apps/clock/<name>/main.be`, alongside a `README.md` describing its purpose, supported AWTRIX version, settings, and input data. Keep app-specific assets with the app; shared icons belong in `assets/icons/` within this directory.

## Available apps

| App | Description |
| --- | --- |
| [cigar](cigar/) | An animated cigar with curling smoke, pulsing embers, and sparks |
| [seagull](seagull/) | A screenshot-derived upside-down seagull with kicking feet |
| [mackerel](mackerel/) | A photo-derived golden mackerel with a waving tail and swimming motion |

Install apps through the Scripts tab in the clock's web UI or the HTTP API shown in the app's README. Automated upload through `clockctl` is planned but not implemented.

For local testing, run `make dev` from the repository root and open http://localhost:8080. Upload the script in the Scripts tab and inspect the live preview. Hardware-dependent features still need testing on a physical TC002.

This directory does not contain Go services or a Docker runtime. See the [development guide](../../docs/development.md) and [contribution guide](../../CONTRIBUTING.md) for details.
