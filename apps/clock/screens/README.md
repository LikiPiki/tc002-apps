# Rotating screens

Visual Berry apps that take turns in AWTRIX's normal rotation. Each app lives at `<name>/main.be` with its own README. These scripts must not declare `@headless true`, `@ondemand`, or `@module`.

Screens may have `loop()` and asynchronous callbacks of their own; a separate service is useful only for independent behavior or data shared by several screens.

| Screen | Description |
| --- | --- |
| [cigar](cigar/) | Animated smoke and glowing embers |
| [seagull](seagull/) | A seagull waving its feet |
| [mackerel](mackerel/) | A swimming pixel mackerel |

From the repository root:

```sh
make clock-list KIND=screens
make clock-deploy KIND=screens
make clock-show APP=mackerel
make clock-check KIND=screens
```

`clock.deploy.screens` selects uploads; `clock.screens.<name>` supplies app settings. Install names stay unchanged even if files move between directories.
