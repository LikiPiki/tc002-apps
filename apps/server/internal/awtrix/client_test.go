package awtrix

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInstallScriptChecksBerryErrorEvenWithHTTP200(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		fail bool
	}{
		{"success", `{"ok":true,"error":null}`, false},
		{"syntax error", `{"ok":true,"error":{"message":"syntax_error"}}`, true},
		{"missing error status", `{"ok":true}`, true},
		{"not confirmed", `{"ok":false,"error":null}`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if r.Method != http.MethodPut || r.URL.Path != "/api/v1/apps/script/demo" || r.Header.Get("Content-Type") != "text/plain" || string(body) != "return Demo()" {
					t.Errorf("unexpected upload: %s %s %q", r.Method, r.URL.Path, body)
				}
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			client, _ := NewClient(server.URL)
			err := client.InstallScript(context.Background(), "demo", []byte("return Demo()"))
			if (err != nil) != test.fail {
				t.Fatalf("error = %v, want failure = %v", err, test.fail)
			}
		})
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
