package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckHandlerRejectsNonPublicTargetBeforeCrawl(t *testing.T) {
	t.Parallel()

	handler := NewCheckHandler(nil)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/check",
		strings.NewReader(`{"url":"http://169.254.169.254/latest/meta-data"}`),
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "public HTTP(S) address") {
		t.Fatalf("body = %q, want public-target validation error", recorder.Body.String())
	}
}

func TestVerifyMCPRejectsNonPublicTargetBeforeProbe(t *testing.T) {
	t.Parallel()

	handler := NewAPIHandler(nil)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/verify-mcp?url=http://127.0.0.1:8080/mcp",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.VerifyMCP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "public HTTP(S) address") {
		t.Fatalf("body = %q, want public-target validation error", recorder.Body.String())
	}
}

func TestSubmitSiteRejectsNonPublicTargetBeforeDatabaseWrite(t *testing.T) {
	t.Parallel()

	handler := NewAPIHandler(nil)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/submit",
		strings.NewReader(`{"url":"http://10.0.0.1/admin"}`),
	)
	recorder := httptest.NewRecorder()

	handler.SubmitSite(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "public HTTP(S) address") {
		t.Fatalf("body = %q, want public-target validation error", recorder.Body.String())
	}
}
