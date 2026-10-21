# Weather provider

Fetches current conditions and three future daily forecasts from Open-Meteo, then publishes a compact versioned reading to `weather-current` and `weather-forecast` over HTTP.

```sh
go run ./apps/server/cmd/weather-provider --print # JSON only; no clock connection
go run ./apps/server/cmd/weather-provider --once # Publish once
sh deploy/deploy.sh weather-provider           # Continuous VPS service
```

Set `[server.weather] latitude` and `longitude` in personal TOML first. See the [weather integration](../../../../docs/integrations/weather.md) for all settings, screen installation, API attribution, data contract, freshness/failure behavior and local end-to-end checks.
