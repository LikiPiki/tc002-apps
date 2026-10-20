package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LikiPiki/tc002-apps/apps/server/internal/awtrix"
	"github.com/pelletier/go-toml/v2"
)

type clockConfig struct {
	URL  string
	Apps []string
}

func loadClockConfig(base, personal string) (clockConfig, error) {
	var config clockConfig
	for index, path := range []string{base, personal} {
		if path == "" && index == 1 {
			continue
		}
		data, err := os.ReadFile(path)
		if index == 1 && errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return config, err
		}
		var fields struct {
			URL  *string   `toml:"url"`
			Apps *[]string `toml:"apps"`
		}
		decoder := toml.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&fields); err != nil {
			return config, fmt.Errorf("%s: %w", path, err)
		}
		if fields.URL != nil {
			config.URL = *fields.URL
		}
		if fields.Apps != nil {
			config.Apps = *fields.Apps
			if len(config.Apps) == 0 {
				return config, fmt.Errorf("%s: apps must be [\"*\"] (all apps) or a nonempty list", path)
			}
			if len(config.Apps) == 1 && config.Apps[0] == "*" {
				config.Apps = nil
			}
		}
	}
	return config, nil
}

type script struct {
	name   string
	path   string
	source []byte
}

func selectScripts(root string, names []string) ([]script, error) {
	if names == nil {
		paths, err := filepath.Glob(filepath.Join(root, "*", "main.be"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			names = append(names, filepath.Base(filepath.Dir(path)))
		}
	}
	var scripts []script
	seen := make(map[string]bool)
	for _, name := range names {
		if err := awtrix.ValidateAppName(name); err != nil {
			return nil, err
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		path := filepath.Join(root, name, "main.be")
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if len(bytes.TrimSpace(source)) == 0 {
			return nil, fmt.Errorf("%s: empty script", path)
		}
		scripts = append(scripts, script{name, path, source})
	}
	if len(scripts) == 0 {
		return nil, fmt.Errorf("no Berry scripts selected in %s", root)
	}
	return scripts, nil
}

func runScripts(command string, args []string) error {
	flags := flag.NewFlagSet("clockctl "+command, flag.ContinueOnError)
	base := flags.String("config", "configs/clock.toml", "shared TOML config")
	personal := flags.String("local-config", "configs/clock.local.toml", "optional personal TOML overrides; empty disables them")
	app := flags.String("app", "", "one clock app; deploy/list otherwise use config selection")
	url := flags.String("url", "", "override target URL (otherwise AWTRIX_URL, then config)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	config, err := loadClockConfig(*base, *personal)
	if err != nil {
		return err
	}
	if env := os.Getenv("AWTRIX_URL"); env != "" {
		config.URL = env
	}
	if *url != "" {
		config.URL = *url
	}
	client, err := awtrix.NewClient(config.URL)
	if err != nil {
		return err
	}
	fmt.Printf("Target: %s\n", config.URL)
	if command == "show" {
		if *app == "" {
			return fmt.Errorf("show requires --app (make clock-show APP=<name>)")
		}
		if err := client.CheckScripting(context.Background()); err != nil {
			return err
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
	if *app != "" {
		config.Apps = []string{*app}
	}
	// Read every selected source before making any writes to the device.
	scripts, err := selectScripts("apps/clock", config.Apps)
	if err != nil {
		return err
	}
	if command == "deploy" {
		if err := client.CheckScripting(context.Background()); err != nil {
			return err
		}
	}
	for _, script := range scripts {
		if command == "list" {
			fmt.Printf("%s: %s\n", script.name, script.path)
			continue
		}
		if err := client.InstallScript(context.Background(), script.name, script.source); err != nil {
			return fmt.Errorf("%s: %w", script.name, err)
		}
		if err := client.CheckScript(context.Background(), script.name); err != nil {
			return err
		}
		fmt.Printf("Installed %s\n", script.name)
	}
	return nil
}
