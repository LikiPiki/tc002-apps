package clockcheck

import (
	"context"
	"fmt"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

const SandboxURL = "http://awtrix-check:8080"

// CheckInvocation restricts the internal runner to the fixed Compose sandbox.
func CheckInvocation(target, marker string) error {
	if marker != "1" || target != SandboxURL {
		return fmt.Errorf("clockctl check is sandbox-only; use make clock-check (custom device URLs are not accepted)")
	}
	return nil
}

// RequireSandbox verifies the endpoint before the runner can write any scripts.
func RequireSandbox(ctx context.Context, c *awtrix.Client) error {
	device, err := c.Device(ctx)
	if err != nil {
		return err
	}
	if device.BoardType != "linux" {
		return fmt.Errorf("refusing runtime checks on boardType %q: expected headless Linux AWTRIX", device.BoardType)
	}
	return nil
}
