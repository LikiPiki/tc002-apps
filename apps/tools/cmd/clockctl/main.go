package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
	"github.com/LikiPiki/tc002-apps/internal/buildinfo"
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
		if os.Args[1] == "build-list" || os.Args[1] == "server-list" {
			if err := runBuildList(os.Args[1], os.Args[2:]); err != nil && err != flag.ErrHelp {
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
		fmt.Fprintln(os.Stderr, "Commands: deploy, rollback, validate, show, start, stop, list, check, doctor, status, build-list, server-list")
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
	client, err := awtrix.NewClient(config.Clock.URL)
	if err == nil {
		ctx, cancel := commandContext(10 * time.Second)
		defer cancel()
		err = client.Notify(ctx, *text)
	}
	if err != nil {
		slog.Error("notification failed", "error", err)
		os.Exit(1)
	}
	slog.Info("notification sent")
}
