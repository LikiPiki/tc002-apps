package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	projectconfig "github.com/LikiPiki/tc002-apps/internal/config"
)

// Explicit URLs allow standalone commands without an implicit project config.
// An explicitly selected config must still exist and parse successfully.
func loadTargetConfig(flags *flag.FlagSet, base, personal, explicitURL string) (projectconfig.Config, error) {
	if explicitURL == "" {
		explicitURL = os.Getenv("AWTRIX_URL")
	}

	configExplicit := false
	flags.Visit(func(option *flag.Flag) {
		if option.Name == "config" {
			configExplicit = true
		}
	})

	_, configFromEnvironment := os.LookupEnv("TC002_CONFIG")
	if explicitURL != "" && !configExplicit && !configFromEnvironment {
		if _, err := os.Stat(base); errors.Is(err, os.ErrNotExist) {
			base = ""
		}
	}

	config, err := projectconfig.Load(base, personal)
	if err != nil {
		return config, err
	}
	if explicitURL != "" {
		config.Clock.URL = explicitURL
	}

	return config, nil
}

func commandContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithTimeout(signalContext, timeout)

	return ctx, func() {
		cancel()
		stop()
	}
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	return encoder.Encode(value)
}
