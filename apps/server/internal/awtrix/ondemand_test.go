package awtrix

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOnDemandCanBeInstalledDormantStartedAndStopped(t *testing.T) {
	running := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/apps":
			fmt.Fprint(w, `[{"name":"timer","origin":"script","enabled":true,"inLoop":false,"ondemand":true,"error":null}]`)
		case "PUT /api/v1/apps/active":
			running = true
			fmt.Fprint(w, `{"ok":true}`)
		case "GET /api/v1/device":
			fmt.Fprint(w, `{"currentApp":"timer"}`)
		case "POST /api/v1/apps/next":
			running = false
			fmt.Fprint(w, `{"ok":true}`)
		default:
			t.Errorf("unexpected operation: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL)
	if err := client.CheckScript(context.Background(), "timer"); err != nil {
		t.Fatalf("dormant on-demand app must be valid: %v", err)
	}
	if err := client.ShowScript(context.Background(), "timer"); err != nil || !running {
		t.Fatalf("start: running=%v error=%v", running, err)
	}
	if err := client.StopOnDemand(context.Background(), "timer"); err != nil || running {
		t.Fatalf("stop: running=%v error=%v", running, err)
	}
}
