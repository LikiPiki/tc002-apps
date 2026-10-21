# Weather Now

A 52×16 current-weather screen: location label, temperature in Celsius, wind in m/s, and a small condition/day/night icon. It runs for ten seconds in normal rotation.

The VPS `weather-provider` supplies script data; the screen does not contact Open-Meteo or manage display power. Freshness is based on the source observation timestamp. Missing data shows `NO DATA`; expired data or unknown clock time shows `STALE`.

Install with `make clock-deploy APP=weather-current`. See the [weather integration](../../../../docs/integrations/weather.md) for location setup and provider commands. Run `make clock-check APP=weather-current` for a runtime/GIF check or `make weather-check` for the complete producer/consumer scenarios.
