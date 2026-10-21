package awtrix

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

func (c *Client) Scripts(ctx context.Context) ([]ScriptInfo, error) {
	var result []ScriptInfo
	err := c.request(ctx, http.MethodGet, "/api/v1/apps", "", nil, &result)
	return result, err
}

func (c *Client) Capabilities(ctx context.Context) (map[string]any, error) {
	var result map[string]any
	err := c.request(ctx, http.MethodGet, "/api/v1/capabilities", "", nil, &result)
	return result, err
}

// Source returns nil only when a script does not exist.
func (c *Client) Source(ctx context.Context, name string) (*string, error) {
	if err := ValidateAppName(name); err != nil {
		return nil, err
	}
	var raw []byte
	err := c.request(ctx, http.MethodGet, "/api/v1/apps/script/"+name, "", nil, &raw)
	var status *HTTPError
	if errors.As(err, &status) && status.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	text := string(raw)
	return &text, nil
}

func (c *Client) ScriptSettings(ctx context.Context, name string) (map[string]any, error) {
	if err := ValidateAppName(name); err != nil {
		return nil, err
	}
	var result struct {
		Fields []struct {
			Key   string `json:"key"`
			Value any    `json:"value"`
		} `json:"fields"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v1/apps/"+name+"/config", "", nil, &result); err != nil {
		return nil, err
	}
	settings := map[string]any{}
	for _, f := range result.Fields {
		settings[f.Key] = f.Value
	}
	return settings, nil
}

func (c *Client) DeleteApp(ctx context.Context, name string) error {
	if err := ValidateAppName(name); err != nil {
		return err
	}
	return c.confirmedCommand(ctx, http.MethodDelete, "/api/v1/apps/"+name, nil)
}

func (c *Client) SetData(ctx context.Context, name string, data any) error {
	if err := ValidateAppName(name); err != nil {
		return err
	}
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var result struct {
		OK    bool            `json:"ok"`
		Error json.RawMessage `json:"error"`
	}
	if err := c.request(ctx, http.MethodPatch, "/api/v1/apps/"+name+"/data", "application/json", body, &result); err != nil {
		return err
	}
	if !result.OK || (len(result.Error) > 0 && string(result.Error) != "null") {
		return &SettingsRestartError{App: name, Detail: string(result.Error)}
	}
	return nil
}
