package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPersonalClockConfigOverridesOnlySpecifiedFields(t *testing.T) {
	dir := t.TempDir()
	base, personal := filepath.Join(dir, "clock.toml"), filepath.Join(dir, "local.toml")
	writeFixture(t, base, "url = \"http://localhost:8080\"\napps = [\"cigar\"]\n")
	writeFixture(t, personal, "# Personal clock\nurl = \"http://192.168.1.50\"\n")
	config, err := loadClockConfig(base, personal)
	if err != nil {
		t.Fatal(err)
	}
	if config.URL != "http://192.168.1.50" || !reflect.DeepEqual(config.Apps, []string{"cigar"}) {
		t.Fatalf("unexpected merged config: %+v", config)
	}
	writeFixture(t, personal, `apps = ["*"]`)
	config, err = loadClockConfig(base, personal)
	if err != nil || config.Apps != nil || config.URL != "http://localhost:8080" {
		t.Fatalf("wildcard should select all apps without changing URL: %+v, %v", config, err)
	}
	config, err = loadClockConfig(base, filepath.Join(dir, "missing.toml"))
	if err != nil || config.URL != "http://localhost:8080" {
		t.Fatalf("missing optional config: %+v, %v", config, err)
	}
}

func TestClockConfigRejectsMistakes(t *testing.T) {
	for _, content := range []string{`apps = []`, `app = ["cigar"]`, `apps = "cigar"`, "url = ["} {
		path := filepath.Join(t.TempDir(), "clock.toml")
		writeFixture(t, path, content)
		if _, err := loadClockConfig(path, ""); err == nil {
			t.Errorf("accepted bad config: %s", content)
		}
	}
}

func TestSelectScriptsDiscoversOnlyBerryAppsAndPreflightsFiles(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "cigar", "main.be"), "return Cigar()")
	writeFixture(t, filepath.Join(root, "seagull", "main.be"), "return Seagull()")
	writeFixture(t, filepath.Join(root, "assets", "README.md"), "not an app")
	scripts, err := selectScripts(root, nil)
	if err != nil || len(scripts) != 2 || scripts[0].name != "cigar" || scripts[1].name != "seagull" {
		t.Fatalf("unexpected discovery: %+v, %v", scripts, err)
	}
	scripts, err = selectScripts(root, []string{"seagull", "seagull"})
	if err != nil || len(scripts) != 1 || scripts[0].name != "seagull" {
		t.Fatalf("unexpected selection: %+v, %v", scripts, err)
	}
	if scripts, err := selectScripts(root, []string{"cigar", "missing"}); err == nil || scripts != nil {
		t.Fatalf("must preflight all sources: %+v, %v", scripts, err)
	}
	writeFixture(t, filepath.Join(root, "empty", "main.be"), " \n")
	if _, err := selectScripts(root, []string{"empty"}); err == nil {
		t.Fatal("accepted an empty script")
	}
}
