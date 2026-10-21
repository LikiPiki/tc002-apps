# Weather

One VPS process fetches [Open-Meteo](https://open-meteo.com/) weather and supplies two independent Berry screens over the AWTRIX HTTP API. The personal/noncommercial API requires no key. Data attribution: Open-Meteo.com, [CC BY 4.0](https://open-meteo.com/en/terms).

## Components

- `apps/server/internal/weatherprovider`: fetches, validates and publishes weather inside the complete `apps/server/cmd/server` process.
- `apps/clock/screens/weather-current`: current temperature, wind speed, location label and a day/night/condition icon.
- `apps/clock/screens/weather-forecast`: three daily forecasts, cycling every three seconds during a nine-second turn. Each page shows a weekday, precipitation probability and minimum/maximum temperature.

The two screens participate in the normal app rotation independently. Deploy both or select only the screen you need. Neither the provider nor the screens change display power or force an active app.

## Configure your location

Add your coordinates to `configs/config.local.toml`. These are an example for central Saint Petersburg, not an automatically detected location:

```toml
[clock]
urls = ["192.168.1.50", "192.168.1.51"]

[server.weather]
latitude = 59.9386
longitude = 30.3141
label = "SPB"
timezone = "Europe/Moscow"
screens = ["weather-current", "weather-forecast"]
```

Both coordinates are required; zero is valid. `label` accepts 1-8 ASCII letters/digits/spaces/underscores/hyphens for the small display font. `timezone = "auto"` uses the coordinates' timezone; an explicit IANA timezone also works. Timezone data is embedded in the provider, so validation and date handling work in minimal containers and across DST changes.

Defaults are `interval = "15m"`, `timeout = "15s"`, `ttl = "2h"`, `attempts = 3`. The TTL must exceed interval + timeout + the source's 15-minute update period. The interval must be at least one minute. The regular interval starts after an update attempt finishes.

## Look at weather on the VPS

```sh
go run ./apps/server/cmd/server --print
```

This fetches one reading and prints formatted JSON without connecting to any clock. It includes temperature, feels-like temperature, humidity, wind, WMO condition code, day/night state and three future daily forecasts. Units are Celsius, metres per second and percent.

## Publish to the clock

```sh
make clock-deploy APP=weather-current
make clock-deploy APP=weather-forecast

# Fetch and publish once:
go run ./apps/server/cmd/server --once

# Show either installed screen:
make clock-show APP=weather-current
make clock-show APP=weather-forecast

# Run continuously on the VPS:
sh deploy/deploy.sh
docker compose -f deploy/compose.yaml logs -f server
```

`screens` names the installed consumers on each clock. For a current-only installation, set `screens = ["weather-current"]`. One VPS fetches weather once per cycle and sends the same reading to every device in `[clock] urls`. Deployment selections and running Compose services remain independent of the Go build selection. `AWTRIX_URL` selects one target instead of the list when publishing; it is ignored by `--print`.

For different clock screen sets, list any number of profile files under `[clock] profiles` in the main config. Profiles contain only clock settings; weather coordinates, source and polling settings remain common. Delivery automatically intersects `server.weather.screens` with each profile's `clock.deploy.screens`. A profile selecting only `weather-current` receives current-weather data without attempting to write `weather-forecast`. Profiles with no selected weather consumers are skipped entirely. See [clock profiles](../configuration.md#different-clock-configurations-from-the-main-config).

The clock's configured HTTP address must be reachable from the VPS. If it is on your home LAN, provide routing/VPN access between the VPS and that network.

For the regular dev clock, deploy the screens to the local instance and run `make dev-server`, or `docker compose -p tc002-dev -f dev/compose.yaml run --rm --build server --once` for one update. The complete server reads your location and overrides only the clock URL with the internal virtual-clock address.

For a local server connected to physical clocks on your LAN, use `make lan`, `make lan-logs`, `make lan-once`, and `make lan-down`. This mode reads `[clock] urls` and personal weather settings. Your VPS may have a different list in its own personal config. See [local server debugging](../development.md#local-server-with-physical-clocks).

## Contract and freshness

The producer writes the same versioned object to each selected consumer using `PATCH /api/v1/apps/<name>/data`:

```json
{
  "weather": {
    "version": 1,
    "label": "SPB",
    "timezone": "Europe/Moscow",
    "updated_at_ms": "1792584000000",
    "ttl_seconds": 7200,
    "current": {
      "temperature_c": 12.5,
      "feels_like_c": 11,
      "humidity_percent": 65,
      "wind_speed_ms": 3.2,
      "weather_code": 1,
      "condition": "CLEAR",
      "is_day": true
    },
    "days": [
      {
        "date": "2026-10-22",
        "weekday": "Thu",
        "minimum_c": 5,
        "maximum_c": 13,
        "precipitation_probability_percent": 20,
        "weather_code": 3,
        "condition": "CLOUD"
      }
    ]
  }
}
```

The shortened example shows one forecast entry; actual readings contain three, starting with tomorrow in the selected timezone. Precipitation probability includes snow as well as rain. Conditions are `CLEAR`, `CLOUD`, `FOG`, `DRIZZLE`, `RAIN`, `SNOW` and `STORM`.

`updated_at_ms` is the **source observation time**, not the time a response was fetched. It is a string to avoid 32-bit Berry integer overflow. Expired or implausibly future source readings, missing/null values, wrong units and incomplete daily arrays are rejected before publication. A source error does not replace the last known weather with invented zero values.

Before the first reading, screens show `NO DATA`. Expired readings, future timestamps or an unsynchronized device clock show `STALE`. The top-right pixel is green for fresh weather and amber otherwise. Stored readings survive restarts; expiry still applies. Synchronize the VPS and AWTRIX clocks.

Network errors, HTTP 429 and 5xx retry with bounded exponential backoff inside one total source timeout. Other HTTP errors and invalid payloads do not retry immediately. Continuous mode logs errors and tries again at the next interval. Publication identifies every target as AWTRIX first, attempts every selected screen independently per clock and warns about skipped/failed devices. `--once` succeeds if at least one clock receives its screens, and fails if none do or the source fetch fails. `--print` has no device dependency. Previously skipped clocks are retried next cycle.

## Checks

```sh
make check
make server-check
make clock-check APP=weather-current LOCAL_CONFIG=
make clock-check APP=weather-forecast LOCAL_CONFIG=
```

`server-check` runs an isolated AWTRIX and a local Open-Meteo-shaped HTTP fixture. It exercises server data processing and both Berry screens, including initial/fresh/stale/unsupported-data states and forecast page rotation. It does not require an API key, internet weather access or physical hardware. CI runs the same integration plus Go tests for retries, timeout, null values, units and DST dates.
