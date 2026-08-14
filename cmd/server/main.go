package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	assets "htmx-showcase"
	"htmx-showcase/internal/demo"
	"htmx-showcase/internal/web"
)

func portFromEnv() string {
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		return port
	}
	return "8080"
}

func newHTTPServer() (*http.Server, error) {
	templateFS, err := fs.Sub(assets.FS, "templates")
	if err != nil {
		return nil, err
	}
	staticFS, err := fs.Sub(assets.FS, "static")
	if err != nil {
		return nil, err
	}
	demoServer, err := demo.New(templateFS)
	if err != nil {
		return nil, err
	}

	health := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	static := http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))
	router := web.NewRouter(web.Dependencies{
		Home:   demoServer,
		Health: health,
		Static: static,
		Demo:   demoServer,
		Events: demoServer,
	})
	return &http.Server{Addr: ":" + portFromEnv(), Handler: router}, nil
}

func main() {
	server, err := newHTTPServer()
	if err != nil {
		slog.Error("build server", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped unexpectedly", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown", "error", err)
	}
}
