package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

func TestExplicitURLWorksWithoutImplicitConfig(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.String("config", "", "")
	missing := filepath.Join(t.TempDir(), "missing.toml")

	config, err := loadTargetConfig(flags, missing, "", "http://localhost:1234")
	if err != nil {
		t.Fatal(err)
	}
	if config.Clock.URL != "http://localhost:1234" {
		t.Fatalf("unexpected target: %s", config.Clock.URL)
	}

	if err := flags.Parse([]string{"--config", missing}); err != nil {
		t.Fatal(err)
	}

	if _, err := loadTargetConfig(flags, missing, "", "http://localhost:1234"); err == nil {
		t.Fatal("explicit missing config was ignored")
	}
}

func TestDoctorReportsDeviceAndRuntimeProblems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/device":
			fmt.Fprint(w, `{"scriptingRunning":true,"currentApp":"demo"}`)
		case "/api/v1/apps":
			fmt.Fprint(w, `[{"name":"demo","origin":"script","enabled":true,"error":{"message":"broken draw"}}]`)
		case "/api/v1/capabilities":
			fmt.Fprint(w, `{"scriptUpdates":false}`)
		case "/api/v1/display/screen":
			fmt.Fprint(w, `{"width":2,"height":1,"pixels":[0,1]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := awtrix.NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	result := inspectTarget(t.Context(), client, server.URL, true)
	if result.Width != 2 || result.Height != 1 || len(result.Problems) != 2 {
		t.Fatalf("unexpected diagnosis: %+v", result)
	}
}
