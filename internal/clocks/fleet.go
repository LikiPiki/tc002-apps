// Package clocks delivers independent operations to configured AWTRIX devices.
package clocks

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
	"github.com/LikiPiki/tc002-apps/internal/config"
)

const DefaultTimeout = 10 * time.Second

type Target struct {
	URL    string
	Index  int
	Client *awtrix.Client
}

type Result struct {
	Target  string `json:"target"`
	Profile string `json:"profile,omitempty"`
	State   string `json:"state"`
	Error   string `json:"error,omitempty"`
	Detail  any    `json:"detail,omitempty"`
}

type Report struct {
	Results []Result `json:"results"`
}

type Action func(context.Context, Target) (any, error)

// Run uses at most four independent device connections. An offline device cannot
// block delivery to healthy clocks. Every operation probes the device before writing.
func Run(ctx context.Context, addresses []string, timeout time.Duration, action Action) Report {
	addresses = uniqueAddresses(addresses)
	report := Report{Results: make([]Result, len(addresses))}
	workers := make(chan struct{}, 4)
	var pending sync.WaitGroup

	for index, address := range addresses {
		pending.Add(1)

		go func() {
			defer pending.Done()

			select {
			case workers <- struct{}{}:
				defer func() { <-workers }()
			case <-ctx.Done():
				report.Results[index] = Result{Target: address, State: "skipped", Error: ctx.Err().Error()}
				return
			}

			report.Results[index] = runTarget(ctx, address, index, timeout, action)
		}()
	}

	pending.Wait()

	return report
}

func runTarget(parent context.Context, address string, index int, timeout time.Duration, action Action) Result {
	result := Result{Target: address, State: "skipped"}
	client, err := awtrix.NewClient(address)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	probe, stopProbe := context.WithTimeout(ctx, 3*time.Second)
	err = identify(probe, client)
	stopProbe()
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Detail, err = action(ctx, Target{URL: address, Index: index, Client: client})
	if err != nil {
		result.State = "failed"
		result.Error = err.Error()
		return result
	}

	result.State = "succeeded"

	return result
}

func identify(ctx context.Context, client *awtrix.Client) error {
	device, err := client.Device(ctx)
	if err != nil {
		return fmt.Errorf("AWTRIX identification: %w", err)
	}

	switch device.BoardType {
	case "tc002", "linux", "esp32", "esp32-s3", "esp32s3":
	default:
		return fmt.Errorf("not an AWTRIX clock: unsupported or missing boardType %q", device.BoardType)
	}

	capabilities, err := client.Capabilities(ctx)
	if err != nil {
		return fmt.Errorf("AWTRIX capabilities: %w", err)
	}
	if _, ok := capabilities["scriptUpdates"].(bool); !ok {
		return fmt.Errorf("not an AWTRIX NG API: missing scriptUpdates capability")
	}

	return nil
}

func uniqueAddresses(addresses []string) []string {
	seen := map[string]bool{}
	var unique []string

	for _, address := range addresses {
		address = config.NormalizeAddress(address)

		if !seen[address] {
			unique = append(unique, address)
			seen[address] = true
		}
	}

	return unique
}

func (report Report) Succeeded() int {
	count := 0
	for _, result := range report.Results {
		if result.State == "succeeded" {
			count++
		}
	}

	return count
}

// Err treats partial delivery as success with warnings, but never hides a total failure.
func (report Report) Err() error {
	if len(report.Results) == 0 {
		return fmt.Errorf("no clock targets configured")
	}
	if report.Succeeded() == 0 {
		return fmt.Errorf("operation did not succeed on any of %d configured clock(s)", len(report.Results))
	}

	return nil
}

func (report Report) Warn(operation string) {
	for _, result := range report.Results {
		if result.State != "succeeded" {
			slog.Warn("clock operation unsuccessful; continuing with other clocks",
				"operation", operation,
				"target", result.Target,
				"profile", result.Profile,
				"state", result.State,
				"error", result.Error)
		}
	}
}
