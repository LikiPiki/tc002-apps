package awtrix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *Client) SystemSettings(ctx context.Context) (map[string]any, error) {
	var settings map[string]any
	err := c.request(ctx, http.MethodGet, "/api/v1/system", "", nil, &settings)
	return settings, err
}

// ConfigureSystem sends only declared keys and checks the persisted values.
func (c *Client) ConfigureSystem(ctx context.Context, settings map[string]any) error {
	if len(settings) == 0 {
		return nil
	}
	body, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	var result map[string]any
	if err := c.request(ctx, http.MethodPut, "/api/v1/system", "application/json", body, &result); err != nil {
		return err
	}
	actual, err := c.SystemSettings(ctx)
	if err != nil {
		return err
	}
	for key, want := range settings {
		left, _ := json.Marshal(want)
		right, _ := json.Marshal(actual[key])
		if string(left) != string(right) {
			return fmt.Errorf("system setting %s was not persisted", key)
		}
	}
	return nil
}
