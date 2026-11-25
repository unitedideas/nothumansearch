package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

// The legacy quota helper still fails closed without a key. Public MCP
// discovery no longer routes through this helper.
func TestLegacyConsumeMCPNoKey(t *testing.T) {
	g := NewUsageGate(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	_, _, _, _, err := g.ConsumeMCP(req, "search_agents")
	if err == nil || err.Error() != "key_required" {
		t.Fatalf("ConsumeMCP with no key err = %v, want key_required", err)
	}
}

func TestLegacySearchEntitledAnonymous(t *testing.T) {
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

func TestSubscribePagePreservesCampaignAttributionInCheckoutPayload(t *testing.T) {
	a := NewAuthService(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodGet, "/subscribe?email=buyer@example.com&qc=campaign-123&utm_source=linkedin&utm_medium=qlimit&utm_campaign=campaign-123", nil)
	rr := httptest.NewRecorder()

	a.SubscribePage(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("subscribe page status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		`value="buyer@example.com"`,
		`id="qc" value="campaign-123"`,
		`id="utm_source" value="linkedin"`,
		`id="utm_medium" value="qlimit"`,
		`"qc":document.getElementById("qc").value`,
		`"utm_campaign":document.getElementById("utm_campaign").value`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("subscribe page missing %q: %s", want, body)
		}
	}
}
