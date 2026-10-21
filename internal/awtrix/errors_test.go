package awtrix

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPErrorsContainOperationAndBoundedDetails(t *testing.T) {
	for _, test := range []struct {
		name   string
		body   string
		status int
		want   []string
	}{
		{
			name:   "structured",
			body:   `{"error":{"code":"validationFailed","message":"invalid notification","field":"text"}}`,
			status: http.StatusUnprocessableEntity,
			want:   []string{"validationFailed", "invalid notification", "field=text"},
		},
		{"plain text", "service temporarily\nunavailable", 503, []string{"service temporarily unavailable"}},
		{"empty", "", 401, nil},
		{"invalid JSON", "{broken", 500, []string{"{broken"}},
		{"large response", strings.Repeat("x", 100000), 502, []string{"..."}},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			client, _ := NewClient(server.URL)
			err := client.Notify(context.Background(), "hello")
			var status *HTTPError
			if !errors.As(err, &status) || status.StatusCode != test.status {
				t.Fatalf("HTTP status information lost: %v", err)
			}
			for _, want := range append(test.want, "POST /api/v1/notifications", fmt.Sprint(test.status)) {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("missing %q in %v", want, err)
				}
			}
			if len(err.Error()) > 1000 {
				t.Fatal("unbounded error response")
			}
		})
	}
}

func TestMalformedSuccessResponseIncludesOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not JSON")
	}))
	defer server.Close()
	client, _ := NewClient(server.URL)
	err := client.CheckScript(context.Background(), "demo")
	if err == nil || !strings.Contains(err.Error(), "GET /api/v1/apps: decode response") {
		t.Fatalf("missing decode context: %v", err)
	}
}

func TestTransportFailureIncludesOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	client, _ := NewClient(server.URL)
	server.Close()
	err := client.Notify(context.Background(), "hello")
	if err == nil || !strings.Contains(err.Error(), "POST /api/v1/notifications") {
		t.Fatalf("missing transport context: %v", err)
	}
}
