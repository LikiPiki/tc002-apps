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
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func Prepare(ctx context.Context, c *awtrix.Client, target string, scripts []clockapps.Script) (Plan, error) {
	p := Plan{Target: target}
	caps, err := c.Capabilities(ctx)
	if err != nil {
		return p, err
	}
	if caps["scriptUpdates"] != true {
		return p, fmt.Errorf("target lacks atomic script updates")
	}
	apps, err := c.Scripts(ctx)
	if err != nil {
		return p, err
	}
	for _, s := range scripts {
		before, err := c.Source(ctx, s.Name)
		if err != nil {
			return p, err
		}
		ch := Change{Name: s.Name, Before: before, After: string(s.Source), SettingsAfter: s.Settings, Enabled: true}
		if before != nil {
			ch.SettingsBefore, err = c.ScriptSettings(ctx, s.Name)
			if err != nil {
				return p, err
			}
			for _, app := range apps {
				if app.Name == s.Name && app.Origin == "script" {
					ch.Enabled = app.Enabled
				}
			}
		}
		ch.SourceChanged = before == nil || *before != ch.After
		for k, v := range s.Settings {
			if !Equal(ch.SettingsBefore[k], v) {
				ch.SettingsChanged = true
			}
		}
		p.Changes = append(p.Changes, ch)
	}
	return p, nil
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
	var p Plan
	data, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	err = json.Unmarshal(data, &p)
	if err == nil && p.Target != target {
		err = fmt.Errorf("backup target %q differs from %q", p.Target, target)
	}
	return p, err
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
func (p Plan) Apply(ctx context.Context, c *awtrix.Client, window time.Duration) (Report, error) {
	r := Report{}
	for _, ch := range p.Changes {
		r.Results = append(r.Results, Result{Name: ch.Name, Source: "not-attempted", Settings: "not-run", Verification: "not-run"})
	}
	for i, ch := range p.Changes {
		result := &r.Results[i]
		result.Source = "unchanged"
		if ch.SourceChanged {
			result.Source = "unconfirmed"
			if err := c.UpdateScript(ctx, ch.Name, ch.Before, []byte(ch.After)); err != nil {
				result.FailedPhase = "source"
				return r, err
			}
			result.Source = "updated"
		}
		result.Settings = "unchanged"
		if !ch.SourceChanged {
			current, err := c.Source(ctx, ch.Name)
			if err != nil {
				return r, err
			}
			if !Equal(current, ch.Before) {
				result.FailedPhase = "source-conflict"
				return r, fmt.Errorf("%s changed since planning", ch.Name)
			}
		}
		// A new schema may reset settings even when old values matched.
		if len(ch.SettingsAfter) > 0 && (ch.SettingsChanged || ch.SourceChanged) {
			result.Settings = "unconfirmed"
			if err := c.ConfigureScript(ctx, ch.Name, ch.SettingsAfter); err != nil {
				result.FailedPhase = "settings"
				var restart *awtrix.SettingsRestartError
				if errors.As(err, &restart) {
					result.Settings = "applied"
					result.FailedPhase = "restart"
					result.Verification = "failed"
				}
				return r, err
			}
			result.Settings = "applied"
		}
		if !ch.Enabled {
			result.Verification = "disabled"
			continue
		}
		if err := verify(ctx, c, ch.Name, window); err != nil {
			result.Verification = "failed"
			result.FailedPhase = "verification"
			return r, err
		}
		result.Verification = "passed"
	}
	return r, nil
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
		if Equal(current, ch.Before) {
			// Settings may still need restoration after a settings-only deployment.
		} else if current == nil || *current != ch.After {
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
