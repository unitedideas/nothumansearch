package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func postMCP(t *testing.T, h http.Handler, method string) *httptest.ResponseRecorder {
	t.Helper()
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + method + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.RemoteAddr = "203.0.113.10:4567"
	req.Header.Set("User-Agent", "python-httpx/0.28.1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestMCPDiscoveryRateLimitBucketsByMethod(t *testing.T) {
	h := NewMCPHandler(nil, "https://nothumansearch.ai")
	h.discoveryRateLimiter = newMCPDiscoveryRateLimiter(2, time.Hour)

	for i := 0; i < 2; i++ {
		rr := postMCP(t, h, "tools/list")
		if rr.Code != http.StatusOK {
			t.Fatalf("tools/list call %d status = %d, want 200; body=%s", i+1, rr.Code, rr.Body.String())
		}
	}
	rr := postMCP(t, h, "tools/list")
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("third tools/list status = %d, want 429; body=%s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header on 429")
	}
	if got := rr.Header().Get("X-RateLimit-Limit"); got != "2" {
		t.Fatalf("X-RateLimit-Limit = %q, want 2", got)
	}

	rr = postMCP(t, h, "initialize")
	if rr.Code != http.StatusOK {
		t.Fatalf("initialize should use a separate method bucket, got status %d; body=%s", rr.Code, rr.Body.String())
	}
}
