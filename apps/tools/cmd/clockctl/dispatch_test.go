package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockdeploy"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

func TestDeploySkipsNonClockAndKeepsSeparateRecoveryRecords(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "screens", "demo")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}

	const source = "return Demo()"
	if err := os.WriteFile(filepath.Join(appDir, "main.be"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	first, firstWrites := deploymentClock(t)
	second, secondWrites := deploymentClock(t)
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error("deployment wrote to an unrelated device")
		}

		fmt.Fprint(w, `{}`)
	}))
	t.Cleanup(other.Close)

	clock := config.ClockConfig{URLs: []string{other.URL, first.URL, second.URL}}
	options := scriptOptions{
		App:     "demo",
		Kind:    "all",
		AppsDir: root,
		Backup:  filepath.Join(root, "recover.json"),
		Timeout: time.Second,
	}

	if err := dispatchScripts(t.Context(), "deploy", clock, options); err != nil {
		t.Fatal(err)
	}

	if firstWrites.Load() != 1 || secondWrites.Load() != 1 {
		t.Fatalf("deployment did not reach both clocks: %d, %d", firstWrites.Load(), secondWrites.Load())
	}

	for index, target := range []string{first.URL, second.URL} {
		path := filepath.Join(root, fmt.Sprintf("recover.clock-%d.json", index+2))
		plan, err := clockdeploy.LoadPlan(path, target)
		if err != nil {
			t.Fatal(err)
		}

		if len(plan.Changes) != 1 || plan.Changes[0].After != source {
			t.Fatalf("wrong recovery record for %s: %+v", target, plan)
		}
	}
}

func deploymentClock(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var lock sync.Mutex
	var source *string
	writes := &atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		defer lock.Unlock()

		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/device":
			fmt.Fprint(w, `{"boardType":"tc002","scriptingRunning":true}`)
		case "GET /api/v1/capabilities":
			fmt.Fprint(w, `{"scriptUpdates":true}`)
		case "GET /api/v1/apps":
			if source == nil {
				fmt.Fprint(w, `[]`)
				return
			}

			fmt.Fprint(w, `[{"name":"demo","origin":"script","enabled":true,"error":null}]`)
		case "GET /api/v1/apps/script/demo":
			if source == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			fmt.Fprint(w, *source)
		case "PUT /api/v1/apps/script-update/demo":
			var update struct {
				Expected *string `json:"expected_source"`
				Source   string  `json:"source"`
			}
			if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if !clockdeploy.Equal(source, update.Expected) {
				w.WriteHeader(http.StatusConflict)
				return
			}

			source = &update.Source
			writes.Add(1)
			fmt.Fprint(w, `{"ok":true}`)
		default:
			t.Errorf("unexpected clock request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	return server, writes
}
