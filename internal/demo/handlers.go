package demo

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"
)

type Server struct {
	templates *template.Template
	state     *State
	mux       *http.ServeMux
}

type pageData struct {
	Snapshot StateSnapshot
	Query    string
	Results  []Command
}

type commandData struct {
	Command  Command
	Snapshot StateSnapshot
}

type errorData struct {
	Title   string
	Message string
}

type eventData struct {
	Sequence int
	Time     time.Time
	Payload  template.HTML
}

type explanationData struct {
	Title  string
	HTMX   string
	Server string
	Client string
	Markup string
	Code   string
}

var explanations = map[string]explanationData{
	"telemetry": {
		Title:  "A GET that returns fresh HTML",
		HTMX:   "The button sends hx-get to /demo/telemetry, shows hx-indicator while waiting, then swaps the response into #telemetry-panel.",
		Server: "Go reads the shared in-memory state, advances the deterministic telemetry values, and renders only the telemetry fragment.",
		Client: "The browser supplies the click and DOM target. HTMX performs the request and replacement; no client-side metric calculation is needed.",
		Markup: `<button hx-get="/demo/telemetry" hx-target="#telemetry-panel" hx-swap="innerHTML" hx-indicator="#telemetry-loading">Refresh telemetry</button>`,
		Code: `func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	s.renderFragment(w, http.StatusOK, "telemetry", s.state.RefreshTelemetry())
}`,
	},
	"search": {
		Title:  "Debounce without a search framework",
		HTMX:   "The input uses hx-trigger=\"input changed delay:300ms\" with hx-get and hx-target, so typing settles before a request is sent.",
		Server: "Go receives q, filters the command catalog, and returns matching command buttons as HTML.",
		Client: "The browser owns the text cursor and input value. HTMX owns the debounce timer, request, loading state, and swap.",
		Markup: `<input name="q" hx-get="/demo/search" hx-trigger="input changed delay:300ms" hx-target="#search-results" hx-indicator="#search-loading">`,
		Code: `func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	s.renderFragment(w, http.StatusOK, "search-results", struct {
		Query   string
		Results []Command
	}{Query: query, Results: s.state.Search(query)})
}`,
	},
	"command": {
		Title:  "One POST, two HTML updates",
		HTMX:   "The form uses hx-post and swaps the command result normally while hx-swap-oob updates #metric-requests from the same response.",
		Server: "Go validates the command, increments the request metric, and renders both the success result and the out-of-band metric markup.",
		Client: "The browser serializes the form. HTMX inserts the main result and finds the OOB element by id; no JSON store exists in the page.",
		Markup: `<form hx-post="/demo/command" hx-target="#command-result" hx-swap="innerHTML" hx-indicator="#command-loading">`,
		Code: `func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Invalid command", "The command payload could not be read.")
		return
	}
	command, snapshot, ok := s.state.ExecuteCommand(r.FormValue("command"))
	if !ok {
		s.renderError(w, r, http.StatusUnprocessableEntity, "Command required", "Command is required before dispatch.")
		return
	}
	s.renderFragment(w, http.StatusOK, "command-result", commandData{Command: command, Snapshot: snapshot})
}`,
	},
	"health": {
		Title:  "Polling that stays server-driven",
		HTMX:   "hx-trigger=\"every 5s\" schedules a GET and hx-target=\"this\" replaces the health panel contents.",
		Server: "Go advances the health check sequence, derives the current state, and returns a small status fragment.",
		Client: "The browser only keeps the polling timer and target element. Health state is never duplicated in JavaScript.",
		Markup: `<section hx-get="/demo/status" hx-trigger="every 5s" hx-target="this" hx-swap="innerHTML">`,
		Code: `func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.renderFragment(w, http.StatusOK, "status", s.state.Status())
}`,
	},
	"activity": {
		Title:  "HTML list mutations",
		HTMX:   "The add form uses hx-post with afterbegin; each delete button uses hx-delete, closest article, hx-swap=delete, and hx-confirm.",
		Server: "Go validates and stores activity items, or removes an id, then returns the new article or an error response.",
		Client: "The browser provides form and confirmation events. HTMX inserts or removes the targeted article without owning the activity list.",
		Markup: `<form hx-post="/demo/activity" hx-target="#activity-list" hx-swap="afterbegin" hx-indicator="#activity-loading">`,
		Code: `func (s *Server) handleAddActivity(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Invalid activity", "The activity payload could not be read.")
		return
	}
	activity, ok := s.state.AddActivity(r.FormValue("message"))
	if !ok {
		s.renderError(w, r, http.StatusUnprocessableEntity, "Activity required", "Activity message is required.")
		return
	}
	s.renderFragment(w, http.StatusOK, "activity-item", activity)
}`,
	},
	"profile": {
		Title:  "PUT editing with an HTML fallback",
		HTMX:   "The edit form uses hx-put to save into #profile-panel; hx-get can load the form again when the display view is shown.",
		Server: "Go validates the profile value, updates shared state, and renders the display fragment after a successful PUT.",
		Client: "The browser owns focus and native required-field validation. HTMX sends the form and replaces the panel with server-rendered markup.",
		Markup: `<form hx-put="/demo/profile" hx-target="#profile-panel" hx-swap="innerHTML">`,
		Code: `func (s *Server) handleProfileUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Invalid profile", "The profile payload could not be read.")
		return
	}
	if !s.state.UpdateProfile(r.FormValue("profile")) {
		s.renderError(w, r, http.StatusUnprocessableEntity, "Profile required", "Profile value is required.")
		return
	}
	s.renderFragment(w, http.StatusOK, "profile", s.state.Snapshot())
}`,
	},
	"lazy": {
		Title:  "Request only when revealed",
		HTMX:   "hx-trigger=\"revealed\" starts the GET when the panel enters the viewport, targeting itself for an innerHTML swap.",
		Server: "Go renders the architecture detail only when /demo/lazy is requested; it is not precomputed into the initial fragment.",
		Client: "The browser observes viewport visibility. HTMX turns that visibility event into a request and then removes the loading copy through the swap.",
		Markup: `<section hx-get="/demo/lazy" hx-trigger="revealed" hx-target="this" hx-swap="innerHTML">`,
		Code: `func (s *Server) handleLazy(w http.ResponseWriter, r *http.Request) {
	s.renderFragment(w, http.StatusOK, "lazy-panel", s.state.Snapshot())
}`,
	},
	"sse": {
		Title:  "Events instead of polling",
		HTMX:   "The local SSE extension opens sse-connect=/events and listens for the signal event named by sse-swap.",
		Server: "Go keeps an event-stream response open, flushes deterministic event/data frames, and stops when the client disconnects.",
		Client: "The browser maintains the EventSource connection. The SSE extension routes each HTML payload into #event-stream; there is no polling timer.",
		Markup: `<section hx-ext="sse" sse-connect="/events" sse-target="#event-stream" sse-swap="signal">`,
		Code: `func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// The handler then flushes each event until the client disconnects.
}`,
	},
	"history": {
		Title:  "Navigation state without a router library",
		HTMX:   "hx-push-url updates the browser address and history after the HTML swap, so Back and Forward remain meaningful.",
		Server: "Go handles /demo/history like any other route and returns a small HTML fragment. The server remains the source of the rendered state.",
		Client: "The browser owns the address bar and history stack. HTMX changes the URL without a full-page reload; there is no client-side router.",
		Markup: `<a href="/demo/history" hx-get="/demo/history" hx-target="#history-result" hx-swap="innerHTML" hx-push-url="true">Open request history</a>`,
		Code: `func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	s.renderFragment(w, http.StatusOK, "history", s.state.Status())
}`,
	},
}

