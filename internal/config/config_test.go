package config

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func fixture(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPersonalOverridesMergeAcrossSections(t *testing.T) {
	dir := t.TempDir()
	base, personal := filepath.Join(dir, "config.toml"), filepath.Join(dir, "local.toml")

	fixture(t, base, `[clock]
url = "http://localhost:8080"
[clock.deploy]
screens = ["cigar"]
[server]
apps = ["worker"]
[server.worker]
interval = "1m"
`)
	fixture(t, personal, "[clock]\nurl = \"http://192.168.1.50\"\n[server.worker]\ninterval = \"30s\"\n")

	c, err := Load(base, personal)
	if err != nil {
		t.Fatal(err)
	}

	if c.Clock.URL != "http://192.168.1.50" ||
		c.Server.Worker.Interval != "30s" ||
		!reflect.DeepEqual(c.Clock.Deploy.Screens, []string{"cigar"}) ||
		!reflect.DeepEqual(c.Server.Apps, []string{"worker"}) {
		t.Fatalf("unexpected merged config: %+v", c)
	}

	fixture(t, personal, "[clock.deploy]\nscreens = [\"*\"]\nservices = []\non_demand = []\n")

	c, err = Load(base, personal)
	if err != nil ||
		c.Clock.Deploy.Screens != nil ||
		c.Clock.Deploy.Services == nil ||
		c.Clock.URL != "http://localhost:8080" ||
		len(c.Server.Apps) != 1 {
		t.Fatalf("wildcard must reset only clock selection: %+v, %v", c, err)
	}

	if _, err := Load(base, filepath.Join(dir, "missing.toml")); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(filepath.Join(dir, "missing.toml"), ""); err == nil {
		t.Fatal("missing base config accepted")
	}
}

func TestRejectInvalidConfig(t *testing.T) {
	for _, content := range []string{
		"[clock]\napps = []",
		"[clock]\napp = [\"cigar\"]",
		"[clock.deploy]\nscreens = \"cigar\"",
		"[server]\napps = true",
		"[clock]\nurl = [",
		"[server.worker]\ninterval = 12",
	} {
		path := filepath.Join(t.TempDir(), "config.toml")
		fixture(t, path, content)

		if _, err := Load(path, ""); err == nil {
			t.Errorf("accepted invalid config: %s", content)
		}
	}
}

func TestLoadDoesNotValidateUnrelatedConsumers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	fixture(t, path, "[clock]\nurl = \"not-a-url\"\n[clock.services.unused]\nsetting = [1,2]\n[server.worker]\ninterval = \"invalid\"\n")

	c, err := Load(path, "")
	if err != nil {
		t.Fatalf("semantic values must be left to consumers: %v", err)
	}

	if _, err := c.WorkerInterval(); err == nil {
		t.Fatal("worker accepted invalid interval")
	}

	c.Server.Worker.Interval = "30s"
	if interval, err := c.WorkerInterval(); err != nil || interval.String() != "30s" {
		t.Fatalf("valid worker config depends on unrelated clock config: %v, %v", interval, err)
	}
}

func TestAppSettingsMergeByField(t *testing.T) {
	dir := t.TempDir()
	base, personal := filepath.Join(dir, "base.toml"), filepath.Join(dir, "personal.toml")

	fixture(t, base, "[clock.services.display-policy]\nschedule_enabled = true\non = \"20:00\"\noff = \"02:00\"\n")
	fixture(t, personal, "[clock.services.display-policy]\nschedule_enabled = false\n")

	c, err := Load(base, personal)
	settings := c.Clock.Services["display-policy"]
	if err != nil || settings["schedule_enabled"] != false || settings["on"] != "20:00" || settings["off"] != "02:00" {
		t.Fatalf("unexpected settings: %+v, %v", settings, err)
	}
}

func TestConfigPathEnvironmentAndExplicitOverride(t *testing.T) {
	t.Setenv("TC002_CONFIG", "custom.toml")
	t.Setenv("TC002_LOCAL_CONFIG", "")

	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	base, personal := Paths(flags)
	if err := flags.Parse([]string{"--config", "explicit.toml"}); err != nil {
		t.Fatal(err)
	}

	if *base != "explicit.toml" || *personal != "" {
		t.Fatalf("unexpected paths: %q, %q", *base, *personal)
	}
}

func TestToolSelectionDoesNotChangeServerSelection(t *testing.T) {
	dir := t.TempDir()
	base, personal := filepath.Join(dir, "base.toml"), filepath.Join(dir, "personal.toml")

	fixture(t, base, "[server]\napps = [\"worker\"]\n[tools]\napps = [\"clockctl\"]\n")
	fixture(t, personal, "[tools]\napps = []\n")

	c, err := Load(base, personal)
	if err != nil || c.Tools.Apps == nil || len(c.Tools.Apps) != 0 || !reflect.DeepEqual(c.Server.Apps, []string{"worker"}) {
		t.Fatalf("tool selection changed server selection: %+v, %v", c, err)
	}

	fixture(t, personal, "[tools]\napps = [\"*\"]\n")

	c, err = Load(base, personal)
	if err != nil || c.Tools.Apps != nil || !reflect.DeepEqual(c.Server.Apps, []string{"worker"}) {
		t.Fatalf("wildcard tools selection failed: %+v, %v", c, err)
	}
}

func TestWeatherSettingsMergeWithoutResettingLocation(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.toml")
	personal := filepath.Join(dir, "personal.toml")
	fixture(t, base, "[server.weather]\nlatitude = 0.0\nlongitude = 37.62\nlabel = \"HOME\"\n")
	fixture(t, personal, "[server.weather]\nlabel = \"MOSCOW\"\nscreens = [\"weather-current\"]\n")

	project, err := Load(base, personal)
	if err != nil {
		t.Fatal(err)
	}

	weather := project.Server.Weather
	if weather.Latitude == nil || *weather.Latitude != 0 ||
		weather.Longitude == nil || *weather.Longitude != 37.62 ||
		weather.Label != "MOSCOW" || weather.Interval != "15m" ||
		!reflect.DeepEqual(weather.Screens, []string{"weather-current"}) {
		t.Fatalf("unexpected merged weather settings: %+v", weather)
	}
}
