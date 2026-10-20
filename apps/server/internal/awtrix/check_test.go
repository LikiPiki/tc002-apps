package awtrix

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestObserveScript(t *testing.T) {
	for _, test := range []struct {
		name    string
		app     string
		frame   string
		runtime bool
		fail    bool
	}{
		{"static black frame is valid", "demo", `{"width":1,"height":1,"pixels":[0]}`, false, false},
		{"draw error", "demo", `{"width":1,"height":1,"pixels":[1]}`, true, true},
		{"wrong app", "Time", `{"width":1,"height":1,"pixels":[1]}`, false, true},
		{"invalid frame", "demo", `{"width":2,"height":1,"pixels":[1]}`, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/apps":
					if test.runtime {
						fmt.Fprint(w, `[{"name":"demo","origin":"script","enabled":true,"error":{"hook":"draw","message":"broken"}}]`)
					} else {
						fmt.Fprint(w, `[{"name":"demo","origin":"script","enabled":true,"error":null}]`)
					}
				case "/api/v1/device":
					fmt.Fprintf(w, `{"currentApp":%q}`, test.app)
				case "/api/v1/display/screen":
					fmt.Fprint(w, test.frame)
				}
			}))
			defer server.Close()
			client, _ := NewClient(server.URL)
			frames, err := client.ObserveScript(context.Background(), "demo", 250*time.Millisecond)
			if (err != nil) != test.fail || (!test.fail && frames < 2) {
				t.Fatalf("frames=%d error=%v, want failure=%v", frames, err, test.fail)
			}
		})
	}
}

func TestObserveScriptCancellation(t *testing.T) {
	client, _ := NewClient("http://127.0.0.1:1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.ObserveScript(ctx, "demo", time.Second); err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}
}
