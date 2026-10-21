package clockcheck

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

// ObserveHeadless checks background hooks without requiring an active display page.
func ObserveHeadless(ctx context.Context, c *awtrix.Client, name string, duration time.Duration) (int, error) {
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

// ObserveScript checks runtime/frame validity and optionally records a preview.
// Static or black frames are valid; visual quality is inspected in the GIF.
func ObserveScript(ctx context.Context, c *awtrix.Client, name string, duration time.Duration, artifacts string) (frames int, err error) {
	if duration < 250*time.Millisecond {
		return 0, fmt.Errorf("observation duration must be at least 250ms")
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	deadline := time.NewTimer(duration)
	defer deadline.Stop()

	var captured []awtrix.Frame
	defer func() {
		if len(captured) > 0 {
			path := filepath.Join(artifacts, name+".gif")
			err = errors.Join(err, SaveGIF(path, captured))
		}
	}()

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

			device, err := c.Device(ctx)
			if err != nil {
				return frames, err
			}
			if device.CurrentApp != name {
				return frames, fmt.Errorf("expected %s on screen, got %q", name, device.CurrentApp)
			}

			screen, err := c.Frame(ctx)
			if err != nil {
				return frames, err
			}
			if screen.Width <= 0 || screen.Width > 128 || screen.Height <= 0 || screen.Height > 32 || len(screen.Pixels) != screen.Width*screen.Height {
				return frames, fmt.Errorf("invalid pixel frame for %s: %dx%d, %d pixels", name, screen.Width, screen.Height, len(screen.Pixels))
			}

			frames++
			if artifacts != "" {
				captured = append(captured, screen)
			}
		}
	}
}
