package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
)

func TestFrontendRootServesIndex(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("failed to chdir repo root: %v", err)
	}

	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	eng := engine.NewEngine(nil, store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("failed to request /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPublicAssetPathsBypassAPIKey(t *testing.T) {
	if !isPublicPath("/css/style.css") {
		t.Fatalf("expected /css/style.css to be public")
	}
	if !isPublicPath("/js/app.js") {
		t.Fatalf("expected /js/app.js to be public")
	}
	if isPublicPath("/css/../../etc/passwd") {
		t.Fatalf("expected traversal css path to require auth")
	}
	if isPublicPath("/js/../../../secrets") {
		t.Fatalf("expected traversal js path to require auth")
	}
	if isPublicPath("/market/ticker") {
		t.Fatalf("expected /market/ticker to require auth")
	}
}

func TestFrontendStaticTraversalBlocked(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("failed to chdir repo root: %v", err)
	}

	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	eng := engine.NewEngine(nil, store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
	defer server.Close()

	resp1, err := http.Get(server.URL + "/css/../../etc/passwd")
	if err != nil {
		t.Fatalf("failed to request traversal css path: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for traversal css path, got %d", resp1.StatusCode)
	}

	resp2, err := http.Get(server.URL + "/js/../../../secrets")
	if err != nil {
		t.Fatalf("failed to request traversal js path: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for traversal js path, got %d", resp2.StatusCode)
	}
}
