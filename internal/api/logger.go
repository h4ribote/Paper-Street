package api

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	if lrw.statusCode == 0 {
		lrw.statusCode = code
		lrw.ResponseWriter.WriteHeader(code)
	}
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.statusCode == 0 {
		lrw.statusCode = http.StatusOK
	}
	lrw.body.Write(b)
	return lrw.ResponseWriter.Write(b)
}

func WithLogging(next http.Handler, name string) http.Handler {
	if strings.ToLower(os.Getenv("ENABLE_API_LOGGING")) != "true" {
		return next
	}
	logPath := filepath.Join("logs", name+"_api.log")
	if envPath := os.Getenv("API_LOG_DIR"); envPath != "" {
		logPath = filepath.Join(envPath, name+"_api.log")
	}

	err := os.MkdirAll(filepath.Dir(logPath), 0755)
	if err != nil {
		log.Printf("failed to create log directory for %s: %v", name, err)
		return next
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("failed to open log file for %s: %v", name, err)
		return next
	}

	logger := log.New(file, "", log.LstdFlags)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var reqBodyBytes []byte
		if r.Body != nil {
			reqBodyBytes, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
		}

		lrw := &loggingResponseWriter{w, 0, &bytes.Buffer{}}
		next.ServeHTTP(lrw, r)

		duration := time.Since(start)

		reqBodyStr := string(reqBodyBytes)
		resBodyStr := lrw.body.String()

		if reqBodyStr == "" {
			reqBodyStr = "-"
		}
		if resBodyStr == "" {
			resBodyStr = "-"
		}

		logger.Printf("[%s] %s %s - %d %v | Req: %s | Res: %s", r.RemoteAddr, r.Method, r.URL.Path, lrw.statusCode, duration, reqBodyStr, resBodyStr)
	})
}
