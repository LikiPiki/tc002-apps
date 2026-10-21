package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDirectCheckRefusesURLWithoutNetworkAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("direct check must fail before contacting the target")
	}))
	defer server.Close()

	t.Setenv("TC002_CHECK_SANDBOX", "")

	err := runScripts("check", []string{"--url", server.URL, "--config", "does-not-exist.toml"})
	if err == nil || !strings.Contains(err.Error(), "sandbox-only") {
		t.Fatalf("unexpected error: %v", err)
	}
}
