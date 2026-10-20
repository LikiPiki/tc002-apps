# Background services

Autonomous Berry apps marked `# @headless true`. They have no display page; setup, loops, timers, and callbacks run while enabled, including while the matrix is dark. Each service lives at `<name>/main.be` with a README.

| Service | Responsibility |
| --- | --- |
| [display-policy](display-policy/) | Owns the daily display power schedule |

```sh
make clock-list KIND=services
make clock-deploy KIND=services
make clock-check KIND=services
```

Use `clock.deploy.services` to select uploads and `clock.services.<name>` for settings. Do not use `clock-show` for a service. Disabling a service in AWTRIX stops its background work; turning off the display does not.

Keep independent responsibilities separate and give each managed resource one owner. For example, schedule, manual overrides, and future night-brightness rules belong in the display controller; a weather fetch failure should not stop morning wake-up.
