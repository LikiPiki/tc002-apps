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

## Clock timezone

For Moscow and Saint Petersburg, add the device timezone to `configs/config.local.toml` (or to a clock profile):

```toml
[clock.system]
tzName = "Europe/Moscow"
tz = "MSK-3"
```

Run `make clock-deploy` to apply the settings along with the selected apps. Moscow time is UTC+3 year-round; the POSIX offset sign is reversed, hence `MSK-3`. You can also select **Europe/Moscow** in the clock's web UI under the system time settings.

The device timezone controls the built-in time/date screens and local schedules. The separate `[server.weather] timezone` setting controls weather dates; it does not configure the clock's timezone. Set it to `"Europe/Moscow"` in your personal weather config for the same local dates. If the clock still shows the wrong time, check its NTP server and network access in the web UI.

## Clock apps

Add Berry apps from `apps/clock/` through the Scripts tab. The Go CLI in `apps/tools/` manages the device; processes in `apps/server/` supply external data. Neither Go family is uploaded to the clock. Reference: [AWTRIX NG scripting](https://blueforcer.github.io/awtrix-ng/guides/scripting/).

Run `make init-config` to create `configs/config.toml` once, then set your clock addresses under `[clock]`, for example `urls = ["192.168.1.50", "192.168.1.51"]`. Host-side templates are under `examples/configs/`; the runtime `configs/` directory stays untracked. Run `make clock-list` to inspect targets, `make clock-deploy` to upload selected apps, and `make clock-show APP=mackerel` to show one. Unreachable/non-clock addresses produce warnings while other clocks continue. See [configuration](configuration.md).
