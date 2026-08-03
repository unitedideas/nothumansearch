package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

func TestPreparedActionTicketDoesNotExposeProviderURL(t *testing.T) {
	t.Parallel()
	encoded, err := json.Marshal(models.ActionTicket{
		ID:                "4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		ActionURLSnapshot: "https://provider.example/private/action",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "action_url") || strings.Contains(string(encoded), "provider.example") {
		t.Fatalf("pre-handoff ticket leaked provider action URL: %s", encoded)
	}
}

func TestTicketPreparationPublishesExplicitBreakingContractVersion(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("provider_exchange_tickets.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if !strings.Contains(text, `"preparation_contract_version"`) ||
		!strings.Contains(text, "models.ProviderActionTicketPreparationV2") {
		t.Fatal("ticket preparation does not publish its explicit v2 machine contract")
	}
}

func TestPrepareProviderActionToolDoesNotConflateTicketWithHandoff(t *testing.T) {
	t.Parallel()
	h := NewMCPHandler(nil, "https://nothumansearch.ai")
	var definition map[string]any
	for _, candidate := range h.toolDefinitions() {
		if candidate["name"] == "prepare_provider_action" {
			definition = candidate
			break
		}
	}
	if definition == nil {
		t.Fatal("prepare_provider_action tool is missing")
	}
	description, _ := definition["description"].(string)
	for _, required := range []string{
		"Prepare a signed action ticket",
		"never returns the provider action URL or records a handoff",
		"ticket-authorization attestation",
	} {
		if !strings.Contains(description, required) {
			t.Fatalf("prepare_provider_action description is missing %q: %s", required, description)
		}
	}
	if strings.Contains(description, "Create a signed action handoff") {
		t.Fatalf("ticket preparation is described as a handoff: %s", description)
	}
	properties := definition["inputSchema"].(map[string]any)["properties"].(map[string]any)
	principalConsent := properties["principal_consent"].(map[string]any)["description"].(string)
	if !strings.Contains(principalConsent, "ticket preparation") ||
		strings.Contains(principalConsent, "authorized this handoff") {
		t.Fatalf("principal_consent description conflates preparation and handoff: %s", principalConsent)
	}
}

func TestTicketPreparationAndHandoffBypassMCPIdentityAndPaidUsageTelemetry(t *testing.T) {
	source, err := os.ReadFile("mcp.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "func (h *MCPHandler) handleToolCall")
	if start < 0 {
		t.Fatal("could not locate handleToolCall")
	}
	endOffset := strings.Index(text[start:], "func isNHSActiveProbeTool")
	if endOffset < 0 {
		t.Fatal("could not isolate handleToolCall")
	}
	handle := text[start : start+endOffset]
	paidAccess := strings.Index(handle, "resolveRequestRateAccess")
	if paidAccess < 0 {
		t.Fatal("could not locate generic paid/identity access path")
	}

	for _, test := range []struct {
		name    string
		rawCall string
	}{
		{name: "prepare_provider_action", rawCall: "toolPrepareProviderActionRaw"},
		{name: "handoff_provider_action", rawCall: "toolHandoffProviderActionRaw"},
	} {
		branch := strings.Index(handle, `if route.Name == "`+test.name+`"`)
		if branch < 0 || branch > paidAccess {
			t.Fatalf("%s privacy branch is not before paid/identity access resolution", test.name)
		}
		between := handle[branch:paidAccess]
		for _, required := range []string{test.rawCall, "flushTo(w)", "return"} {
			if !strings.Contains(between, required) {
				t.Fatalf("%s early branch missing %q", test.name, required)
			}
		}
		for _, forbidden := range []string{"reservePriorityUnit", "LogMCPRequest", "LogIntentEvent", "mcpAnalyticsArguments"} {
			if strings.Contains(between, forbidden) {
				t.Fatalf("%s early branch contains forbidden telemetry/quota path %q", test.name, forbidden)
			}
		}
	}
}

func TestDisabledProviderExchangeMCPGatePrecedesArgumentValidation(t *testing.T) {
	t.Parallel()
	h := NewMCPHandler(nil, "https://nothumansearch.ai")
	h.ProviderExchangeEnabled = false
	// Deliberately leave a non-nil but unusable provider handler behind. The
	// recovery-mode gate, not configuration or input validation, must decide
	// every provider-funded mutation response.
	h.ProviderExchange = &ProviderExchangeHandler{}

	tests := []struct {
		name      string
		tool      string
		arguments any
	}{
		{name: "prepare invalid shape", tool: "prepare_provider_action", arguments: []any{}},
		{name: "prepare valid shape", tool: "prepare_provider_action", arguments: map[string]any{
			"offer_id": "00000000-0000-4000-8000-000000000000", "search_id": "nhs_sr_fixture",
			"demand_topic": "developer-tools", "principal_consent": true,
			"consent_version": models.ProviderPrincipalConsentV1,
		}},
		{name: "handoff invalid shape", tool: "handoff_provider_action", arguments: []any{}},
		{name: "handoff valid shape", tool: "handoff_provider_action", arguments: map[string]any{
			"ticket_id": "00000000-0000-4000-8000-000000000000", "attribution_token": "fixture-bearer",
			"principal_handoff_consent": true, "handoff_consent_version": models.ProviderActionHandoffConsentV1,
		}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			body, err := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"id":      91,
				"method":  "tools/call",
				"params": map[string]any{
					"name": test.tool, "arguments": test.arguments,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			if rr.Header().Get("Cache-Control") != "private, no-store" || rr.Header().Get("Pragma") != "no-cache" {
				t.Fatalf("private response headers=%v", rr.Header())
			}
			var response struct {
				Result struct {
					IsError           bool             `json:"isError"`
					Content           []map[string]any `json:"content"`
					StructuredContent map[string]any   `json:"structuredContent"`
				} `json:"result"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if !response.Result.IsError || len(response.Result.Content) != 1 ||
				response.Result.Content[0]["type"] != "text" ||
				response.Result.Content[0]["text"] != providerExchangeDisabledMCPMessage ||
				response.Result.StructuredContent["error"] != providerExchangeDisabledMCPErrorCode ||
				response.Result.StructuredContent["writes_enabled"] != false ||
				len(response.Result.StructuredContent) != 2 {
				t.Fatalf("disabled MCP result=%s", rr.Body.String())
			}
			serialized := rr.Body.String()
			if strings.Contains(serialized, "attribution_token") || strings.Contains(serialized, `"action_url"`) {
				t.Fatalf("disabled MCP response exposed mutation capability: %s", serialized)
			}
		})
	}
}

func TestActionTicketHandoffReturnsURLOnlyAfterDurableReceipt(t *testing.T) {
	t.Parallel()
	h := &ProviderExchangeHandler{
		handoffLimit: newMCPDiscoveryRateLimiter(10, time.Hour),
		recordActionHandoff: func(_ *sql.DB, input models.ProviderActionHandoffInput) (*models.ActionTicket, *models.ProviderActionHandoffReceipt, error) {
			if input.ActionTicketID != "4b69ca8e-d61d-47e2-91dd-fecd9f711234" || input.AttributionToken != "exact-bearer" ||
				!input.PrincipalHandoffConsent || input.HandoffConsentVersion != models.ProviderActionHandoffConsentV1 ||
				!input.PrincipalControlledIntentDisclosureConsent ||
				input.ControlledIntentDisclosureConsentVersion != models.ProviderControlledIntentDisclosureConsentV1 {
				t.Fatalf("handoff input=%#v", input)
			}
			return &models.ActionTicket{
					ID:                input.ActionTicketID,
					ActionURLSnapshot: "https://provider.example/start",
					Status:            "redirected",
				}, &models.ProviderActionHandoffReceipt{
					ID:                      "4b69ca8e-d61d-47e2-91dd-fecd9f711235",
					ActionTicketID:          input.ActionTicketID,
					PrincipalHandoffConsent: true,
					HandoffConsentVersion:   models.ProviderActionHandoffConsentV1,
					PrincipalControlledIntentDisclosureConsent: true,
					ControlledIntentDisclosureConsentVersion:   models.ProviderControlledIntentDisclosureConsentV1,
					EventContractVersion:                       models.ProviderActionHandoffContractV1,
				}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/action-tickets/handoff", bytes.NewBufferString(`{
		"ticket_id":"4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		"attribution_token":" exact-bearer ",
		"principal_handoff_consent":true,
		"handoff_consent_version":"nhs-provider-handoff-consent-v1",
		"principal_controlled_intent_disclosure_consent":true,
		"controlled_intent_disclosure_consent_version":"nhs-provider-controlled-intent-disclosure-consent-v1"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ActionTicketHandoff(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("handoff status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{`"observed_handoff":true`, `"provider_charged":false`, `"principal_charged":false`, `"principal_controlled_intent_disclosure_consent":true`, `"controlled_intent_disclosure_consent_version":"nhs-provider-controlled-intent-disclosure-consent-v1"`, `https://provider.example/start?nhs_attribution=exact-bearer`} {
		if !strings.Contains(body, want) {
			t.Fatalf("handoff response missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "presented_token_hash") {
		t.Fatalf("handoff response leaked token hash: %s", body)
	}
}

func TestActionTicketHandoffReturnsMachineReadableReviewPendingWithoutURL(t *testing.T) {
	t.Parallel()
	called := 0
	h := &ProviderExchangeHandler{
		handoffLimit: newMCPDiscoveryRateLimiter(10, time.Hour),
		recordActionHandoff: func(_ *sql.DB, input models.ProviderActionHandoffInput) (*models.ActionTicket, *models.ProviderActionHandoffReceipt, error) {
			called++
			if input.ActionTicketID != "4b69ca8e-d61d-47e2-91dd-fecd9f711234" ||
				input.AttributionToken != "exact-bearer" {
				t.Fatalf("handoff input=%#v", input)
			}
			return nil, nil, models.ErrProviderPilotReviewRequired
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/action-tickets/handoff", bytes.NewBufferString(`{
		"ticket_id":"4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		"attribution_token":"exact-bearer",
		"principal_handoff_consent":true,
		"handoff_consent_version":"nhs-provider-handoff-consent-v1"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ActionTicketHandoff(rr, req)
	if called != 1 || rr.Code != http.StatusConflict {
		t.Fatalf("handoff calls=%d status=%d body=%s", called, rr.Code, rr.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"status":                         "review_pending",
		"review_pending":                 true,
		"review_contract_version":        models.ProviderPilotReviewContractV1,
		"review_type":                    "ticket",
		"subject_id":                     "4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		"retryable":                      true,
		"observed_handoff":               false,
		"handoff_receipt_created":        false,
		"action_url_available":           false,
		"principal_charged":              false,
		"provider_charged":               false,
		"direct_provider_access_is_free": true,
	} {
		if response[key] != want {
			t.Fatalf("%s=%#v want %#v; response=%s", key, response[key], want, rr.Body.String())
		}
	}
	for _, forbidden := range []string{"action_url", "handoff_receipt", "attribution_token"} {
		if _, exists := response[forbidden]; exists {
			t.Fatalf("review-pending response exposed %s: %s", forbidden, rr.Body.String())
		}
	}
}

func TestHandoffProviderActionMCPIsReachableWithoutRESTFallback(t *testing.T) {
	t.Parallel()
	provider := &ProviderExchangeHandler{
		handoffLimit: newMCPDiscoveryRateLimiter(10, time.Hour),
		recordActionHandoff: func(_ *sql.DB, input models.ProviderActionHandoffInput) (*models.ActionTicket, *models.ProviderActionHandoffReceipt, error) {
			if input.PrincipalControlledIntentDisclosureConsent || input.ControlledIntentDisclosureConsentVersion != "" {
				t.Fatalf("optional controlled-intent disclosure was not safely declined: %#v", input)
			}
			return &models.ActionTicket{
					ID: input.ActionTicketID, ActionURLSnapshot: "https://provider.example/mcp", Status: "redirected",
				}, &models.ProviderActionHandoffReceipt{
					ID: "4b69ca8e-d61d-47e2-91dd-fecd9f711236", ActionTicketID: input.ActionTicketID,
					PrincipalHandoffConsent: true, HandoffConsentVersion: models.ProviderActionHandoffConsentV1,
					EventContractVersion: models.ProviderActionHandoffContractV1,
				}, nil
		},
	}
	h := &MCPHandler{ProviderExchange: provider}
	if !isKnownMCPTool("handoff_provider_action") {
		t.Fatal("handoff_provider_action is not advertised as a known MCP tool")
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	h.toolHandoffProviderActionRaw(rr, json.RawMessage(`7`), json.RawMessage(`{
		"ticket_id":"4b69ca8e-d61d-47e2-91dd-fecd9f711234",
		"attribution_token":"mcp-bearer",
		"principal_handoff_consent":true,
		"handoff_consent_version":"nhs-provider-handoff-consent-v1"
	}`), req)
	if !strings.Contains(rr.Body.String(), `"observed_handoff":true`) ||
		!strings.Contains(rr.Body.String(), `https://provider.example/mcp?nhs_attribution=mcp-bearer`) {
		t.Fatalf("MCP handoff response=%s", rr.Body.String())
	}
	if rr.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("MCP handoff cache policy=%q", rr.Header().Get("Cache-Control"))
	}
}

func TestActionTicketPreparationMapsProviderCapacityToConflictNotPayment(t *testing.T) {
	t.Parallel()
	for _, err := range []error{
		models.ErrInsufficientProviderFunds,
		models.ErrProviderTermsCreditLimit,
		models.ErrProviderBudgetLimit,
	} {
		status, message := actionTicketPreparationStatus(err)
		if status != http.StatusConflict || !strings.Contains(message, "provider-funded commercial capacity") ||
			!strings.Contains(message, "principal is not charged") {
			t.Fatalf("err=%v status=%d message=%q", err, status, message)
		}
	}
	if status, _ := actionTicketPreparationStatus(models.ErrInvalidProviderExchange); status != http.StatusBadRequest {
		t.Fatalf("invalid input status=%d, want=%d", status, http.StatusBadRequest)
	}
}

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
