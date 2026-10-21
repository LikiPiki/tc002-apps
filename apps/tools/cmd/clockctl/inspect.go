package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
	"github.com/LikiPiki/tc002-apps/internal/clocks"
	projectconfig "github.com/LikiPiki/tc002-apps/internal/config"
)

type inspection struct {
	Target       string              `json:"target"`
	Device       awtrix.DeviceInfo   `json:"device"`
	Apps         []awtrix.ScriptInfo `json:"apps"`
	Capabilities map[string]any      `json:"capabilities,omitempty"`
	Width        int                 `json:"width,omitempty"`
	Height       int                 `json:"height,omitempty"`
	Problems     []string            `json:"problems"`
}

func runInspection(command string, args []string) error {
	flags := flag.NewFlagSet("clockctl "+command, flag.ContinueOnError)
	base, personal := projectconfig.Paths(flags)
	url := flags.String("url", "", "AWTRIX URL (or AWTRIX_URL/config)")
	jsonOutput := flags.Bool("json", false, "print machine-readable JSON")
	timeout := flags.Duration("timeout", 15*time.Second, "timeout per clock")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *timeout <= 0 {
		return fmt.Errorf("unexpected arguments or nonpositive timeout")
	}

	config, err := loadTargetConfig(flags, *base, *personal, *url)
	if err != nil {
		return err
	}

	ctx, cancel := commandContext(2 * time.Minute)
	defer cancel()

	profiles, err := config.Clock.ResolveProfiles()
	if err != nil {
		return err
	}

	report, err := clocks.RunProfiles(ctx, profiles, *timeout, func(ctx context.Context, target clocks.Target, profile projectconfig.ClockProfile) (any, error) {
		result := inspectTarget(ctx, target.Client, target.URL, command == "doctor")
		if len(result.Problems) > 0 {
			return result, fmt.Errorf("%s found %d problem(s)", command, len(result.Problems))
		}

		return result, nil
	})
	if err != nil {
		return err
	}

	return finishDelivery(ctx, report, command, *jsonOutput)
}

func (result inspection) String() string {
	var output strings.Builder
	fmt.Fprintf(&output, "Active app: %s\nBerry running: %v\n",
		result.Device.CurrentApp, result.Device.ScriptingRunning)

	if result.Capabilities != nil {
		fmt.Fprintf(&output, "Display: %dx%d\nAtomic updates: %v\n",
			result.Width, result.Height, result.Capabilities["scriptUpdates"])
	}

	for _, app := range result.Apps {
		fmt.Fprintf(&output, "  %s (%s): enabled=%v headless=%v on-demand=%v error=%s\n",
			app.Name, app.Origin, app.Enabled, app.Headless, app.OnDemand, app.Error)
	}
	for _, problem := range result.Problems {
		fmt.Fprintf(&output, "Problem: %s\n", problem)
	}

	return output.String()
}

func inspectTarget(ctx context.Context, client *awtrix.Client, target string, doctor bool) inspection {
	result := inspection{Target: target, Problems: []string{}}
	var err error
	result.Device, err = client.Device(ctx)
	if err != nil {
		result.Problems = append(result.Problems, err.Error())
	} else if !result.Device.ScriptingRunning {
		result.Problems = append(result.Problems, "Berry scripting is not running")
	}

	result.Apps, err = client.Scripts(ctx)
	if err != nil {
		result.Problems = append(result.Problems, err.Error())
	}

	for _, app := range result.Apps {
		if app.Origin == "script" && len(app.Error) > 0 && string(app.Error) != "null" {
			result.Problems = append(result.Problems, fmt.Sprintf("%s: %s", app.Name, app.Error))
		}
	}

	if !doctor {
		return result
	}

	result.Capabilities, err = client.Capabilities(ctx)
	if err != nil {
		result.Problems = append(result.Problems, err.Error())
	} else if result.Capabilities["scriptUpdates"] != true {
		result.Problems = append(result.Problems, "atomic script updates are unavailable")
	}

	frame, err := client.Frame(ctx)
	if err != nil {
		result.Problems = append(result.Problems, err.Error())
	} else {
		result.Width = frame.Width
		result.Height = frame.Height
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height {
			result.Problems = append(result.Problems, "invalid display frame")
		}
	}

	return result
}
