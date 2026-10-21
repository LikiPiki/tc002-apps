package clockcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	production = strings.NewReplacer("hour()", "check_hour()", "minute()", "check_minute()", "display.", "check_state.", "shared.", "check_state.").Replace(production)
	test := "# @headless true\n" + parts[0] + production + "\nclass PolicyCheck" + parts[1]
	if err := c.InstallScript(ctx, "policy-behavior", []byte(test)); err != nil {
		return err
	}
	return c.CheckScript(ctx, "policy-behavior")
}

type Expectations struct {
	Nonempty bool `json:"nonempty"`
	Animated bool `json:"animated"`
}

// CheckFrames verifies app-specific expectations and saves an inspectable preview.
func CheckFrames(ctx context.Context, c *awtrix.Client, name, path, artifacts string) error {
	data, err := os.ReadFile(filepath.Join(filepath.Dir(path), "check.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var want Expectations
	if err := json.Unmarshal(data, &want); err != nil {
		return err
	}
	var frames []awtrix.Frame
	for i := 0; i < 12; i++ {
		frame, err := c.Frame(ctx)
		if err != nil {
			return err
		}
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height {
			return fmt.Errorf("invalid frame")
		}
		frames = append(frames, frame)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	if artifacts != "" {
		if err := SaveGIF(filepath.Join(artifacts, name+".gif"), frames); err != nil {
			return err
		}
	}
	lit, changed := false, false
	for _, f := range frames {
		for i, p := range f.Pixels {
			lit = lit || p != 0
			changed = changed || f.Width != frames[0].Width || f.Height != frames[0].Height || i >= len(frames[0].Pixels) || p != frames[0].Pixels[i]
		}
	}
	if want.Nonempty && !lit {
		return fmt.Errorf("%s rendered only black frames", name)
	}
	if want.Animated && !changed {
		return fmt.Errorf("%s animation did not change", name)
	}
	return nil
}

func SaveGIF(path string, frames []awtrix.Frame) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out := &gif.GIF{LoopCount: 0}
	for _, frame := range frames {
		rgba := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
		for i, p := range frame.Pixels {
			rgba.SetRGBA(i%frame.Width, i/frame.Width, color.RGBA{uint8(p >> 16), uint8(p >> 8), uint8(p), 255})
		}
		img := image.NewPaletted(rgba.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(img, img.Rect, rgba, image.Point{})
		out.Image = append(out.Image, img)
		out.Delay = append(out.Delay, 15)
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
