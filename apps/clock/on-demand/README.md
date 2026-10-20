# On-demand apps

Interactive tools and games that run only when explicitly started. They are installed in the AWTRIX device menu, not in the normal rotation. There are no on-demand apps bundled yet; this directory and the deployment commands support adding them.

Create `<name>/main.be` with a leading `# @ondemand` header and a `draw()` method. For example:

```berry
# @name My Tool
# @ondemand

class Tool
  def draw()
    clear()
    text(1, 6, "TOOL", 0x00FF88)
  end
end

return Tool()
```

## Lifecycle

- Deployment compiles and stores the source without starting it.
- Start from the clock's Scripts menu, the web UI, or `make clock-start APP=<name>`.
- Each start creates a fresh instance: `init()` and `setup()` run. The app owns the display while active; normal rotation waits.
- Loops, timers, and callbacks run only during that session. Settings and stored data persist across sessions.
- Hold select to exit, or run `make clock-stop APP=<name>`. Stopping ends the session and releases its instance memory; it does not uninstall the app.
- Starting an app does not override display-policy's power schedule.

```sh
make clock-list KIND=on-demand
make clock-deploy KIND=on-demand
make clock-start APP=<name>
make clock-stop APP=<name>
make clock-check KIND=on-demand
```

`clock.deploy.on_demand` selects uploads and `clock.on_demand.<name>` supplies settings. The local check starts each selected tool, observes frames/errors, then ends its session in the disposable sandbox. An empty category is a valid no-op.
