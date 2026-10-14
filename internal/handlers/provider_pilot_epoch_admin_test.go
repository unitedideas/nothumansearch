package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func providerPilotAdminRequest(t *testing.T, path, body string) *http.Request {
	t.Helper()
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-key")
	return req
}

func TestAdminProviderPilotEpochActionRejectsCrossActionFields(t *testing.T) {
	cases := []string{
		`{"action":"authorize","provider_pilot_epoch_id":"4b69ca8e-d61d-47e2-91dd-fecd9f711234","demand_topic":"developer-tools","cohort_limit":10,"provider_ticket_cap":5,"total_ticket_cap":10,"owner_reference":"owner:case-1","evidence_reference":"evidence:case-1"}`,
		`{"action":"enroll","provider_pilot_epoch_id":"4b69ca8e-d61d-47e2-91dd-fecd9f711234","provider_claim_id":"5b69ca8e-d61d-47e2-91dd-fecd9f711234","demand_topic":"jobs","owner_reference":"owner:case-1","evidence_reference":"evidence:case-1"}`,
		`{"action":"activate","provider_pilot_epoch_id":"4b69ca8e-d61d-47e2-91dd-fecd9f711234","provider_claim_id":"5b69ca8e-d61d-47e2-91dd-fecd9f711234","owner_reference":"owner:case-1","evidence_reference":"evidence:case-1"}`,
	}
	for _, body := range cases {
		rr := httptest.NewRecorder()
		(&ProviderExchangeHandler{}).AdminProviderPilotEpochAction(
			rr, providerPilotAdminRequest(t, "/api/v1/admin/provider-pilot/action", body),
		)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("cross-action shape status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func TestAdminProviderPilotEpochActionIsOwnerGatedAndStrict(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	unauthorized := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/provider-pilot/action",
		strings.NewReader(`{"action":"close"}`),
	)
	unauthorized.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	(&ProviderExchangeHandler{}).AdminProviderPilotEpochAction(rr, unauthorized)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d body=%s", rr.Code, rr.Body.String())
	}

	unknown := providerPilotAdminRequest(t, "/api/v1/admin/provider-pilot/action", `{
		"action":"close",
		"provider_pilot_epoch_id":"4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		"owner_reference":"owner:case-1",
		"evidence_reference":"evidence:case-1",
		"query":"must-not-be-accepted"
	}`)
	rr = httptest.NewRecorder()
	(&ProviderExchangeHandler{}).AdminProviderPilotEpochAction(rr, unknown)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unknown-field status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminProviderPilotEpochStatusRequiresExactPilotID(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/provider-pilot/epoch", nil)
	req.Header.Set("Authorization", "Bearer test-admin-key")
	rr := httptest.NewRecorder()
	(&ProviderExchangeHandler{}).AdminProviderPilotEpochStatus(rr, req)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "exact pilot_id required") {
		t.Fatalf("missing-pilot status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestProviderPilotEpochMutationResponseDisclaimsProof(t *testing.T) {
	response := providerPilotEpochMutationResponse("activate", map[string]string{"id": "opaque"})
	if response["commercial_proof_created"] != false || response["action"] != "activate" {
		t.Fatalf("pilot mutation response=%#v", response)
	}
}
