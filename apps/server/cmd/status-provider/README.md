# HTTP status provider

Checks `server.status.url` once per cycle and publishes a versioned reading to `service-status` on every device in `[clock] urls`. Configure it under `[server.status]` in project TOML. `AWTRIX_URL` selects one clock instead of the list. Unknown/unreachable devices produce warnings while other clocks continue updating.

```sh
go run ./apps/server/cmd/status-provider --once
sh deploy/deploy.sh status-provider
```

See the [integration guide](../../../../docs/integrations/service-status.md) for the full configuration, contract, bounded retries, freshness semantics, setup and `make status-check` verification.
