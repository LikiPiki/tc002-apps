package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/LikiPiki/tc002-apps/apps/server/internal/weatherprovider"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

type fixedWeather struct {
	calls int
}

func (source *fixedWeather) Fetch(context.Context) (weatherprovider.Reading, error) {
	source.calls++

	return weatherprovider.Reading{Version: 1, Label: "SPB"}, nil
}

func TestOneVPSFetchPublishesToEveryClockDespitePartialFailure(t *testing.T) {
	dir := t.TempDir()
	var references []string
	var writeCounts []*atomic.Int32
	var expectedWrites []int32

	for index := 0; index < 12; index++ {
		clock, writes := weatherClock(t, index == 0)
		name := fmt.Sprintf("clock-%d.local.toml", index)
		screens := `["weather-current"]`
		count := int32(1)
		if index%2 == 1 {
			screens = `["weather-current", "weather-forecast"]`
			count = 2
		}

		profile := fmt.Sprintf("[clock]\nurls = [%q]\n[clock.deploy]\nscreens = %s\n", clock.URL, screens)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(profile), 0o600); err != nil {
			t.Fatal(err)
		}

		references = append(references, strconv.Quote(name))
		writeCounts = append(writeCounts, writes)
		expectedWrites = append(expectedWrites, count)
	}

	mainConfig := "[clock]\nprofiles = [" + strings.Join(references, ", ") + "]\n"
	path := filepath.Join(dir, "main.toml")
	if err := os.WriteFile(path, []byte(mainConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	project, err := config.Load(path, "")
	if err != nil {
		t.Fatal(err)
	}

	profiles, err := project.Clock.ResolveProfiles()
	if err != nil {
		t.Fatal(err)
	}

	source := &fixedWeather{}
	settings := weatherprovider.Settings{
		Screens: []string{"weather-current", "weather-forecast"},
	}

	if err := serve(t.Context(), source, profiles, settings, true); err != nil {
		t.Fatal(err)
	}

	if source.calls != 1 || len(profiles) != len(references) {
		t.Fatalf("source fetches=%d, profiles loaded=%d", source.calls, len(profiles))
	}

	for index, writes := range writeCounts {
		if writes.Load() != expectedWrites[index] {
			t.Errorf("profile %d received %d writes; expected %d", index, writes.Load(), expectedWrites[index])
		}
	}
}

func weatherClock(t *testing.T, failPublication bool) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	writes := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/device":
			fmt.Fprint(w, `{"boardType":"tc002","scriptingRunning":true}`)
		case "/api/v1/capabilities":
			fmt.Fprint(w, `{"scriptUpdates":true}`)
		case "/api/v1/apps/weather-current/data", "/api/v1/apps/weather-forecast/data":
			writes.Add(1)

			var payload map[string]weatherprovider.Reading
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if payload["weather"].Label != "SPB" {
				t.Error("clock received a different weather reading")
			}

			if failPublication {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			fmt.Fprint(w, `{"ok":true,"error":null}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	return server, writes
}
