# Clock setup

This project targets the **Ulanzi TC002 running AWTRIX NG**. The stock Ulanzi firmware and AWTRIX 3 are not supported.

1. Install AWTRIX NG using the [official TC002 instructions](https://blueforcer.github.io/awtrix-ng/tc002/getting-started/tc002/).
2. Connect the clock to Wi-Fi and open its web UI.
3. Find the clock's IP address and check that it is reachable from the machine running the Go commands.
4. Send a notification:

   ```sh
   AWTRIX_URL=http://192.168.1.50 go run ./apps/tools/cmd/clockctl --text 'Hello TC002'
   ```

The initial client expects an HTTP API without authentication. If your setup requires authentication, client support must be added before use.

Add Berry apps from `apps/clock/` through the Scripts tab. The Go CLI in `apps/tools/` manages the device; processes in `apps/server/` supply external data. Neither Go family is uploaded to the clock. Reference: [AWTRIX NG scripting](https://blueforcer.github.io/awtrix-ng/guides/scripting/).

Run `make init-config` to create `configs/config.toml` once, then set your clock addresses under `[clock]`, for example `urls = ["192.168.1.50", "192.168.1.51"]`. Host-side templates are under `examples/configs/`; the runtime `configs/` directory stays untracked. Run `make clock-list` to inspect targets, `make clock-deploy` to upload selected apps, and `make clock-show APP=mackerel` to show one. Unreachable/non-clock addresses produce warnings while other clocks continue. See [configuration](configuration.md).
