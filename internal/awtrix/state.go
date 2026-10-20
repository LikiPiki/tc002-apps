package awtrix

import (
	"context"
	"net/http"
)

type DeviceInfo struct {
	CurrentApp       string `json:"currentApp"`
	ScriptingRunning bool   `json:"scriptingRunning"`
}

type Frame struct {
	Width  int      `json:"width"`
	Height int      `json:"height"`
	Pixels []uint32 `json:"pixels"`
}

func (c *Client) Device(ctx context.Context) (DeviceInfo, error) {
	var result DeviceInfo
	err := c.request(ctx, http.MethodGet, "/api/v1/device", "", nil, &result)
	return result, err
}

func (c *Client) Frame(ctx context.Context) (Frame, error) {
	var result Frame
	err := c.request(ctx, http.MethodGet, "/api/v1/display/screen", "", nil, &result)
	return result, err
}
