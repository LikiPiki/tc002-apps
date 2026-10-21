package config

type WeatherConfig struct {
	Latitude  *float64
	Longitude *float64
	Label     string
	Timezone  string
	Screens   []string
	Interval  string
	Timeout   string
	TTL       string
	Attempts  int
}

type weatherOverrides struct {
	Latitude  *float64  `toml:"latitude"`
	Longitude *float64  `toml:"longitude"`
	Label     *string   `toml:"label"`
	Timezone  *string   `toml:"timezone"`
	Screens   *[]string `toml:"screens"`
	Interval  *string   `toml:"interval"`
	Timeout   *string   `toml:"timeout"`
	TTL       *string   `toml:"ttl"`
	Attempts  *int      `toml:"attempts"`
}

func (overrides weatherOverrides) apply(config *WeatherConfig) {
	if overrides.Latitude != nil {
		config.Latitude = overrides.Latitude
	}
	if overrides.Longitude != nil {
		config.Longitude = overrides.Longitude
	}
	if overrides.Screens != nil {
		config.Screens = *overrides.Screens
	}
	if overrides.Attempts != nil {
		config.Attempts = *overrides.Attempts
	}

	fields := []struct {
		value  *string
		target *string
	}{
		{overrides.Label, &config.Label},
		{overrides.Timezone, &config.Timezone},
		{overrides.Interval, &config.Interval},
		{overrides.Timeout, &config.Timeout},
		{overrides.TTL, &config.TTL},
	}

	for _, field := range fields {
		if field.value != nil {
			*field.target = *field.value
		}
	}
}
