package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	projectconfig "github.com/LikiPiki/tc002-apps/internal/config"
)

func main() {
	base, personal := projectconfig.Paths(flag.CommandLine)
	once := flag.Bool("once", false, "run once and exit")
	interval := flag.Duration("interval", 0, "override server.worker.interval from TOML")
	flag.Parse()
	config, err := projectconfig.Load(*base, *personal)
	if err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}
	overridden := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "interval" {
			overridden = true
		}
	})
	if !overridden {
		*interval, err = config.WorkerInterval()
		if err != nil {
			slog.Error("invalid worker config", "error", err)
			os.Exit(1)
		}
	}
	if *interval <= 0 {
		slog.Error("interval must be positive")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	run := func() {
		// TODO: fetch data, transform it and publish through the server's awtrix client.
		slog.Info("worker tick", "mode", "placeholder", "interval", interval.String())
	}
	run()
	if *once {
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopped")
			return
		case <-ticker.C:
			run()
		}
	}
}
