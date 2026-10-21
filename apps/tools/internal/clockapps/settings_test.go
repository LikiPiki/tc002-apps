package clockapps

import (
	"testing"

	"github.com/LikiPiki/tc002-apps/internal/config"
)

func TestCatalogEnforcesConstraints(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "screens", "demo", "# @config caption text maxlen=5\nreturn Demo()")

	c := config.ClockConfig{Screens: config.AppSettings{"demo": {"caption": "too long"}}}

	if _, err := Select(root, c, "demo", "all"); err == nil {
		t.Fatal("catalog bypassed constraints")
	}
}

func TestSettingsConstraints(t *testing.T) {
	source := []byte(`# @config caption text "Label with spaces" maxlen=5
# @config mode select options="slow, fast" default="slow"
# @config speed slider min=1 max=10 step=1 default=5
# @config ink color default="#abcdef"
return Demo()`)

	tests := []struct {
		key   string
		value any
		bad   bool
	}{
		{"caption", "hello", false},
		{"caption", "longer", true},
		{"mode", "fast", false},
		{"mode", "unknown", true},
		{"speed", int64(10), false},
		{"speed", int64(11), true},
		{"speed", int64(0), true},
		{"ink", "#abc123", false},
		{"ink", "#gggggg", true},
		{"ink", int64(-1), true},
		{"ink", int64(0x1000000), true},
	}

	for _, tc := range tests {
		if err := validateSettings(source, map[string]any{tc.key: tc.value}); (err != nil) != tc.bad {
			t.Errorf("%s=%v: %v", tc.key, tc.value, err)
		}
	}
}

func TestScheduleRejectedBeforeDeployment(t *testing.T) {
	tests := [][2]string{
		{"99:00", "20:00"},
		{"10:00", "10:00"},
		{"1:00", "20:00"},
		{"10:00", "20:60"},
	}

	for _, times := range tests {
		script := Script{
			Name: "display-policy",
			Kind: "services",
			Settings: map[string]any{
				"on":  times[0],
				"off": times[1],
			},
		}

		if err := validateDomain(script); err == nil {
			t.Errorf("accepted %v", times)
		}
	}
}
