package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
