package clockdeploy

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/LikiPiki/tc002-apps/apps/tools/internal/clockapps"
	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

type fakeAPI struct {
	failPhase string
	cause     error
	calls     []string
}

func TestSettingsSavedButRestartFailedIsNotReportedAsUnconfirmed(t *testing.T) {
	cause := &awtrix.SettingsRestartError{App: "second", Detail: "bad schedule"}
	api := &fakeAPI{failPhase: "settings", cause: cause}
	report, err := Run(context.Background(), api, []clockapps.Script{{Name: "second", Kind: "services", Settings: map[string]any{"enabled": true}}})
	result := report.Results[0]
	if !errors.Is(err, cause) || result.Source != "updated" || result.Settings != "applied" || result.FailedPhase != "restart" || result.Verification != "failed" {
		t.Fatalf("confirmed settings save was hidden: %+v, %v", result, err)
	}
	if strings.Contains(report.String(), "Unconfirmed writes") {
		t.Fatal("confirmed failure must not be described as an uncertain write")
	}
}

func (f *fakeAPI) step(name, phase string) error {
	f.calls = append(f.calls, name+":"+phase)
	if name == "second" && phase == f.failPhase {
		return f.cause
	}
	return nil
}
func (f *fakeAPI) InstallScript(_ context.Context, name string, _ []byte) error {
	return f.step(name, "source")
}
func (f *fakeAPI) ConfigureScript(_ context.Context, name string, _ map[string]any) error {
	return f.step(name, "settings")
}
func (f *fakeAPI) CheckScript(_ context.Context, name string) error {
	return f.step(name, "verification")
}

func TestPartialDeploymentReportsConfirmedAndPendingPhases(t *testing.T) {
	for _, phase := range []string{"source", "settings", "verification"} {
		t.Run(phase, func(t *testing.T) {
			cause := errors.New("device failure")
			api := &fakeAPI{failPhase: phase, cause: cause}
			scripts := []clockapps.Script{
				{Name: "first", Kind: "screens"},
				{Name: "second", Kind: "services", Settings: map[string]any{"enabled": true}},
				{Name: "third", Kind: "screens"},
			}
			report, err := Run(context.Background(), api, scripts)
			if !errors.Is(err, cause) || report.Results[0].Verification != "passed" || report.Results[1].FailedPhase != phase || report.Results[2].Source != "not-attempted" {
				t.Fatalf("unexpected failure report: %+v, %v", report, err)
			}
			if phase != "source" && report.Results[1].Source != "updated" {
				t.Fatal("confirmed source update was hidden")
			}
			for _, call := range api.calls {
				if strings.HasPrefix(call, "third:") {
					t.Fatal("batch continued after failure")
				}
			}
			if !strings.Contains(report.String(), "1/3 apps verified") || (phase != "verification" && !strings.Contains(report.String(), "unconfirmed")) {
				t.Fatal(report.String())
			}
		})
	}
}

func TestSuccessfulDeploymentDoesNotInventSettingsUpdate(t *testing.T) {
	api := &fakeAPI{}
	report, err := Run(context.Background(), api, []clockapps.Script{{Name: "first", Kind: "screens"}})
	if err != nil || report.Results[0].Settings != "not-configured" || !reflect.DeepEqual(api.calls, []string{"first:source", "first:verification"}) {
		t.Fatalf("unexpected report: %+v, %v, %v", report, err, api.calls)
	}
}
