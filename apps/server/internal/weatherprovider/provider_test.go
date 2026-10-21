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

func TestFetchUsesCoordinatesAndNormalizesThreeFutureDays(t *testing.T) {
	now := time.Date(2026, time.March, 28, 12, 0, 0, 0, time.UTC)
	body := encodeForecast(t, forecastFixture(t, now, "Europe/Berlin"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("latitude") != "55.75" || query.Get("longitude") != "37.62" ||
			query.Get("wind_speed_unit") != "ms" || query.Get("temperature_unit") != "celsius" ||
			query.Get("forecast_days") != "4" || query.Get("timeformat") != "unixtime" ||
			!strings.Contains(query.Get("current"), "apparent_temperature") {
			t.Errorf("unexpected query: %v", query)
		}

		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)

	provider := New(testSettings())
	provider.endpoint = server.URL
	provider.now = func() time.Time { return now }

	reading, err := provider.Fetch(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if reading.Current.TemperatureC != 12.5 || reading.Current.Condition != "CLEAR" ||
		reading.Current.WindSpeedMS != 3.2 || len(reading.Days) != 3 {
		t.Fatalf("unexpected weather: %+v", reading)
	}

	// The forecast crosses the DST boundary and must still use consecutive local dates.
	if reading.Days[0].Date != "2026-03-29" || reading.Days[2].Date != "2026-03-31" {
		t.Fatalf("incorrect local forecast dates: %+v", reading.Days)
	}
}

func TestFetchRejectsIncompleteAndExpiredDataWithoutRetry(t *testing.T) {
	now := time.Date(2026, time.October, 21, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		change func(map[string]any)
	}{
		{"missing current", func(raw map[string]any) {
			delete(raw, "current")
		}},
		{"null temperature", func(raw map[string]any) {
			raw["current"].(map[string]any)["temperature_2m"] = nil
		}},
		{"missing forecast day", func(raw map[string]any) {
			raw["daily"].(map[string]any)["temperature_2m_min"] = []float64{1, 2}
		}},
		{"wrong units", func(raw map[string]any) {
			raw["current_units"].(map[string]string)["wind_speed_10m"] = "km/h"
		}},
		{"unknown condition", func(raw map[string]any) {
			raw["current"].(map[string]any)["weather_code"] = 999
		}},
		{"expired source", func(raw map[string]any) {
			raw["current"].(map[string]any)["time"] = now.Add(-3 * time.Hour).Unix()
		}},
		{"future source", func(raw map[string]any) {
			raw["current"].(map[string]any)["time"] = now.Add(time.Hour).Unix()
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
	body := encodeForecast(t, forecastFixture(t, now, "UTC"))
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

	err = Publish(t.Context(), client, testSettings(), Reading{Version: 1})
	if err == nil || !strings.Contains(err.Error(), "weather-current") || requests.Load() != 2 {
		t.Fatalf("partial publication: requests=%d error=%v", requests.Load(), err)
	}
}

func testSettings() Settings {
	return Settings{
		Latitude:  55.75,
		Longitude: 37.62,
		Label:     "HOME",
		Timezone:  "auto",
		Screens:   []string{"weather-current", "weather-forecast"},
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
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	timestamps := make([]int64, 4)
	for index := range timestamps {
		timestamps[index] = midnight.AddDate(0, 0, index).Unix()
	}

	return map[string]any{
		"timezone": timezone,
		"current_units": map[string]string{
			"temperature_2m":       "°C",
			"apparent_temperature": "°C",
			"relative_humidity_2m": "%",
			"wind_speed_10m":       "m/s",
		},
		"daily_units": map[string]string{
			"temperature_2m_min":            "°C",
			"temperature_2m_max":            "°C",
			"precipitation_probability_max": "%",
		},
		"current": map[string]any{
			"time":                 now.Unix(),
			"temperature_2m":       12.5,
			"apparent_temperature": 11.0,
			"relative_humidity_2m": 65,
			"wind_speed_10m":       3.2,
			"weather_code":         1,
			"is_day":               1,
		},
		"daily": map[string]any{
			"time":                          timestamps,
			"temperature_2m_min":            []float64{4, 5, 6, 7},
			"temperature_2m_max":            []float64{12, 13, 14, 15},
			"precipitation_probability_max": []float64{0, 20, 60, 80},
			"weather_code":                  []int{1, 3, 61, 71},
		},
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
