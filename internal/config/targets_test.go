package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestClockListsReplaceAcrossLayersAndSingleOverrides(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.toml")
	personal := filepath.Join(dir, "personal.toml")
	fixture(t, base, "[clock]\nurls = [\"192.168.0.1\", \"192.168.0.2\"]\n")
	fixture(t, personal, "[clock]\nurls = [\"192.168.0.121\", \"192.168.0.122\"]\n")

	project, err := Load(base, personal)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"192.168.0.121", "192.168.0.122"}
	if !reflect.DeepEqual(project.Clock.Targets(), want) {
		t.Fatalf("unexpected targets: %v", project.Clock.Targets())
	}

	project.Clock.SetTarget("http://override")
	if !reflect.DeepEqual(project.Clock.Targets(), []string{"http://override"}) {
		t.Fatal("single override did not replace the entire list")
	}

	fixture(t, personal, "[clock]\nurl = \"http://legacy-clock\"\n")
	project, err = Load(base, personal)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(project.Clock.Targets(), []string{"http://legacy-clock"}) {
		t.Fatal("legacy personal clock.url did not replace shared clock.urls")
	}
}

func TestEmptyClockListIsExplicitAndMixedFormsAreRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	fixture(t, path, "[clock]\nurls = []\n")

	project, err := Load(path, "")
	if err != nil || len(project.Clock.Targets()) != 0 {
		t.Fatalf("explicit empty target list was replaced by a default: %+v, %v", project.Clock, err)
	}

	fixture(t, path, "[clock]\nurl = \"http://clock\"\nurls = [\"http://other\"]\n")
	if _, err := Load(path, ""); err == nil {
		t.Fatal("ambiguous single and list configuration was accepted")
	}
}