func New(templateFS fs.FS) (*Server, error) {
	return NewWithState(templateFS, NewState())
}

func NewWithState(templateFS fs.FS, state *State) (*Server, error) {
	templates, err := template.New("index.html").Funcs(template.FuncMap{
		"formatTime": func(value time.Time) string { return value.Local().Format("15:04:05") },
		"formatAge":  func(value time.Time) string { return age(value) },
		"dict": func(values ...any) map[string]any {
			result := make(map[string]any, len(values)/2)
			for index := 0; index+1 < len(values); index += 2 {
				key, ok := values[index].(string)
				if ok {
					result[key] = values[index+1]
				}
			}
			return result
		},
	}).ParseFS(templateFS, "index.html", "partials/*.html", "fragments/*.html")
	if err != nil {
		return nil, fmt.Errorf("load templates: %w", err)
	}

	server := &Server{templates: templates, state: state, mux: http.NewServeMux()}
	server.mux.HandleFunc("GET /", server.handleHome)
	server.mux.HandleFunc("GET /demo/telemetry", server.handleTelemetry)
	server.mux.HandleFunc("GET /demo/search", server.handleSearch)
	server.mux.HandleFunc("POST /demo/command", server.handleCommand)
	server.mux.HandleFunc("POST /demo/activity", server.handleAddActivity)
	server.mux.HandleFunc("DELETE /demo/activity/{id}", server.handleDeleteActivity)
	server.mux.HandleFunc("POST /demo/activity/{id}", server.handleActivityFallback)
	server.mux.HandleFunc("GET /demo/profile", server.handleProfile)
	server.mux.HandleFunc("PUT /demo/profile", server.handleProfileUpdate)
	server.mux.HandleFunc("POST /demo/profile", server.handleProfileFallback)
	server.mux.HandleFunc("GET /demo/status", server.handleStatus)
	server.mux.HandleFunc("GET /demo/lazy", server.handleLazy)
	server.mux.HandleFunc("GET /demo/explain", server.handleExplain)
	server.mux.HandleFunc("GET /demo/history", server.handleHistory)
	server.mux.HandleFunc("GET /events", server.handleEvents)
	return server, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.render(w, http.StatusOK, "index.html", pageData{Snapshot: s.state.Snapshot()})
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	s.renderFragment(w, http.StatusOK, "telemetry", s.state.RefreshTelemetry())
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	s.renderFragment(w, http.StatusOK, "search-results", struct {
		Query   string
		Results []Command
	}{Query: query, Results: s.state.Search(query)})
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Invalid command", "The command payload could not be read.")
		return
	}
	command, snapshot, ok := s.state.ExecuteCommand(r.FormValue("command"))
	if !ok {
		s.renderError(w, r, http.StatusUnprocessableEntity, "Command required", "Command is required before dispatch.")
		return
	}
	s.renderFragment(w, http.StatusOK, "command-result", commandData{Command: command, Snapshot: snapshot})
}

