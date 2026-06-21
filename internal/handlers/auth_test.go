package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

// The MCP endpoint is a hard wall: a tools/call with no API key must be rejected
// with key_required, before any DB access (so this is safe with a nil DB).
func TestConsumeMCPHardWallNoKey(t *testing.T) {
	g := NewUsageGate(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	_, _, _, _, err := g.ConsumeMCP(req, "search_agents")
	if err == nil || err.Error() != "key_required" {
		t.Fatalf("ConsumeMCP with no key err = %v, want key_required", err)
	}
}

// An unauthenticated request (no session, no API key) is not entitled to full
// search results.
func TestSearchEntitledAnonymous(t *testing.T) {
	a := NewAuthService(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=payments", nil)
	entitled, via, key := a.SearchEntitled(req)
	if entitled {
		t.Fatalf("anonymous SearchEntitled = true (via=%q), want false", via)
	}
	if key != nil {
		t.Fatalf("anonymous SearchEntitled returned a key")
	}
}

// The single public plan is $9.99/mo with a 50k soft cap, regardless of the name
// requested (legacy starter/pro/scale all collapse to it).
func TestAPIPlanForUnlimited(t *testing.T) {
	for _, name := range []string{"", "unlimited", "starter", "pro", "scale", "bogus"} {
		p := models.APIPlanFor(name)
		if p.PriceCents != 999 || p.MonthlyLimit != 50000 || p.Name != "unlimited" {
			t.Fatalf("APIPlanFor(%q) = %+v, want {unlimited 50000 999}", name, p)
		}
	}
	if got := models.APIPlans(); len(got) != 1 {
		t.Fatalf("APIPlans() len = %d, want 1", len(got))
	}
}
