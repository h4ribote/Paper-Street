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
		if r.URL.Path == "/health" || r.URL.Path == "/auth/login" || r.URL.Path == "/auth/callback" {
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
