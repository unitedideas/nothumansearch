package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIIndexPublishesNHSProviderExchangeWithoutChangingDiscoveryAuth(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	NewAPIHandler(nil).Index(rr, httptest.NewRequest(http.MethodGet, "/api/v1", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("API index status = %d; body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Description string            `json:"description"`
		Endpoints   map[string]string `json:"endpoints"`
		Auth        string            `json:"auth"`
		Exchange    map[string]any    `json:"provider_exchange"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"provider_claims", "provider_offers", "action_tickets", "provider_outcomes", "receipt_verify"} {
		if payload.Endpoints[key] == "" {
			t.Fatalf("API index missing %q", key)
		}
	}
	if !strings.Contains(payload.Auth, "none for discovery") || payload.Exchange["organic_rank_sold"] != false || payload.Exchange["raw_queries_sold"] != false || payload.Exchange["agent_identities_sold"] != false {
		t.Fatalf("API index boundary is incomplete: %#v", payload)
	}
	if strings.Contains(strings.ToLower(rr.Body.String()), "aidev") {
		t.Fatalf("NHS API index contains AI Dev Board scope: %s", rr.Body.String())
	}
}
