package config

type StatusConfig struct {
	URL      string
	App      string
	Label    string
	Interval string
	Timeout  string
	TTL      string
	Attempts int
}

type statusOverrides struct {
	URL      *string `toml:"url"`
	App      *string `toml:"app"`
	Label    *string `toml:"label"`
	Interval *string `toml:"interval"`
	Timeout  *string `toml:"timeout"`
	TTL      *string `toml:"ttl"`
	Attempts *int    `toml:"attempts"`
}

func (overrides statusOverrides) apply(config *StatusConfig) {
	fields := []struct {
		value  *string
		target *string
	}{
		{overrides.URL, &config.URL},
		{overrides.App, &config.App},
		{overrides.Label, &config.Label},
		{overrides.Interval, &config.Interval},
		{overrides.Timeout, &config.Timeout},
		{overrides.TTL, &config.TTL},
	}
	for _, field := range fields {
		if field.value != nil {
			*field.target = *field.value
		}
	}
	if overrides.Attempts != nil {
		config.Attempts = *overrides.Attempts
	}
}
