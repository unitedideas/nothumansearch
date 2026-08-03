package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminProviderActionRequiresEvidenceReference(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/provider-offers/action", bytes.NewBufferString(`{
		"action":"activate",
		"offer_id":"4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		"evidence_reference":"notes with free form"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-key")
	rr := httptest.NewRecorder()
	(&ProviderExchangeHandler{}).AdminOfferAction(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("admin action status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestLegacyFundingActionsFailClosedInTermsOnlyPilot(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	for _, action := range []string{"fund", "adjust"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/provider-offers/action", bytes.NewBufferString(`{
			"action":"`+action+`",
			"offer_id":"4b69ca8e-d61d-47e2-91dd-fecd9f711234",
			"amount_cents":100,
			"currency":"usd",
			"operator_reference":"operator-case-1",
			"evidence_reference":"owner-case-1"
		}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-admin-key")
		rr := httptest.NewRecorder()
		(&ProviderExchangeHandler{}).AdminOfferAction(rr, req)
		if rr.Code != http.StatusConflict {
			t.Fatalf("%s status=%d body=%s", action, rr.Code, rr.Body.String())
		}
	}
}
