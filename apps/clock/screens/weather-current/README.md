# Weather Now

A single-page 52×16 daily weather summary. The top row shows the current temperature (white), a person with the feels-like temperature (amber), and a full wind icon with speed in m/s (green). The person and feels-like reading sit close to the wind group; spacing adapts to negative values.

The bottom row shows today's minimum (blue) and maximum (amber), with a large 13-pixel condition icon between them. The thin blue-to-amber range gauge has a white current-temperature marker, clamped to the forecast range; a constant-temperature day uses the midpoint. Temperatures are rounded Celsius values. No precipitation percentage or future-day pages are shown. The screen runs for ten seconds in normal rotation.

The unified VPS server supplies wttr.in weather data, including the `today` forecast. The screen does not contact the weather source or manage display power. Freshness is based on the source observation timestamp. Missing data shows `NO DATA`; expired data or unknown clock time shows `STALE`.

Install with `make clock-deploy APP=weather-current`. See the [weather integration](../../../../docs/integrations/weather.md) for location setup and server commands. Run `make clock-check APP=weather-current` for a runtime/GIF check or `make server-check` for the complete producer/consumer scenarios.

When upgrading from the old paged forecast, remove `weather-forecast` from your local deployment and weather consumer selections, and disable or delete that old script on the clock. The bundled weather catalog now contains only `weather-current`.
