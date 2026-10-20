package main

import (
	"os"
	"path/filepath"
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
