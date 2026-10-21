package weatherprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

const forecastURL = "https://api.open-meteo.com/v1/forecast"

type Provider struct {
	settings Settings
	http     *http.Client
	endpoint string
	now      func() time.Time
}

func New(settings Settings) *Provider {
	return &Provider{
		settings: settings,
		http:     &http.Client{Timeout: settings.Timeout},
		endpoint: forecastURL,
		now:      time.Now,
	}
}

func (provider *Provider) Fetch(ctx context.Context) (Reading, error) {
	ctx, cancel := context.WithTimeout(ctx, provider.settings.Timeout)
	defer cancel()

	endpoint, err := provider.requestURL()
	if err != nil {
		return Reading{}, err
	}

	for attempt := 0; attempt < provider.settings.Attempts; attempt++ {
		reading, retry, err := provider.fetchOnce(ctx, endpoint)
		if err == nil {
			return reading, nil
		}
		if !retry || attempt+1 == provider.settings.Attempts {
			return Reading{}, err
		}

		select {
		case <-ctx.Done():
			return Reading{}, ctx.Err()
		case <-time.After(time.Duration(1<<attempt) * 250 * time.Millisecond):
		}
	}

	return Reading{}, fmt.Errorf("weather fetch requires at least one attempt")
}

func (provider *Provider) requestURL() (string, error) {
	endpoint, err := url.Parse(provider.endpoint)
	if err != nil {
		return "", err
	}

	query := endpoint.Query()
	query.Set("latitude", strconv.FormatFloat(provider.settings.Latitude, 'f', -1, 64))
	query.Set("longitude", strconv.FormatFloat(provider.settings.Longitude, 'f', -1, 64))
	query.Set("timezone", provider.settings.Timezone)
	query.Set("temperature_unit", "celsius")
	query.Set("wind_speed_unit", "ms")
	query.Set("timeformat", "unixtime")
	query.Set("forecast_days", "4")
	query.Set("current", "temperature_2m,apparent_temperature,relative_humidity_2m,weather_code,wind_speed_10m,is_day")
	query.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max")
	endpoint.RawQuery = query.Encode()

	return endpoint.String(), nil
}

func (provider *Provider) fetchOnce(ctx context.Context, endpoint string) (Reading, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Reading{}, false, err
	}
	request.Header.Set("User-Agent", "tc002-apps/server")

	response, err := provider.http.Do(request)
	if err != nil {
		return Reading{}, true, fmt.Errorf("fetch Open-Meteo: %w", err)
	}
	defer response.Body.Close()

	const limit = 1 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return Reading{}, true, fmt.Errorf("read Open-Meteo: %w", err)
	}
	if len(body) > limit {
		return Reading{}, false, fmt.Errorf("Open-Meteo response exceeds 1 MiB")
	}

	if response.StatusCode != http.StatusOK {
		detail := strings.TrimSpace(string(body))
		if len(detail) > 256 {
			detail = detail[:256]
		}

		retry := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
		return Reading{}, retry, fmt.Errorf("Open-Meteo HTTP %d: %s", response.StatusCode, detail)
	}

	var forecast forecastResponse
	if err := json.Unmarshal(body, &forecast); err != nil {
		return Reading{}, false, fmt.Errorf("decode Open-Meteo: %w", err)
	}

	reading, err := normalize(forecast, provider.settings, provider.now())

	return reading, false, err
}

// Publish attempts every selected screen, reporting partial delivery explicitly.
// On fetch failure the caller keeps the previous reading on the device unchanged.
func Publish(ctx context.Context, client *awtrix.Client, settings Settings, reading Reading) error {
	var failures []error

	for _, name := range settings.Screens {
		if err := client.SetData(ctx, name, map[string]any{"weather": reading}); err != nil {
			failures = append(failures, fmt.Errorf("publish weather to %s: %w", name, err))
		}
	}

	return errors.Join(failures...)
}
