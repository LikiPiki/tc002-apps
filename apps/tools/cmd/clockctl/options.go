package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/clocks"
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
		if err := config.Clock.OverrideTarget(explicitURL); err != nil {
			return config, err
		}
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

func finishDelivery(ctx context.Context, report clocks.Report, operation string, jsonOutput bool) error {
	report.Warn(operation)

	if jsonOutput {
		if err := printJSON(report); err != nil {
			return err
		}
	} else {
		for _, result := range report.Results {
			if result.Profile != "" {
				fmt.Printf("Profile: %s\n", result.Profile)
			}

			fmt.Printf("Target: %s [%s]\n", result.Target, result.State)
			if result.Detail != nil {
				fmt.Println(result.Detail)
			}
			if result.Error != "" {
				fmt.Printf("  %s\n", result.Error)
			}
		}

		fmt.Printf("Clocks: %d/%d completed\n", report.Succeeded(), len(report.Results))
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return report.Err()
}
