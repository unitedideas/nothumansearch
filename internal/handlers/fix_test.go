package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestScoreFixEligibleRequiresHardSignal(t *testing.T) {
	if scoreFixEligible(nil) {
		t.Fatal("nil site should not be score-fix eligible")
	}
	if scoreFixEligible(&models.Site{HasLLMsTxt: true, HasRobotsAI: true, HasSchemaOrg: true}) {
		t.Fatal("passive-only site should not be score-fix eligible")
	}
	if !scoreFixEligible(&models.Site{HasOpenAPI: true}) {
		t.Fatal("hard-signal site should be score-fix eligible")
	}
	if scoreFixEligible(&models.Site{AgenticScore: fixTargetScore, HasOpenAPI: true}) {
		t.Fatal("site already at the target score should not be score-fix eligible")
	}
}

func TestCommerceCatalogIncludesAPIPlans(t *testing.T) {
	h := NewFixHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
	rr := httptest.NewRecorder()

	h.CommerceCatalog(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("catalog status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Products []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Plan string `json:"plan"`
		} `json:"products"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	got := map[string]bool{}
	for _, product := range payload.Products {
		got[product.ID] = true
	}
	for _, id := range []string{"nhs_geo_fix_my_score", "nhs_api_unlimited"} {
		if !got[id] {
			t.Fatalf("catalog missing product %q; got=%v", id, got)
		}
	}
}

func TestCommerceQuoteSupportsAPIPlans(t *testing.T) {
	h := NewFixHandler(nil, "https://nothumansearch.ai")
	body := bytes.NewBufferString(`{"product_id":"nhs_api_unlimited"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/quote", body)
	rr := httptest.NewRecorder()

	h.CommerceQuote(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("quote status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		ProductID          string `json:"product_id"`
		Plan               string `json:"plan"`
		Amount             int64  `json:"amount"`
		MonthlyLimit       int    `json:"monthly_limit"`
		CheckoutEndpoint   string `json:"checkout_endpoint"`
		ActivationEndpoint string `json:"activation_endpoint"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode quote: %v", err)
	}
	if payload.ProductID != "nhs_api_unlimited" || payload.Plan != "unlimited" {
		t.Fatalf("quote product/plan = %q/%q, want nhs_api_unlimited/unlimited", payload.ProductID, payload.Plan)
	}
	if payload.Amount != 999 || payload.MonthlyLimit != 50000 {
		t.Fatalf("quote amount/limit = %d/%d, want 999/50000", payload.Amount, payload.MonthlyLimit)
	}
	if payload.CheckoutEndpoint != "https://nothumansearch.ai/api/v1/api-keys/subscribe" {
		t.Fatalf("checkout endpoint = %q", payload.CheckoutEndpoint)
	}
	if payload.ActivationEndpoint == "" {
		t.Fatalf("activation endpoint missing")
	}
}

func TestGeoFixAdminActionRequiresBearerAuth(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	h := NewFixHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/geo-jobs/action", bytes.NewBufferString(`{"id":1}`))
	rr := httptest.NewRecorder()

	h.AdminAction(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("AdminAction status = %d, want 401; body=%s", rr.Code, rr.Body.String())
	}
}

func TestGeoFixAdminActionRejectsInvalidActionBeforeDB(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	h := NewFixHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/geo-jobs/action", bytes.NewBufferString(`{
		"id": 1,
		"action": "send_followup",
		"operator": "business-agent-not-human-search"
	}`))
	req.Header.Set("Authorization", "Bearer test-admin-key")
	rr := httptest.NewRecorder()

	h.AdminAction(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("AdminAction status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
}

func TestGeoFixAdminActionRequiresOperatorBeforeDB(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	h := NewFixHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/geo-jobs/action", bytes.NewBufferString(`{
		"id": 1,
		"action": "mark_internal_test"
	}`))
	req.Header.Set("Authorization", "Bearer test-admin-key")
	rr := httptest.NewRecorder()

	h.AdminAction(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("AdminAction status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
}

func TestGeoFixAdminActionRequiresConfiguredAdminKey(t *testing.T) {
	os.Unsetenv("ADMIN_API_KEY")
	h := NewFixHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/geo-jobs/action", bytes.NewBufferString(`{"id":1}`))
	rr := httptest.NewRecorder()

	h.AdminAction(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("AdminAction status = %d, want 503; body=%s", rr.Code, rr.Body.String())
	}
}

func TestFixPreviewBlock(t *testing.T) {
	// A site missing llms.txt + robots.txt but having OpenAPI: the missing ones must
	// show their point values + a domain-templated snippet; the present one must show
	// a green "already present" row and NOT a snippet.
	site := &models.Site{
		Domain:       "example.com",
		AgenticScore: 20,
		HasOpenAPI:   true, // present
		// HasLLMsTxt, HasRobotsAI, etc. false → missing
	}
	out := fixPreviewBlock(site)

	if !strings.Contains(out, "+25") {
		t.Errorf("expected llms.txt point value +25 for a missing signal; got:\n%s", out)
	}
	if !strings.Contains(out, "/llms.txt for example.com") {
		t.Errorf("expected a domain-templated llms.txt snippet for example.com")
	}
	if !strings.Contains(out, "already present") {
		t.Errorf("expected a green 'already present' row for the present OpenAPI signal")
	}
	// Projected score = 20 + (all missing weights). Must mention the current score anchor.
	if !strings.Contains(out, "20") {
		t.Errorf("expected current score 20 anchored in the projection")
	}
	// A fully-loaded site gains nothing and projects to its current score.
	full := &models.Site{
		Domain: "done.com", AgenticScore: 100,
		HasLLMsTxt: true, HasAIPlugin: true, HasOpenAPI: true, HasStructuredAPI: true,
		HasMCPServer: true, HasRobotsAI: true, HasSchemaOrg: true,
	}
	full_out := fixPreviewBlock(full)
	if strings.Contains(full_out, "+25") || strings.Contains(full_out, "+20") {
		t.Errorf("a fully-loaded site should show no point gains; got:\n%s", full_out)
	}
}
