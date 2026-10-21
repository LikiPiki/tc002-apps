package weatherprovider

import (
	"testing"

	"github.com/LikiPiki/tc002-apps/internal/config"
)

func TestWeatherSettingsRequireLocationAndValidTiming(t *testing.T) {
	project, err := config.Load("", "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Parse(project.Server.Weather); err == nil {
		t.Fatal("missing coordinates were accepted")
	}

	latitude, longitude := 0.0, 0.0
	raw := project.Server.Weather
	raw.Latitude = &latitude
	raw.Longitude = &longitude

	if _, err := Parse(raw); err != nil {
		t.Fatalf("zero coordinates must be valid: %v", err)
	}

	raw.TTL = "15m"
	if _, err := Parse(raw); err == nil {
		t.Fatal("TTL shorter than the update budget was accepted")
	}
}
