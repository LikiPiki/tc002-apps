// Package clockapps maps repository layouts to AWTRIX script lifecycle modes.
package clockapps

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

type Script struct {
	Name     string
	Kind     string
	Path     string
	Source   []byte
	Settings map[string]any
}

func Select(root string, c config.ClockConfig, app, kind string) ([]Script, error) {
	if kind == "" {
		kind = "all"
	}
	if kind != "all" && kind != "screens" && kind != "services" && kind != "on-demand" {
		return nil, fmt.Errorf("unknown kind %q: use all, screens, services, or on-demand", kind)
	}
	groups := []struct {
		kind     string
		names    []string
		settings config.AppSettings
	}{
		{"services", c.Deploy.Services, c.Services},
		{"screens", c.Deploy.Screens, c.Screens},
		{"on-demand", c.Deploy.OnDemand, c.OnDemand},
	}
	catalog := map[string]Script{}
	byKind := map[string][]string{}
	for _, group := range groups {
		paths, err := filepath.Glob(filepath.Join(root, group.kind, "*", "main.be"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			name := filepath.Base(filepath.Dir(path))
			if err := awtrix.ValidateAppName(name); err != nil {
				return nil, err
			}
			if previous, ok := catalog[name]; ok {
				return nil, fmt.Errorf("duplicate AWTRIX install name %s: %s and %s", name, previous.Path, path)
			}
			catalog[name] = Script{Name: name, Kind: group.kind, Path: path, Settings: group.settings[name]}
			byKind[group.kind] = append(byKind[group.kind], name)
		}
		for name := range group.settings {
			if entry, ok := catalog[name]; !ok || entry.Kind != group.kind {
				return nil, fmt.Errorf("settings reference unknown %s app %s", group.kind, name)
			}
		}
	}
	if app != "" {
		entry, ok := catalog[app]
		if !ok || (kind != "all" && entry.Kind != kind) {
			return nil, fmt.Errorf("unknown %s app %q", kind, app)
		}
		return prepare([]Script{entry})
	}
	var selected []Script
	for _, group := range groups {
		if kind != "all" && kind != group.kind {
			continue
		}
		names := group.names
		if names == nil {
			names = byKind[group.kind]
		}
		seen := map[string]bool{}
		for _, name := range names {
			entry, ok := catalog[name]
			if !ok || entry.Kind != group.kind {
				return nil, fmt.Errorf("unknown %s app %q", group.kind, name)
			}
			if !seen[name] {
				selected = append(selected, entry)
				seen[name] = true
			}
		}
	}
	return prepare(selected)
}

func prepare(scripts []Script) ([]Script, error) {
	for i := range scripts {
		entry := &scripts[i]
		source, err := os.ReadFile(entry.Path)
		if err != nil {
			return nil, err
		}
		if len(bytes.TrimSpace(source)) == 0 {
			return nil, fmt.Errorf("%s: empty script", entry.Path)
		}
		if err := validateHeader(entry.Kind, source, entry.Settings); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Path, err)
		}
		if err := validateDomain(*entry); err != nil {
			return nil, err
		}
		// Device config responses expose RGB values as numbers, regardless of input encoding.
		for _, line := range strings.Split(string(source), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if !strings.HasPrefix(line, "#") {
				break
			}
			fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, "#")))
			if len(fields) >= 3 && fields[0] == "@config" && fields[2] == "color" {
				if value, ok := entry.Settings[fields[1]]; ok {
					rgb, _ := Color(value)
					entry.Settings[fields[1]] = int64(rgb)
				}
			}
		}
		entry.Source = source
	}
	return scripts, nil
}

func validateHeader(kind string, source []byte, settings map[string]any) error {
	headless, onDemand := false, false
	for _, line := range strings.Split(string(source), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			break
		}
		fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, "#")))
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "@module":
			return fmt.Errorf("Berry modules are not supported by this app catalog yet")
		case "@headless":
			if len(fields) < 2 {
				return fmt.Errorf("@headless requires an explicit boolean value")
			}
			switch strings.ToLower(fields[1]) {
			case "true", "1", "yes":
				headless = true
			case "false", "0", "no":
				headless = false
			default:
				return fmt.Errorf("invalid @headless marker")
			}
		case "@ondemand":
			if len(fields) > 1 && fields[1] != "#" {
				return fmt.Errorf("@ondemand is a flag; omit its value")
			}
			onDemand = true
		}
	}
	if (headless && onDemand) || (kind == "services" && !headless) || (kind == "on-demand" && !onDemand) || (kind == "screens" && (headless || onDemand)) || (kind == "services" && onDemand) || (kind == "on-demand" && headless) {
		return fmt.Errorf("Berry lifecycle markers do not match directory kind %s", kind)
	}
	return validateSettings(source, settings)
}
