package clockdeploy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockapps"
	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

func TestPlanIdempotencyAndRecovery(t *testing.T) {
	source := "old"
	value := float64(1)
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			writes++
		}
		switch r.URL.Path {
		case "/api/v1/capabilities":
			fmt.Fprint(w, `{"scriptUpdates":true}`)
		case "/api/v1/apps":
			fmt.Fprint(w, `[{"name":"demo","origin":"script","enabled":true,"error":null}]`)
		case "/api/v1/apps/script/demo":
			fmt.Fprint(w, source)
		case "/api/v1/apps/script-update/demo":
			var p struct {
				Expected string `json:"expected_source"`
				Source   string `json:"source"`
			}
			json.NewDecoder(r.Body).Decode(&p)
			if p.Expected != source {
				w.WriteHeader(409)
				return
			}
			source = p.Source
			fmt.Fprint(w, `{"ok":true}`)
		case "/api/v1/apps/demo/config":
			if r.Method == "PATCH" {
				var p map[string]float64
				json.NewDecoder(r.Body).Decode(&p)
				value = p["n"]
				fmt.Fprint(w, `{"ok":true,"error":null}`)
			} else {
				fmt.Fprintf(w, `{"fields":[{"key":"n","value":%v}]}`, value)
			}
		case "/api/v1/apps/demo/enabled":
			fmt.Fprint(w, `{"ok":true}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	c, _ := awtrix.NewClient(server.URL)
	ctx := context.Background()
	scripts := []clockapps.Script{{Name: "demo", Source: []byte("old"), Settings: map[string]any{"n": int64(1)}}}
	p, err := Prepare(ctx, c, server.URL, scripts)
	if err != nil {
		t.Fatal(err)
	}
	if writes != 0 {
		t.Fatal("planning wrote to device")
	}
	if _, err = p.Apply(ctx, c, 0); err != nil {
		t.Fatal(err)
	}
	if writes != 0 {
		t.Fatal("unchanged deployment wrote to device")
	}
	scripts[0].Source = []byte("new")
	scripts[0].Settings["n"] = int64(2)
	p, err = Prepare(ctx, c, server.URL, scripts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Apply(ctx, c, 0); err != nil {
		t.Fatal(err)
	}
	if source != "new" || value != 2 {
		t.Fatal("changes missing")
	}
	if err = p.Rollback(ctx, c); err != nil {
		t.Fatal(err)
	}
	if source != "old" || value != 1 {
		t.Fatal("recovery failed")
	}
	source = "someone else's update"
	if err = p.Rollback(ctx, c); err == nil {
		t.Fatal("rollback overwrote concurrent edit")
	}
}
