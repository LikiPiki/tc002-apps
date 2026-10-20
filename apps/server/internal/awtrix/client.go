package awtrix

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

var appName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

func ValidateAppName(name string) error {
	if !appName.MatchString(name) {
		return fmt.Errorf("invalid app name %q: use 1-32 letters, digits, hyphens or underscores", name)
	}
	switch name {
	case "active", "next", "previous", "order":
		return fmt.Errorf("reserved app name %q", name)
	}
	return nil
}

func (c *Client) request(ctx context.Context, method, path, contentType string, body []byte, result any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s %s: create request: %w", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return responseError(method, path, resp.StatusCode, resp.Status, detail)
	}
	if raw, ok := result.(*[]byte); ok {
		*raw, err = io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
		if err == nil && len(*raw) > 1<<20 {
			return fmt.Errorf("%s %s: AWTRIX source exceeds 1 MiB", method, path)
		}
		if err != nil {
			return fmt.Errorf("%s %s: read source: %w", method, path, err)
		}
		return nil
	}
	if result != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(result); err != nil {
			return fmt.Errorf("%s %s: decode response: %w", method, path, err)
		}
		return nil
	}
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("%s %s: read response: %w", method, path, err)
	}
	return nil
}

func (c *Client) InstallScript(ctx context.Context, name string, source []byte) error {
	if err := ValidateAppName(name); err != nil {
		return err
	}
	var capabilities struct {
		ScriptUpdates bool `json:"scriptUpdates"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v1/capabilities", "", nil, &capabilities); err != nil {
		return err
	}
	if !capabilities.ScriptUpdates {
		return fmt.Errorf("target does not support atomic script updates; update AWTRIX NG first")
	}
	var oldSource []byte
	var expected *string
	err := c.request(ctx, http.MethodGet, "/api/v1/apps/script/"+name, "", nil, &oldSource)
	if err == nil {
		text := string(oldSource)
		expected = &text
	} else {
		var status *HTTPError
		if !errors.As(err, &status) || status.StatusCode != http.StatusNotFound {
			return err
		}
	}
	payload, err := json.Marshal(struct {
		Expected *string `json:"expected_source"`
		Source   string  `json:"source"`
	}{expected, string(source)})
	if err != nil {
		return err
	}
	var result struct {
		OK    bool            `json:"ok"`
		Error json.RawMessage `json:"error"`
	}
	if err := c.request(ctx, http.MethodPut, "/api/v1/apps/script-update/"+name, "application/json", payload, &result); err != nil {
		return err
	}
	if !result.OK || (len(result.Error) != 0 && string(result.Error) != "null") {
		return fmt.Errorf("script installation failed: ok=%v error=%s", result.OK, result.Error)
	}
	return nil
}

type ScriptInfo struct {
	Name     string          `json:"name"`
	Origin   string          `json:"origin"`
	Enabled  bool            `json:"enabled"`
	Headless bool            `json:"headless"`
	OnDemand bool            `json:"ondemand"`
	Error    json.RawMessage `json:"error"`
}

func (app ScriptInfo) check() error {
	if !app.Enabled || string(app.Error) != "null" {
		return fmt.Errorf("script %s: enabled=%v error=%s", app.Name, app.Enabled, app.Error)
	}
	return nil
}

func (c *Client) StopOnDemand(ctx context.Context, name string) error {
	info, err := c.Script(ctx, name)
	if err != nil {
		return err
	}
	if !info.OnDemand {
		return fmt.Errorf("script %s is not an on-demand app", name)
	}
	var device struct {
		CurrentApp string `json:"currentApp"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v1/device", "", nil, &device); err != nil {
		return err
	}
	if device.CurrentApp != name {
		return fmt.Errorf("on-demand app %s is not running", name)
	}
	return c.confirmedCommand(ctx, http.MethodPost, "/api/v1/apps/next", nil)
}

func (c *Client) Script(ctx context.Context, name string) (ScriptInfo, error) {
	var apps []ScriptInfo
	if err := c.request(ctx, http.MethodGet, "/api/v1/apps", "", nil, &apps); err != nil {
		return ScriptInfo{}, err
	}
	for _, app := range apps {
		if app.Name == name && app.Origin == "script" {
			return app, nil
		}
	}
	return ScriptInfo{}, fmt.Errorf("script %s is missing from AWTRIX", name)
}

func (c *Client) CheckScript(ctx context.Context, name string) error {
	info, err := c.Script(ctx, name)
	if err != nil {
		return err
	}
	return info.check()
}

func (c *Client) ConfigureScript(ctx context.Context, name string, settings map[string]any) error {
	if err := ValidateAppName(name); err != nil {
		return err
	}
	if len(settings) == 0 {
		return fmt.Errorf("script settings must not be empty")
	}
	body, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	var result struct {
		OK    bool            `json:"ok"`
		Error json.RawMessage `json:"error"`
	}
	if err := c.request(ctx, http.MethodPatch, "/api/v1/apps/"+name+"/config", "application/json", body, &result); err != nil {
		return err
	}
	if !result.OK || string(result.Error) != "null" {
		return fmt.Errorf("script %s settings failed: ok=%v error=%s", name, result.OK, result.Error)
	}
	return nil
}

func (c *Client) confirmedCommand(ctx context.Context, method, path string, payload []byte) error {
	var result struct {
		OK bool `json:"ok"`
	}
	if err := c.request(ctx, method, path, "application/json", payload, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("%s %s: command was not confirmed", method, path)
	}
	return nil
}

func (c *Client) SetDisplayPower(ctx context.Context, on bool) error {
	body, err := json.Marshal(struct {
		Power bool `json:"power"`
	}{on})
	if err != nil {
		return err
	}
	return c.confirmedCommand(ctx, http.MethodPatch, "/api/v1/display", body)
}

func (c *Client) SetScriptEnabled(ctx context.Context, name string, enabled bool) error {
	if err := ValidateAppName(name); err != nil {
		return err
	}
	body, err := json.Marshal(enabled)
	if err != nil {
		return err
	}
	return c.confirmedCommand(ctx, http.MethodPut, "/api/v1/apps/"+name+"/enabled", body)
}

func (c *Client) CheckScripting(ctx context.Context) error {
	var device struct {
		Running bool `json:"scriptingRunning"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v1/device", "", nil, &device); err != nil {
		return err
	}
	if !device.Running {
		return fmt.Errorf("Berry scripting is not running on the target")
	}
	return nil
}

func (c *Client) ShowScript(ctx context.Context, name string) error {
	if err := ValidateAppName(name); err != nil {
		return err
	}
	info, err := c.Script(ctx, name)
	if err != nil {
		return err
	}
	if err := info.check(); err != nil {
		return err
	}
	if info.Headless {
		return fmt.Errorf("script %s is headless and has no display page", name)
	}
	body, err := json.Marshal(struct {
		Name string `json:"name"`
		Fast bool   `json:"fast"`
	}{name, true})
	if err != nil {
		return err
	}
	var result struct {
		OK bool `json:"ok"`
	}
	if err := c.request(ctx, http.MethodPut, "/api/v1/apps/active", "application/json", body, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("AWTRIX did not confirm showing %s", name)
	}
	return nil
}

func NewClient(baseURL string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("AWTRIX_URL must be an absolute http(s) URL")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("AWTRIX_URL must not contain a query or fragment")
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (c *Client) Notify(ctx context.Context, text string) error {
	payload, err := json.Marshal(struct {
		Text string `json:"text"`
	}{Text: text})
	if err != nil {
		return err
	}
	return c.request(ctx, http.MethodPost, "/api/v1/notifications", "application/json", payload, nil)
}
