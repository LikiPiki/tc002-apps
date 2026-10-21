# Release builds

`clockctl --version` (or `clockctl version`) prints the version and source commit without loading configuration or accessing a clock. `make build` and `make docker-build` embed Git-derived metadata; `VERSION` and `COMMIT` can override it. Direct `go build` uses `dev` and, when available, Go's VCS metadata.

## Build locally

```sh
make release VERSION=v0.1.0
```

This cross-compiles standalone `clockctl` for Linux, macOS and Windows on amd64/arm64 with CGO disabled. Archives and relative SHA-256 checksums are written to `dist/release/<version>/`. Each archive contains the binary and repository license; Windows binaries have an `.exe` suffix. Go, Git and `tar` are required, plus either `sha256sum` or `shasum`. No cross-compilation toolchain or Docker is needed.

Extract an archive and put `clockctl` on your PATH. For example:

```sh
sha256sum --ignore-missing -c SHA256SUMS
tar -xzf clockctl-v0.1.0-linux-amd64.tar.gz
./clockctl --version
./clockctl doctor --url http://192.168.1.50
```

On macOS use `shasum -a 256 <archive>` and compare its digest with `SHA256SUMS`. For Berry deployment, clone this repository and use absolute `--config`, `--local-config` and `--apps-dir` paths when running outside the checkout.

## Publish

After CI passes for the chosen commit, pushing a `v*` tag triggers `.github/workflows/release.yml`. It reruns Go checks, builds all six archives, and publishes a GitHub Release with generated notes and checksums using the workflow token. Normal pushes to `main` run CI without creating releases.

CI checks the module's minimum Go 1.24 and the current stable Go release. Its Berry job checks every catalog app with `CHECK_ALL=true`, independent of the normal deploy selection, plus display-policy behavior and the weather producer/consumer integration. GIF previews are uploaded for visual inspection.
