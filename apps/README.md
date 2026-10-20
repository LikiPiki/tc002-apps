# On-device apps

Each app lives in its own directory: `apps/<name>/main.be`, alongside a `README.md` describing its settings and input data.

Apps run in the Berry interpreter embedded in AWTRIX NG. This directory is currently a placeholder; the first app will target a specific use case and firmware version.

Install apps through the Scripts tab in the clock's web UI. Automated upload through `clockctl` is planned but not implemented yet.

To test without hardware, run `make dev` from the repository root and open http://localhost:8080. Upload the script in the Scripts tab and inspect the live display. See the [development guide](../docs/development.md) and [contribution guide](../CONTRIBUTING.md) for details.
