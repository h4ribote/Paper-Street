package api

import (
	"net/http"
	"strings"
)

const apiKeyHeader = "X-API-Key"

func (s *Server) withAPIKeyAuth(next http.Handler) http.Handler {
	if s == nil || s.APIKeys == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		apiKey := strings.TrimSpace(r.Header.Get(apiKeyHeader))
		if apiKey == "" {
			respondError(w, http.StatusUnauthorized, "API key required")
			return
		}
		if !s.APIKeys.ContainsHex(apiKey) {
			respondError(w, http.StatusUnauthorized, "Invalid API key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isPublicPath(path string) bool {
	switch path {
	case "/", "/index.html", "/health", "/auth/login", "/auth/bot", "/auth/callback", "/ws":
		return true
	}
	if strings.HasPrefix(path, "/css/") || strings.HasPrefix(path, "/js/") {
		return true
	}
	return false
}
