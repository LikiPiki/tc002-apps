package clocks

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOfflineAndNonClockTargetsDoNotBlockDelivery(t *testing.T) {
	var writes atomic.Int32
	clock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/device":
			fmt.Fprint(w, `{"boardType":"tc002","scriptingRunning":true}`)
		case "/api/v1/capabilities":
			fmt.Fprint(w, `{"scriptUpdates":true}`)
		case "/api/v1/notifications":
			writes.Add(1)
			fmt.Fprint(w, `{"ok":true}`)
		default:
			t.Errorf("unexpected clock request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(clock.Close)

	var wrongDeviceWrites atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			wrongDeviceWrites.Add(1)
		}

		fmt.Fprint(w, `{"boardType":"router"}`)
	}))
	t.Cleanup(other.Close)

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(slow.Close)

	targets := []string{slow.URL, other.URL, strings.TrimPrefix(clock.URL, "http://"), clock.URL + "/"}
	report := Run(t.Context(), targets, 200*time.Millisecond, func(ctx context.Context, target Target) (any, error) {
		return nil, target.Client.Notify(ctx, "hello")
	})

	if report.Err() != nil || report.Succeeded() != 1 || len(report.Results) != 3 {
		t.Fatalf("unexpected partial delivery: %+v", report)
	}
	if writes.Load() != 1 || wrongDeviceWrites.Load() != 0 {
		t.Fatalf("clock writes=%d, non-clock writes=%d", writes.Load(), wrongDeviceWrites.Load())
	}
	if report.Results[0].State != "skipped" || report.Results[1].State != "skipped" ||
		report.Results[0].Error == "" || report.Results[1].Error == "" {
		t.Fatalf("skipped clocks must retain warning details: %+v", report)
	}
}

func TestAWTRIXShapedDeviceWithoutCapabilitiesIsNotWritten(t *testing.T) {
	var actionCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/device" {
			fmt.Fprint(w, `{"boardType":"linux"}`)
			return
		}

		fmt.Fprint(w, `{}`)
	}))
	t.Cleanup(server.Close)

	report := Run(t.Context(), []string{server.URL}, time.Second, func(ctx context.Context, target Target) (any, error) {
		actionCalled.Store(true)
		return nil, nil
	})

	if actionCalled.Load() || report.Err() == nil || report.Results[0].State != "skipped" {
		t.Fatalf("non-AWTRIX endpoint accepted: %+v", report)
	}
}

func TestInvalidOrEmptyTargetSetsFailWithoutDelivery(t *testing.T) {
	for _, targets := range [][]string{nil, {""}, {"ftp://clock"}} {
		report := Run(t.Context(), targets, time.Second, func(ctx context.Context, target Target) (any, error) {
			t.Error("invalid clock target reached the action")
			return nil, nil
		})

		if report.Err() == nil {
			t.Fatalf("total failure hidden for %v", targets)
		}
	}
}
