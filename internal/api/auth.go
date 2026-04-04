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
	cleanPath := normalizePath(path)
	switch cleanPath {
	case "/", "/index.html", "/health", "/auth/login", "/auth/bot", "/auth/callback", "/ws":
		return true
	}
	if strings.HasPrefix(cleanPath, "/css/") || strings.HasPrefix(cleanPath, "/js/") {
		return true
	}
	return false
}

func normalizePath(path string) string {
	cleaned := path
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	cleaned = strings.ReplaceAll(cleaned, "\\", "/")
	for strings.Contains(cleaned, "//") {
		cleaned = strings.ReplaceAll(cleaned, "//", "/")
	}
	segments := strings.Split(cleaned, "/")
	stack := make([]string, 0, len(segments))
	for _, segment := range segments {
		switch segment {
		case "", ".":
			continue
		case "..":
			if len(stack) == 0 {
				return ""
			}
			stack = stack[:len(stack)-1]
		default:
			stack = append(stack, segment)
		}
	}
	return "/" + strings.Join(stack, "/")
}
