package demo

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func newTestApp(t *testing.T) *Server {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	rootFS := os.DirFS(root)
	templateFS, err := fs.Sub(rootFS, "templates")
	if err != nil {
		t.Fatalf("template fs: %v", err)
	}
	app, err := NewWithState(templateFS, NewState())
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}
