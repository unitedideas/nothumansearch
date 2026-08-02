package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("OpenAPI priority/free contract missing %q", required)
		}
	}
	for _, forbidden := range []string{"Free search abuse limit exceeded", "nhs_api_starter", "nhs_api_pro", "nhs_api_scale"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("OpenAPI retained obsolete contract %q", forbidden)
		}
	}
}
