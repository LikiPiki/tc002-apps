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
	fixture(t, base, "[clock]\nurl = \"http://localhost:8080\"\napps = [\"cigar\"]\n[server]\napps = [\"worker\"]\n[server.worker]\ninterval = \"1m\"\n")
	fixture(t, personal, "[clock]\nurl = \"http://192.168.1.50\"\n[server.worker]\ninterval = \"30s\"\n")
	c, err := Load(base, personal)
	if err != nil {
		t.Fatal(err)
	}
	if c.Clock.URL != "http://192.168.1.50" || c.Server.Worker.Interval != "30s" || !reflect.DeepEqual(c.Clock.Apps, []string{"cigar"}) || !reflect.DeepEqual(c.Server.Apps, []string{"worker"}) {
		t.Fatalf("unexpected merged config: %+v", c)
	}
	fixture(t, personal, "[clock]\napps = [\"*\"]\n")
	c, err = Load(base, personal)
	if err != nil || c.Clock.Apps != nil || c.Clock.URL != "http://localhost:8080" || len(c.Server.Apps) != 1 {
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
		"[clock]\napps = []", "[server]\napps = []", "[clock]\napp = [\"cigar\"]",
		"[clock]\napps = \"cigar\"", "[server]\napps = [\"*\", \"worker\"]",
		"[clock]\nurl = \"not-a-url\"", "[server.worker]\ninterval = \"0s\"",
		"[server.worker]\ninterval = \"nope\"", "[clock]\nurl = [",
		"[clock.schedule]\non = \"9:00\"", "[clock.schedule]\noff = \"24:00\"",
		"[clock.schedule]\non = \"20:00\"\noff = \"20:00\"",
	} {
		path := filepath.Join(t.TempDir(), "config.toml")
		fixture(t, path, content)
		if _, err := Load(path, ""); err == nil {
			t.Errorf("accepted invalid config: %s", content)
		}
	}
}

func TestScheduleOverridesAndOvernightWindow(t *testing.T) {
	dir := t.TempDir()
	base, personal := filepath.Join(dir, "base.toml"), filepath.Join(dir, "personal.toml")
	fixture(t, base, "[clock.schedule]\nenabled = false\non = \"20:00\"\noff = \"02:00\"\n")
	fixture(t, personal, "[clock.schedule]\nenabled = true\n")
	c, err := Load(base, personal)
	if err != nil || !c.Clock.Schedule.Enabled || c.Clock.Schedule.On != "20:00" || c.Clock.Schedule.Off != "02:00" {
		t.Fatalf("unexpected schedule: %+v, %v", c.Clock.Schedule, err)
	}
}

func TestServerSelectionAndDefaultDiscovery(t *testing.T) {
	root := t.TempDir()
	fixture(t, filepath.Join(root, "worker", "main.go"), "package main")
	fixture(t, filepath.Join(root, "clockctl", "main.go"), "package main")
	all, err := ServerApps(root, nil)
	if err != nil || !reflect.DeepEqual(all, []string{"clockctl", "worker"}) {
		t.Fatalf("unexpected discovery: %v, %v", all, err)
	}
	selected, err := ServerApps(root, []string{"worker", "worker"})
	if err != nil || !reflect.DeepEqual(selected, []string{"worker"}) {
		t.Fatalf("unexpected selection: %v, %v", selected, err)
	}
	if _, err := ServerApps(root, []string{"missing"}); err == nil {
		t.Fatal("unknown command accepted")
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
