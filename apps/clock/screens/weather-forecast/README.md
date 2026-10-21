# Weather Forecast

A 52×16 forecast screen showing the next three days. It cycles through three-second pages during each nine-second rotation turn: weekday, precipitation probability, and minimum/maximum temperature in Celsius. Probability includes both rain and snow.

The VPS `weather-provider` supplies versioned script data. The screen performs no network requests and does not manage display power. Before data arrives it shows `NO DATA`; expired data or unknown device time shows `STALE`.

Install with `make clock-deploy APP=weather-forecast`. See the [weather integration](../../../../docs/integrations/weather.md) for setup, contract, and provider commands. `make weather-check` verifies publication, state handling and page rotation against the actual Berry runtime.
