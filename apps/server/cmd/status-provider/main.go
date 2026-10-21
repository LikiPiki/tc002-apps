package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LikiPiki/tc002-apps/apps/server/internal/statusprovider"
	"github.com/LikiPiki/tc002-apps/internal/clocks"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("status provider failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	base, personal := config.Paths(flag.CommandLine)
	once := flag.Bool("once", false, "probe and publish once")
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}

	project, err := config.Load(*base, *personal)
	if err != nil {
		return err
	}
	if target := os.Getenv("AWTRIX_URL"); target != "" {
		if err := project.Clock.OverrideTarget(target); err != nil {
			return err
		}
	}

	settings, err := statusprovider.Parse(project.Server.Status)
	if err != nil {
		return err
	}

	profiles, err := project.Clock.ResolveProfiles()
	if err != nil {
		return err
	}

	var audience []config.ClockProfile
	for _, profile := range profiles {
		if profile.Clock.ScreenSelected(settings.App) && len(profile.Clock.Targets()) > 0 {
			audience = append(audience, profile)
		}
	}
	if len(audience) == 0 {
		return fmt.Errorf("no clocks with selected status screens configured")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	for {
		reading, err := statusprovider.Probe(ctx, settings)
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return nil
		}

		delivery, err := clocks.RunProfiles(ctx, audience, clocks.DefaultTimeout, func(ctx context.Context, target clocks.Target, profile config.ClockProfile) (any, error) {
			return nil, statusprovider.Publish(ctx, target.Client, settings, reading)
		})
		delivery.Warn("status publication")
		if err == nil {
			err = delivery.Err()
		}
		if err != nil {
			if *once {
				return err
			}

			slog.Error("status publication failed", "error", err)
		} else {
			slog.Info("status published",
				"state", reading.State,
				"http_status", reading.HTTPStatus,
				"latency_ms", reading.LatencyMS,
				"clocks_updated", delivery.Succeeded(),
				"clocks_total", len(delivery.Results))
		}

		if *once {
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(settings.Interval):
		}
	}
}
