package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/buildinfo"
	"github.com/LikiPiki/tc002-apps/internal/clocks"
	projectconfig "github.com/LikiPiki/tc002-apps/internal/config"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println(buildinfo.String())
		return
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "doctor" || os.Args[1] == "status" {
			if err := runInspection(os.Args[1], os.Args[2:]); err != nil && err != flag.ErrHelp {
				slog.Error("inspection failed", "error", err)
				os.Exit(1)
			}
			return
		}

		if os.Args[1] == "tool-list" {
			if err := runToolList(os.Args[2:]); err != nil && err != flag.ErrHelp {
				slog.Error("build selection failed", "error", err)
				os.Exit(1)
			}
			return
		}

		switch os.Args[1] {
		case "deploy", "show", "start", "stop", "list", "check", "validate", "rollback":
			if err := runScripts(os.Args[1], os.Args[2:]); err != nil {
				if err == flag.ErrHelp {
					return
				}

				slog.Error("clock command failed", "error", err)
				os.Exit(1)
			}
			return
		}
	}

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: clockctl [notification flags] | <command> [flags]")
		fmt.Fprintln(os.Stderr, "Commands: deploy, rollback, validate, show, start, stop, list, check, doctor, status, tool-list")
		fmt.Fprintln(os.Stderr, "Run clockctl deploy --help for Berry upload options.")
		flag.PrintDefaults()
	}

	base, personal := projectconfig.Paths(flag.CommandLine)
	url := flag.String("url", os.Getenv("AWTRIX_URL"), "AWTRIX NG base URL (or AWTRIX_URL)")
	text := flag.String("text", "Hello from Go", "notification text")
	flag.Parse()
	if flag.NArg() != 0 {
		slog.Error("unexpected arguments; use deploy, show, list, or notification flags")
		os.Exit(1)
	}

	config, err := loadTargetConfig(flag.CommandLine, *base, *personal, *url)
	if err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := commandContext(2 * time.Minute)
	defer cancel()

	profiles, err := config.Clock.ResolveProfiles()
	if err != nil {
		slog.Error("invalid clock profiles", "error", err)
		os.Exit(1)
	}

	report, err := clocks.RunProfiles(ctx, profiles, clocks.DefaultTimeout, func(ctx context.Context, target clocks.Target, profile projectconfig.ClockProfile) (any, error) {
		return "Notification sent", target.Client.Notify(ctx, *text)
	})
	if err == nil {
		err = finishDelivery(ctx, report, "notification", false)
	}

	if err != nil {
		slog.Error("notification failed", "error", err)
		os.Exit(1)
	}

	slog.Info("notification sent")
}
