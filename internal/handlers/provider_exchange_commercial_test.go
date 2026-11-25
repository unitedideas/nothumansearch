package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

const (
	commercialClaimID      = "4b69ca8e-d61d-47e2-91dd-fecd9f711234"
	commercialOfferID      = "a5a84ae1-62aa-4be0-91aa-1ed8a48ed321"
	commercialAcceptanceID = "552aee31-02ef-4fe2-94bd-086495341234"
	commercialCommitmentID = "7c85a0a4-f068-4f5a-a18f-135099a01234"
	commercialTicketID     = "c5a84ae1-62aa-4be0-91aa-1ed8a48ed321"
)

func decodeCommercialResponse(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %q: %v", rr.Body.String(), err)
	}
	return response
}

func TestProviderCommercialAcceptanceIsAuthenticatedAndNotProof(t *testing.T) {
	t.Parallel()
	var got models.ProviderCommercialAcceptanceInput
	h := &ProviderExchangeHandler{
		resolveProviderKey: func(_ *sql.DB, raw string) (*models.ProviderAPIKey, error) {
			if raw != "nhs_provider_test" {
				t.Fatalf("provider key = %q", raw)
			}
			return &models.ProviderAPIKey{ID: 17, ProviderClaimID: commercialClaimID}, nil
		},
		recordCommercialAcceptance: func(_ *sql.DB, key *models.ProviderAPIKey, input models.ProviderCommercialAcceptanceInput) (*models.ProviderCommercialAcceptanceEvent, bool, error) {
			if key.ID != 17 || key.ProviderClaimID != commercialClaimID {
				t.Fatalf("provider key = %#v", key)
			}
			got = input
			return &models.ProviderCommercialAcceptanceEvent{
				ID: commercialAcceptanceID, ProviderClaimID: commercialClaimID,
				ProviderAPIKeyID: 17, EventType: "pilot_company",
				ProviderAcceptanceReference: "provider-contract-1",
			}, true, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/commercial-acceptances", bytes.NewBufferString(`{
		"event_type":"pilot_company",
		"provider_acceptance_reference":"provider-contract-1"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NHS-Provider-Key", "nhs_provider_test")
	req.Header.Set("Idempotency-Key", "company-acceptance-1")
	rr := httptest.NewRecorder()
	h.ProviderCommercialAcceptances(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("acceptance status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got.EventType != "pilot_company" || got.IdempotencyKey != "company-acceptance-1" ||
		got.ProviderReference != "provider-contract-1" {
		t.Fatalf("acceptance input = %#v", got)
	}
	response := decodeCommercialResponse(t, rr)
	if response["provider_authenticated"] != true || response["owner_verification_required"] != true ||
		response["commercial_proof_created"] != false {
		t.Fatalf("acceptance evidence boundary = %#v", response)
	}
	if scope, _ := response["evidence_scope"].(string); !strings.Contains(scope, "cannot count") {
		t.Fatalf("acceptance evidence scope = %q", scope)
	}
}

func TestProviderCommercialAcceptanceFailsClosedBeforeModelWithoutKey(t *testing.T) {
	t.Parallel()
	called := false
	h := &ProviderExchangeHandler{
		recordCommercialAcceptance: func(*sql.DB, *models.ProviderAPIKey, models.ProviderCommercialAcceptanceInput) (*models.ProviderCommercialAcceptanceEvent, bool, error) {
			called = true
			return nil, false, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/commercial-acceptances", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ProviderCommercialAcceptances(rr, req)
	if rr.Code != http.StatusUnauthorized || called {
		t.Fatalf("unauthenticated status=%d called=%t body=%s", rr.Code, called, rr.Body.String())
	}
}

func TestProviderTermsAcceptanceCarriesExactOfferPrecondition(t *testing.T) {
	t.Parallel()
	exactHash := strings.Repeat("b", 64)
	called := false
	h := &ProviderExchangeHandler{
		resolveProviderKey: func(_ *sql.DB, raw string) (*models.ProviderAPIKey, error) {
			if raw != "nhs_provider_terms" {
				t.Fatalf("provider key = %q", raw)
			}
			return &models.ProviderAPIKey{ID: 23, ProviderClaimID: commercialClaimID}, nil
		},
		recordCommercialAcceptance: func(_ *sql.DB, _ *models.ProviderAPIKey, input models.ProviderCommercialAcceptanceInput) (*models.ProviderCommercialAcceptanceEvent, bool, error) {
			called = true
			if input.EventType != "terms_acceptance" || input.ProviderOfferID != commercialOfferID ||
				input.ExpectedOfferVersion != 7 || input.ExpectedExactTermsSHA256 != exactHash {
				t.Fatalf("exact terms precondition = %#v", input)
			}
			version := input.ExpectedOfferVersion
			return &models.ProviderCommercialAcceptanceEvent{
				ID: commercialAcceptanceID, ProviderClaimID: commercialClaimID,
				ProviderOfferID: commercialOfferID, OfferVersionSnapshot: &version,
				ExactTermsSHA256: exactHash, EventType: "terms_acceptance",
			}, true, nil
		},
	}
	body := `{"event_type":"terms_acceptance","offer_id":"` + commercialOfferID +
		`","offer_version":7,"exact_terms_sha256":"` + exactHash +
		`","provider_acceptance_reference":"provider-terms-7"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/commercial-acceptances", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NHS-Provider-Key", "nhs_provider_terms")
	req.Header.Set("Idempotency-Key", "terms-acceptance-7")
	rr := httptest.NewRecorder()
	h.ProviderCommercialAcceptances(rr, req)
	if rr.Code != http.StatusCreated || !called {
		t.Fatalf("terms acceptance status=%d called=%t body=%s", rr.Code, called, rr.Body.String())
	}
}

func TestAdminCommercialCompanyVerificationRequiresProviderAcceptance(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	var gotAcceptanceID, gotCompanyHash, gotOperator, gotEvidence string
	h := &ProviderExchangeHandler{
		verifyPilotCompany: func(_ *sql.DB, acceptanceID, companyHash, operator, evidence string) (*models.ProviderPilotCompany, bool, error) {
			gotAcceptanceID, gotCompanyHash, gotOperator, gotEvidence = acceptanceID, companyHash, operator, evidence
			return &models.ProviderPilotCompany{
				ID: commercialCommitmentID, ProviderClaimID: commercialClaimID,
				ProviderAcceptanceEventID: commercialAcceptanceID,
			}, true, nil
		},
	}
	body := `{
		"action":"verify_company",
		"provider_acceptance_event_id":"` + commercialAcceptanceID + `",
		"company_key_hash":"` + strings.Repeat("a", 64) + `",
		"operator_reference":"operator-case-1",
		"identity_evidence_reference":"identity-evidence-1"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/provider-commercial/action", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-key")
	rr := httptest.NewRecorder()
	h.AdminCommercialAction(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("company verification status=%d body=%s", rr.Code, rr.Body.String())
	}
	if gotAcceptanceID != commercialAcceptanceID || gotCompanyHash != strings.Repeat("a", 64) ||
		gotOperator != "operator-case-1" || gotEvidence != "identity-evidence-1" {
		t.Fatalf("company verification args=%q %q %q %q", gotAcceptanceID, gotCompanyHash, gotOperator, gotEvidence)
	}
	response := decodeCommercialResponse(t, rr)
	if response["commercial_evidence_recorded"] != true || response["pilot_threshold_evaluated"] != false {
		t.Fatalf("company verification response = %#v", response)
	}
	if _, exposed := response["company_key_hash"]; exposed ||
		strings.Contains(rr.Body.String(), strings.Repeat("a", 64)) ||
		strings.Contains(rr.Body.String(), "Private Company Name") {
		t.Fatalf("company verification response exposed identity material: %s", rr.Body.String())
	}
}

func TestAdminCommercialActionsRouteExactTypedEvidence(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	effectiveAt := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	called := false
	h := &ProviderExchangeHandler{
		recordVerifiedTerms: func(_ *sql.DB, input models.VerifiedProviderTermsInput) (*models.ProviderCommercialCommitmentEvent, bool, error) {
			called = true
			if input.ProviderOfferID != commercialOfferID || input.ProviderAcceptanceEventID != commercialAcceptanceID ||
				input.RelatedCommitmentEventID != commercialCommitmentID || input.SourceSystem != "contract-store" ||
				!input.SourceEffectiveAt.Equal(effectiveAt) {
				t.Fatalf("verified terms input = %#v", input)
			}
			return &models.ProviderCommercialCommitmentEvent{ID: commercialCommitmentID, EventType: "terms_renewal"}, false, nil
		},
	}
	body := `{"action":"verify_terms","offer_id":"` + commercialOfferID + `","provider_acceptance_event_id":"` + commercialAcceptanceID + `","related_commitment_event_id":"` + commercialCommitmentID + `","source_system":"contract-store","source_event_id":"terms-event-1","source_effective_at":"` + effectiveAt.Format(time.RFC3339) + `","operator_reference":"operator-case-1","owner_evidence_reference":"owner-terms-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/provider-commercial/action", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-key")
	rr := httptest.NewRecorder()
	h.AdminCommercialAction(rr, req)
	if rr.Code != http.StatusOK || !called {
		t.Fatalf("verified terms status=%d called=%t body=%s", rr.Code, called, rr.Body.String())
	}
	response := decodeCommercialResponse(t, rr)
	if response["commitment"] == nil || response["commercial_evidence_recorded"] != true || response["pilot_threshold_evaluated"] != false {
		t.Fatalf("verified terms response = %#v", response)
	}
}

func TestPrepaidCommercialAdminActionsFailClosedInTermsOnlyPilot(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	effectiveAt := time.Date(2026, time.August, 2, 12, 0, 0, 0, time.UTC)
	for name, body := range map[string]string{
		"verify funding":  `{"action":"verify_funding","offer_id":"` + commercialOfferID + `","amount_cents":5000,"currency":"usd","source_system":"stripe-pilot","source_event_id":"settlement-event-1","source_effective_at":"` + effectiveAt.Format(time.RFC3339) + `","qualifying_action_ticket_id":"` + commercialTicketID + `","operator_reference":"operator-case-1","owner_evidence_reference":"owner-receipt-1"}`,
		"reverse funding": `{"action":"reverse_funding","related_commitment_event_id":"` + commercialCommitmentID + `","amount_cents":2500,"source_system":"stripe-pilot","source_event_id":"reversal-event-1","source_effective_at":"` + effectiveAt.Format(time.RFC3339) + `","operator_reference":"operator-case-1","owner_evidence_reference":"owner-reversal-1"}`,
	} {
		t.Run(name, func(t *testing.T) {
			called := false
			h := &ProviderExchangeHandler{
				recordVerifiedFunding: func(*sql.DB, models.VerifiedProviderFundingInput) (*models.ProviderCommercialCommitmentEvent, bool, error) {
					called = true
					return nil, false, nil
				},
				reverseVerifiedFunding: func(*sql.DB, models.ProviderFundingReversalInput) (*models.ProviderCommercialCommitmentEvent, bool, error) {
					called = true
					return nil, false, nil
				},
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/provider-commercial/action", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-admin-key")
			rr := httptest.NewRecorder()
			h.AdminCommercialAction(rr, req)
			if rr.Code != http.StatusConflict || called || !strings.Contains(rr.Body.String(), "terms-only") {
				t.Fatalf("%s status=%d called=%t body=%s", name, rr.Code, called, rr.Body.String())
			}
		})
	}
}

func TestAdminCommercialActionRejectsIrrelevantAndIdentityFields(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	for name, body := range map[string]string{
		"unrelated action field": `{"action":"verify_company","provider_acceptance_event_id":"` + commercialAcceptanceID + `","company_key_hash":"` + strings.Repeat("a", 64) + `","amount_cents":1,"operator_reference":"operator-case-1","identity_evidence_reference":"identity-evidence-1"}`,
		"raw company identity":   `{"action":"verify_company","provider_acceptance_event_id":"` + commercialAcceptanceID + `","company_key_hash":"` + strings.Repeat("a", 64) + `","company_name":"Private Company Name","operator_reference":"operator-case-1","identity_evidence_reference":"identity-evidence-1"}`,
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/provider-commercial/action", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-admin-key")
			rr := httptest.NewRecorder()
			(&ProviderExchangeHandler{}).AdminCommercialAction(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("%s status=%d body=%s", name, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestCommercialEvidenceRequiredMapsToStateConflict(t *testing.T) {
	t.Parallel()
	status, message := providerExchangeStatus(models.ErrProviderCommercialEvidenceRequired)
	if status != http.StatusConflict || !strings.Contains(message, "provider-authenticated") ||
		!strings.Contains(message, "owner-verified") {
		t.Fatalf("commercial evidence mapping = %d %q", status, message)
	}
}

func TestProviderCommercialRoutesAndLegacyProgressScopeRemainSeparate(t *testing.T) {
	t.Parallel()
	serverSource, err := os.ReadFile(filepath.Join("..", "..", "cmd", "server", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{
		`/api/v1/provider/commercial-acceptances`,
		`/api/v1/admin/provider-commercial/action`,
	} {
		if !strings.Contains(string(serverSource), route) {
			t.Fatalf("server route missing %q", route)
		}
	}
	legacySource, err := os.ReadFile("provider_exchange_admin.go")
	if err != nil {
		t.Fatal(err)
	}
	legacy := string(legacySource)
	for _, required := range []string{
		"prepaid and legacy funding writes are disabled for the terms-only pilot",
		"operator references alone cannot count as proof",
		`"verified_provider_companies"`,
		`"verified_provider_renewals"`,
		"legacy fields remain diagnostic observations",
	} {
		if !strings.Contains(legacy, required) {
			t.Fatalf("legacy progress boundary missing %q", required)
		}
	}
}

func TestAdminCommercialActionReturnsTruthfulModelFailure(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	h := &ProviderExchangeHandler{
		verifyPilotCompany: func(*sql.DB, string, string, string, string) (*models.ProviderPilotCompany, bool, error) {
			return nil, false, models.ErrProviderCommercialEvidenceRequired
		},
	}
	body := `{"action":"verify_company","provider_acceptance_event_id":"` + commercialAcceptanceID + `","company_key_hash":"` + strings.Repeat("a", 64) + `","operator_reference":"operator-case-1","identity_evidence_reference":"identity-evidence-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/provider-commercial/action", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-key")
	rr := httptest.NewRecorder()
	h.AdminCommercialAction(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("model failure status=%d body=%s", rr.Code, rr.Body.String())
	}
	response := decodeCommercialResponse(t, rr)
	if response["commercial_proof_created"] != false {
		t.Fatalf("model failure response = %#v", response)
	}
}
