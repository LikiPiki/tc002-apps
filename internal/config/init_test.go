package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

func initScript(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("config initialization requires a POSIX shell")
	}

	return filepath.Join("..", "..", "scripts", "init-config.sh")
}

func TestInitConfigPreservesExistingFileAndPermissions(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.toml")
	template := filepath.Join(dir, "config.toml.example")
	fixture(t, template, "[clock]\nurls = [\"template\"]\n")
	fixture(t, target, "# personal configuration\n[clock]\nurls = [\"my-clock\"]\n")

	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}

	for attempt := 0; attempt < 2; attempt++ {
		output, err := exec.Command("sh", initScript(t), target, template).CombinedOutput()
		if err != nil {
			t.Fatalf("initialization failed: %v\n%s", err, output)
		}
	}

	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}

	if string(after) != string(before) || updated.Mode().Perm() != info.Mode().Perm() {
		t.Fatal("initialization replaced the working config or changed its permissions")
	}
}

func TestConcurrentInitializationCreatesOneUsableConfig(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.toml")
	template := filepath.Join(dir, "config.toml.example")
	contents := "[clock]\nurls = [\"192.168.1.50\"]\n"
	fixture(t, template, contents)
	script := initScript(t)

	var pending sync.WaitGroup
	for attempt := 0; attempt < 8; attempt++ {
		pending.Add(1)
		go func() {
			defer pending.Done()

			output, err := exec.Command("sh", script, target, template).CombinedOutput()
			if err != nil {
				t.Errorf("concurrent initialization failed: %v\n%s", err, output)
			}
		}()
	}
	pending.Wait()

	created, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(created) != contents {
		t.Fatal("working config was truncated or replaced during concurrent initialization")
	}

	if _, err := Load(target, ""); err != nil {
		t.Fatalf("initialized config is not usable: %v", err)
	}
}
