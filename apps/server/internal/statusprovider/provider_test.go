package statusprovider

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/config"
)

func TestProbeRetriesTransientFailure(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	settings := Settings{
		URL:      server.URL,
		Label:    "SITE",
		Timeout:  time.Second,
		TTL:      time.Minute,
		Attempts: 3,
	}
	reading, err := Probe(t.Context(), settings)
	if err != nil {
		t.Fatal(err)
	}
	if reading.State != "up" || reading.HTTPStatus != 204 || requests.Load() != 3 {
		t.Fatalf("unexpected probe result: %+v, requests=%d", reading, requests.Load())
	}
}

func TestProbeDoesNotRetryClientErrors(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	reading, err := Probe(t.Context(), Settings{
		URL:      server.URL,
		Timeout:  time.Second,
		Attempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reading.State != "down" || requests.Load() != 1 {
		t.Fatalf("unexpected result: %+v, requests=%d", reading, requests.Load())
	}
}

func TestSettingsRejectExpiryBeforeNextUpdate(t *testing.T) {
	_, err := Parse(config.StatusConfig{
		URL:      "https://example.com/health",
		App:      "service-status",
		Label:    "SITE",
		Interval: "30s",
		Timeout:  "5s",
		TTL:      "30s",
		Attempts: 3,
	})
	if err == nil {
		t.Fatal("accepted ttl shorter than update budget")
	}
}
