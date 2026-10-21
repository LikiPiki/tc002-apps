# Clock apps by lifecycle

All code here runs on the TC002 inside AWTRIX NG. Choose the lifecycle before adding an app:

| Directory | Firmware marker | Behavior |
| --- | --- | --- |
| [screens/](screens/) | No special marker | Visual apps in normal rotation |
| [services/](services/) | `# @headless true` | Background controllers with no page |
| [on-demand/](on-demand/) | `# @ondemand` | Tools or games started from the menu or API |

Each app has `<kind>/<name>/main.be` and a README. The directory name is the install name and must be unique across all three categories. Deployment rejects mismatched lifecycle headers and duplicate names before contacting the clock.

## Selection and settings

One shared TOML and optional personal overrides control all types:

```toml
[clock.deploy]
screens = ["*"]
services = ["display-policy"]
on_demand = []

[clock.services.display-policy]
schedule_enabled = true
on = "10:00"
off = "20:00"
```

Selection and behavior are separate: putting a service in the deployment list installs it; its own settings decide what it does. Per-app settings use `clock.screens.<name>`, `clock.services.<name>`, or `clock.on_demand.<name>` and match the app's `@config` keys. The CLI applies them through the generic AWTRIX config API.

```sh
make clock-list
make clock-deploy KIND=screens
make clock-deploy APP=display-policy
make clock-show APP=mackerel
make clock-start APP=<tool>
make clock-stop APP=<tool>
make clock-check
```

For manual local preview, run `make dev`, then deploy with `CONFIG=configs/config.toml.example LOCAL_CONFIG=` and leave `AWTRIX_URL` unset. `clock-check` always uses its own disposable local target.

See [configuration](../../docs/configuration.md), [architecture](../../docs/architecture.md), and [commands](../../docs/commands.md).
