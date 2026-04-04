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
	eng := engine.NewEngine(store)
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
	if isPublicPath("/market/ticker") {
		t.Fatalf("expected /market/ticker to require auth")
	}
}
