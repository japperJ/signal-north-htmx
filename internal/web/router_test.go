package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouterForwardsRequiredRoutes(t *testing.T) {
	marker := func(label string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == "/demo/missing" {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, label)
		})
	}

	router := NewRouter(Dependencies{
		Home:   marker("home"),
		Health: marker("health"),
		Static: marker("static"),
		Demo:   marker("demo"),
		Events: marker("events"),
	})

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"home", http.MethodGet, "/", "home"},
		{"health", http.MethodGet, "/healthz", "health"},
		{"static", http.MethodGet, "/static/app.css", "static"},
		{"telemetry", http.MethodGet, "/demo/telemetry", "demo"},
		{"search", http.MethodGet, "/demo/search", "demo"},
		{"command", http.MethodPost, "/demo/command", "demo"},
		{"activity", http.MethodPost, "/demo/activity", "demo"},
		{"profile-read", http.MethodGet, "/demo/profile", "demo"},
		{"profile-update", http.MethodPut, "/demo/profile", "demo"},
		{"activity-delete", http.MethodDelete, "/demo/activity/1", "demo"},
		{"status", http.MethodGet, "/demo/status", "demo"},
		{"lazy", http.MethodGet, "/demo/lazy", "demo"},
		{"explanation", http.MethodGet, "/demo/explain?demo=telemetry", "demo"},
		{"events", http.MethodGet, "/events", "events"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", res.Code)
			}
			if strings.TrimSpace(res.Body.String()) != tc.body {
				t.Fatalf("body = %q, want %q", res.Body.String(), tc.body)
			}
		})
	}
}

func TestRouterDoesNotRewriteMissingFragment(t *testing.T) {
	router := NewRouter(Dependencies{
		Home: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "homepage")
		}),
		Demo: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}),
	})

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/missing", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.Code)
	}
	if strings.Contains(res.Body.String(), "homepage") {
		t.Fatal("missing fragment was rewritten to the homepage")
	}
}
