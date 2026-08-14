package demo

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDemoState(t *testing.T) {
	snapshot := NewState().Snapshot()
	if len(snapshot.Telemetry) < 3 {
		t.Fatalf("telemetry values = %d, want at least 3", len(snapshot.Telemetry))
	}
	if len(snapshot.Commands) < 3 {
		t.Fatalf("commands = %d, want at least 3", len(snapshot.Commands))
	}
	if len(snapshot.Activities) < 3 {
		t.Fatalf("activities = %d, want at least 3", len(snapshot.Activities))
	}
	if snapshot.Health == "" {
		t.Fatal("health state is empty")
	}
	if snapshot.Profile == "" {
		t.Fatal("profile value is empty")
	}
}

func TestTemplatesAndHomePage(t *testing.T) {
	app := newTestApp(t)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	for _, text := range []string{
		"Signal North",
		"zero custom application JavaScript",
		"Server-rendered HTML",
		"HTMX handles browser behavior",
		"Refresh telemetry",
		"Search commands",
		"Send command",
		"Activity stream",
		"Health monitor",
		"Lazy-loaded panel",
		"Inline editing",
		"SSE stream",
		"hx-get",
		"hx-post",
		"hx-put",
		"hx-delete",
		"hx-target",
		"hx-swap",
		"hx-trigger",
		"hx-indicator",
		"hx-confirm",
		"/static/css/app.css",
		"/static/vendor/htmx.min.js",
		"/static/vendor/htmx-ext-sse.js",
	} {
		if !strings.Contains(body, text) {
			t.Errorf("homepage does not contain %q", text)
		}
	}
}

func TestStyles(t *testing.T) {
	data, err := os.ReadFile("../../static/css/app.css")
	if err != nil {
		t.Fatalf("read stylesheet: %v", err)
	}
	css := string(data)
	for _, text := range []string{":focus-visible", "@media", ".htmx-request", "prefers-reduced-motion"} {
		if !strings.Contains(css, text) {
			t.Errorf("stylesheet does not contain %q", text)
		}
	}
}

func TestTelemetry(t *testing.T) {
	app := newTestApp(t)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/telemetry", nil))

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Telemetry refreshed") || strings.Contains(body, "<html") || strings.Contains(body, "<body") {
		t.Fatalf("unexpected telemetry fragment: %s", body)
	}
}

func TestSearch(t *testing.T) {
	app := newTestApp(t)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/search?q=deploy", nil))
	body := res.Body.String()
	if res.Code != http.StatusOK || !strings.Contains(body, "deploy") {
		t.Fatalf("search response = %d %s", res.Code, body)
	}

	res = httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/search?q=", nil))
	if !strings.Contains(res.Body.String(), "Type a command") {
		t.Fatalf("empty search response = %s", res.Body.String())
	}
}

func TestCommand(t *testing.T) {
	app := newTestApp(t)
	form := url.Values{"command": {"deploy api"}}
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/demo/command", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "Command accepted") || !strings.Contains(res.Body.String(), "hx-swap-oob") {
		t.Fatalf("command response = %d %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/demo/command", strings.NewReader("command="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ServeHTTP(res, req)
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), "Command is required") {
		t.Fatalf("empty command response = %d %s", res.Code, res.Body.String())
	}
}

func TestActivityAndProfile(t *testing.T) {
	app := newTestApp(t)
	form := url.Values{"message": {"New deployment started"}}
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/demo/activity", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "New deployment started") {
		t.Fatalf("activity response = %d %s", res.Code, res.Body.String())
	}

	id := app.state.Snapshot().Activities[0].ID
	res = httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodDelete, "/demo/activity/"+id, nil))
	if res.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", res.Code)
	}

	res = httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/profile", nil))
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "Save profile") {
		t.Fatalf("profile form = %d %s", res.Code, res.Body.String())
	}

	form = url.Values{"profile": {"production-east"}}
	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/demo/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "production-east") {
		t.Fatalf("profile update = %d %s", res.Code, res.Body.String())
	}
}

func TestStatusAndLazy(t *testing.T) {
	app := newTestApp(t)
	for _, path := range []string{"/demo/status", "/demo/lazy"} {
		res := httptest.NewRecorder()
		app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		if res.Code != http.StatusOK || strings.Contains(res.Body.String(), "<html") {
			t.Fatalf("%s response = %d %s", path, res.Code, res.Body.String())
		}
	}
}

