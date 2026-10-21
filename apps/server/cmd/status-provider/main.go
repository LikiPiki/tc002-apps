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
	"github.com/LikiPiki/tc002-apps/internal/awtrix"
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
		project.Clock.URL = target
	}
	settings, err := statusprovider.Parse(project.Server.Status)
	if err != nil {
		return err
	}
	client, err := awtrix.NewClient(project.Clock.URL)
	if err != nil {
		return err
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
		err = statusprovider.Publish(ctx, client, settings, reading)
		if err != nil {
			if *once {
				return err
			}
			slog.Error("status publication failed", "error", err)
		} else {
			slog.Info("status published", "state", reading.State, "http_status", reading.HTTPStatus, "latency_ms", reading.LatencyMS)
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
