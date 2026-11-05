package weatherprovider

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

func TestWTTRObservationAcrossUTCMidnight(t *testing.T) {
	now := time.Date(2026, 11, 5, 0, 10, 0, 0, time.UTC)
	raw := forecastFixture(t, now, "Europe/Moscow")
	raw["current_condition"].([]map[string]any)[0]["observation_time"] = "11:55 PM"
	var response forecastResponse
	if err := json.Unmarshal([]byte(encodeForecast(t, raw)), &response); err != nil {
		t.Fatal(err)
	}
	reading, err := normalize(response, testSettings(), now)
	if err != nil {
		t.Fatal(err)
	}
	want := strconv.FormatInt(now.Add(-15*time.Minute).UnixMilli(), 10)
	if reading.UpdatedAtMS != want || reading.Current.IsDay {
		t.Fatalf("incorrect midnight observation: %+v", reading)
	}
}

func TestWTTRRejectsInvalidForecast(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	for _, scenario := range []struct {
		name   string
		change func(map[string]any)
	}{
		{"old forecast", func(raw map[string]any) {
			raw["weather"].([]map[string]any)[0]["date"] = "2026-11-04"
		}},
		{"reversed range", func(raw map[string]any) {
			raw["weather"].([]map[string]any)[0]["mintempC"] = "30"
		}},
		{"missing hours", func(raw map[string]any) {
			raw["weather"].([]map[string]any)[0]["hourly"] = nil
		}},
		{"missing sunrise", func(raw map[string]any) {
			raw["weather"].([]map[string]any)[0]["astronomy"] = nil
		}},
		{"null current", func(raw map[string]any) {
			raw["current_condition"] = nil
		}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			raw := forecastFixture(t, now, "Europe/Moscow")
			scenario.change(raw)
			var response forecastResponse
			if err := json.Unmarshal([]byte(encodeForecast(t, raw)), &response); err != nil {
				t.Fatal(err)
			}
			if _, err := normalize(response, testSettings(), now); err == nil {
				t.Fatal("invalid source data was accepted")
			}
		})
	}
}

func TestWTTRConditionTranslation(t *testing.T) {
	for source, want := range map[string]string{
		"113": "CLEAR", "116": "CLOUD", "248": "FOG", "266": "DRIZZLE",
		"296": "RAIN", "338": "SNOW", "395": "STORM",
	} {
		code, err := wttrCode(source)
		if err != nil || condition(code) != want {
			t.Errorf("%s: condition=%s error=%v", source, condition(code), err)
		}
	}
}
