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
