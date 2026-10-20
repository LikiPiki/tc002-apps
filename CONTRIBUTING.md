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

1. Add the entry point at `cmd/<name>/main.go`. Use a directory name containing only letters, digits, hyphens, or underscores.
2. Put reusable or app-specific processing logic in `internal/` and add meaningful tests for its behavior.
3. Document configuration, flags, data sources, and whether the command is one-off or long-running.
4. Build it with `make build` or `make docker-build`. If you have a local `configs/apps.list`, add the app to it or remove the config to build all apps.
5. For managed deployment, add a service in `deploy/compose.yaml` using build argument `APP: <name>` and image `tc002-apps/<name>:local`. Use the `tools` profile for one-off utilities. Add a service in `dev/compose.yaml` if the app needs a containerized local integration workflow.

CI currently builds the Go binaries and the Docker services declared in production Compose. Add the Compose service when adding a deployable app so its image is covered too.

## Adding a Berry app

1. Add `apps/<name>/main.be` and a `README.md` describing its purpose, supported AWTRIX version, settings, and expected input data.
2. Place shared icons in `assets/icons/`, with source and license information.
3. Run `make dev`, upload the script through the Scripts tab, and inspect its output and runtime errors.
4. Check hardware-dependent features on a physical TC002. The headless environment does not emulate all hardware.

## Before opening a pull request

- Run `make check` and `make build` for Go changes.
- Build affected Docker images when changing container or deployment files.
- Run `make dev-check` when changing the HTTP integration or development environment.
- Update the relevant guides and examples when behavior or configuration changes.
- Keep actual API credentials, `deploy/.env`, and local app-selection settings out of commits.

Open a branch and submit a pull request describing the change and how you verified it. CI checks Go formatting, vet, tests, native builds, production Compose configuration, and production image builds. It does not currently run the virtual-clock integration or automatically test Berry apps.
