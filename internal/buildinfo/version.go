package buildinfo

import (
	"fmt"
	"runtime/debug"
)

// Release builds set these values with -ldflags -X.
var Version = "dev"
var Commit = "unknown"

func String() string {
	commit := Commit
	if commit == "unknown" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					commit = setting.Value
				}
			}
		}
	}
	return fmt.Sprintf("clockctl %s (%s)", Version, commit)
}
