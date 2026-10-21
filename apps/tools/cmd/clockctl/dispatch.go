package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockapps"
	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockcheck"
	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockdeploy"
	"github.com/LikiPiki/tc002-apps/internal/awtrix"
	"github.com/LikiPiki/tc002-apps/internal/clocks"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

type scriptOptions struct {
	App      string
	Kind     string
	AppsDir  string
	Backup   string
	Duration time.Duration
	Timeout  time.Duration
	Observe  time.Duration
	DryRun   bool
	JSON     bool
	All      bool
}

func dispatchScripts(ctx context.Context, command string, clock config.ClockConfig, options scriptOptions) error {
	profiles, err := clock.ResolveProfiles()
	if err != nil {
		return err
	}

	targetCount := 0
	for _, profile := range profiles {
		targetCount += len(profile.Clock.Targets())
	}

	switch command {
	case "show", "start", "stop":
		if options.App == "" {
			return fmt.Errorf("%s requires APP=<name>", command)
		}

		report, err := clocks.RunProfiles(ctx, profiles, options.Timeout, func(ctx context.Context, target clocks.Target, profile config.ClockProfile) (any, error) {
			return controlScript(ctx, target.Client, command, options.App)
		})
		if err != nil {
			return err
		}

		return finishDelivery(ctx, report, command, options.JSON)

	case "rollback":
		if options.Backup == "" || targetCount != 1 {
			return fmt.Errorf("rollback requires --backup and one clock target; select it with --url")
		}

		report, err := clocks.RunProfiles(ctx, profiles, options.Timeout, func(ctx context.Context, target clocks.Target, profile config.ClockProfile) (any, error) {
			plan, err := clockdeploy.LoadPlan(options.Backup, target.URL)
			if err != nil {
				return nil, err
			}

			return "Restoring " + options.Backup, plan.Rollback(ctx, target.Client)
		})
		if err != nil {
			return err
		}

		return finishDelivery(ctx, report, command, options.JSON)
	}

	catalogs := map[string][]clockapps.Script{}
	appCount := 0
	for _, profile := range profiles {
		selection := profile.Clock
		if options.All {
			selection.Deploy = config.ClockDeploy{}
		}

		scripts, err := clockapps.Select(options.AppsDir, selection, options.App, options.Kind)
		if err != nil {
			return fmt.Errorf("profile %s: %w", profile.Name, err)
		}

		catalogs[profile.Name] = scripts
		appCount += len(scripts)
	}

	if command == "list" || command == "validate" {
		return printCatalog(command, profiles, catalogs, options.JSON)
	}
	if appCount == 0 {
		if options.JSON {
			return printJSON(clocks.Report{Results: []clocks.Result{}})
		}

		fmt.Println("No clock apps selected")
		return nil
	}
	if command == "check" {
		if len(profiles) != 1 {
			return fmt.Errorf("sandbox checks require a single sandbox target")
		}

		return checkSandbox(ctx, catalogs[profiles[0].Name], options.Duration)
	}

	var audience []config.ClockProfile
	for _, profile := range profiles {
		if len(catalogs[profile.Name]) > 0 {
			audience = append(audience, profile)
		}
	}

	report, err := clocks.RunProfiles(ctx, audience, options.Timeout, func(ctx context.Context, target clocks.Target, profile config.ClockProfile) (any, error) {
		scripts := catalogs[profile.Name]
		return deployTarget(ctx, target, scripts, options, targetCount > 1)
	})
	if err != nil {
		return err
	}

	return finishDelivery(ctx, report, command, options.JSON)
}

type deploymentDetail struct {
	Plan   clockdeploy.Plan    `json:"plan"`
	Report *clockdeploy.Report `json:"report,omitempty"`
	Backup string              `json:"backup,omitempty"`
}

func (detail deploymentDetail) String() string {
	text := detail.Plan.String()
	if detail.Backup != "" {
		text += "Recovery record: " + detail.Backup + "\n"
	}
	if detail.Report != nil {
		text += detail.Report.String()
	}

	return text
}

