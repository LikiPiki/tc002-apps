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

type AppSettings map[string]map[string]any

type ClockDeploy struct {
	Screens  []string
	Services []string
	OnDemand []string
}

type ClockConfig struct {
	URL      string
	Deploy   ClockDeploy
	Screens  AppSettings
	Services AppSettings
	OnDemand AppSettings
}

type Config struct {
	Clock  ClockConfig
	Server struct {
		Apps   []string
		Worker struct{ Interval string }
	}
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
	for index, path := range []string{base, personal} {
		if index == 1 && path == "" {
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
			Clock struct {
				URL    *string `toml:"url"`
				Deploy struct {
					Screens  *[]string `toml:"screens"`
					Services *[]string `toml:"services"`
					OnDemand *[]string `toml:"on_demand"`
				} `toml:"deploy"`
				Screens  AppSettings `toml:"screens"`
				Services AppSettings `toml:"services"`
				OnDemand AppSettings `toml:"on_demand"`
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
			return c, fmt.Errorf("%s: %w", path, err)
		}
		if fields.Clock.URL != nil {
			c.Clock.URL = *fields.Clock.URL
		}
		for _, field := range []struct {
			name string
			src  *[]string
			dst  *[]string
		}{
			{"screens", fields.Clock.Deploy.Screens, &c.Clock.Deploy.Screens},
			{"services", fields.Clock.Deploy.Services, &c.Clock.Deploy.Services},
			{"on_demand", fields.Clock.Deploy.OnDemand, &c.Clock.Deploy.OnDemand},
		} {
			if field.src == nil {
				continue
			}
			if len(*field.src) == 0 {
				*field.dst = []string{}
			} else if *field.dst, err = selection(*field.src); err != nil {
				return c, fmt.Errorf("%s: clock.deploy.%s: %w", path, field.name, err)
			}
		}
		for _, field := range []struct {
			name string
			src  AppSettings
			dst  AppSettings
		}{
			{"screens", fields.Clock.Screens, c.Clock.Screens},
			{"services", fields.Clock.Services, c.Clock.Services},
			{"on_demand", fields.Clock.OnDemand, c.Clock.OnDemand},
		} {
			if err := mergeSettings(field.dst, field.src); err != nil {
				return c, fmt.Errorf("%s: clock.%s: %w", path, field.name, err)
			}
		}
		if fields.Server.Apps != nil {
			c.Server.Apps, err = selection(*fields.Server.Apps)
			if err != nil {
				return c, fmt.Errorf("%s: server.apps: %w", path, err)
			}
		}
		if fields.Server.Worker.Interval != nil {
			c.Server.Worker.Interval = *fields.Server.Worker.Interval
		}
	}
	u, err := url.Parse(c.Clock.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" {
		return c, fmt.Errorf("clock.url must be an absolute HTTP(S) base URL without query or fragment")
	}
	interval, err := time.ParseDuration(c.Server.Worker.Interval)
	if err != nil || interval <= 0 {
		return c, fmt.Errorf("server.worker.interval must be a positive Go duration")
	}
	return c, nil
}

func mergeSettings(dst, src AppSettings) error {
	for name, fields := range src {
		if !namePattern.MatchString(name) || len(name) > 32 {
			return fmt.Errorf("invalid app name %q", name)
		}
		if dst[name] == nil {
			dst[name] = map[string]any{}
		}
		for key, value := range fields {
			switch value.(type) {
			case string, bool, int64, float64:
				dst[name][key] = value
			default:
				return fmt.Errorf("%s.%s must be a scalar app setting", name, key)
			}
		}
	}
	return nil
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
