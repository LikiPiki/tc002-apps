package statusprovider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

type Settings struct {
	URL      string
	App      string
	Label    string
	Interval time.Duration
	Timeout  time.Duration
	TTL      time.Duration
	Attempts int
}

type Reading struct {
	Version     int    `json:"version"`
	State       string `json:"state"`
	Label       string `json:"label"`
	UpdatedAtMS string `json:"updated_at_ms"`
	TTLSeconds  int64  `json:"ttl_seconds"`
	LatencyMS   int64  `json:"latency_ms"`
	HTTPStatus  int    `json:"http_status"`
}

var labelPattern = regexp.MustCompile(`^[A-Za-z0-9 _-]{1,8}$`)

func Parse(raw config.StatusConfig) (Settings, error) {
	settings := Settings{URL: raw.URL, App: raw.App, Label: raw.Label, Attempts: raw.Attempts}
	parsedURL, err := url.Parse(raw.URL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return settings, fmt.Errorf("server.status.url must be an absolute HTTP(S) URL")
	}
	if err := awtrix.ValidateAppName(raw.App); err != nil {
		return settings, err
	}
	if !labelPattern.MatchString(raw.Label) {
		return settings, fmt.Errorf("server.status.label requires 1-8 ASCII letters, digits, spaces, _ or -")
	}
	if raw.Attempts < 1 || raw.Attempts > 5 {
		return settings, fmt.Errorf("server.status.attempts must be 1-5")
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
			return settings, fmt.Errorf("server.status.%s must be between 1s and 24h", duration.name)
		}
		*duration.target = value
	}
	if settings.TTL <= settings.Interval+settings.Timeout {
		return settings, fmt.Errorf("server.status.ttl must exceed interval + timeout")
	}
	return settings, nil
}

func Probe(ctx context.Context, settings Settings) (Reading, error) {
	ctx, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	reading := Reading{
		Version:    1,
		State:      "down",
		Label:      settings.Label,
		TTLSeconds: int64(settings.TTL / time.Second),
	}
	client := &http.Client{Timeout: settings.Timeout}
	started := time.Now()
	for attempt := 0; attempt < settings.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			break
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, settings.URL, nil)
		if err != nil {
			return reading, err
		}
		response, err := client.Do(request)
		if err == nil {
			reading.HTTPStatus = response.StatusCode
			// The health check only needs headers; bound body draining for reuse.
			io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
			response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				reading.State = "up"
				break
			}
			if response.StatusCode < 500 {
				break
			}
		}
		if attempt+1 < settings.Attempts {
			select {
			case <-ctx.Done():
			case <-time.After(time.Duration(1<<attempt) * 100 * time.Millisecond):
			}
		}
	}
	reading.LatencyMS = time.Since(started).Milliseconds()
	reading.UpdatedAtMS = strconv.FormatInt(time.Now().UnixMilli(), 10)
	return reading, nil
}

func Publish(ctx context.Context, client *awtrix.Client, settings Settings, reading Reading) error {
	return client.SetData(ctx, settings.App, map[string]any{"status": reading})
}
