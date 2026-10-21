package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockapps"
	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockcheck"
	"github.com/LikiPiki/tc002-apps/internal/awtrix"
	projectconfig "github.com/LikiPiki/tc002-apps/internal/config"
)

func runScripts(command string, args []string) error {
	flags := flag.NewFlagSet("clockctl "+command, flag.ContinueOnError)
	base, personal := projectconfig.Paths(flags)
	app := flags.String("app", "", "one clock app by unique install name")
	kind := flags.String("kind", "all", "all, screens, services, or on-demand")
	url := flags.String("url", "", "override target URL (otherwise AWTRIX_URL, then config)")
	duration := flags.Duration("duration", 2*time.Second, "check: observation duration (headless requires at least 1500ms)")
	timeout := flags.Duration("timeout", 2*time.Minute, "whole command timeout")
	dryRun := flags.Bool("dry-run", false, "deploy: inspect changes without writing")
	backup := flags.String("backup", "", "deploy: new recovery file; rollback: recovery file to restore")
	observe := flags.Duration("observe", 1500*time.Millisecond, "deploy: verification window per enabled app")
	appsDir := flags.String("apps-dir", "apps/clock", "clock app catalog directory")
	jsonOutput := flags.Bool("json", false, "JSON for list, validate, or deploy")
	all := flags.Bool("all", false, "check: include every catalog app regardless of deploy selection")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if *timeout <= 0 || *observe < 0 {
		return fmt.Errorf("timeout must be positive and observe nonnegative")
	}
	if *all && command != "check" {
		return fmt.Errorf("--all is only supported by check")
	}
	if *jsonOutput && command != "list" && command != "validate" && command != "deploy" {
		return fmt.Errorf("--json is supported by list, validate, deploy, doctor and status")
	}

	ctx, cancel := commandContext(*timeout)
	defer cancel()

	if command == "check" {
		if err := clockcheck.CheckInvocation(*url, os.Getenv("TC002_CHECK_SANDBOX")); err != nil {
			return err
		}
		if *duration < 250*time.Millisecond {
			return fmt.Errorf("check requires --duration >= 250ms")
		}
	}

	config, err := loadTargetConfig(flags, *base, *personal, *url)
	if err != nil {
		return err
	}

	options := scriptOptions{
		App:      *app,
		Kind:     *kind,
		AppsDir:  *appsDir,
		Backup:   *backup,
		Duration: *duration,
		Timeout:  *timeout,
		Observe:  *observe,
		DryRun:   *dryRun,
		JSON:     *jsonOutput,
		All:      *all,
	}

	return dispatchScripts(ctx, command, config.Clock, options)
}

func checkScript(parent context.Context, client *awtrix.Client, script clockapps.Script, duration time.Duration) error {
	ctx, cancel := context.WithTimeout(parent, duration+10*time.Second)
	defer cancel()

	if script.Kind == "services" {
		if script.Name == "display-policy" {
			if err := clockcheck.PolicyBehavior(ctx, client, script.Path, script.Source); err != nil {
				return fmt.Errorf("policy behavior: %w", err)
			}
		}

		checks, err := clockcheck.ObserveHeadless(ctx, client, script.Name, duration)
		if err != nil {
			return err
		}

		if err := client.SetScriptEnabled(ctx, script.Name, false); err != nil {
			return err
		}

		fmt.Printf("Checked services/%s: %d status checks, no runtime errors\n", script.Name, checks)
		return nil
	}

	if err := client.SetDisplayPower(ctx, true); err != nil {
		return err
	}
	if err := client.ShowScript(ctx, script.Name); err != nil {
		return err
	}

	frames, err := clockcheck.ObserveScript(ctx, client, script.Name, duration, os.Getenv("TC002_CHECK_ARTIFACTS"))
	if err != nil {
		return err
	}

	if script.Kind == "on-demand" {
		if err := client.StopOnDemand(ctx, script.Name); err != nil {
			return err
		}
	}

	fmt.Printf("Checked %s/%s: %d frames, no runtime errors\n", script.Kind, script.Name, frames)
	return nil
}
