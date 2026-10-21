package clockcheck

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

// CheckSnapshots freezes animation time in a sandbox-only copy of the source.
// Golden hashes compare raw pixels, independent of GIF palette quantization.
func CheckSnapshots(ctx context.Context, client *awtrix.Client, name, path string, source []byte, artifacts string) error {
	data, err := os.ReadFile(filepath.Join(filepath.Dir(path), "check.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var expectations struct {
		Snapshots map[string]string `json:"snapshots"`
	}
	if err := json.Unmarshal(data, &expectations); err != nil {
		return err
	}
	var timestamps []string
	for timestamp := range expectations.Snapshots {
		timestamps = append(timestamps, timestamp)
	}
	sort.Strings(timestamps)

	actual := map[string]string{}
	var failures []error
	var frames []awtrix.Frame
	for _, timestamp := range timestamps {
		if _, err := strconv.ParseUint(timestamp, 10, 31); err != nil {
			return fmt.Errorf("invalid snapshot time %q", timestamp)
		}
		frozen := strings.ReplaceAll(string(source), "now_ms()", timestamp)
		if err := client.InstallScript(ctx, "snapshot", []byte(frozen)); err != nil {
			return err
		}
		if err := client.ShowScript(ctx, "snapshot"); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
		if err := client.CheckScript(ctx, "snapshot"); err != nil {
			return err
		}
		frame, err := client.Frame(ctx)
		if err != nil {
			return err
		}
		frames = append(frames, frame)
		encoded, err := json.Marshal(frame)
		if err != nil {
			return err
		}
		digest := fmt.Sprintf("%x", sha256.Sum256(encoded))
		actual[timestamp] = digest
		if digest != expectations.Snapshots[timestamp] {
			failures = append(failures, fmt.Errorf("%s snapshot at %sms: got %s", name, timestamp, digest))
		}
	}

	if artifacts != "" && len(frames) > 0 {
		if err := SaveGIF(filepath.Join(artifacts, name+"-snapshots.gif"), frames); err != nil {
			return err
		}
		report, err := json.MarshalIndent(actual, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(artifacts, name+"-snapshots.json"), report, 0o644); err != nil {
			return err
		}
	}
	return errors.Join(failures...)
}
