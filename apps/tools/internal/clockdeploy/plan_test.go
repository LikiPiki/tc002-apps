package clockdeploy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockapps"
	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

func TestUnchangedPlanDoesNotWrite(t *testing.T) {
	clock, client := newPlanTestClock(t)
	scripts := []clockapps.Script{
		{
			Name:     "demo",
			Source:   []byte("old"),
			Settings: map[string]any{"n": int64(1)},
		},
	}

	plan, err := Prepare(t.Context(), client, clock.url, scripts)
	if err != nil {
		t.Fatal(err)
	}
	if clock.writeCount() != 0 {
		t.Fatal("planning wrote to device")
	}

	if _, err := plan.Apply(t.Context(), client, 0); err != nil {
		t.Fatal(err)
	}
	if clock.writeCount() != 0 {
		t.Fatal("unchanged deployment wrote to device")
	}
}

func TestPlanRestoresSourceAndSettings(t *testing.T) {
	clock, client := newPlanTestClock(t)
	scripts := []clockapps.Script{
		{
			Name:     "demo",
			Source:   []byte("new"),
			Settings: map[string]any{"n": int64(2)},
		},
	}

	plan, err := Prepare(t.Context(), client, clock.url, scripts)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := plan.Apply(t.Context(), client, 0); err != nil {
		t.Fatal(err)
	}
	clock.assertState(t, "new", 2)

	if err := plan.Rollback(t.Context(), client); err != nil {
		t.Fatal(err)
	}
	clock.assertState(t, "old", 1)
}

func TestRollbackPreservesConcurrentSourceEdit(t *testing.T) {
	clock, client := newPlanTestClock(t)
	scripts := []clockapps.Script{
		{Name: "demo", Source: []byte("new")},
	}

	plan, err := Prepare(t.Context(), client, clock.url, scripts)
	if err != nil {
		t.Fatal(err)
	}

	clock.mu.Lock()
	clock.source = "someone else's update"
	clock.mu.Unlock()

	if err := plan.Rollback(t.Context(), client); err == nil {
		t.Fatal("rollback overwrote concurrent edit")
	}
	clock.assertState(t, "someone else's update", 1)
}

type planTestClock struct {
	mu     sync.Mutex
	url    string
	source string
	value  float64
	writes int
}

func newPlanTestClock(t *testing.T) (*planTestClock, *awtrix.Client) {
	t.Helper()

	clock := &planTestClock{source: "old", value: 1}
	server := httptest.NewServer(http.HandlerFunc(clock.serveHTTP))
	t.Cleanup(server.Close)
	clock.url = server.URL

	client, err := awtrix.NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return clock, client
}

func (clock *planTestClock) serveHTTP(w http.ResponseWriter, r *http.Request) {
	clock.mu.Lock()
	defer clock.mu.Unlock()

	if r.Method != http.MethodGet {
		clock.writes++
	}

	switch r.Method + " " + r.URL.Path {
	case "GET /api/v1/capabilities":
		fmt.Fprint(w, `{"scriptUpdates":true}`)
	case "GET /api/v1/apps":
		fmt.Fprint(w, `[{"name":"demo","origin":"script","enabled":true,"error":null}]`)
	case "GET /api/v1/apps/script/demo":
		fmt.Fprint(w, clock.source)
	case "PUT /api/v1/apps/script-update/demo":
		var update struct {
			Expected string `json:"expected_source"`
			Source   string `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if update.Expected != clock.source {
			w.WriteHeader(http.StatusConflict)
			return
		}

		clock.source = update.Source
		fmt.Fprint(w, `{"ok":true}`)
	case "GET /api/v1/apps/demo/config":
		fmt.Fprintf(w, `{"fields":[{"key":"n","value":%v}]}`, clock.value)
	case "PATCH /api/v1/apps/demo/config":
		var settings map[string]float64
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		clock.value = settings["n"]
		fmt.Fprint(w, `{"ok":true,"error":null}`)
	case "PUT /api/v1/apps/demo/enabled":
		fmt.Fprint(w, `{"ok":true}`)
	default:
		http.Error(w, "unexpected request", http.StatusNotFound)
	}
}

func (clock *planTestClock) writeCount() int {
	clock.mu.Lock()
	defer clock.mu.Unlock()

	return clock.writes
}

func (clock *planTestClock) assertState(t *testing.T, source string, value float64) {
	t.Helper()

	clock.mu.Lock()
	defer clock.mu.Unlock()

	if clock.source != source || clock.value != value {
		t.Fatalf("device state: source=%q value=%v; want source=%q value=%v", clock.source, clock.value, source, value)
	}
}
