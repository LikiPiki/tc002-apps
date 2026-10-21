# Contributing

Issues and pull requests are welcome. Use English for documentation, comments, examples, and contribution descriptions.

## Development setup

Clone the repository and follow the [local development guide](docs/development.md). For native Go work, run:

```sh
make check
make build
```

To exercise the HTTP integration with a virtual clock:

```sh
make dev-check
# Inspect the display at http://localhost:8080.
make dev-down
```

The first virtual-clock build downloads and compiles AWTRIX NG and can take several minutes.

## Adding a server feature or Go utility

1. Add server functionality under `apps/server/internal/` and wire it into the existing `apps/server/cmd/server` process. Add independent utilities under `apps/tools/cmd/<name>/main.go`.
2. Put family-specific logic in that family's `internal/`. Root `internal/` is for shared API/config code, never utility-specific checking/discovery. Server applications must not import tools. Add meaningful behavior tests.
3. Document configuration, flags, data sources, and whether the command is one-off or long-running.
4. Use `make build-server` / `make docker-build-server` for the complete server, or `make build-tools` / `make docker-build-tools` for utilities. `tools.apps` controls utility artifacts only.
5. All server features ship in `tc002-apps/server:local` and run through the one production `server` service. Do not add per-feature Docker images, Compose services, selection flags or generated server build sets.

CI checks all Go packages, builds the complete server plus selected utilities, and validates production and dev Compose. Run `make server-check` when changing server data processing or clock-consumer integration. Screen/VPS features should have a documented [integration contract](docs/integrations.md) and cross-linked component READMEs.

## Adding a Berry app

1. Choose `screens`, `services`, or `on-demand` and add `apps/clock/<kind>/<name>/main.be` with a README describing purpose, AWTRIX compatibility, settings, and inputs. Services require `# @headless true`; on-demand tools require `# @ondemand`. Ordinary screens use neither marker. Names must be unique across categories.
2. Place shared icons in `apps/clock/assets/icons/`, with source and license information. Keep app-specific assets alongside the app.
3. Run `make dev`, then `make clock-deploy APP=<name> LOCAL_CONFIG=` and `make clock-show APP=<name> LOCAL_CONFIG=`. Keep `AWTRIX_URL` unset so these commands use the shared local URL. Inspect the live output and runtime errors. You can also use the Scripts tab.
4. Check hardware-dependent features on a physical TC002. The headless environment does not emulate all hardware.

Run `make clock-check APP=<name>` for an isolated runtime smoke check before uploading to physical hardware. It checks installation and a short period of drawing without changing your regular dev instance. It requires Docker, not a host Go installation. See [runtime checks](docs/development.md#runtime-checks).

Register per-app settings under `clock.<category>.<name>` in TOML (`on_demand` for that category) and declare matching `@config` fields in the Berry header. `make clock-validate` checks declarations, types, defaults, text lengths, options, numeric bounds and RGB colors without contacting a clock. Numeric values outside declared bounds are rejected locally instead of relying on device clamping; `step` is a UI increment, not a quantization rule. Add app-domain validation in the catalog and retain validation in Berry for settings edited on-device. See [architecture](docs/architecture.md).

## Before opening a pull request

Keep code readable before running a formatter: one operation per line, blank lines between logical steps, descriptive names for domain values, and multiline literals for nontrivial structs and test cases. Split long functions by responsibility and handle request-decoding errors in test servers. Go code must pass `gofmt`; Berry code uses two-space indentation and multiline methods.

- Run `make check` and `make build` for Go changes.
- Build affected Docker images when changing container or deployment files.
- Run `make dev-check` when changing the HTTP integration or development environment.
- Update the relevant guides and examples when behavior or configuration changes.
- Keep personal settings and actual credentials in the ignored `configs/config.local.toml`, not in commits. All code reads the [same project TOML](docs/configuration.md).

Open a branch and submit a pull request describing the change and how you verified it. CI checks Go formatting, vet, tests, native builds, Compose configuration, isolated server/tool image builds, and Berry runtime smoke checks in a cached, disposable AWTRIX. It does not evaluate visual quality or physical-device behavior. A first cold AWTRIX build is slower than cached runs.
