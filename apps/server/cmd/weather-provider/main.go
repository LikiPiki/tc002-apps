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
	"github.com/LikiPiki/tc002-apps/internal/awtrix"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

func main() {
	if err := run(os.Args[1:]); err != nil && err != flag.ErrHelp {
		slog.Error("weather provider failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("weather-provider", flag.ContinueOnError)
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
		project.Clock.URL = target
	}

	client, err := awtrix.NewClient(project.Clock.URL)
	if err != nil {
		return err
	}

	return serve(ctx, provider, client, settings, *once)
}

func serve(ctx context.Context, provider *weatherprovider.Provider, client *awtrix.Client, settings weatherprovider.Settings, once bool) error {
	for {
		reading, err := provider.Fetch(ctx)
		if err == nil {
			err = weatherprovider.Publish(ctx, client, settings, reading)
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
