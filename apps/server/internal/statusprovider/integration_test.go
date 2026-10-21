package statusprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

func TestClockIntegration(t *testing.T) {
	url := os.Getenv("TC002_INTEGRATION_URL")
	if url == "" {
		t.Skip("run make status-check for the disposable AWTRIX integration")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	client, err := awtrix.NewClient(url)
	if err != nil {
		t.Fatal(err)
	}

	source, err := os.ReadFile("/repo/apps/clock/screens/service-status/main.be")
	if err != nil {
		t.Fatal(err)
	}

	if err := client.InstallScript(ctx, "service-status", source); err != nil {
		t.Fatal(err)
	}
	if err := client.SetDisplayPower(ctx, true); err != nil {
		t.Fatal(err)
	}

	assertStatusColor(t, ctx, client, 0xFFAA22)

	var responseCode atomic.Int32
	responseCode.Store(http.StatusOK)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(responseCode.Load()))
	}))
	t.Cleanup(server.Close)

	settings := Settings{
		URL:      server.URL,
		App:      "service-status",
		Label:    "SITE",
		Timeout:  time.Second,
		TTL:      90 * time.Second,
		Attempts: 2,
	}

	for _, scenario := range []struct {
		name  string
		code  int32
		color uint32
	}{
		{"healthy", http.StatusOK, 0x22CC66},
		{"unavailable", http.StatusServiceUnavailable, 0xFF4444},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			responseCode.Store(scenario.code)

			reading, err := Probe(ctx, settings)
			if err != nil {
				t.Fatal(err)
			}
			if err := Publish(ctx, client, settings, reading); err != nil {
				t.Fatal(err)
			}

			assertStatusColor(t, ctx, client, scenario.color)
		})
	}

	reading := Reading{
		Version:     1,
		State:       "up",
		Label:       "SITE",
		UpdatedAtMS: strconv.FormatInt(time.Now().Add(-time.Hour).UnixMilli(), 10),
		TTLSeconds:  90,
	}

	if err := Publish(ctx, client, settings, reading); err != nil {
		t.Fatal(err)
	}

	assertStatusColor(t, ctx, client, 0xFFAA22)
}

func assertStatusColor(t *testing.T, ctx context.Context, client *awtrix.Client, expected uint32) {
	t.Helper()

	if err := client.ShowScript(ctx, "service-status"); err != nil {
		t.Fatal(err)
	}

	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()

	for {
		if err := client.CheckScript(ctx, "service-status"); err != nil {
			t.Fatal(err)
		}

		frame, err := client.Frame(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(frame.Pixels) > 0 && frame.Pixels[0] == expected {
			return
		}

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-deadline.C:
			t.Fatalf("expected status color %#x; frame size=%dx%d", expected, frame.Width, frame.Height)
		case <-time.After(100 * time.Millisecond):
		}
	}
}
