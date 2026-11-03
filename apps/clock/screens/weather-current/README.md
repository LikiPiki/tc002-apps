# Weather Now

A single-page 52×16 weather screen. The top row shows a condition/day/night icon and the current temperature in Celsius. The bottom row shows today's minimum/maximum temperature on the left and precipitation probability on the right. It runs for ten seconds in normal rotation, without paging, a location label or wind readings.

The unified VPS server supplies weather data, including the `today` forecast; update the server together with this screen. The screen does not contact Open-Meteo or manage display power. Freshness is based on the source observation timestamp. Missing data shows `NO DATA`; expired data or unknown clock time shows `STALE`.

Install with `make clock-deploy APP=weather-current`. See the [weather integration](../../../../docs/integrations/weather.md) for location setup and server commands. Run `make clock-check APP=weather-current` for a runtime/GIF check or `make server-check` for the complete producer/consumer scenarios.
