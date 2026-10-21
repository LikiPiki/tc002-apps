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

To deploy from the repository, copy `configs/config.local.toml.example` to `configs/config.local.toml` and set `urls` under `[clock]` to your clock addresses, for example `urls = ["192.168.1.50", "192.168.1.51"]`. Run `make clock-list` to check the targets, `make clock-deploy` to upload selected apps to all reachable AWTRIX devices, and `make clock-show APP=mackerel` to show one. Unreachable/non-clock addresses produce warnings while other clocks continue. Personal config is not committed. Notifications and VPS providers use the same list. See [configuration](configuration.md) and [Berry deployment](commands.md#berry-deployment).
