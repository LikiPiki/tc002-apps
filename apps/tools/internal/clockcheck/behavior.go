package clockcheck

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"
	"path/filepath"
	"strings"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

// PolicyBehavior runs the real policy class with a deterministic clock and display.
// Substitution is limited to the disposable sandbox; deployed sources are unchanged.
func PolicyBehavior(ctx context.Context, c *awtrix.Client, path string, source []byte) error {
	harness, err := os.ReadFile(filepath.Join(filepath.Dir(path), "check.be"))
	if err != nil {
		return err
	}

	parts := strings.SplitN(string(harness), "class PolicyCheck", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid policy harness")
	}

	production := strings.TrimSuffix(strings.TrimSpace(string(source)), "return DisplayPolicy()")
	replaceAPIs := strings.NewReplacer(
		"hour()", "check_hour()",
		"minute()", "check_minute()",
		"display.", "check_state.",
		"shared.", "check_state.",
	)
	production = replaceAPIs.Replace(production)

	test := "# @headless true\n" + parts[0] + production + "\nclass PolicyCheck" + parts[1]
	if err := c.InstallScript(ctx, "policy-behavior", []byte(test)); err != nil {
		return err
	}

	return c.CheckScript(ctx, "policy-behavior")
}

func SaveGIF(path string, frames []awtrix.Frame) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	out := &gif.GIF{LoopCount: 0}
	for _, frame := range frames {
		rgba := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
		for i, pixel := range frame.Pixels {
			rgb := color.RGBA{
				R: uint8(pixel >> 16),
				G: uint8(pixel >> 8),
				B: uint8(pixel),
				A: 255,
			}
			rgba.SetRGBA(i%frame.Width, i/frame.Width, rgb)
		}

		img := image.NewPaletted(rgba.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(img, img.Rect, rgba, image.Point{})
		out.Image = append(out.Image, img)
		out.Delay = append(out.Delay, 10)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	if err := gif.EncodeAll(f, out); err != nil {
		f.Close()
		return err
	}

	return f.Close()
}
