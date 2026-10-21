package awtrix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInstallScriptUsesAtomicCompareAndSwap(t *testing.T) {
	oldSource := "# Preserve exact whitespace\nreturn Old()\n"

	for _, test := range []struct {
		name string
		old  *string
	}{
		{"new script", nil},
		{"existing script", &oldSource},
	} {
		t.Run(test.name, func(t *testing.T) {
			updated := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method + " " + r.URL.Path {
				case "GET /api/v1/capabilities":
					fmt.Fprint(w, `{"scriptUpdates":true}`)
				case "GET /api/v1/apps/script/demo":
					if test.old == nil {
						w.WriteHeader(http.StatusNotFound)
					} else {
						fmt.Fprint(w, *test.old)
					}
				case "PUT /api/v1/apps/script-update/demo":
					var body struct {
						Expected *string `json:"expected_source"`
						Source   string  `json:"source"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if r.Header.Get("Content-Type") != "application/json" || body.Source != "return Demo()" || (body.Expected == nil) != (test.old == nil) {
						t.Errorf("unexpected update: %+v", body)
					}
					if body.Expected != nil && *body.Expected != *test.old {
						t.Error("old source was changed before comparison")
					}
					updated = true
					fmt.Fprint(w, `{"ok":true}`)
				default:
					t.Errorf("unsafe or unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			defer server.Close()
			client, _ := NewClient(server.URL)
			err := client.InstallScript(context.Background(), "demo", []byte("return Demo()"))
			if err != nil || !updated {
				t.Fatalf("update = %v, error = %v", updated, err)
			}
		})
	}
}

func TestAtomicUpdateNeverFallsBackOnFailure(t *testing.T) {
	for _, status := range []int{409, 422} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			writes := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/capabilities":
					fmt.Fprint(w, `{"scriptUpdates":true}`)
				case "/api/v1/apps/script/demo":
					if r.Method != http.MethodGet {
						t.Error("unsafe replacement attempted")
					}
					fmt.Fprint(w, "return Working()")
				case "/api/v1/apps/script-update/demo":
					writes++
					w.WriteHeader(status)
				default:
					t.Error("unexpected request")
				}
			}))
			defer server.Close()
			client, _ := NewClient(server.URL)
			if err := client.InstallScript(context.Background(), "demo", []byte("bad source")); err == nil || writes != 1 {
				t.Fatalf("writes=%d error=%v", writes, err)
			}
		})
	}
}

func TestAtomicUpdatesRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/capabilities" {
			t.Error("must stop before reading or writing scripts")
		}
		fmt.Fprint(w, `{"scriptUpdates":false}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL)
	if err := client.InstallScript(context.Background(), "demo", []byte("return Demo()")); err == nil {
		t.Fatal("accepted unsupported firmware")
	}
}

func TestCheckScriptDetectsRuntimeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"demo","origin":"script","enabled":true,"error":{"message":"bad draw","hook":"draw"}}]`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL)
	if err := client.CheckScript(context.Background(), "demo"); err == nil {
		t.Fatal("runtime error was ignored")
	}
}

func TestInstallScriptRejectsUnsafeOrReservedNames(t *testing.T) {
	client, _ := NewClient("http://127.0.0.1:1")
	for _, name := range []string{"../demo", "active", "", "a/b"} {
		if err := ValidateAppName(name); err == nil {
			t.Errorf("accepted invalid name %q", name)
		}
	}
	if err := client.InstallScript(context.Background(), "../demo", nil); err == nil {
		t.Fatal("unsafe upload accepted")
	}
}
