# Service status

A complete HTTP health integration: a VPS provider checks one endpoint and updates a Berry screen over the AWTRIX HTTP API. It needs no API keys or broker.

## Components and ownership

- Producer: `apps/server/cmd/status-provider`.
- Consumer: `apps/clock/screens/service-status/main.be`.
- Delivery: `PATCH /api/v1/apps/<server.status.app>/data` on each device in `[clock] urls`.
- The provider owns health data only. It never changes display power or the active app.
- The display policy remains the sole automatic power controller.

## Run it

Configure the clock URL and health endpoint in `configs/config.local.toml`:

```toml
[clock]
urls = ["192.168.1.50", "192.168.1.51"]

[server.status]
url = "https://example.com/health"
label = "SITE"
```

```sh
make clock-deploy APP=service-status
go run ./apps/server/cmd/status-provider --once
make clock-show APP=service-status

# Continuous VPS process:
sh deploy/deploy.sh status-provider
```

`server.status.app` defaults to `service-status`; it must match the installed consumer name. All transport addresses live in TOML. The source endpoint must be reachable from the VPS/container. A healthy service returns 2xx. Other HTTP statuses and network failures publish `down`. Redirects follow Go's normal HTTP policy. No response body is stored.

## Version 1 contract

```json
{
  "status": {
    "version": 1,
    "state": "up",
    "label": "SITE",
    "updated_at_ms": "1792576800000",
    "ttl_seconds": 90,
    "latency_ms": 12,
    "http_status": 200
  }
}
```

- `state`: `up` or `down`.
- `label`: 1-8 ASCII letters/digits/spaces/underscores/hyphens.
- `updated_at_ms`: UTC Unix milliseconds as a string, avoiding 32-bit Berry integer overflow.
- `ttl_seconds`: validity measured from the probe completion time.
- `latency_ms`: total probe duration, including retries.
- `http_status`: last observed status, or 0 if no response arrived.

The consumer initially shows `WAIT / NO DATA`, green for fresh `UP`, red for fresh `DOWN`, and amber `STALE` after expiry, with unknown clock time or a future timestamp. Unknown payload versions remain `NO DATA`. Device and VPS clocks should be synchronized. AWTRIX persists the latest reading; expiry continues across device reboots.

## Cadence and failure handling

Defaults: `interval = "30s"`, `timeout = "5s"`, `ttl = "90s"`, `attempts = 3`. The interval starts after each publication. TTL must exceed interval + probe timeout. Durations are bounded to 1 second through 24 hours; attempts to 1-5. Network errors/5xx retry within one total timeout using bounded exponential backoff. 4xx do not retry.

The service probes its health endpoint once per cycle, then identifies and publishes to every configured AWTRIX clock. Unreachable/non-clock addresses are skipped with warnings and checked again next cycle; other clocks continue updating. `--once` exits nonzero only if no clock was updated (or the source operation failed). A monitored service being down is a valid reading, not a provider failure.

## End-to-end verification

`make status-check` builds the real producer test runner and a disposable, port-free AWTRIX. It checks the initial state, probes a local HTTP fixture, publishes real API data and verifies the screen's status pixels for healthy, failed and expired readings. No physical clock or external endpoint is used. CI runs the same check.
