package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LikiPiki/tc002-apps/apps/server/internal/weatherprovider"
	"github.com/LikiPiki/tc002-apps/internal/buildinfo"
	"github.com/LikiPiki/tc002-apps/internal/clocks"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(buildinfo.StringFor("tc002-server"))
		return
	}

	if err := run(os.Args[1:]); err != nil && err != flag.ErrHelp {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("tc002-server", flag.ContinueOnError)
	base, personal := config.Paths(flags)
	once := flags.Bool("once", false, "fetch and publish once")
	printOnly := flags.Bool("print", false, "fetch once and print JSON without connecting to a clock")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}

	project, err := config.Load(*base, *personal)
	if err != nil {
		return err
	}

	settings, err := weatherprovider.Parse(project.Server.Weather)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	provider := weatherprovider.New(settings)
	if *printOnly {
		reading, err := provider.Fetch(ctx)
		if err != nil {
			return err
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		return encoder.Encode(reading)
	}

	if target := os.Getenv("AWTRIX_URL"); target != "" {
		if err := project.Clock.OverrideTarget(target); err != nil {
			return err
		}
	}

	profiles, err := project.Clock.ResolveProfiles()
	if err != nil {
		return err
	}

	return serve(ctx, provider, profiles, settings, *once)
}

type weatherSource interface {
	Fetch(context.Context) (weatherprovider.Reading, error)
}

func serve(ctx context.Context, provider weatherSource, profiles []config.ClockProfile, settings weatherprovider.Settings, once bool) error {
	var audience []config.ClockProfile
	for _, profile := range profiles {
		if len(settingsForClock(settings, profile.Clock).Screens) > 0 && len(profile.Clock.Targets()) > 0 {
			audience = append(audience, profile)
		}
	}
	if len(audience) == 0 {
		return fmt.Errorf("no clocks with selected weather screens configured")
	}

	for {
		reading, err := provider.Fetch(ctx)
		var delivery clocks.Report

		if err == nil {
			delivery, err = clocks.RunProfiles(ctx, audience, clocks.DefaultTimeout, func(ctx context.Context, target clocks.Target, profile config.ClockProfile) (any, error) {
				return nil, weatherprovider.Publish(ctx, target.Client, settingsForClock(settings, profile.Clock), reading)
			})
			delivery.Warn("weather publication")
			if err == nil {
				err = delivery.Err()
			}
		}

		if ctx.Err() != nil {
			return nil
		}

		if err != nil {
			if once {
				return err
			}

			slog.Error("weather update failed", "error", err)
		} else {
			slog.Info("weather published",
				"place", reading.Label,
				"temperature_c", reading.Current.TemperatureC,
				"condition", reading.Current.Condition,
				"clocks_updated", delivery.Succeeded(),
				"clocks_total", len(delivery.Results),
				"screens", settings.Screens)
		}

		if once {
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(settings.Interval):
		}
	}
}

func settingsForClock(settings weatherprovider.Settings, clock config.ClockConfig) weatherprovider.Settings {
	selected := settings
	selected.Screens = nil

	for _, name := range settings.Screens {
		if clock.ScreenSelected(name) {
			selected.Screens = append(selected.Screens, name)
		}
	}

	return selected
}
