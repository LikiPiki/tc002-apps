package weatherprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

func TestFetchUsesCoordinatesAndNormalizesTwoFutureDays(t *testing.T) {
	now := time.Date(2026, time.March, 28, 12, 0, 0, 0, time.UTC)
	body := encodeForecast(t, forecastFixture(t, now, "Europe/Berlin"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.URL.Path != "/55.75,37.62" || query.Get("format") != "j1" {
			t.Errorf("unexpected query: %v", query)
		}

		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)

	provider := New(testSettings())
	provider.settings.Timezone = "Europe/Berlin"
	provider.endpoint = server.URL
	provider.now = func() time.Time { return now }

	reading, err := provider.Fetch(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if reading.Current.TemperatureC != 12.5 || reading.Current.Condition != "CLEAR" ||
		reading.Current.WindSpeedMS != 3 || len(reading.Days) != 2 || !reading.Current.IsDay {
		t.Fatalf("unexpected weather: %+v", reading)
	}

	// The forecast crosses the DST boundary and must still use consecutive local dates.
	if reading.Today.Date != "2026-03-28" || reading.Days[0].Date != "2026-03-29" || reading.Days[1].Date != "2026-03-30" {
		t.Fatalf("incorrect local forecast dates: %+v", reading.Days)
	}
	if reading.Today.MinimumC != 4 || reading.Today.MaximumC != 12 || reading.Today.PrecipitationProbabilityPercent != 0 {
		t.Fatalf("incorrect today's forecast: %+v", reading.Today)
	}
}

func TestFetchRejectsIncompleteAndExpiredDataWithoutRetry(t *testing.T) {
	now := time.Date(2026, time.October, 21, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		change func(map[string]any)
	}{
		{"missing current", func(raw map[string]any) {
			delete(raw, "current_condition")
		}},
		{"missing temperature", func(raw map[string]any) {
			raw["current_condition"].([]map[string]any)[0]["temp_C"] = ""
		}},
		{"missing forecast day", func(raw map[string]any) {
			raw["weather"] = raw["weather"].([]map[string]any)[:2]
		}},
		{"invalid wind", func(raw map[string]any) {
			raw["current_condition"].([]map[string]any)[0]["windspeedKmph"] = "NaN"
		}},
		{"unknown condition", func(raw map[string]any) {
			raw["current_condition"].([]map[string]any)[0]["weatherCode"] = "999"
		}},
		{"expired source", func(raw map[string]any) {
			raw["current_condition"].([]map[string]any)[0]["observation_time"] = now.Add(-3 * time.Hour).UTC().Format("03:04 PM")
		}},
		{"future source", func(raw map[string]any) {
			raw["current_condition"].([]map[string]any)[0]["observation_time"] = now.Add(time.Hour).UTC().Format("03:04 PM")
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := forecastFixture(t, now, "Europe/Moscow")
			test.change(raw)
			body := encodeForecast(t, raw)
			var requests atomic.Int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				fmt.Fprint(w, body)
			}))
			t.Cleanup(server.Close)

			provider := New(testSettings())
			provider.endpoint = server.URL
			provider.now = func() time.Time { return now }

			if _, err := provider.Fetch(t.Context()); err == nil {
				t.Fatal("accepted invalid weather")
			}

			if requests.Load() != 1 {
				t.Fatal("invalid data should not trigger HTTP retries")
			}
		})
	}
}

func TestFetchRetriesTransientHTTPFailure(t *testing.T) {
	now := time.Now()
	body := encodeForecast(t, forecastFixture(t, now, "Europe/Moscow"))
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)

	provider := New(testSettings())
	provider.endpoint = server.URL
	provider.now = func() time.Time { return now }

	if _, err := provider.Fetch(t.Context()); err != nil {
		t.Fatal(err)
	}

	if requests.Load() != 2 {
		t.Fatalf("expected one retry, got %d requests", requests.Load())
	}
}

func TestFetchHonorsTotalTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)

	settings := testSettings()
	settings.Timeout = 50 * time.Millisecond
	provider := New(settings)
	provider.endpoint = server.URL

	if _, err := provider.Fetch(t.Context()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected total timeout, got %v", err)
	}
}

func TestPublishAttemptsEveryScreenAfterPartialFailure(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if strings.Contains(r.URL.Path, "weather-current") {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		fmt.Fprint(w, `{"ok":true,"error":null}`)
	}))
	t.Cleanup(server.Close)

	client, err := awtrix.NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	settings := testSettings()
	settings.Screens = []string{"weather-current", "weather-copy"}
	err = Publish(t.Context(), client, settings, Reading{Version: 1})
	if err == nil || !strings.Contains(err.Error(), "weather-current") || requests.Load() != 2 {
		t.Fatalf("partial publication: requests=%d error=%v", requests.Load(), err)
	}
}

func testSettings() Settings {
	return Settings{
		Latitude:  55.75,
		Longitude: 37.62,
		Label:     "HOME",
		Timezone:  "Europe/Moscow",
		Screens:   []string{"weather-current"},
		Interval:  15 * time.Minute,
		Timeout:   5 * time.Second,
		TTL:       2 * time.Hour,
		Attempts:  3,
	}
}

func forecastFixture(t *testing.T, now time.Time, timezone string) map[string]any {
	t.Helper()

	location, err := time.LoadLocation(timezone)
	if err != nil {
		t.Fatal(err)
	}

	local := now.In(location)
	days := make([]map[string]any, 3)
	for index := range days {
		hours := make([]map[string]any, 8)
		for hour := range hours {
			hours[hour] = map[string]any{
				"time": fmt.Sprint(hour * 300), "weatherCode": "113",
				"chanceofrain": fmt.Sprint(index * 20), "chanceofsnow": "0",
			}
		}
		days[index] = map[string]any{
			"date":     local.AddDate(0, 0, index).Format("2006-01-02"),
			"mintempC": fmt.Sprint(4 + index), "maxtempC": fmt.Sprint(12 + index),
			"astronomy": []map[string]any{{"sunrise": "06:00 AM", "sunset": "08:00 PM"}},
			"hourly":    hours,
		}
	}
	return map[string]any{
		"current_condition": []map[string]any{{
			"observation_time": now.UTC().Format("03:04 PM"),
			"temp_C":           "12.5", "FeelsLikeC": "11", "humidity": "65", "windspeedKmph": "10.8", "weatherCode": "113",
		}},
		"weather": days,
	}
}

func encodeForecast(t *testing.T, raw map[string]any) string {
	t.Helper()

	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}
