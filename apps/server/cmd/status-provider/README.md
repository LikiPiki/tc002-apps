# HTTP status provider

Checks `server.status.url` and publishes a versioned reading to the installed `service-status` Berry screen. Configure it under `[server.status]` in the shared/personal project TOML. The clock target uses `[clock] url` or `AWTRIX_URL`.

```sh
go run ./apps/server/cmd/status-provider --once
sh deploy/deploy.sh status-provider
```

See the [integration guide](../../../../docs/integrations/service-status.md) for the full configuration, contract, bounded retries, freshness semantics, setup and `make status-check` verification.
