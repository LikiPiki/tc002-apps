package config

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Clock struct {
		URL      string
		Apps     []string
		Schedule DisplaySchedule
	}
	Server struct {
		Apps   []string
		Worker struct{ Interval string }
	}
}

type DisplaySchedule struct {
	Enabled bool
	On      string
	Off     string
}

func validateScheduleTime(value string) error {
	t, err := time.Parse("15:04", value)
	if err != nil || t.Format("15:04") != value {
		return fmt.Errorf("must use HH:MM between 00:00 and 23:59")
	}
	return nil
}

// Paths registers the same config flags for every command, including containers.
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
	var config Config
	config.Clock.URL = "http://localhost:8080"
	config.Clock.Schedule.On = "10:00"
	config.Clock.Schedule.Off = "20:00"
	config.Server.Worker.Interval = "1m"
	for index, path := range []string{base, personal} {
		if index == 1 && path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if index == 1 && errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return config, err
		}
		var fields struct {
			Clock struct {
				URL      *string   `toml:"url"`
				Apps     *[]string `toml:"apps"`
				Schedule struct {
					Enabled *bool   `toml:"enabled"`
					On      *string `toml:"on"`
					Off     *string `toml:"off"`
				} `toml:"schedule"`
			} `toml:"clock"`
			Server struct {
				Apps   *[]string `toml:"apps"`
				Worker struct {
					Interval *string `toml:"interval"`
				} `toml:"worker"`
			} `toml:"server"`
		}
		decoder := toml.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&fields); err != nil {
			return config, fmt.Errorf("%s: %w", path, err)
		}
		if fields.Clock.URL != nil {
			config.Clock.URL = *fields.Clock.URL
		}
		if fields.Clock.Schedule.Enabled != nil {
			config.Clock.Schedule.Enabled = *fields.Clock.Schedule.Enabled
		}
		if fields.Clock.Schedule.On != nil {
			config.Clock.Schedule.On = *fields.Clock.Schedule.On
		}
		if fields.Clock.Schedule.Off != nil {
			config.Clock.Schedule.Off = *fields.Clock.Schedule.Off
		}
		if fields.Clock.Apps != nil {
			config.Clock.Apps, err = selection(*fields.Clock.Apps)
			if err != nil {
				return config, fmt.Errorf("%s: clock.apps: %w", path, err)
			}
		}
		if fields.Server.Apps != nil {
			config.Server.Apps, err = selection(*fields.Server.Apps)
			if err != nil {
				return config, fmt.Errorf("%s: server.apps: %w", path, err)
			}
		}
		if fields.Server.Worker.Interval != nil {
			config.Server.Worker.Interval = *fields.Server.Worker.Interval
		}
	}
	u, err := url.Parse(config.Clock.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" {
		return config, fmt.Errorf("clock.url must be an absolute HTTP(S) base URL without query or fragment")
	}
	interval, err := time.ParseDuration(config.Server.Worker.Interval)
	if err != nil || interval <= 0 {
		return config, fmt.Errorf("server.worker.interval must be a positive Go duration")
	}
	if err := validateScheduleTime(config.Clock.Schedule.On); err != nil {
		return config, fmt.Errorf("clock.schedule.on: %w", err)
	}
	if err := validateScheduleTime(config.Clock.Schedule.Off); err != nil {
		return config, fmt.Errorf("clock.schedule.off: %w", err)
	}
	if config.Clock.Schedule.On == config.Clock.Schedule.Off {
		return config, fmt.Errorf("clock.schedule.on and off must be different")
	}
	return config, nil
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func selection(names []string) ([]string, error) {
	if len(names) == 1 && names[0] == "*" {
		return nil, nil
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("use [\"*\"] for all apps or a nonempty list of names")
	}
	seen := make(map[string]bool)
	var selected []string
	for _, name := range names {
		if !namePattern.MatchString(name) {
			return nil, fmt.Errorf("invalid app name %q", name)
		}
		if !seen[name] {
			selected = append(selected, name)
			seen[name] = true
		}
	}
	return selected, nil
}

func ServerApps(root string, names []string) ([]string, error) {
	if names == nil {
		paths, err := filepath.Glob(filepath.Join(root, "*", "main.go"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			names = append(names, filepath.Base(filepath.Dir(path)))
		}
	}
	names, err := selection(names)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(root, name, "main.go")); err != nil {
			return nil, fmt.Errorf("server app %s: %w", name, err)
		}
	}
	return names, nil
}
