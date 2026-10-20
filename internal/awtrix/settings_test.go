package awtrix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfigureScriptChecksRestartError(t *testing.T) {
	for _, failed := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var settings map[string]any
			if r.Method != "PATCH" || r.URL.Path != "/api/v1/apps/display-policy/config" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&settings); err != nil || settings["schedule_enabled"] != true || settings["on"] != "10:00" {
				t.Errorf("unexpected settings: %v, %v", settings, err)
			}
			if failed {
				fmt.Fprint(w, `{"ok":true,"error":{"hook":"setup","message":"bad settings"}}`)
			} else {
				fmt.Fprint(w, `{"ok":true,"error":null}`)
			}
		}))
		client, _ := NewClient(server.URL)
		err := client.ConfigureScript(context.Background(), "display-policy", map[string]any{"schedule_enabled": true, "on": "10:00"})
		server.Close()
		if (err != nil) != failed {
			t.Fatalf("failed=%v error=%v", failed, err)
		}
	}
}

func TestHeadlessCannotBeShown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/apps" || r.Method != "GET" {
			t.Errorf("unexpected display access: %s %s", r.Method, r.URL.Path)
		}
		fmt.Fprint(w, `[{"name":"policy","origin":"script","enabled":true,"headless":true,"error":null}]`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL)
	if err := client.ShowScript(context.Background(), "policy"); err == nil {
		t.Fatal("headless script was shown")
	}
}
