# tc002-server

The complete server executable. All server functionality is compiled into this entry point and shipped in the one `tc002-apps/server` image.

```sh
go run ./apps/server/cmd/server          # Continuous updates
go run ./apps/server/cmd/server --once   # One update for all configured clocks
go run ./apps/server/cmd/server --print  # Inspect one reading without clock writes
```

Runtime values come from the shared/personal TOML files, including all referenced clock profiles. Weather processing lives under `apps/server/internal/weatherprovider`; additional server features belong under `internal/` and run in this same process. See [server builds](../../README.md), [deployment](../../../../docs/deployment.md) and [weather](../../../../docs/integrations/weather.md).
