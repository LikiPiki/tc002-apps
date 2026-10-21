package clockapps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LikiPiki/tc002-apps/internal/config"
)

func fixture(t *testing.T, root, kind, name, source string) {
	t.Helper()

	path := filepath.Join(root, kind, name, "main.be")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogSelectionAndSettings(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "screens", "cigar", "return Cigar()")
	fixture(t, root, "services", "policy", "# @headless true\n# @config enabled bool\nreturn Policy()")
	fixture(t, root, "on-demand", "timer", "# @ondemand\nreturn Timer()")

	c := config.ClockConfig{Services: config.AppSettings{"policy": {"enabled": true}}}

	scripts, err := Select(root, c, "", "all")
	if err != nil || len(scripts) != 3 {
		t.Fatalf("unexpected catalog: %+v, %v", scripts, err)
	}
	if scripts[0].Kind != "services" ||
		scripts[1].Kind != "screens" ||
		scripts[2].Kind != "on-demand" ||
		scripts[0].Settings["enabled"] != true {
		t.Fatalf("unexpected catalog order or settings: %+v", scripts)
	}

	c.Deploy.Screens = []string{}
	c.Deploy.Services = []string{}

	scripts, err = Select(root, c, "", "all")
	if err != nil || len(scripts) != 1 || scripts[0].Name != "timer" {
		t.Fatalf("empty category did not opt out: %+v, %v", scripts, err)
	}

	scripts, err = Select(root, c, "cigar", "screens")
	if err != nil || len(scripts) != 1 {
		t.Fatalf("explicit APP must override selection: %+v, %v", scripts, err)
	}

	if _, err := Select(root, c, "cigar", "services"); err == nil {
		t.Fatal("wrong explicit kind accepted")
	}
}

func TestCatalogRejectsMistakesBeforeDeployment(t *testing.T) {
	for _, test := range []struct {
		kind   string
		source string
	}{
		{"services", "return NoHeadless()"},
		{"services", "# @headless\nreturn NoValue()"},
		{"screens", "# @headless true\nreturn Hidden()"},
		{"on-demand", "return NoOnDemand()"},
		{"screens", "# @ondemand false\nreturn FlagIsAlwaysTrue()"},
		{"on-demand", "# @headless true\n# @ondemand\nreturn Conflict()"},
		{"screens", "# @module\nreturn Helpers()"},
		{"screens", " "},
	} {
		root := t.TempDir()
		fixture(t, root, test.kind, "demo", test.source)

		if _, err := Select(root, config.ClockConfig{}, "demo", "all"); err == nil {
			t.Errorf("accepted %s: %s", test.kind, test.source)
		}
	}

	root := t.TempDir()
	fixture(t, root, "screens", "same", "return One()")
	fixture(t, root, "services", "same", "# @headless true\nreturn Two()")

	if _, err := Select(root, config.ClockConfig{}, "", "all"); err == nil {
		t.Fatal("duplicate install name accepted")
	}
}

func TestCatalogRejectsUnknownSettingsAndApps(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "screens", "demo", "# @config enabled bool\nreturn Demo()")

	for key, value := range map[string]any{"typo": true, "enabled": "not a bool"} {
		c := config.ClockConfig{Screens: config.AppSettings{"demo": {key: value}}}
		if _, err := Select(root, c, "demo", "all"); err == nil {
			t.Errorf("accepted setting %s=%v", key, value)
		}
	}

	c := config.ClockConfig{Screens: config.AppSettings{"missing": {"enabled": true}}}

	if _, err := Select(root, c, "", "all"); err == nil {
		t.Fatal("unknown configured app accepted")
	}
}
