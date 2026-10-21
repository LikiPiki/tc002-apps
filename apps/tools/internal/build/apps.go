// Package build selects utility artifacts. The server has one fixed build entry point.
package build

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func Select(root string, names []string) ([]string, error) {
	if names == nil {
		paths, err := filepath.Glob(filepath.Join(root, "cmd", "*", "main.go"))
		if err != nil {
			return nil, err
		}

		for _, path := range paths {
			names = append(names, filepath.Base(filepath.Dir(path)))
		}
	}

	var selected []string
	seen := map[string]bool{}
	for _, name := range names {
		if !namePattern.MatchString(name) {
			return nil, fmt.Errorf("invalid utility name %q", name)
		}

		path := filepath.Join(root, "cmd", name, "main.go")
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("utility %s: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular Go entry point", path)
		}

		if !seen[name] {
			selected = append(selected, name)
			seen[name] = true
		}
	}

	return selected, nil
}
