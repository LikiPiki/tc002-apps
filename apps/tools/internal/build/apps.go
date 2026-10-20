// Package build selects repository build artifacts. Runtime config never scans source trees.
package build

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type Target struct {
	Group string
	Name  string
}

func (t Target) String() string { return t.Group + "/" + t.Name }

var namePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func Select(root string, server, tools []string, group string) ([]Target, error) {
	if group != "all" && group != "server" && group != "tools" {
		return nil, fmt.Errorf("unknown Go group %q: use all, server, or tools", group)
	}
	var targets []Target
	for _, entry := range []struct {
		group string
		names []string
	}{{"server", server}, {"tools", tools}} {
		if group != "all" && group != entry.group {
			continue
		}
		names := entry.names
		if names == nil {
			paths, err := filepath.Glob(filepath.Join(root, entry.group, "cmd", "*", "main.go"))
			if err != nil {
				return nil, err
			}
			for _, path := range paths {
				names = append(names, filepath.Base(filepath.Dir(path)))
			}
		}
		seen := map[string]bool{}
		for _, name := range names {
			if !namePattern.MatchString(name) {
				return nil, fmt.Errorf("invalid Go app name %q", name)
			}
			path := filepath.Join(root, entry.group, "cmd", name, "main.go")
			info, err := os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("%s/%s: %w", entry.group, name, err)
			}
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("%s is not a regular Go entry point", path)
			}
			if !seen[name] {
				targets = append(targets, Target{entry.group, name})
				seen[name] = true
			}
		}
	}
	return targets, nil
}
