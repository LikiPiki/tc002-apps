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
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if command == "check" {
		if err := clockcheck.CheckInvocation(*url, os.Getenv("TC002_CHECK_SANDBOX")); err != nil {
			return err
		}
		if *duration < 250*time.Millisecond {
			return fmt.Errorf("check requires --duration >= 250ms")
		}
	}
	config, err := projectconfig.Load(*base, *personal)
	if err != nil {
		return err
	}
	if env := os.Getenv("AWTRIX_URL"); env != "" {
		config.Clock.URL = env
	}
	if *url != "" {
		config.Clock.URL = *url
	}
	client, err := awtrix.NewClient(config.Clock.URL)
	if err != nil {
		return err
	}
	fmt.Printf("Target: %s\n", config.Clock.URL)
	if command == "check" {
		if err := clockcheck.RequireSandbox(context.Background(), client); err != nil {
			return err
		}
	}
	if command == "show" || command == "start" || command == "stop" {
		if *app == "" {
			return fmt.Errorf("%s requires APP=<name>", command)
		}
		if err := client.CheckScripting(context.Background()); err != nil {
			return err
		}
		info, err := client.Script(context.Background(), *app)
		if err != nil {
			return err
		}
		if command == "stop" {
			if err := client.StopOnDemand(context.Background(), *app); err != nil {
				return err
			}
			fmt.Printf("Stopped %s\n", *app)
			return nil
		}
		if (command == "start") != info.OnDemand {
			return fmt.Errorf("use clock-start for on-demand apps and clock-show for rotating screens")
		}
		if err := client.ShowScript(context.Background(), *app); err != nil {
			return err
		}
		time.Sleep(300 * time.Millisecond)
		if err := client.CheckScript(context.Background(), *app); err != nil {
			return err
		}
		fmt.Printf("Showing %s\n", *app)
		return nil
	}
	scripts, err := clockapps.Select("apps/clock", config.Clock, *app, *kind)
	if err != nil {
		return err
	}
	if len(scripts) == 0 {
		fmt.Println("No clock apps selected")
		return nil
	}
	if command != "list" {
		if err := client.CheckScripting(context.Background()); err != nil {
			return err
		}
	}
	if command == "deploy" {
		report, err := clockdeploy.Run(context.Background(), client, scripts)
		fmt.Print(report.String())
		return err
	}
	for _, script := range scripts {
		if command == "list" {
			fmt.Printf("%s/%s: %s\n", script.Kind, script.Name, script.Path)
			continue
		}
		report, err := clockdeploy.Run(context.Background(), client, []clockapps.Script{script})
		if err != nil {
			fmt.Print(report.String())
			return err
		}
		if command == "check" {
			if err := checkScript(client, script, *duration); err != nil {
				return fmt.Errorf("check %s: %w", script.Name, err)
			}
		} else {
			fmt.Printf("Installed %s/%s\n", script.Kind, script.Name)
		}
	}
	return nil
}

func checkScript(client *awtrix.Client, script clockapps.Script, duration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), duration+10*time.Second)
	defer cancel()
	if script.Kind == "services" {
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
	if script.Kind == "on-demand" {
		if err := client.StopOnDemand(ctx, script.Name); err != nil {
			return err
		}
	}
	fmt.Printf("Checked %s/%s: %d frames, no runtime errors\n", script.Kind, script.Name, frames)
	return nil
}
