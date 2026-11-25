package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func providerStatusTestHandler() *ProviderExchangeHandler {
	return &ProviderExchangeHandler{
		resolveProviderKey: func(_ *sql.DB, raw string) (*models.ProviderAPIKey, error) {
			if raw != "test-provider-key" {
				return nil, sql.ErrNoRows
			}
			return &models.ProviderAPIKey{
				ID:              7,
				ProviderClaimID: "4b69ca8e-d61d-47e2-91dd-fecd9f711234",
			}, nil
		},
	}
}

func TestProviderReadEndpointsRequireGETAndProviderKey(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		path string
		call func(*ProviderExchangeHandler, http.ResponseWriter, *http.Request)
	}{
		{"status", "/api/v1/provider/pilot-status", (*ProviderExchangeHandler).ProviderPilotStatus},
		{"demand", "/api/v1/provider/demand", (*ProviderExchangeHandler).ProviderDemand},
	} {
		t.Run(test.name+"_method", func(t *testing.T) {
			rr := httptest.NewRecorder()
			test.call(providerStatusTestHandler(), rr, httptest.NewRequest(http.MethodPost, test.path, nil))
			if rr.Code != http.StatusMethodNotAllowed || rr.Header().Get("Allow") != "GET" {
				t.Fatalf("status=%d allow=%q body=%s", rr.Code, rr.Header().Get("Allow"), rr.Body.String())
			}
		})
		t.Run(test.name+"_auth", func(t *testing.T) {
			rr := httptest.NewRecorder()
			test.call(providerStatusTestHandler(), rr, httptest.NewRequest(http.MethodGet, test.path, nil))
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestProviderReadParametersFailClosedBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		path string
		call func(*ProviderExchangeHandler, http.ResponseWriter, *http.Request)
	}{
		{"/api/v1/provider/pilot-status?limit=101", (*ProviderExchangeHandler).ProviderPilotStatus},
		{"/api/v1/provider/demand?days=0", (*ProviderExchangeHandler).ProviderDemand},
		{"/api/v1/provider/demand?days=thirty", (*ProviderExchangeHandler).ProviderDemand},
	} {
		req := httptest.NewRequest(http.MethodGet, test.path, nil)
		req.Header.Set("X-NHS-Provider-Key", "test-provider-key")
		rr := httptest.NewRecorder()
		test.call(providerStatusTestHandler(), rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d body=%s", test.path, rr.Code, rr.Body.String())
		}
	}
}

func TestProviderDemandDerivesClaimAndNeverAcceptsDomainParameter(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("provider_exchange_status.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "func (h *ProviderExchangeHandler) ProviderDemand")
	end := strings.Index(text[start:], "func (h *ProviderExchangeHandler) AdminProviderPilotQueue")
	if start < 0 || end < 1 {
		t.Fatal("could not isolate ProviderDemand")
	}
	body := text[start : start+end]
	if !strings.Contains(body, "GetProviderDemandAnalyticsForClaim(h.DB, key.ProviderClaimID, days)") {
		t.Fatal("provider demand is not derived from the authenticated claim")
	}
	for _, forbidden := range []string{`Query().Get("domain")`, `FormValue("domain")`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("provider demand accepts caller-selected domain via %s", forbidden)
		}
	}
}

func TestAdminProviderPilotQueueRequiresAdminAndValidState(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	unauthorized := httptest.NewRecorder()
	(&ProviderExchangeHandler{}).AdminProviderPilotQueue(
		unauthorized,
		httptest.NewRequest(http.MethodGet, "/api/v1/admin/provider-pilot-queue", nil),
	)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/provider-pilot-queue?state=raw_queries", nil)
	req.Header.Set("Authorization", "Bearer test-admin-key")
	rr := httptest.NewRecorder()
	(&ProviderExchangeHandler{}).AdminProviderPilotQueue(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid state status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestProviderPilotStatusJSONContractExcludesSensitiveMaterial(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(&models.ProviderPilotStatus{
		ProviderClaimID:        "4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		Offers:                 []models.ProviderPilotOfferStatus{},
		RecentObservedHandoffs: []models.ProviderPilotRecentEvent{},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, forbidden := range []string{
		"attribution_token", "token_hash", "action_url", "search_receipt",
		"controlled_intent", "raw_query", "company_key_hash", "email", "ip_address",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("provider status JSON exposed forbidden field %q: %s", forbidden, text)
		}
	}
}
