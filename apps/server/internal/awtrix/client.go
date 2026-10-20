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

type HTTPError struct {
	StatusCode int
	Status     string
}

func (e *HTTPError) Error() string {
	return "AWTRIX returned " + e.Status
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
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	if raw, ok := result.(*[]byte); ok {
		*raw, err = io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
		if err == nil && len(*raw) > 1<<20 {
			return fmt.Errorf("AWTRIX source exceeds 1 MiB")
		}
		return err
	}
	if result != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(result)
	}
	_, err = io.Copy(io.Discard, resp.Body)
	return err
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

func (c *Client) CheckScript(ctx context.Context, name string) error {
	var apps []struct {
		Name    string          `json:"name"`
		Origin  string          `json:"origin"`
		Enabled bool            `json:"enabled"`
		Error   json.RawMessage `json:"error"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v1/apps", "", nil, &apps); err != nil {
		return err
	}
	for _, app := range apps {
		if app.Name == name && app.Origin == "script" {
			if !app.Enabled || string(app.Error) != "null" {
				return fmt.Errorf("script %s: enabled=%v error=%s", name, app.Enabled, app.Error)
			}
			return nil
		}
	}
	return fmt.Errorf("script %s is missing from AWTRIX", name)
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
	if err := c.CheckScript(ctx, name); err != nil {
		return err
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/notifications", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("AWTRIX returned %s", resp.Status)
	}
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}
