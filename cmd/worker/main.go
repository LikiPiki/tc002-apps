package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	once := flag.Bool("once", false, "run once and exit")
	interval := flag.Duration("interval", time.Minute, "interval between runs")
	flag.Parse()
	if *interval <= 0 {
		slog.Error("interval must be positive")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	run := func() {
		// TODO: fetch data, transform it and publish through internal/awtrix.
		slog.Info("worker tick", "mode", "placeholder")
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
