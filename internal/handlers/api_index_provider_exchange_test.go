package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestProviderExchangeOfferSidecarsDefaultOnAndCanFailClosed(t *testing.T) {
	api := NewAPIHandler(nil)
	mcp := NewMCPHandler(nil, "https://nothumansearch.ai")
	if !api.ProviderExchangeEnabled || !mcp.ProviderExchangeEnabled {
		t.Fatal("provider exchange offer sidecars must default on for compatibility")
	}
	api.ProviderExchangeEnabled = false
	mcp.ProviderExchangeEnabled = false
	if api.ProviderExchangeEnabled || mcp.ProviderExchangeEnabled {
		t.Fatal("provider exchange offer sidecars did not enter fail-closed mode")
	}
}

func TestAPIIndexPublishesNHSProviderExchangeWithoutChangingDiscoveryAuth(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	NewAPIHandler(nil).Index(rr, httptest.NewRequest(http.MethodGet, "/api/v1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("API index status = %d; body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Version       string            `json:"version"`
		VersionPolicy string            `json:"version_policy"`
		Description   string            `json:"description"`
		Endpoints     map[string]string `json:"endpoints"`
		Auth          string            `json:"auth"`
		Exchange      map[string]any    `json:"provider_exchange"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"provider_claims",
		"provider_offers",
		"provider_commercial_acceptances",
		"provider_pilot_status",
		"provider_demand",
		"provider_controlled_intent",
		"action_tickets",
		"action_ticket_handoff",
		"provider_outcomes",
		"receipt_verify",
	} {
		if payload.Endpoints[key] == "" {
			t.Fatalf("API index missing %q", key)
		}
	}
	if !strings.Contains(payload.Endpoints["action_tickets"], "not the provider URL") ||
		!strings.Contains(payload.Endpoints["action_ticket_handoff"], "durable privacy-safe handoff receipt") ||
		!strings.Contains(payload.Endpoints["action_ticket_handoff"], "nhs-provider-handoff-consent-v1") ||
		!strings.Contains(payload.Endpoints["provider_commercial_acceptances"], "provider-authenticated acceptance only") ||
		!strings.Contains(payload.Endpoints["provider_pilot_status"], "claim-scoped") ||
		!strings.Contains(payload.Endpoints["provider_demand"], "authenticated claim domain only") ||
		!strings.Contains(payload.Endpoints["provider_demand"], "privacy-thresholded") ||
		!strings.Contains(payload.Endpoints["provider_controlled_intent"], "optional separately consented") ||
		!strings.Contains(payload.Endpoints["provider_controlled_intent"], "no query, identity, contact, charge, or proof") {
		t.Fatalf("API index exchange workflow is incomplete: %#v", payload.Endpoints)
	}
	if !strings.Contains(payload.Auth, "none for discovery") ||
		!strings.Contains(payload.Auth, "claim-scoped status and privacy-thresholded demand reports") ||
		payload.Exchange["organic_rank_sold"] != false || payload.Exchange["raw_queries_sold"] != false || payload.Exchange["agent_identities_sold"] != false {
		t.Fatalf("API index boundary is incomplete: %#v", payload)
	}
	if payload.Exchange["handoff_contract_version"] != "nhs-action-handoff-v1" ||
		payload.Exchange["handoff_consent_version"] != "nhs-provider-handoff-consent-v1" ||
		payload.Exchange["ticket_preparation_contract"] != "nhs-action-ticket-preparation-v2" ||
		payload.Exchange["ticket_or_handoff_charge"] != false ||
		payload.Exchange["controlled_intent_disclosure_optional"] != true ||
		payload.Exchange["controlled_intent_disclosure_default"] != false ||
		payload.Exchange["controlled_intent_disclosure_consent_version"] != "nhs-provider-controlled-intent-disclosure-consent-v1" ||
		payload.Exchange["controlled_intent_resolver_contract"] != "nhs-provider-controlled-intent-resolver-v1" ||
		payload.Exchange["controlled_intent_resolver_charge"] != false ||
		payload.Exchange["controlled_intent_resolver_creates_proof"] != false ||
		payload.Exchange["provider_status_endpoint"] != "GET /api/v1/provider/pilot-status" ||
		payload.Exchange["provider_status_scope"] != "authenticated_claim_only" ||
		payload.Exchange["provider_demand_endpoint"] != "GET /api/v1/provider/demand?days=" ||
		payload.Exchange["provider_demand_scope"] != "authenticated_claim_domain_only" ||
		payload.Exchange["provider_demand_privacy_threshold_receipts"] != float64(models.ProviderDemandPrivacyThreshold) ||
		payload.Exchange["provider_demand_returns_individual_receipts"] != false ||
		payload.Exchange["provider_read_rate_limit"] != "240/hour/provider key per read surface" {
		t.Fatalf("API index handoff contract is incomplete: %#v", payload.Exchange)
	}
	if payload.Version != "1.1.0" ||
		!strings.Contains(payload.VersionPolicy, "not a semantic-compatibility promise") {
		t.Fatalf("API index version/cutover policy is incomplete: %#v", payload)
	}
	if strings.Contains(strings.ToLower(rr.Body.String()), "aidev") {
		t.Fatalf("NHS API index contains AI Dev Board scope: %s", rr.Body.String())
	}
}
