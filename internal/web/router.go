package web

import "net/http"

type Dependencies struct {
	Home   http.Handler
	Health http.Handler
	Static http.Handler
	Demo   http.Handler
	Events http.Handler
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", fallback(deps.Health))
	mux.Handle("/static/", fallback(deps.Static))
	mux.Handle("GET /events", fallback(deps.Events))
	mux.Handle("/events/", http.NotFoundHandler())
	mux.Handle("/demo/", fallback(deps.Demo))
	mux.Handle("/", fallback(deps.Home))
	return mux
}

func fallback(handler http.Handler) http.Handler {
	if handler == nil {
		return http.NotFoundHandler()
	}
	return handler
}
