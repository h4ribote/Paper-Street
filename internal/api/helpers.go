package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parseQueryInt64(r *http.Request, key string) int64 {
	if r == nil {
		return 0
	}
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseUserID(r *http.Request) int64 {
	return parseQueryInt64(r, "user_id")
}

func (s *Server) resolveUserID(r *http.Request, requestedUserID int64, required bool) (int64, int, string) {
	authenticatedUserID := s.userIDFromRequest(r)
	apiKeyProvided := false
	if r != nil {
		apiKeyProvided = strings.TrimSpace(r.Header.Get(apiKeyHeader)) != ""
	}
	if authenticatedUserID != 0 {
		if requestedUserID != 0 && requestedUserID != authenticatedUserID {
			return 0, http.StatusUnauthorized, "user_id does not match authenticated user"
		}
		return authenticatedUserID, 0, ""
	}
	if apiKeyProvided && s != nil && s.APIKeys != nil {
		return 0, http.StatusUnauthorized, "authenticated user not found"
	}
	if requestedUserID != 0 {
		return requestedUserID, 0, ""
	}
	if required {
		return 0, http.StatusBadRequest, "user_id required"
	}
	return 0, 0, ""
}

func parseLimit(r *http.Request, fallback int) int {
	if r == nil {
		return fallback
	}
	value := strings.TrimSpace(r.URL.Query().Get("limit"))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseOffset(r *http.Request, fallback int) int {
	if r == nil {
		return fallback
	}
	value := strings.TrimSpace(r.URL.Query().Get("offset"))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func parseTimeframe(value string) (time.Duration, bool) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return 0, false
	}
	switch trimmed {
	case "1m":
		return time.Minute, true
	case "5m":
		return 5 * time.Minute, true
	case "15m":
		return 15 * time.Minute, true
	case "30m":
		return 30 * time.Minute, true
	case "1h":
		return time.Hour, true
	case "4h":
		return 4 * time.Hour, true
	case "1d":
		return 24 * time.Hour, true
	default:
		return 0, false
	}
}

func parseUnixMillis(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.UnixMilli(parsed).UTC(), true
}

func parsePathID(path, prefix string) (int64, []string, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return 0, nil, errors.New("id required")
	}
	segments := strings.Split(trimmed, "/")
	if len(segments) == 0 || segments[0] == "" {
		return 0, nil, errors.New("id required")
	}
	id, err := parseID(segments[0])
	if err != nil {
		return 0, nil, err
	}
	return id, segments[1:], nil
}

func parseID(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("id required")
	}
	if strings.Contains(value, "/") {
		return 0, errors.New("invalid id path")
	}
	return strconv.ParseInt(value, 10, 64)
}
