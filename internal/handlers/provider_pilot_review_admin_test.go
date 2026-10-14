package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminProviderPilotReviewRequiresOwnerAuthentication(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/provider-pilot-review?pilot_id=623e4567-e89b-42d3-a456-426614174000&review_type=provider&subject_id=123e4567-e89b-42d3-a456-426614174000",
		nil,
	)
	recorder := httptest.NewRecorder()
	(&ProviderExchangeHandler{}).AdminProviderPilotReview(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized review status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAdminProviderPilotReviewGETRequiresExactBoundedIdentity(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	for _, query := range []string{
		"",
		"?pilot_id=623e4567-e89b-42d3-a456-426614174000&review_type=unknown&subject_id=123e4567-e89b-42d3-a456-426614174000",
		"?pilot_id=623e4567-e89b-42d3-a456-426614174000&review_type=ticket&subject_id=not-a-uuid",
	} {
		request := httptest.NewRequest(
			http.MethodGet, "/api/v1/admin/provider-pilot-review"+query, nil,
		)
		request.Header.Set("Authorization", "Bearer test-admin-key")
		recorder := httptest.NewRecorder()
		(&ProviderExchangeHandler{}).AdminProviderPilotReview(recorder, request)
		if recorder.Code != http.StatusBadRequest ||
			!strings.Contains(recorder.Body.String(), "invalid provider exchange request") {
			t.Fatalf("invalid review query status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	}
}

func TestAdminProviderPilotReviewPOSTIsStrictAndRequiresSnapshotDigest(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	for _, body := range []string{
		`{"provider_pilot_epoch_id":"623e4567-e89b-42d3-a456-426614174000","review_type":"ticket","subject_id":"123e4567-e89b-42d3-a456-426614174000","owner_reference":"owner:review:ticket","evidence_reference":"evidence:review:ticket"}`,
		`{"provider_pilot_epoch_id":"623e4567-e89b-42d3-a456-426614174000","review_type":"ticket","subject_id":"123e4567-e89b-42d3-a456-426614174000","expected_snapshot_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","owner_reference":"owner:review:ticket","evidence_reference":"evidence:review:ticket","query":"must-not-be-accepted"}`,
	} {
		request := httptest.NewRequest(
			http.MethodPost, "/api/v1/admin/provider-pilot-review",
			bytes.NewBufferString(body),
		)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer test-admin-key")
		recorder := httptest.NewRecorder()
		(&ProviderExchangeHandler{}).AdminProviderPilotReview(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("invalid review body status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	}
}

func TestAdminProviderPilotReviewRejectsUnsupportedMethods(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/provider-pilot-review", nil)
	request.Header.Set("Authorization", "Bearer test-admin-key")
	recorder := httptest.NewRecorder()
	(&ProviderExchangeHandler{}).AdminProviderPilotReview(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("unsupported review method status=%d allow=%q body=%s",
			recorder.Code, recorder.Header().Get("Allow"), recorder.Body.String())
	}
}

func TestProviderPilotReviewEvidenceScopeDisclaimsCommercialProofAndSensitiveData(t *testing.T) {
	for _, required := range []string{
		"no search receipt", "principal or agent identity/contact/network metadata",
		"does not create", "3/5/2/1 proof",
	} {
		if !strings.Contains(providerPilotReviewEvidenceScope, required) {
			t.Fatalf("review evidence scope missing %q: %s", required, providerPilotReviewEvidenceScope)
		}
	}
}
