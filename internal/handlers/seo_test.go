package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestMCPManifestAdvertisesCanonicalMCPTools(t *testing.T) {
	seo := NewSEOHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodGet, "/.well-known/mcp.json", nil)
	rr := httptest.NewRecorder()

	seo.MCPManifest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("MCPManifest status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode MCP manifest: %v", err)
	}

	got := make([]string, 0, len(payload.Tools))
	for _, tool := range payload.Tools {
		got = append(got, tool.Name)
	}
	want := NewMCPHandler(nil, "https://nothumansearch.ai").toolNames()

	gotSet := map[string]bool{}
	for _, name := range got {
		gotSet[name] = true
	}
	if len(gotSet) != len(want) {
		t.Fatalf("manifest tools = %d unique, want %d: got=%v want=%v", len(gotSet), len(want), got, want)
	}
	for _, name := range want {
		if !gotSet[name] {
			t.Fatalf("manifest missing canonical tool %q; got=%v", name, got)
		}
	}
}

func TestSearchCategoryVocabularyStaysConsistent(t *testing.T) {
	if len(publicSearchCategories) != 12 {
		t.Fatalf("public categories = %d, want 12: %v", len(publicSearchCategories), publicSearchCategories)
	}
	public := publicSearchCategoryCSV()
	for _, want := range []string{"ai-tools", "developer", "finance", "ecommerce", "security", "news"} {
		if !strings.Contains(public, want) {
			t.Fatalf("public category list missing %q: %s", want, public)
		}
	}
	desc := searchCategoryDescription()
	for _, want := range []string{"other", "spam", "not promoted"} {
		if !strings.Contains(desc, want) {
			t.Fatalf("category description missing %q: %s", want, desc)
		}
	}

	seo := NewSEOHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rr := httptest.NewRecorder()
	seo.OpenAPISpec(rr, req)

	body := rr.Body.String()
	for _, want := range []string{"news", "other", "spam", "Do not treat audit-only buckets as promoted discovery inventory"} {
		if !strings.Contains(body, want) {
			t.Fatalf("OpenAPI category copy missing %q", want)
		}
	}
}

func TestLLMsTxtCategoryCopyUsesExpectedArguments(t *testing.T) {
	seo := NewSEOHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodGet, "/llms.txt", nil)
	rr := httptest.NewRecorder()
	seo.LLMsTxt(rr, req)

	body := rr.Body.String()
	for _, want := range []string{
		"Base URL: https://nothumansearch.ai/api/v1",
		"Tools (13):",
		"record_action_interest",
		"It does not contact the",
		"Public categories: ai-tools, developer, data, finance, ecommerce, jobs, security, health, education, communication, productivity, news.",
		"Live scorer: https://nothumansearch.ai/score",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("llms.txt missing %q; body=%s", want, body)
		}
	}
}

func TestOpenAPIDescribesFreeFallbackAndOptionalPriorityThroughput(t *testing.T) {
	seo := NewSEOHandler(nil, "https://nothumansearch.ai")
	rr := httptest.NewRecorder()
	seo.OpenAPISpec(rr, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	body := rr.Body.String()
	if strings.Contains(body, "\t") {
		t.Fatal("OpenAPI YAML contains tab indentation")
	}
	for _, required := range []string{
		"receipt_recorded",
		"free access resumes after the reset",
		"optional priority key raises this to 100/hour",
		"enum: [nhs_geo_fix_my_score, nhs_api_unlimited]",
		"enum: [unlimited]",
		"is_featured:",
		"deprecated: true",
		"never affects organic score or ordering",
		"paid_offers_available",
		"#/components/schemas/PublicProviderOffer",
		"/provider/claims:",
		"/provider/offers:",
		"/action-interests:",
		"/action-tickets:",
		"/provider/outcomes:",
		"/action-receipts/verify:",
		"signature_valid",
		"within_validity_window",
		"nhs-principal-consent-v1",
		"nhs-action-interest-v1",
		"ActionInterestRequest:",
		"ActionInterestResponse:",
		"It expires with the source search, no later than 30 days after that search.",
		"caller_attests_principal_interest",
		`schema: { $ref: "#/components/schemas/ActionInterestResponse" }`,
		"commercial proof",
		"organic_rank_paid",
		"Update an owned draft offer",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("OpenAPI priority/free contract missing %q", required)
		}
	}
	for _, forbidden := range []string{"Free search abuse limit exceeded", "nhs_api_starter", "nhs_api_pro", "nhs_api_scale", "draft or paused"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("OpenAPI retained obsolete contract %q", forbidden)
		}
	}
}

func TestOpenAPIDescribesPersistentDNSOwnershipFreshness(t *testing.T) {
	t.Parallel()
	seo := NewSEOHandler(nil, "https://nothumansearch.ai")
	rr := httptest.NewRecorder()
	seo.OpenAPISpec(rr, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	body := rr.Body.String()

	for _, required := range []string{
		"ProviderClaim:",
		"OwnershipFreshness:",
		"ProviderClaimChallengeResponse:",
		"ProviderClaimVerifyResponse:",
		"record_must_remain_published:",
		"stored_challenge_material:",
		"raw_dns_answers_retained:",
		"verification_last_succeeded_at:",
		"verification_next_check_at:",
		fmt.Sprintf("recheck_interval_seconds: { type: integer, enum: [%d]", int64(models.ProviderClaimDNSRecheckInterval/time.Second)),
		fmt.Sprintf("paid_actions_stop_after_consecutive_failures: { type: integer, enum: [%d]", models.ProviderClaimDNSFailureLimit),
		fmt.Sprintf("paid_actions_stop_when_last_success_age_reaches_seconds: { type: integer, enum: [%d]", int64(models.ProviderClaimVerificationFreshness/time.Second)),
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("OpenAPI persistent DNS contract missing %q", required)
		}
	}
	if strings.Contains(body, "%!") {
		t.Fatalf("OpenAPI contains a formatting error: %s", body)
	}
}
