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

func TestBuildIgnoresUnrelatedClockAndWorkerSettings(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "server", "worker")

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[clock]\nurl = \"invalid URL\"\n[server.worker]\ninterval = \"invalid interval\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	c, err := config.Load(path, "")
	if err != nil {
		t.Fatal(err)
	}

	if targets, err := Select(root, c.Server.Apps, c.Tools.Apps, "server"); err != nil || len(targets) != 1 {
		t.Fatalf("unrelated runtime fields blocked build: %v, %v", targets, err)
	}

	if _, err := Select(root, []string{}, nil, "server"); err == nil {
		t.Fatal("build consumer accepted empty server selection")
	}
}

func TestServerSelectionAndDefaultDiscovery(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "server", "worker")
	fixture(t, root, "tools", "clockctl")

	all, err := Select(root, nil, nil, "all")
	if err != nil || !reflect.DeepEqual(all, []Target{{"server", "worker"}, {"tools", "clockctl"}}) {
		t.Fatalf("unexpected discovery: %v, %v", all, err)
	}

	selected, err := Select(root, []string{"worker", "worker"}, nil, "server")
	if err != nil || !reflect.DeepEqual(selected, []Target{{"server", "worker"}}) {
		t.Fatalf("unexpected server selection: %v, %v", selected, err)
	}

	selected, err = Select(root, nil, []string{}, "all")
	if err != nil || len(selected) != 1 || selected[0].Group != "server" {
		t.Fatalf("tools opt-out failed: %v, %v", selected, err)
	}

	for _, name := range []string{"missing", "../clockctl"} {
		if _, err := Select(root, []string{name}, nil, "server"); err == nil {
			t.Errorf("accepted %q", name)
		}
	}
}

func TestGroupsHaveSeparateArtifactNames(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "server", "same")
	fixture(t, root, "tools", "same")

	targets, err := Select(root, nil, nil, "all")
	if err != nil || len(targets) != 2 || targets[0].String() == targets[1].String() {
		t.Fatalf("artifact name collision: %v, %v", targets, err)
	}
}
