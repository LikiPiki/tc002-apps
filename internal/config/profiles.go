package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type ClockProfile struct {
	Name  string
	Clock ClockConfig
}

// ResolveProfiles reads only clock settings. Server settings remain global, so
// one VPS can fetch once and distribute the same data to different screen sets.
func (clock ClockConfig) ResolveProfiles() ([]ClockProfile, error) {
	if clock.Profiles == nil {
		name := clock.ProfileName
		if name == "" {
			name = "default"
		}

		return []ClockProfile{{Name: name, Clock: clock}}, nil
	}
	if len(clock.Profiles) == 0 {
		return nil, fmt.Errorf("no clock profiles configured")
	}

	var profiles []ClockProfile
	names := map[string]bool{}
	addresses := map[string]string{}

	for _, path := range clock.Profiles {
		profile, err := readProfile(path, clock)
		if err != nil {
			return nil, err
		}
		if names[profile.Name] {
			return nil, fmt.Errorf("duplicate clock profile name %q", profile.Name)
		}
		names[profile.Name] = true

		for _, address := range profile.Clock.Targets() {
			address = NormalizeAddress(address)
			if previous, exists := addresses[address]; exists && previous != profile.Name {
				return nil, fmt.Errorf("clock %s appears in profiles %s and %s", address, previous, profile.Name)
			}
			addresses[address] = profile.Name
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func readProfile(path string, defaults ClockConfig) (ClockProfile, error) {
	var profile ClockProfile
	data, err := os.ReadFile(path)
	if err != nil {
		return profile, fmt.Errorf("clock profile %s: %w", path, err)
	}

	var fields struct {
		Clock clockOverrides `toml:"clock"`
	}
	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return profile, fmt.Errorf("clock profile %s: %w", path, err)
	}
	if fields.Clock.Profiles != nil {
		return profile, fmt.Errorf("%s: nested clock profiles are not supported", path)
	}
	if fields.Clock.URL == nil && fields.Clock.URLs == nil {
		return profile, fmt.Errorf("%s: clock profile requires its own urls", path)
	}

	profile.Name = strings.TrimSuffix(filepath.Base(path), ".toml")
	profile.Name = strings.TrimSuffix(profile.Name, ".local")
	profile.Clock = defaults
	profile.Clock.SetTarget("")
	profile.Clock.ProfileName = profile.Name
	profile.Clock.Screens = cloneSettings(defaults.Screens)
	profile.Clock.Services = cloneSettings(defaults.Services)
	profile.Clock.OnDemand = cloneSettings(defaults.OnDemand)

	if err := fields.Clock.apply(&profile.Clock, path); err != nil {
		return profile, err
	}
	if len(profile.Clock.Targets()) == 0 {
		return profile, fmt.Errorf("%s: clock profile has no targets", path)
	}

	return profile, nil
}

func cloneSettings(source AppSettings) AppSettings {
	copy := AppSettings{}
	for name, fields := range source {
		copy[name] = map[string]any{}
		for key, value := range fields {
			copy[name][key] = value
		}
	}

	return copy
}

// ScreenSelected lets data providers match their consumers to each profile's
// declared screens without discovering or importing the Berry source catalog.
func (clock ClockConfig) ScreenSelected(name string) bool {
	if clock.Deploy.Screens == nil {
		return true
	}

	for _, selected := range clock.Deploy.Screens {
		if selected == name {
			return true
		}
	}

	return false
}