func deployTarget(ctx context.Context, target clocks.Target, scripts []clockapps.Script, options scriptOptions, multiple bool) (any, error) {
	if err := target.Client.CheckScripting(ctx); err != nil {
		return nil, err
	}

	plan, err := clockdeploy.Prepare(ctx, target.Client, target.URL, scripts)
	if err != nil {
		return nil, err
	}

	detail := deploymentDetail{Plan: plan}
	if options.DryRun {
		return detail, nil
	}

	path := options.Backup
	if path == "" {
		name := time.Now().UTC().Format("20060102T150405.000000000") + ".json"
		path = filepath.Join("dist", "deploy", name)
	}
	if multiple {
		extension := filepath.Ext(path)
		path = strings.TrimSuffix(path, extension) + fmt.Sprintf(".clock-%d", target.Index+1) + extension
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return detail, err
	}
	if err := plan.Save(path); err != nil {
		return detail, err
	}
	detail.Backup = path

	report, err := plan.Apply(ctx, target.Client, options.Observe)
	detail.Report = &report

	return detail, err
}

func controlScript(ctx context.Context, client *awtrix.Client, command, name string) (any, error) {
	if err := client.CheckScripting(ctx); err != nil {
		return nil, err
	}

	info, err := client.Script(ctx, name)
	if err != nil {
		return nil, err
	}
	if command == "stop" {
		return "Stopped " + name, client.StopOnDemand(ctx, name)
	}
	if (command == "start") != info.OnDemand {
		return nil, fmt.Errorf("use clock-start for on-demand apps and clock-show for rotating screens")
	}

	if err := client.ShowScript(ctx, name); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(300 * time.Millisecond):
	}

	return "Showing " + name, client.CheckScript(ctx, name)
}

func checkSandbox(ctx context.Context, scripts []clockapps.Script, duration time.Duration) error {
	client, err := awtrix.NewClient(clockcheck.SandboxURL)
	if err != nil {
		return err
	}
	if err := clockcheck.RequireSandbox(ctx, client); err != nil {
		return err
	}
	if err := client.CheckScripting(ctx); err != nil {
		return err
	}

	fmt.Printf("Target: %s\n", clockcheck.SandboxURL)
	for _, script := range scripts {
		plan, err := clockdeploy.Prepare(ctx, client, clockcheck.SandboxURL, []clockapps.Script{script})
		if err != nil {
			return err
		}

		report, err := plan.Apply(ctx, client, 0)
		if err != nil {
			fmt.Print(report.String())
			return err
		}
		if err := checkScript(ctx, client, script, duration); err != nil {
			return fmt.Errorf("check %s: %w", script.Name, err)
		}
	}

	return nil
}

func printCatalog(command string, profiles []config.ClockProfile, catalogs map[string][]clockapps.Script, jsonOutput bool) error {
	var output []map[string]any

	for _, profile := range profiles {
		scripts := catalogs[profile.Name]
		catalog := make([]map[string]string, 0, len(scripts))
		for _, script := range scripts {
			catalog = append(catalog, map[string]string{
				"name": script.Name,
				"kind": script.Kind,
				"path": script.Path,
			})
		}

		output = append(output, map[string]any{
			"profile":   profile.Name,
			"targets":   profile.Clock.Targets(),
			"apps":      catalog,
			"validated": len(scripts),
		})
		if jsonOutput {
			continue
		}

		fmt.Printf("Profile: %s\nTargets: %s\n", profile.Name, strings.Join(profile.Clock.Targets(), ", "))
		if command == "validate" {
			fmt.Printf("Validated %d clock apps\n", len(scripts))
			continue
		}

		for _, script := range scripts {
			fmt.Printf("%s/%s: %s\n", script.Kind, script.Name, script.Path)
		}
	}

	if jsonOutput {
		return printJSON(map[string]any{"profiles": output})
	}

	return nil
}
