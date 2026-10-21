// Package clockdeploy reports confirmed phases of a sequential, non-transactional batch.
package clockdeploy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockapps"
	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

type API interface {
	InstallScript(context.Context, string, []byte) error
	ConfigureScript(context.Context, string, map[string]any) error
	CheckScript(context.Context, string) error
}

type Result struct {
	Name         string
	Kind         string
	Source       string
	Settings     string
	Verification string
	FailedPhase  string
}

type Report struct{ Results []Result }

func Run(ctx context.Context, api API, scripts []clockapps.Script) (Report, error) {
	report := Report{Results: make([]Result, len(scripts))}
	for i, script := range scripts {
		report.Results[i] = Result{script.Name, script.Kind, "not-attempted", "not-run", "not-run", ""}
	}
	for i, script := range scripts {
		result := &report.Results[i]
		result.Source = "unconfirmed"
		if err := api.InstallScript(ctx, script.Name, script.Source); err != nil {
			result.FailedPhase = "source"
			return report, fmt.Errorf("%s/%s source update: %w", script.Kind, script.Name, err)
		}
		result.Source = "updated"
		result.Settings = "not-configured"
		if len(script.Settings) != 0 {
			result.Settings = "unconfirmed"
			if err := api.ConfigureScript(ctx, script.Name, script.Settings); err != nil {
				result.FailedPhase = "settings"
				var restart *awtrix.SettingsRestartError
				if errors.As(err, &restart) {
					result.Settings = "applied"
					result.Verification = "failed"
					result.FailedPhase = "restart"
				}
				return report, fmt.Errorf("%s/%s settings (source already updated): %w", script.Kind, script.Name, err)
			}
			result.Settings = "applied"
		}
		result.Verification = "unconfirmed"
		if err := api.CheckScript(ctx, script.Name); err != nil {
			result.Verification = "failed"
			result.FailedPhase = "verification"
			return report, fmt.Errorf("%s/%s verification (source already updated): %w", script.Kind, script.Name, err)
		}
		result.Verification = "passed"
	}
	return report, nil
}

func (r Report) String() string {
	var out strings.Builder
	verified := 0
	uncertain := false
	for _, result := range r.Results {
		if result.Verification == "passed" {
			verified++
		}
		uncertain = uncertain || result.Source == "unconfirmed" || result.Settings == "unconfirmed"
	}
	fmt.Fprintf(&out, "Deployment: %d/%d apps verified\n", verified, len(r.Results))
	for _, result := range r.Results {
		fmt.Fprintf(&out, "  %s/%s: source=%s settings=%s verification=%s", result.Kind, result.Name, result.Source, result.Settings, result.Verification)
		if result.FailedPhase != "" {
			fmt.Fprintf(&out, " FAILED at %s", result.FailedPhase)
		}
		out.WriteByte('\n')
	}
	if verified != len(r.Results) {
		out.WriteString("Confirmed updates remain installed; no batch rollback was performed.\n")
		if uncertain {
			out.WriteString("Unconfirmed writes may have reached the device; inspect its state before retrying.\n")
		}
	}
	return out.String()
}
