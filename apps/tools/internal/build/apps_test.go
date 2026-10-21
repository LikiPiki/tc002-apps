package build

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/LikiPiki/tc002-apps/internal/config"
)

func fixture(t *testing.T, root, group, name string) {
	t.Helper()

	dir := filepath.Join(root, group, "cmd", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestToolBuildIgnoresRuntimeSettings(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "tools", "clockctl")

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[clock]\nurl = \"invalid URL\"\n[server.weather]\ninterval = \"invalid interval\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	c, err := config.Load(path, "")
	if err != nil {
		t.Fatal(err)
	}

	if targets, err := Select(filepath.Join(root, "tools"), c.Tools.Apps); err != nil || len(targets) != 1 {
		t.Fatalf("unrelated runtime fields blocked build: %v, %v", targets, err)
	}

}

func TestUtilitySelectionAndDiscovery(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "server", "server")
	fixture(t, root, "tools", "clockctl")
	fixture(t, root, "tools", "spritegen")

	all, err := Select(filepath.Join(root, "tools"), nil)
	if err != nil || !reflect.DeepEqual(all, []string{"clockctl", "spritegen"}) {
		t.Fatalf("unexpected discovery: %v, %v", all, err)
	}

	selected, err := Select(filepath.Join(root, "tools"), []string{"clockctl", "clockctl"})
	if err != nil || !reflect.DeepEqual(selected, []string{"clockctl"}) {
		t.Fatalf("unexpected utility selection: %v, %v", selected, err)
	}

	selected, err = Select(filepath.Join(root, "tools"), []string{})
	if err != nil || len(selected) != 0 {
		t.Fatalf("tools opt-out failed: %v, %v", selected, err)
	}

	for _, name := range []string{"missing", "../clockctl"} {
		if _, err := Select(filepath.Join(root, "tools"), []string{name}); err == nil {
			t.Errorf("accepted %q", name)
		}
	}
}
