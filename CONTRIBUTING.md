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

## Adding a Go app

1. Add the entry point at `apps/server/cmd/<name>/main.go`. These programs run on a VPS or computer, not on the clock. Use a directory name containing only letters, digits, hyphens, or underscores.
2. Put reusable or app-specific processing logic in `apps/server/internal/` and add meaningful tests for its behavior.
3. Document configuration, flags, data sources, and whether the command is one-off or long-running.
4. Build it with `make build` or `make docker-build`. If your personal TOML restricts `server.apps`, add the app or use `["*"]` to build all commands.
5. For managed deployment, add a service in `deploy/compose.yaml` using build argument `APP: <name>` and image `tc002-apps/<name>:local`. Use the `tools` profile for one-off utilities. Add a service in `dev/compose.yaml` if the app needs a containerized local integration workflow.

CI currently builds the Go binaries and the Docker services declared in production Compose. Add the Compose service when adding a deployable app so its image is covered too.

## Adding a Berry app

1. Choose `screens`, `services`, or `on-demand` and add `apps/clock/<kind>/<name>/main.be` with a README describing purpose, AWTRIX compatibility, settings, and inputs. Services require `# @headless true`; on-demand tools require `# @ondemand`. Ordinary screens use neither marker. Names must be unique across categories.
2. Place shared icons in `apps/clock/assets/icons/`, with source and license information. Keep app-specific assets alongside the app.
3. Run `make dev`, then `make clock-deploy APP=<name> LOCAL_CONFIG=` and `make clock-show APP=<name> LOCAL_CONFIG=`. Keep `AWTRIX_URL` unset so these commands use the shared local URL. Inspect the live output and runtime errors. You can also use the Scripts tab.
4. Check hardware-dependent features on a physical TC002. The headless environment does not emulate all hardware.

Run `make clock-check APP=<name>` for an isolated runtime smoke check before uploading to physical hardware. It checks installation and a short period of drawing without changing your regular dev instance. It requires Docker, not a host Go installation. See [runtime checks](docs/development.md#runtime-checks).

Register per-app settings under `clock.<category>.<name>` in TOML (`on_demand` for that category) and declare matching `@config` fields in the Berry header. The generic CLI validates names/basic types and passes settings to AWTRIX; domain validation belongs in the app. Add tests when changing the catalog or lifecycle rules. See [architecture](docs/architecture.md).

## Before opening a pull request

- Run `make check` and `make build` for Go changes.
- Build affected Docker images when changing container or deployment files.
- Run `make dev-check` when changing the HTTP integration or development environment.
- Update the relevant guides and examples when behavior or configuration changes.
- Keep personal settings and actual credentials in the ignored `configs/config.local.toml`, not in commits. All code reads the [same project TOML](docs/configuration.md).

Open a branch and submit a pull request describing the change and how you verified it. CI checks Go formatting, vet, tests, native builds, production Compose configuration, and production image builds. It does not currently run the virtual-clock integration or automatically test Berry apps.
