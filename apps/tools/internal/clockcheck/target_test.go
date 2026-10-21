package clockcheck

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LikiPiki/tc002-apps/internal/awtrix"
)

func TestCheckInvocationRejectsDeviceURLs(t *testing.T) {
	for _, pair := range [][2]string{
		{"http://192.168.1.50", ""},
		{"http://192.168.1.50", "1"},
		{SandboxURL, ""},
		{"http://localhost:8080", "1"},
	} {
		if err := CheckInvocation(pair[0], pair[1]); err == nil {
			t.Errorf("accepted target=%q marker=%q", pair[0], pair[1])
		}
	}

	if err := CheckInvocation(SandboxURL, "1"); err != nil {
		t.Fatal(err)
	}
}

func TestRequireSandboxRejectsHardwareBeforeWrites(t *testing.T) {
	for _, board := range []string{"tc002", "esp32", "", "linux"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" || r.URL.Path != "/api/v1/device" {
				t.Errorf("unexpected write/access: %s %s", r.Method, r.URL.Path)
			}
			fmt.Fprintf(w, `{"boardType":%q}`, board)
		}))

		client, _ := awtrix.NewClient(server.URL)

		err := RequireSandbox(context.Background(), client)
		server.Close()

		if (err == nil) != (board == "linux") {
			t.Errorf("board=%q error=%v", board, err)
		}
	}
}
