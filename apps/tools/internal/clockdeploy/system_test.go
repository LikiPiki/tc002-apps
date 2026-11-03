package clockdeploy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

func TestSystemSettingsPlanApplyAndRollback(t *testing.T) {
	settings := map[string]any{"tzName": "Europe/Berlin", "tz": "CET-1", "ntpServer": "pool.ntp.org"}
	writes := 0
	persist := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/system" {
			t.Errorf("unexpected endpoint: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == http.MethodPut {
			writes++
			var patch map[string]any
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				t.Error(err)
			}
			if len(patch) != 2 || patch["ntpServer"] != nil {
				t.Errorf("write included undeclared settings: %v", patch)
			}
			if persist {
				for key, value := range patch {
					settings[key] = value
				}
			}
		}
		json.NewEncoder(w).Encode(settings)
	}))
	defer server.Close()
	client, err := awtrix.NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	desired := map[string]any{"tzName": "Europe/Moscow", "tz": "MSK-3"}
	plan := Plan{Target: server.URL}
	if err := plan.PrepareSystem(t.Context(), client, desired); err != nil {
		t.Fatal(err)
	}
	if writes != 0 || len(plan.SystemBefore) != 2 {
		t.Fatal("planning changed device state or captured unrelated settings")
	}
	path := filepath.Join(t.TempDir(), "recovery.json")
	if err := plan.Save(path); err != nil {
		t.Fatal(err)
	}
	report, err := plan.Apply(t.Context(), client, 0)
	if err != nil || report.System != "applied" || writes != 1 || settings["tz"] != "MSK-3" {
		t.Fatalf("apply: report=%+v writes=%d error=%v", report, writes, err)
	}
	unchanged := Plan{}
	if err := unchanged.PrepareSystem(t.Context(), client, desired); err != nil {
		t.Fatal(err)
	}
	report, err = unchanged.Apply(t.Context(), client, 0)
	if err != nil || report.System != "unchanged" || writes != 1 {
		t.Fatalf("unchanged settings were rewritten: %+v %v", report, err)
	}
	recovery, err := LoadPlan(path, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := recovery.Rollback(t.Context(), client); err != nil {
		t.Fatal(err)
	}
	if settings["tz"] != "CET-1" || settings["ntpServer"] != "pool.ntp.org" {
		t.Fatalf("rollback: %v", settings)
	}
	persist = false
	if _, err := plan.Apply(t.Context(), client, 0); err == nil {
		t.Fatal("accepted an unpersisted system update")
	}
	if err := plan.PrepareSystem(t.Context(), client, map[string]any{"typo": true}); err == nil {
		t.Fatal("accepted an unknown system setting")
	}
}
