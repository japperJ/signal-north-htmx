package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestPortFromEnvDefaultsTo8080(t *testing.T) {
	t.Setenv("PORT", "")
	if got := portFromEnv(); got != "8080" {
		t.Fatalf("port = %q, want 8080", got)
	}
}

func TestPortFromEnvHonorsConfiguredPort(t *testing.T) {
	t.Setenv("PORT", "9090")
	if got := portFromEnv(); got != "9090" {
		t.Fatalf("port = %q, want 9090", got)
	}
}

func TestServerHealthEndpoint(t *testing.T) {
	old := os.Getenv("PORT")
	t.Cleanup(func() { _ = os.Setenv("PORT", old) })
	_ = os.Setenv("PORT", "8080")

	server, err := newHTTPServer()
	if err != nil {
		t.Fatalf("newHTTPServer() error = %v", err)
	}

	ts := httptest.NewServer(server.Handler)
	t.Cleanup(ts.Close)

	res, err := ts.Client().Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz error = %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
}
