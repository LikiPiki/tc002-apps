package weatherprovider

import (
	"fmt"
	"math"
	"regexp"
	"time"
	_ "time/tzdata" // IANA timezone validation must also work in minimal containers.

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

type Settings struct {
	Latitude  float64
	Longitude float64
	Label     string
	Timezone  string
	Screens   []string
	Interval  time.Duration
	Timeout   time.Duration
	TTL       time.Duration
	Attempts  int
}

var labelPattern = regexp.MustCompile(`^[A-Za-z0-9 _-]{1,8}$`)

func Parse(raw config.WeatherConfig) (Settings, error) {
	settings := Settings{
		Label:    raw.Label,
		Timezone: raw.Timezone,
		Attempts: raw.Attempts,
	}

	if raw.Latitude == nil || raw.Longitude == nil {
		return settings, fmt.Errorf("set server.weather.latitude and longitude in your project config")
	}
	if !inRange(*raw.Latitude, -90, 90) || !inRange(*raw.Longitude, -180, 180) {
		return settings, fmt.Errorf("weather coordinates must be finite: latitude -90..90, longitude -180..180")
	}
	settings.Latitude = *raw.Latitude
	settings.Longitude = *raw.Longitude

	if !labelPattern.MatchString(raw.Label) {
		return settings, fmt.Errorf("server.weather.label requires 1-8 ASCII letters, digits, spaces, _ or -")
	}
	if raw.Timezone != "auto" {
		if _, err := time.LoadLocation(raw.Timezone); err != nil || raw.Timezone == "" || raw.Timezone == "Local" {
			return settings, fmt.Errorf("server.weather.timezone must be auto or an IANA timezone")
		}
	}
	if raw.Attempts < 1 || raw.Attempts > 5 {
		return settings, fmt.Errorf("server.weather.attempts must be 1-5")
	}

	seen := map[string]bool{}
	for _, name := range raw.Screens {
		if err := awtrix.ValidateAppName(name); err != nil {
			return settings, err
		}
		if !seen[name] {
			settings.Screens = append(settings.Screens, name)
			seen[name] = true
		}
	}
	if len(settings.Screens) == 0 {
		return settings, fmt.Errorf("server.weather.screens must contain at least one installed screen")
	}

	durations := []struct {
		name   string
		raw    string
		target *time.Duration
	}{
		{"interval", raw.Interval, &settings.Interval},
		{"timeout", raw.Timeout, &settings.Timeout},
		{"ttl", raw.TTL, &settings.TTL},
	}

	for _, duration := range durations {
		value, err := time.ParseDuration(duration.raw)
		if err != nil || value < time.Second || value > 24*time.Hour {
			return settings, fmt.Errorf("server.weather.%s must be between 1s and 24h", duration.name)
		}
		*duration.target = value
	}

	if settings.Interval < time.Minute {
		return settings, fmt.Errorf("server.weather.interval must be at least 1m")
	}
	if settings.TTL <= settings.Interval+settings.Timeout+15*time.Minute {
		return settings, fmt.Errorf("server.weather.ttl must exceed interval + timeout + the 15m source update period")
	}

	return settings, nil
}

func inRange(value, minimum, maximum float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= minimum && value <= maximum
}
