package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestUIEndpointWorksOutsideRepoCWD(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tmp, err := os.MkdirTemp("", "mesnada-ui-cwd-*")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// /ui should be served from embedded assets, not from the filesystem.
	{
		req := httptest.NewRequest(http.MethodGet, "/ui", nil)
		w := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected /ui status 200, got %d (body=%q)", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, "<title>Mesnada") {
			t.Fatalf("expected /ui body to contain title, got prefix=%q", body[:min(len(body), 200)])
		}
	}

	// UI partials should also render using embedded templates.
	{
		req := httptest.NewRequest(http.MethodGet, "/ui/partials/tasks", nil)
		w := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected /ui/partials/tasks status 200, got %d (body=%q)", w.Code, w.Body.String())
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
