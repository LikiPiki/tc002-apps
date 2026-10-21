package clockdeploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockapps"
	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

type Change struct {
	Name            string         `json:"name"`
	Before          *string        `json:"before"`
	After           string         `json:"after"`
	SettingsBefore  map[string]any `json:"settings_before"`
	SettingsAfter   map[string]any `json:"settings_after"`
	Enabled         bool           `json:"enabled"`
	SourceChanged   bool           `json:"source_changed"`
	SettingsChanged bool           `json:"settings_changed"`
}

type Plan struct {
	Target  string   `json:"target"`
	Changes []Change `json:"changes"`
}

func Equal(a, b any) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}

func Prepare(ctx context.Context, c *awtrix.Client, target string, scripts []clockapps.Script) (Plan, error) {
	plan := Plan{Target: target}
	capabilities, err := c.Capabilities(ctx)
	if err != nil {
		return plan, err
	}
	if capabilities["scriptUpdates"] != true {
		return plan, fmt.Errorf("target lacks atomic script updates")
	}

	apps, err := c.Scripts(ctx)
	if err != nil {
		return plan, err
	}

	for _, script := range scripts {
		change, err := prepareChange(ctx, c, script, apps)
		if err != nil {
			return plan, err
		}
		plan.Changes = append(plan.Changes, change)
	}
	return plan, nil
}

func prepareChange(ctx context.Context, client *awtrix.Client, script clockapps.Script, apps []awtrix.ScriptInfo) (Change, error) {
	before, err := client.Source(ctx, script.Name)
	if err != nil {
		return Change{}, err
	}

	change := Change{
		Name:          script.Name,
		Before:        before,
		After:         string(script.Source),
		SettingsAfter: script.Settings,
		Enabled:       true,
	}
	if before != nil {
		change.SettingsBefore, err = client.ScriptSettings(ctx, script.Name)
		if err != nil {
			return change, err
		}
		for _, app := range apps {
			if app.Name == script.Name && app.Origin == "script" {
				change.Enabled = app.Enabled
			}
		}
	}

	change.SourceChanged = before == nil || *before != change.After
	for key, value := range script.Settings {
		if !Equal(change.SettingsBefore[key], value) {
			change.SettingsChanged = true
		}
	}
	return change, nil
}

func (p Plan) String() string {
	out := "Deployment plan for " + p.Target + "\n"
	for _, ch := range p.Changes {
		out += fmt.Sprintf("  %s: source-change=%v settings-change=%v\n", ch.Name, ch.SourceChanged, ch.SettingsChanged)
	}
	return out
}

// Save writes a private recovery record before any device writes.
func (p Plan) Save(path string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func LoadPlan(path, target string) (Plan, error) {
	var plan Plan
	data, err := os.ReadFile(path)
	if err != nil {
		return plan, err
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		return plan, err
	}
	if plan.Target != target {
		return plan, fmt.Errorf("backup target %q differs from %q", plan.Target, target)
	}
	return plan, nil
}

func verify(ctx context.Context, c *awtrix.Client, name string, window time.Duration) error {
	timer := time.NewTimer(window)
	defer timer.Stop()
	for {
		if err := c.CheckScript(ctx, name); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return c.CheckScript(ctx, name)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// Apply stops on first failure. The returned report preserves confirmed phases.
func (p Plan) Apply(ctx context.Context, client *awtrix.Client, window time.Duration) (Report, error) {
	report := Report{}
	for _, change := range p.Changes {
		report.Results = append(report.Results, Result{
			Name:         change.Name,
			Source:       "not-attempted",
			Settings:     "not-run",
			Verification: "not-run",
		})
	}

	for i, change := range p.Changes {
		if err := change.apply(ctx, client, window, &report.Results[i]); err != nil {
			return report, err
		}
	}
	return report, nil
}

func (change Change) apply(ctx context.Context, client *awtrix.Client, window time.Duration, result *Result) error {
	if err := change.applySource(ctx, client, result); err != nil {
		return err
	}
	if err := change.applySettings(ctx, client, result); err != nil {
		return err
	}

	if !change.Enabled {
		result.Verification = "disabled"
		return nil
	}
	if err := verify(ctx, client, change.Name, window); err != nil {
		result.Verification = "failed"
		result.FailedPhase = "verification"
		return err
	}
	result.Verification = "passed"
	return nil
}

func (change Change) applySource(ctx context.Context, client *awtrix.Client, result *Result) error {
	if change.SourceChanged {
		result.Source = "unconfirmed"
		if err := client.UpdateScript(ctx, change.Name, change.Before, []byte(change.After)); err != nil {
			result.FailedPhase = "source"
			return err
		}
		result.Source = "updated"
		return nil
	}

	result.Source = "unchanged"
	current, err := client.Source(ctx, change.Name)
	if err != nil {
		return err
	}
	if !Equal(current, change.Before) {
		result.FailedPhase = "source-conflict"
		return fmt.Errorf("%s changed since planning", change.Name)
	}
	return nil
}

func (change Change) applySettings(ctx context.Context, client *awtrix.Client, result *Result) error {
	result.Settings = "unchanged"

	// A new schema may reset settings even when old values matched.
	needsUpdate := change.SettingsChanged || change.SourceChanged
	if len(change.SettingsAfter) == 0 || !needsUpdate {
		return nil
	}

	result.Settings = "unconfirmed"
	if err := client.ConfigureScript(ctx, change.Name, change.SettingsAfter); err != nil {
		result.FailedPhase = "settings"
		var restart *awtrix.SettingsRestartError
		if errors.As(err, &restart) {
			result.Settings = "applied"
			result.FailedPhase = "restart"
			result.Verification = "failed"
		}
		return err
	}
	result.Settings = "applied"
	return nil
}

// Rollback restores existing scripts and their declared settings. Newly installed
// apps are disabled instead of deleted because AWTRIX has no atomic delete API.
func (p Plan) Rollback(ctx context.Context, c *awtrix.Client) error {
	for i := len(p.Changes) - 1; i >= 0; i-- {
		ch := p.Changes[i]
		if !ch.SourceChanged && !ch.SettingsChanged {
			continue
		}
		current, err := c.Source(ctx, ch.Name)
		if err != nil {
			return err
		}
		sourceUnchanged := Equal(current, ch.Before)
		sourceDeployed := current != nil && *current == ch.After
		if !sourceUnchanged && !sourceDeployed {
			return fmt.Errorf("%s changed since deployment; rollback stopped", ch.Name)
		}
		if ch.Before == nil {
			if current != nil {
				if err := c.SetScriptEnabled(ctx, ch.Name, false); err != nil {
					return err
				}
			}
			continue
		}
		if !Equal(current, ch.Before) {
			if err := c.UpdateScript(ctx, ch.Name, current, []byte(*ch.Before)); err != nil {
				return err
			}
		}
		if len(ch.SettingsBefore) > 0 {
			if err := c.ConfigureScript(ctx, ch.Name, ch.SettingsBefore); err != nil {
				return err
			}
		}
		if err := c.SetScriptEnabled(ctx, ch.Name, ch.Enabled); err != nil {
			return err
		}
	}
	return nil
}
