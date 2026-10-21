package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

type clockOverrides struct {
	URL      *string   `toml:"url"`
	URLs     *[]string `toml:"urls"`
	Profiles *[]string `toml:"profiles"`
	Deploy   struct {
		Screens  *[]string `toml:"screens"`
		Services *[]string `toml:"services"`
		OnDemand *[]string `toml:"on_demand"`
	} `toml:"deploy"`
	Screens  AppSettings `toml:"screens"`
	Services AppSettings `toml:"services"`
	OnDemand AppSettings `toml:"on_demand"`
}

func (fields clockOverrides) apply(clock *ClockConfig, path string) error {
	forms := 0
	for _, present := range []bool{fields.URL != nil, fields.URLs != nil, fields.Profiles != nil} {
		if present {
			forms++
		}
	}
	if forms > 1 {
		return fmt.Errorf("%s: use only one of clock.url, clock.urls or clock.profiles", path)
	}

	if fields.URL != nil {
		clock.SetTarget(*fields.URL)
	}
	if fields.URLs != nil {
		clock.SetTarget("")
		clock.URLs = append([]string{}, (*fields.URLs)...)
	}
	if fields.Profiles != nil {
		clock.SetTarget("")
		clock.Profiles = []string{}
		for _, reference := range *fields.Profiles {
			if !filepath.IsAbs(reference) {
				reference = filepath.Join(filepath.Dir(path), reference)
			}
			clock.Profiles = append(clock.Profiles, filepath.Clean(reference))
		}
	}

	for _, field := range []struct {
		source *[]string
		target *[]string
	}{
		{fields.Deploy.Screens, &clock.Deploy.Screens},
		{fields.Deploy.Services, &clock.Deploy.Services},
		{fields.Deploy.OnDemand, &clock.Deploy.OnDemand},
	} {
		if field.source != nil {
			*field.target = selection(*field.source)
		}
	}

	mergeSettings(clock.Screens, fields.Screens)
	mergeSettings(clock.Services, fields.Services)
	mergeSettings(clock.OnDemand, fields.OnDemand)

	return nil
}

// Targets resolves the list form while retaining compatibility with clock.url.
func (clock ClockConfig) Targets() []string {
	if clock.URLs != nil {
		return append([]string{}, clock.URLs...)
	}
	if clock.URL == "" {
		return nil
	}

	return []string{clock.URL}
}

// SetTarget applies a single-device flag/environment override to the whole set.
func (clock *ClockConfig) SetTarget(address string) {
	clock.URL = address
	clock.URLs = nil
	clock.Profiles = nil
}

func NormalizeAddress(address string) string {
	address = strings.TrimSpace(address)
	if address != "" && !strings.Contains(address, "://") {
		address = "http://" + address
	}

	return strings.TrimRight(address, "/")
}

// OverrideTarget retains a matching device profile's settings for explicit --url.
func (clock *ClockConfig) OverrideTarget(address string) error {
	if clock.Profiles != nil {
		profiles, err := clock.ResolveProfiles()
		if err != nil {
			return err
		}

		for _, profile := range profiles {
			for _, candidate := range profile.Clock.Targets() {
				if NormalizeAddress(candidate) == NormalizeAddress(address) {
					*clock = profile.Clock
					clock.SetTarget(address)
					return nil
				}
			}
		}
	}

	clock.SetTarget(address)

	return nil
}
