package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMonitorAdminActionRequiresBearerAuth(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	h := NewMonitorHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/monitors/action", bytes.NewBufferString(`{"id":1}`))
	rr := httptest.NewRecorder()

	h.AdminAction(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("AdminAction status = %d, want 401; body=%s", rr.Code, rr.Body.String())
	}
}

func TestMonitorAdminActionRejectsInvalidActionBeforeDB(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	h := NewMonitorHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/monitors/action", bytes.NewBufferString(`{
		"id": 1,
		"action": "delete_monitor",
		"operator": "qlimit-test"
	}`))
	req.Header.Set("Authorization", "Bearer test-admin-key")
	rr := httptest.NewRecorder()

	h.AdminAction(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("AdminAction status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
}

func TestMonitorAdminActionCountsRequiresAuth(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	h := NewMonitorHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/monitors/actions", nil)
	rr := httptest.NewRecorder()

	h.AdminActionCounts(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("AdminActionCounts status = %d, want 401; body=%s", rr.Code, rr.Body.String())
	}
}

func TestMonitorAdminActionRejectsMissingOperatorBeforeDB(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	h := NewMonitorHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/monitors/action", bytes.NewBufferString(`{
		"id": 1,
		"action": "approve_monitoring"
	}`))
	req.Header.Set("Authorization", "Bearer test-admin-key")
	rr := httptest.NewRecorder()

	h.AdminAction(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("AdminAction status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
}

func TestMonitorAdminActionRequiresConfiguredAdminKey(t *testing.T) {
	os.Unsetenv("ADMIN_API_KEY")
	h := NewMonitorHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/monitors/action", bytes.NewBufferString(`{"id":1}`))
	rr := httptest.NewRecorder()

	h.AdminAction(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("AdminAction status = %d, want 503; body=%s", rr.Code, rr.Body.String())
	}
}

func TestMonitorAdminActionErrorResponsesDoNotExposePrivateFields(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	h := NewMonitorHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/monitors/action", bytes.NewBufferString(`{
		"id": 1,
		"action": "delete_monitor",
		"operator": "qlimit-test",
		"notes": "private row note"
	}`))
	req.Header.Set("Authorization", "Bearer test-admin-key")
	rr := httptest.NewRecorder()

	h.AdminAction(rr, req)

	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["error"] == "" {
		t.Fatalf("expected error response, got %s", rr.Body.String())
	}
	if bytes.Contains(rr.Body.Bytes(), []byte("private row note")) {
		t.Fatalf("error response exposed private notes: %s", rr.Body.String())
	}
}

func TestMonitorLandingPrefillsDomainFromScoreHandoff(t *testing.T) {
	h := NewMonitorHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodGet, "/monitor?domain=example.com", nil)
	rr := httptest.NewRecorder()

	h.LandingPage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("LandingPage status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("new URLSearchParams(window.location.search)")) {
		t.Fatalf("landing page does not read query params: %s", rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("document.getElementById('monitor-domain').value = monitorDomain")) {
		t.Fatalf("landing page does not prefill monitor domain: %s", rr.Body.String())
	}
}
