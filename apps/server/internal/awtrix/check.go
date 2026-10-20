package awtrix

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ObserveHeadless checks background hooks without requiring an active display page.
func (c *Client) ObserveHeadless(ctx context.Context, name string, duration time.Duration) (int, error) {
	if duration < 1500*time.Millisecond {
		return 0, fmt.Errorf("headless observation requires at least 1500ms to exercise loop()")
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(duration)
	defer deadline.Stop()
	checks := 0
	for {
		select {
		case <-ctx.Done():
			return checks, ctx.Err()
		case <-deadline.C:
			return checks, c.CheckScript(ctx, name)
		case <-ticker.C:
			if err := c.CheckScript(ctx, name); err != nil {
				return checks, err
			}
			checks++
		}
	}
}

// ObserveScript samples an already displayed app. Static or black frames are valid.
func (c *Client) ObserveScript(ctx context.Context, name string, duration time.Duration) (int, error) {
	if duration < 250*time.Millisecond {
		return 0, fmt.Errorf("observation duration must be at least 250ms")
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(duration)
	defer deadline.Stop()
	frames := 0
	for {
		select {
		case <-ctx.Done():
			return frames, ctx.Err()
		case <-deadline.C:
			if frames < 2 {
				return frames, fmt.Errorf("not enough frames observed for %s", name)
			}
			return frames, c.CheckScript(ctx, name)
		case <-ticker.C:
			if err := c.CheckScript(ctx, name); err != nil {
				return frames, err
			}
			var device struct {
				CurrentApp string `json:"currentApp"`
			}
			if err := c.request(ctx, http.MethodGet, "/api/v1/device", "", nil, &device); err != nil {
				return frames, err
			}
			if device.CurrentApp != name {
				return frames, fmt.Errorf("expected %s on screen, got %q", name, device.CurrentApp)
			}
			var screen struct {
				Width  int      `json:"width"`
				Height int      `json:"height"`
				Pixels []uint32 `json:"pixels"`
			}
			if err := c.request(ctx, http.MethodGet, "/api/v1/display/screen", "", nil, &screen); err != nil {
				return frames, err
			}
			if screen.Width <= 0 || screen.Width > 128 || screen.Height <= 0 || screen.Height > 32 || len(screen.Pixels) != screen.Width*screen.Height {
				return frames, fmt.Errorf("invalid pixel frame for %s: %dx%d, %d pixels", name, screen.Width, screen.Height, len(screen.Pixels))
			}
			frames++
		}
	}
}
