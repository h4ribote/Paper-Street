package bots

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDecodeAPIErrorLimitsBodyRead(t *testing.T) {
	oversized := strings.Repeat("x", maxAPIErrorBodyBytes*2)
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(oversized)),
	}

	err := decodeAPIError(resp)
	if err == nil {
		t.Fatal("expected api error")
	}
	remaining, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("failed to read remaining body: %v", readErr)
	}
	if len(remaining) != maxAPIErrorBodyBytes {
		t.Fatalf("expected %d remaining bytes, got %d", maxAPIErrorBodyBytes, len(remaining))
	}
}
