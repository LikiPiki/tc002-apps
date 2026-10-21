package clocks

import (
	"context"
	"fmt"
	"time"

	"github.com/LikiPiki/tc002-apps/internal/config"
)

type ProfileAction func(context.Context, Target, config.ClockProfile) (any, error)

func RunProfiles(ctx context.Context, profiles []config.ClockProfile, timeout time.Duration, action ProfileAction) (Report, error) {
	var addresses []string
	owners := map[string]config.ClockProfile{}

	for _, profile := range profiles {
		for _, address := range profile.Clock.Targets() {
			address = config.NormalizeAddress(address)
			if previous, exists := owners[address]; exists && previous.Name != profile.Name {
				return Report{}, fmt.Errorf("clock %s has conflicting profiles", address)
			}

			addresses = append(addresses, address)
			owners[address] = profile
		}
	}

	report := Run(ctx, addresses, timeout, func(ctx context.Context, target Target) (any, error) {
		return action(ctx, target, owners[target.URL])
	})

	for index := range report.Results {
		report.Results[index].Profile = owners[report.Results[index].Target].Name
	}

	return report, nil
}
