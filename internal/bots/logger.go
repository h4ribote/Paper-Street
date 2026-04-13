package bots

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

type loggingRoundTripper struct {
	next   http.RoundTripper
	logger *log.Logger
}

func (lrt *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	var reqBodyBytes []byte
	if req.Body != nil {
		reqBodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
	}

	res, err := lrt.next.RoundTrip(req)
	duration := time.Since(start)

	var resBodyBytes []byte
	if err == nil && res != nil && res.Body != nil {
		resBodyBytes, _ = io.ReadAll(res.Body)
		res.Body = io.NopCloser(bytes.NewBuffer(resBodyBytes))
	}

	reqBodyStr := string(reqBodyBytes)
	resBodyStr := string(resBodyBytes)

	if reqBodyStr == "" {
		reqBodyStr = "-"
	}
	if resBodyStr == "" {
		resBodyStr = "-"
	}

	if err != nil {
		lrt.logger.Printf("ERROR %s %s - %v - %v | Req: %s", req.Method, req.URL.Path, err, duration, reqBodyStr)
	} else {
		lrt.logger.Printf("%s %s - %d %v | Req: %s | Res: %s", req.Method, req.URL.Path, res.StatusCode, duration, reqBodyStr, resBodyStr)
	}

	return res, err
}

func getLoggingTransport(role string) http.RoundTripper {
	if strings.ToLower(os.Getenv("ENABLE_API_LOGGING")) != "true" {
		return http.DefaultTransport
	}

	logPath := filepath.Join("logs", role+"_api.log")
	if envPath := os.Getenv("API_LOG_DIR"); envPath != "" {
		logPath = filepath.Join(envPath, role+"_api.log")
	}

	err := os.MkdirAll(filepath.Dir(logPath), 0755)
	if err != nil {
		log.Printf("failed to create log directory for %s: %v", role, err)
		return http.DefaultTransport
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("failed to open log file for %s: %v", role, err)
		return http.DefaultTransport
	}

	logger := log.New(file, "", log.LstdFlags)
	return &loggingRoundTripper{
		next:   http.DefaultTransport,
		logger: logger,
	}
}
