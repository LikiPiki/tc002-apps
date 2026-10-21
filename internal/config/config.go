package config

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type AppSettings map[string]map[string]any

type ClockDeploy struct {
	Screens  []string
	Services []string
	OnDemand []string
}

type ClockConfig struct {
	URL         string
	URLs        []string
	Profiles    []string
	ProfileName string
	Deploy      ClockDeploy
	Screens     AppSettings
	Services    AppSettings
	OnDemand    AppSettings
}

type Config struct {
	Clock  ClockConfig
	Server struct {
		Apps    []string
		Worker  struct{ Interval string }
		Weather WeatherConfig
	}
	Tools struct{ Apps []string }
}

func Paths(flags *flag.FlagSet) (base, personal *string) {
	basePath, ok := os.LookupEnv("TC002_CONFIG")
	if !ok {
		basePath = "configs/config.toml"
	}

	localPath, ok := os.LookupEnv("TC002_LOCAL_CONFIG")
	if !ok {
		localPath = "configs/config.local.toml"
	}

	return flags.String("config", basePath, "shared project TOML config"),
		flags.String("local-config", localPath, "optional personal TOML overrides; empty disables them")
}

func Load(base, personal string) (Config, error) {
	var c Config
	c.Clock.URL = "http://localhost:8080"
	c.Clock.Deploy.Services = []string{}
	c.Clock.Deploy.OnDemand = []string{}
	c.Clock.Screens, c.Clock.Services, c.Clock.OnDemand = AppSettings{}, AppSettings{}, AppSettings{}
	c.Server.Worker.Interval = "1m"

	c.Server.Weather = WeatherConfig{
		Label:    "HOME",
		Timezone: "auto",
		Screens:  []string{"weather-current", "weather-forecast"},
		Interval: "15m",
		Timeout:  "15s",
		TTL:      "2h",
		Attempts: 3,
	}

	for index, path := range []string{base, personal} {
		if path == "" {
			continue
		}

		data, err := os.ReadFile(path)
		if index == 1 && errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return c, err
		}

		var fields struct {
			Clock  clockOverrides `toml:"clock"`
			Server struct {
				Apps   *[]string `toml:"apps"`
				Worker struct {
					Interval *string `toml:"interval"`
				} `toml:"worker"`
				Weather weatherOverrides `toml:"weather"`
			} `toml:"server"`
			Tools struct {
				Apps *[]string `toml:"apps"`
			} `toml:"tools"`
		}

		decoder := toml.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&fields); err != nil {
			return c, fmt.Errorf("%s: %w", path, err)
		}

		if err := fields.Clock.apply(&c.Clock, path); err != nil {
			return c, err
		}

		if fields.Server.Apps != nil {
			c.Server.Apps = selection(*fields.Server.Apps)
		}
		if fields.Server.Worker.Interval != nil {
			c.Server.Worker.Interval = *fields.Server.Worker.Interval
		}

		fields.Server.Weather.apply(&c.Server.Weather)

		if fields.Tools.Apps != nil {
			c.Tools.Apps = selection(*fields.Tools.Apps)
		}
	}

	return c, nil
}

// WorkerInterval validates only worker runtime semantics, after explicit overrides.
func (c Config) WorkerInterval() (time.Duration, error) {
	interval, err := time.ParseDuration(c.Server.Worker.Interval)
	if err != nil || interval <= 0 {
		return 0, fmt.Errorf("server.worker.interval %q must be a positive Go duration", c.Server.Worker.Interval)
	}

	return interval, nil
}

func mergeSettings(dst, src AppSettings) {
	for name, fields := range src {
		if dst[name] == nil {
			dst[name] = map[string]any{}
		}

		for key, value := range fields {
			dst[name][key] = value
		}
	}
}

// Preserve invalid semantic values for the consumer to validate; parsing stays independent.
func selection(names []string) []string {
	if len(names) == 1 && names[0] == "*" {
		return nil
	}

	seen := make(map[string]bool)
	selected := make([]string, 0, len(names))
	for _, name := range names {
		if !seen[name] {
			selected = append(selected, name)
			seen[name] = true
		}
	}

	return selected
}