func (s *Server) handleAddActivity(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Invalid activity", "The activity payload could not be read.")
		return
	}
	activity, ok := s.state.AddActivity(r.FormValue("message"))
	if !ok {
		s.renderError(w, r, http.StatusUnprocessableEntity, "Activity required", "Activity message is required.")
		return
	}
	s.renderFragment(w, http.StatusOK, "activity-item", activity)
}

func (s *Server) handleDeleteActivity(w http.ResponseWriter, r *http.Request) {
	if !s.state.DeleteActivity(r.PathValue("id")) {
		s.renderError(w, r, http.StatusNotFound, "Activity not found", "That activity item no longer exists.")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleActivityFallback(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("_method") != http.MethodDelete {
		http.NotFound(w, r)
		return
	}
	s.handleDeleteActivity(w, r)
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	s.renderFragment(w, http.StatusOK, "edit-form", s.state.Snapshot())
}

func (s *Server) handleProfileUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Invalid profile", "The profile payload could not be read.")
		return
	}
	if !s.state.UpdateProfile(r.FormValue("profile")) {
		s.renderError(w, r, http.StatusUnprocessableEntity, "Profile required", "Profile value is required.")
		return
	}
	s.renderFragment(w, http.StatusOK, "profile", s.state.Snapshot())
}

func (s *Server) handleProfileFallback(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("_method") != http.MethodPut {
		http.NotFound(w, r)
		return
	}
	s.handleProfileUpdate(w, r)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.renderFragment(w, http.StatusOK, "status", s.state.Status())
}

func (s *Server) handleLazy(w http.ResponseWriter, r *http.Request) {
	s.renderFragment(w, http.StatusOK, "lazy-panel", s.state.Snapshot())
}

func (s *Server) handleExplain(w http.ResponseWriter, r *http.Request) {
	explanation, ok := explanations[r.URL.Query().Get("demo")]
	if !ok {
		s.renderError(w, r, http.StatusNotFound, "Explanation not found", "That demo does not have an explanation.")
		return
	}
	s.renderFragment(w, http.StatusOK, "explanation", explanation)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	s.renderFragment(w, http.StatusOK, "history", s.state.Status())
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	sequence := 1
	if err := s.writeEvent(w, flusher, sequence); err != nil {
		return
	}
	maxEvents := 0
	if os.Getenv("VERCEL") == "1" {
		maxEvents = 3
	}
	ticker := time.NewTicker(900 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			sequence++
			if err := s.writeEvent(w, flusher, sequence); err != nil {
				return
			}
			if maxEvents > 0 && sequence >= maxEvents {
				return
			}
		}
	}
}

func (s *Server) writeEvent(w http.ResponseWriter, flusher http.Flusher, sequence int) error {
	var builder strings.Builder
	now := time.Now().UTC()
	payload := template.HTML(fmt.Sprintf(`<div class="sse-event"><span class="status-dot status-dot-cyan"></span><span>Signal %d received from north-01</span><time>%s UTC</time></div>`, sequence, now.Local().Format("15:04:05")))
	if err := s.templates.ExecuteTemplate(&builder, "sse-event", eventData{Sequence: sequence, Time: now, Payload: payload}); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, builder.String()); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func (s *Server) renderFragment(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = s.templates.ExecuteTemplate(w, name, data)
}

func (s *Server) render(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = s.templates.ExecuteTemplate(w, name, data)
}

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, status int, title, message string) {
	if r.Header.Get("HX-Request") == "true" {
		s.renderFragment(w, status, "error", errorData{Title: title, Message: message})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, "<!doctype html><html lang=\"en\"><head><title>%s</title></head><body><main><h1>%s</h1><p>%s</p></main></body></html>", template.HTMLEscapeString(title), template.HTMLEscapeString(title), template.HTMLEscapeString(message))
}

func age(value time.Time) string {
	minutes := int(time.Since(value).Minutes())
	if minutes < 1 {
		return "just now"
	}
	return fmt.Sprintf("%dm ago", minutes)
}
