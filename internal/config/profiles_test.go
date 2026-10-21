package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestClockProfilesInheritSettingsWithoutSharingMutableMaps(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.toml")
	fixture(t, main, `[clock]
profiles = ["clocks/first.local.toml", "clocks/second.local.toml"]
[clock.deploy]
screens = ["weather-current"]
services = ["display-policy"]
[clock.services.display-policy]
schedule_enabled = false
off = "20:00"
[server.weather]
label = "SPB"
`)
	fixture(t, filepath.Join(dir, "clocks", "first.local.toml"), `[clock]
urls = ["192.168.1.50"]
[clock.deploy]
screens = ["weather-current", "weather-forecast"]
services = []
`)
	fixture(t, filepath.Join(dir, "clocks", "second.local.toml"), `[clock]
urls = ["192.168.1.51"]
[clock.services.display-policy]
schedule_enabled = true
`)

	project, err := Load(main, "")
	if err != nil {
		t.Fatal(err)
	}

	profiles, err := project.Clock.ResolveProfiles()
	if err != nil {
		t.Fatal(err)
	}

	if len(profiles) != 2 || profiles[0].Name != "first" || profiles[1].Name != "second" {
		t.Fatalf("unexpected profiles: %+v", profiles)
	}

	first := profiles[0].Clock
	second := profiles[1].Clock
	if len(first.Deploy.Services) != 0 || !second.ScreenSelected("weather-current") ||
		!first.ScreenSelected("weather-forecast") || second.ScreenSelected("weather-forecast") {
		t.Fatalf("profile screen selections were not merged independently: %+v", profiles)
	}

	if first.Services["display-policy"]["schedule_enabled"] != false ||
		second.Services["display-policy"]["schedule_enabled"] != true ||
		second.Services["display-policy"]["off"] != "20:00" ||
		project.Clock.Services["display-policy"]["schedule_enabled"] != false {
		t.Fatal("profile overrides leaked into the defaults or another profile")
	}

	if project.Server.Weather.Label != "SPB" {
		t.Fatal("clock profiles changed shared VPS settings")
	}

	if err := project.Clock.OverrideTarget("http://192.168.1.51/"); err != nil {
		t.Fatal(err)
	}

	if project.Clock.ProfileName != "second" ||
		!reflect.DeepEqual(project.Clock.Targets(), []string{"http://192.168.1.51/"}) ||
		project.Clock.ScreenSelected("weather-forecast") {
		t.Fatal("explicit device override lost its matching profile")
	}
}

func TestProfilesRejectOverlappingDevicesAndVPSOverrides(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.toml")
	first := filepath.Join(dir, "first.toml")
	second := filepath.Join(dir, "second.toml")
	fixture(t, main, "[clock]\nprofiles = [\"first.toml\", \"second.toml\"]\n")
	fixture(t, first, "[clock]\nurls = [\"192.168.1.50\"]\n")
	fixture(t, second, "[clock]\nurls = [\"http://192.168.1.50/\"]\n")

	project, err := Load(main, "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := project.Clock.ResolveProfiles(); err == nil {
		t.Fatal("one clock was assigned conflicting profiles")
	}

	fixture(t, second, "[clock]\nurls = [\"192.168.1.51\"]\n[server.weather]\nlabel = \"OTHER\"\n")
	if _, err := project.Clock.ResolveProfiles(); err == nil {
		t.Fatal("clock profile was allowed to override the shared VPS weather")
	}

	fixture(t, second, "[clock]\nprofiles = [\"first.toml\"]\n")
	if _, err := project.Clock.ResolveProfiles(); err == nil {
		t.Fatal("nested profile references were accepted")
	}
}
