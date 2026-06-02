package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCheckRateLimitResponseAdvertisesPaidAPIHandoff(t *testing.T) {
	h := NewCheckHandler(nil)
	h.counts[hashIP(httptest.NewRequest(http.MethodPost, "/api/v1/check", strings.NewReader(`{}`)))] = checkFreeLimit
	// resetAt must be in the future relative to the test's wall-clock, or
	// allow() (check.go: now.After(h.resetAt)) wipes the pre-loaded count and
	// returns 200. A hardcoded epoch is a time-bomb; derive it from now.
	resetAt := time.Now().Add(time.Hour).Truncate(time.Second)
	h.resetAt = resetAt

	req := httptest.NewRequest(http.MethodPost, "/api/v1/check", strings.NewReader(`{"url":"stripe.com"}`))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429; body=%s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("X-RateLimit-Limit") != "10" {
		t.Fatalf("X-RateLimit-Limit = %q, want 10", rr.Header().Get("X-RateLimit-Limit"))
	}

	var payload struct {
		Error           string   `json:"error"`
		PlansURL        string   `json:"plans_url"`
		SubscribeURL    string   `json:"subscribe_url"`
		SubscribeMethod string   `json:"subscribe_method"`
		SubscribeFields []string `json:"subscribe_fields"`
		ResetAtUnix     int64    `json:"reset_at_unix"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error != "rate limit exceeded" {
		t.Fatalf("error = %q", payload.Error)
	}
	if payload.PlansURL != "https://nothumansearch.ai/api/v1/api-keys/subscribe" {
		t.Fatalf("plans_url = %q", payload.PlansURL)
	}
	if payload.SubscribeURL != payload.PlansURL || payload.SubscribeMethod != "POST" {
		t.Fatalf("subscribe handoff = %q %q", payload.SubscribeMethod, payload.SubscribeURL)
	}
	if len(payload.SubscribeFields) != 2 || payload.SubscribeFields[0] != "email" || payload.SubscribeFields[1] != "plan" {
		t.Fatalf("subscribe_fields = %#v", payload.SubscribeFields)
	}
	if payload.ResetAtUnix != resetAt.Unix() {
		t.Fatalf("reset_at_unix = %d", payload.ResetAtUnix)
	}
}
