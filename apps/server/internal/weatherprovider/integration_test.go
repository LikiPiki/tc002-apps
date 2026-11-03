package weatherprovider

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

func TestWeatherClockIntegration(t *testing.T) {
	url := os.Getenv("TC002_INTEGRATION_URL")
	if url == "" {
		t.Skip("run make server-check for the disposable AWTRIX integration")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	client, err := awtrix.NewClient(url)
	if err != nil {
		t.Fatal(err)
	}

	settings := testSettings()
	for _, name := range settings.Screens {
		path := "/repo/apps/clock/screens/" + name + "/main.be"
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		if err := client.InstallScript(ctx, name, source); err != nil {
			t.Fatal(err)
		}
	}

	if err := client.SetDisplayPower(ctx, true); err != nil {
		t.Fatal(err)
	}

	for _, name := range settings.Screens {
		showWeatherScreen(t, ctx, client, name)
		waitWeatherFrame(t, ctx, client, name, 0xFFAA22, nil)
	}

	now := time.Now()
	body := encodeForecast(t, forecastFixture(t, now, "Europe/Moscow"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)

	provider := New(settings)
	provider.endpoint = server.URL
	reading, err := provider.Fetch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err := Publish(ctx, client, settings, reading); err != nil {
		t.Fatal(err)
	}

	for _, name := range settings.Screens {
		showWeatherScreen(t, ctx, client, name)
		waitWeatherFrame(t, ctx, client, name, 0x22CC66, nil)
	}

	// The summary stays on one page, with all metrics visible together.
	first := waitWeatherFrame(t, ctx, client, "weather-current", 0x22CC66, nil)
	time.Sleep(3200 * time.Millisecond)
	second := waitWeatherFrame(t, ctx, client, "weather-current", 0x22CC66, nil)
	if !slices.Equal(first.Pixels, second.Pixels) {
		t.Fatal("weather summary changed pages")
	}
	for _, shade := range []uint32{0xEEF1EA, 0xEFBB66, 0x62C891, 0x65B9EC} {
		if !slices.Contains(first.Pixels, shade) {
			t.Fatalf("weather summary is missing metric color %#x", shade)
		}
	}
	for _, scenario := range []struct {
		condition                    string
		day                          bool
		temp, feels, wind, low, high float64
	}{
		{"CLOUD", true, 5.9, 0.8, 6.2, 4.4, 7.1},
		{"CLEAR", true, 23, 25, 2, 17, 26},
		{"CLEAR", false, 0, 0, 0, 0, 0},
		{"SNOW", true, -100, -120, 150, -100, -99},
		{"RAIN", true, 8, 4, 9, 5, 11},
		{"DRIZZLE", true, 8, 4, 9, 5, 11},
		{"STORM", true, 8, 4, 9, 5, 11},
		{"FOG", true, 8, 4, 9, 5, 11},
	} {
		reading.Current.Condition = scenario.condition
		reading.Current.IsDay = scenario.day
		reading.Current.TemperatureC = scenario.temp
		reading.Current.FeelsLikeC = scenario.feels
		reading.Current.WindSpeedMS = scenario.wind
		reading.Today.MinimumC = scenario.low
		reading.Today.MaximumC = scenario.high
		if err := Publish(ctx, client, settings, reading); err != nil {
			t.Fatal(err)
		}
		showWeatherScreen(t, ctx, client, "weather-current")
		frame := waitWeatherFrame(t, ctx, client, "weather-current", 0x22CC66, nil)
		position := 0.5
		if scenario.high != scenario.low {
			position = (scenario.temp - scenario.low) / (scenario.high - scenario.low)
		}
		marker := 17 + int(math.Round(16*max(0, min(1, position))))
		if frame.Pixels[15*52+marker] != 0xEEF1EA {
			t.Fatalf("%s: range marker is misplaced", scenario.condition)
		}
	}

	reading.UpdatedAtMS = strconv.FormatInt(now.Add(-3*time.Hour).UnixMilli(), 10)
	if err := Publish(ctx, client, settings, reading); err != nil {
		t.Fatal(err)
	}

	for _, name := range settings.Screens {
		showWeatherScreen(t, ctx, client, name)
		waitWeatherFrame(t, ctx, client, name, 0xFFAA22, nil)
	}

	// Unsupported payloads must display NO DATA instead of breaking draw().
	unsupported := map[string]any{
		"weather": map[string]any{"version": 99},
	}

	for _, name := range settings.Screens {
		if err := client.SetData(ctx, name, unsupported); err != nil {
			t.Fatal(err)
		}

		showWeatherScreen(t, ctx, client, name)
		waitWeatherFrame(t, ctx, client, name, 0xFFAA22, nil)
	}
}

func showWeatherScreen(t *testing.T, ctx context.Context, client *awtrix.Client, name string) {
	t.Helper()

	if err := client.ShowScript(ctx, name); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	case <-time.After(100 * time.Millisecond):
	}
}

func waitWeatherFrame(t *testing.T, ctx context.Context, client *awtrix.Client, name string, marker uint32, previous []uint32) awtrix.Frame {
	t.Helper()

	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()

	for {
		if err := client.CheckScript(ctx, name); err != nil {
			t.Fatal(err)
		}

		device, err := client.Device(ctx)
		if err != nil {
			t.Fatal(err)
		}

		frame, err := client.Frame(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if device.CurrentApp == name && frame.Width == 52 && frame.Height == 16 &&
			len(frame.Pixels) == 52*16 && frame.Pixels[51] == marker &&
			(previous == nil || !slices.Equal(frame.Pixels, previous)) {
			return frame
		}

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-deadline.C:
			t.Fatalf("%s did not render the expected weather frame with marker %#x", name, marker)
		case <-time.After(100 * time.Millisecond):
		}
	}
}
