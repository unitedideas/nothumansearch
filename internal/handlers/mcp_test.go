package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestMCPGetAdvertisesCanonicalToolList(t *testing.T) {
	h := NewMCPHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /mcp status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Tools []string `json:"tools"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode GET /mcp JSON: %v", err)
	}

	want := h.toolNames()
	if len(payload.Tools) != len(want) {
		t.Fatalf("GET /mcp advertises %d tools, want %d: got=%v want=%v", len(payload.Tools), len(want), payload.Tools, want)
	}
	for i := range want {
		if payload.Tools[i] != want[i] {
			t.Fatalf("GET /mcp tool %d = %q, want %q", i, payload.Tools[i], want[i])
		}
	}
}

func TestMCPRejectsRequestBodyOver64KiB(t *testing.T) {
	h := NewMCPHandler(nil, "https://nothumansearch.ai")
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","padding":"` +
		strings.Repeat("x", int(mcpMaxRequestBodyBytes)+1) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized MCP request status = %d, want 413; body=%s", rr.Code, rr.Body.String())
	}
	var response rpcResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode oversized MCP response: %v", err)
	}
	if response.Error == nil || response.Error.Code != -32000 {
		t.Fatalf("oversized MCP response error = %#v, want code -32000", response.Error)
	}
}

func TestMCPRejectsMultipleJSONValuesBeforeExecuting(t *testing.T) {
	h := NewMCPHandler(nil, "https://nothumansearch.ai")
	h.discoveryRateLimiter = newMCPDiscoveryRateLimiter(1, time.Hour)
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.RemoteAddr = "203.0.113.10:4567"
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	var response rpcResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode multiple-value MCP response: %v", err)
	}
	if response.Error == nil || response.Error.Code != -32700 {
		t.Fatalf("multiple-value MCP response error = %#v, want parse error -32700", response.Error)
	}

	// A valid call from the same client must still receive the sole allowance.
	// That proves the first JSON value was rejected before tool dispatch.
	valid := postMCP(t, h, "tools/list")
	if valid.Code != http.StatusOK {
		t.Fatalf("valid call after rejected multiple values status = %d, want 200; body=%s", valid.Code, valid.Body.String())
	}
}
