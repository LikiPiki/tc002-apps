package weatherprovider

import (
	"context"
	"fmt"
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
		t.Skip("run make weather-check for the disposable AWTRIX integration")
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

	// The forecast is one installed screen that cycles through the three days.
	first := waitWeatherFrame(t, ctx, client, "weather-forecast", 0x22CC66, nil)
	waitWeatherFrame(t, ctx, client, "weather-forecast", 0x22CC66, first.Pixels)

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
			t.Fatalf("%s did not render expected marker %#x or a changed forecast page", name, marker)
		case <-time.After(100 * time.Millisecond):
		}
	}
}
