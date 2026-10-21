package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockapps"
	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockcheck"
	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockdeploy"
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
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if *timeout <= 0 || *observe < 0 {
		return fmt.Errorf("timeout must be positive and observe nonnegative")
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
	client, err := awtrix.NewClient(config.Clock.URL)
	if err != nil {
		return err
	}
	if !*jsonOutput {
		fmt.Printf("Target: %s\n", config.Clock.URL)
	}
	if command == "check" {
		if err := clockcheck.RequireSandbox(ctx, client); err != nil {
			return err
		}
	}
	if command == "rollback" {
		if *backup == "" {
			return fmt.Errorf("rollback requires --backup")
		}
		plan, err := clockdeploy.LoadPlan(*backup, config.Clock.URL)
		if err != nil {
			return err
		}
		return plan.Rollback(ctx, client)
	}
	if command == "show" || command == "start" || command == "stop" {
		if *app == "" {
			return fmt.Errorf("%s requires APP=<name>", command)
		}
		if err := client.CheckScripting(ctx); err != nil {
			return err
		}
		info, err := client.Script(ctx, *app)
		if err != nil {
			return err
		}
		if command == "stop" {
			if err := client.StopOnDemand(ctx, *app); err != nil {
				return err
			}
			fmt.Printf("Stopped %s\n", *app)
			return nil
		}
		if (command == "start") != info.OnDemand {
			return fmt.Errorf("use clock-start for on-demand apps and clock-show for rotating screens")
		}
		if err := client.ShowScript(ctx, *app); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
		if err := client.CheckScript(ctx, *app); err != nil {
			return err
		}
		fmt.Printf("Showing %s\n", *app)
		return nil
	}
	scripts, err := clockapps.Select(*appsDir, config.Clock, *app, *kind)
	if err != nil {
		return err
	}
	if len(scripts) == 0 {
		if *jsonOutput {
			return printJSON(map[string]any{"target": config.Clock.URL, "apps": []string{}})
		}
		fmt.Println("No clock apps selected")
		return nil
	}
	if command == "validate" {
		if *jsonOutput {
			return printJSON(map[string]any{"target": config.Clock.URL, "validated": len(scripts)})
		}
		fmt.Printf("Validated %d clock apps\n", len(scripts))
		return nil
	}
	if command == "list" && *jsonOutput {
		catalog := make([]map[string]string, 0, len(scripts))
		for _, script := range scripts {
			catalog = append(catalog, map[string]string{"name": script.Name, "kind": script.Kind, "path": script.Path})
		}
		return printJSON(map[string]any{"target": config.Clock.URL, "apps": catalog})
	}
	if command != "list" {
		if err := client.CheckScripting(ctx); err != nil {
			return err
		}
	}
	if command == "deploy" {
		plan, err := clockdeploy.Prepare(ctx, client, config.Clock.URL, scripts)
		if err != nil {
			return err
		}
		if !*jsonOutput {
			fmt.Print(plan.String())
		}
		if *dryRun {
			if *jsonOutput {
				return printJSON(plan)
			}
			return nil
		}
		if *backup == "" {
			*backup = filepath.Join("dist", "deploy", time.Now().UTC().Format("20060102T150405.000000000")+".json")
		}
		if err := os.MkdirAll(filepath.Dir(*backup), 0o700); err != nil {
			return err
		}
		if err := plan.Save(*backup); err != nil {
			return err
		}
		if !*jsonOutput {
			fmt.Printf("Recovery record: %s\n", *backup)
		}
		report, err := plan.Apply(ctx, client, *observe)
		if *jsonOutput {
			if outputErr := printJSON(map[string]any{"plan": plan, "report": report, "backup": *backup}); outputErr != nil {
				return outputErr
			}
		} else {
			fmt.Print(report.String())
		}
		return err
	}
	for _, script := range scripts {
		if command == "list" {
			fmt.Printf("%s/%s: %s\n", script.Kind, script.Name, script.Path)
			continue
		}
		plan, err := clockdeploy.Prepare(ctx, client, config.Clock.URL, []clockapps.Script{script})
		if err != nil {
			return err
		}
		report, err := plan.Apply(ctx, client, 0)
		if err != nil {
			fmt.Print(report.String())
			return err
		}
		if command == "check" {
			if err := checkScript(ctx, client, script, *duration); err != nil {
				return fmt.Errorf("check %s: %w", script.Name, err)
			}
		} else {
			fmt.Printf("Installed %s/%s\n", script.Kind, script.Name)
		}
	}
	return nil
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
	frames, err := clockcheck.ObserveScript(ctx, client, script.Name, duration)
	if err != nil {
		return err
	}
	if err := clockcheck.CheckFrames(ctx, client, script.Name, script.Path, os.Getenv("TC002_CHECK_ARTIFACTS")); err != nil {
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
