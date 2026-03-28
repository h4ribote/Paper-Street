package api

import (
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
