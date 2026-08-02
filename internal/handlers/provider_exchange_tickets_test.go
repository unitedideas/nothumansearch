package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

func TestActionTicketRejectsUnknownContactFieldBeforeDatabase(t *testing.T) {
	t.Parallel()
	signer, err := providerexchange.NewSigner("test-admin-key-0123456789abcdef012345")
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{
		"offer_id":"4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		"search_id":"nhs_sr_example",
		"demand_topic":"developer-tools",
		"principal_consent":true,
		"consent_version":"nhs-principal-consent-v1",
		"contact":"private@example.com"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/action-tickets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	(&ProviderExchangeHandler{
		Signer:      signer,
		ticketLimit: newMCPDiscoveryRateLimiter(120, time.Hour),
	}).ActionTickets(rr, req)
	if rr.Code != http.StatusBadRequest || !bytes.Contains(rr.Body.Bytes(), []byte("contact details")) {
		t.Fatalf("ticket response status=%d body=%s", rr.Code, rr.Body.String())
	}
	for name, want := range map[string]string{
		"Cache-Control":   "private, no-store",
		"Pragma":          "no-cache",
		"Referrer-Policy": "no-referrer",
	} {
		if got := rr.Header().Get(name); got != want {
			t.Fatalf("ticket response %s = %q, want %q", name, got, want)
		}
	}
}

func TestPrepareProviderActionMCPUsesDedicatedTicketLimit(t *testing.T) {
	t.Parallel()
	signer, err := providerexchange.NewSigner("test-admin-key-0123456789abcdef012345")
	if err != nil {
		t.Fatal(err)
	}
	provider := &ProviderExchangeHandler{
		Signer:      signer,
		ticketLimit: newMCPDiscoveryRateLimiter(1, time.Hour),
	}
	h := &MCPHandler{ProviderExchange: provider}
	args := map[string]any{
		"offer_id":          "4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		"search_id":         "nhs_sr_example",
		"demand_topic":      "developer-tools",
		"principal_consent": true,
		"consent_version":   "nhs-principal-consent-v1",
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.RemoteAddr = "203.0.113.9:4321"

	first := httptest.NewRecorder()
	h.toolPrepareProviderAction(first, json.RawMessage(`1`), args, req)
	if strings.Contains(first.Body.String(), "safety limit exceeded") {
		t.Fatalf("first MCP action attempt was unexpectedly limited: %s", first.Body.String())
	}

	second := httptest.NewRecorder()
	h.toolPrepareProviderAction(second, json.RawMessage(`2`), args, req)
	if !strings.Contains(second.Body.String(), "action ticket safety limit exceeded") {
		t.Fatalf("second MCP action attempt bypassed dedicated limit: %s", second.Body.String())
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("MCP action ticket limit omitted Retry-After")
	}
}

func TestPrepareProviderActionMCPArgumentsRejectUnknownAndWrongTypes(t *testing.T) {
	t.Parallel()
	valid := map[string]any{
		"offer_id":          "4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		"search_id":         "nhs_sr_example",
		"demand_topic":      "developer-tools",
		"requirement_flags": []any{"mcp", "api_access"},
		"principal_consent": true,
		"consent_version":   "nhs-principal-consent-v1",
	}
	if err := validatePrepareProviderActionArguments(valid); err != nil {
		t.Fatalf("valid MCP provider action arguments rejected: %v", err)
	}

	withContact := map[string]any{}
	for key, value := range valid {
		withContact[key] = value
	}
	withContact["contact"] = "private@example.com"
	if err := validatePrepareProviderActionArguments(withContact); err == nil {
		t.Fatal("unknown contact field was accepted")
	}

	wrongFlags := map[string]any{}
	for key, value := range valid {
		wrongFlags[key] = value
	}
	wrongFlags["requirement_flags"] = "mcp"
	if err := validatePrepareProviderActionArguments(wrongFlags); err == nil {
		t.Fatal("string requirement_flags was silently accepted")
	}
}