func TestExplanations(t *testing.T) {
	app := newTestApp(t)
	for _, demo := range []string{"telemetry", "search", "command", "health", "activity", "profile", "lazy", "sse", "history", "shaping", "sync", "headers", "transition", "validate"} {
		res := httptest.NewRecorder()
		app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/explain?demo="+demo, nil))
		body := res.Body.String()
		if res.Code != http.StatusOK || !strings.Contains(body, "HTMX") || !strings.Contains(body, "Server") || !strings.Contains(body, "Browser") || !strings.Contains(body, "Show exact Go function") || !strings.Contains(body, "Show exact HTMX markup") || !strings.Contains(body, "func (s *Server) handle") || !strings.Contains(body, "hx-") {
			t.Fatalf("explanation %q = %d %s", demo, res.Code, body)
		}
	}

	res := httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/explain?demo=unknown", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("unknown explanation status = %d, want 404", res.Code)
	}
}

func TestHistory(t *testing.T) {
	app := newTestApp(t)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/history", nil))
	if res.Code != http.StatusOK || strings.Contains(res.Body.String(), "<html") || !strings.Contains(res.Body.String(), "History entry loaded") {
		t.Fatalf("history response = %d %s", res.Code, res.Body.String())
	}
}

func TestAdvancedHTMXFeatures(t *testing.T) {
	app := newTestApp(t)

	res := httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/shaping?context=north-1&mode=safe", nil))
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "north-1") {
		t.Fatalf("shaping response = %d %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/demo/sync", strings.NewReader("service=api")))
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "Synchronized check accepted") {
		t.Fatalf("sync response = %d %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/headers", nil))
	if res.Code != http.StatusOK || res.Header().Get("HX-Trigger") != "server-signal" {
		t.Fatalf("header response = %d trigger=%q", res.Code, res.Header().Get("HX-Trigger"))
	}

	res = httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/transition", nil))
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "Transition complete") {
		t.Fatalf("transition response = %d %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	validateRequest := httptest.NewRequest(http.MethodPost, "/demo/validate", strings.NewReader("service=api-service"))
	validateRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	validateRequest.Header.Set("HX-Request", "true")
	app.ServeHTTP(res, validateRequest)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "Validation passed") {
		t.Fatalf("validation response = %d %s", res.Code, res.Body.String())
	}
}

func TestErrorsAndMissingRoutes(t *testing.T) {
	app := newTestApp(t)

	res := httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/missing", nil))
	if res.Code != http.StatusNotFound || strings.Contains(res.Body.String(), "Signal North") {
		t.Fatalf("missing demo route = %d %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/demo/command", strings.NewReader("command="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	app.ServeHTTP(res, req)
	if res.Code != http.StatusUnprocessableEntity || strings.Contains(res.Body.String(), "<html") {
		t.Fatalf("HTMX validation = %d %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/demo/missing", nil))
	if res.Code != http.StatusNotFound || !strings.Contains(res.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("ordinary missing route = %d content-type=%q", res.Code, res.Header().Get("Content-Type"))
	}
}

func TestSSE(t *testing.T) {
	app := newTestApp(t)
	ts := httptest.NewServer(app)
	t.Cleanup(ts.Close)

	res, err := ts.Client().Get(ts.URL + "/events")
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	if got := res.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("content type = %q", got)
	}
	reader := bufio.NewReader(res.Body)
	var lines []string
	for len(lines) < 3 {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			t.Fatalf("read SSE event: %v", readErr)
		}
		lines = append(lines, line)
	}
	data := strings.Join(lines, "")
	if !strings.Contains(data, "event: signal") || !strings.Contains(data, "data:") {
		t.Fatalf("invalid SSE event: %s", data)
	}
}

func TestVercelSSEStreamIsBounded(t *testing.T) {
	t.Setenv("VERCEL", "1")
	app := newTestApp(t)
	ts := httptest.NewServer(app)
	t.Cleanup(ts.Close)

	client := ts.Client()
	client.Timeout = 3 * time.Second
	res, err := client.Get(ts.URL + "/events")
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read bounded SSE stream: %v", err)
	}
	if !strings.Contains(string(data), "event: signal") {
		t.Fatalf("bounded stream did not contain an event: %s", data)
	}
}
