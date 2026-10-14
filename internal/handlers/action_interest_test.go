package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

const validActionInterestJSON = `{"search_id":"nhs_sr_AAAAAAAAAAAAAAAA","domain":"example.com","action_type":"quote","caller_attests_principal_interest":true,"confirmation_version":"nhs-action-interest-v1"}`

func newActionInterestRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/action-interests", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.80:4321"
	return req
}

func TestActionInterestRESTIsStrictPrivateAndTruthful(t *testing.T) {
	for name, test := range map[string]struct {
		body   string
		status int
	}{
		"unknown field":               {strings.TrimSuffix(validActionInterestJSON, "}") + `,"notes":"call me"}`, http.StatusBadRequest},
		"false confirmation":          {`{"search_id":"nhs_sr_AAAAAAAAAAAAAAAA","domain":"example.com","action_type":"quote","caller_attests_principal_interest":false,"confirmation_version":"nhs-action-interest-v1"}`, http.StatusBadRequest},
		"url domain":                  {`{"search_id":"nhs_sr_AAAAAAAAAAAAAAAA","domain":"https://example.com/private?token=secret","action_type":"quote","caller_attests_principal_interest":true,"confirmation_version":"nhs-action-interest-v1"}`, http.StatusBadRequest},
		"valid but unavailable store": {validActionInterestJSON, http.StatusServiceUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			handler := NewActionInterestHandler(nil, "https://nothumansearch.ai")
			rr := httptest.NewRecorder()
			handler.Record(rr, newActionInterestRequest(test.body))
			if rr.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, test.status, rr.Body.String())
			}
			for header, want := range map[string]string{
				"Cache-Control":   "private, no-store",
				"Pragma":          "no-cache",
				"Referrer-Policy": "no-referrer",
			} {
				if got := rr.Header().Get(header); got != want {
					t.Fatalf("%s = %q, want %q", header, got, want)
				}
			}
		})
	}

	handler := NewActionInterestHandler(nil, "https://nothumansearch.ai")
	rr := httptest.NewRecorder()
	req := newActionInterestRequest(validActionInterestJSON)
	req.Method = http.MethodGet
	handler.Record(rr, req)
	if rr.Code != http.StatusMethodNotAllowed || rr.Header().Get("Allow") != "POST" {
		t.Fatalf("GET status=%d allow=%q", rr.Code, rr.Header().Get("Allow"))
	}

	response := actionInterestResponse(&models.ActionInterestReceipt{})
	for _, key := range []string{
		"provider_contacted", "row_level_shared_with_provider", "action_ticket_created",
		"charge_created", "provider_or_principal_charged", "commercial_proof",
		"organic_rank_affected", "rank_or_score_input",
	} {
		if value, ok := response[key].(bool); !ok || value {
			t.Fatalf("truth field %s = %#v, want false", key, response[key])
		}
	}
}

func TestActionInterestAttemptOutcomesAreStableAndNonCommercial(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{nil, "created"},
		{errActionInterestRateLimited, "rate_limited"},
		{errActionInterestCrossOrigin, "cross_origin"},
		{models.ErrInvalidActionInterest, "invalid_request"},
		{models.ErrActionInterestUnavailable, "unavailable"},
		{models.ErrActionInterestConflict, "conflict"},
		{models.ErrActionInterestStoreUnavailable, "store_unavailable"},
		{errors.New("database unavailable"), "internal_error"},
	} {
		if got := actionInterestAttemptOutcome(test.err); got != test.want {
			t.Fatalf("attempt outcome for %v = %q, want %q", test.err, got, test.want)
		}
	}
}

func TestSelectedActionInterestOpportunityIsExactAndNonCommercial(t *testing.T) {
	site := &models.Site{Domain: "Example.COM"}
	opportunity := selectedActionInterestOpportunity("https://nothumansearch.ai/", "nhs_sr_AAAAAAAAAAAAAAAA", site)
	if available, _ := opportunity["available"].(bool); !available {
		t.Fatalf("selected opportunity unavailable: %#v", opportunity)
	}
	if got := opportunity["search_id"]; got != "nhs_sr_AAAAAAAAAAAAAAAA" {
		t.Fatalf("search_id = %#v", got)
	}
	domains, ok := opportunity["eligible_domains"].([]string)
	if !ok || len(domains) != 1 || domains[0] != "example.com" {
		t.Fatalf("eligible domains = %#v, want exact selected domain", opportunity["eligible_domains"])
	}
	for _, key := range []string{"provider_contacted", "commercial_proof", "organic_rank_affected"} {
		if value, ok := opportunity[key].(bool); !ok || value {
			t.Fatalf("truth field %s = %#v, want false", key, opportunity[key])
		}
	}
	if condition, _ := opportunity["invocation_condition"].(string); !strings.Contains(condition, "do not infer interest") {
		t.Fatalf("invocation condition lost anti-inference boundary: %#v", condition)
	}
	contract, ok := opportunity["call_contract"].(map[string]any)
	if !ok || contract["available"] != true || contract["tool"] != "record_action_interest" ||
		contract["invoke_only_if"] != actionInterestInvocationCondition ||
		contract["executable_without_explicit_principal_intent"] != false ||
		contract["query_prompt_contact_identity_fields_are_accepted"] != false {
		t.Fatalf("selected call contract = %#v", opportunity["call_contract"])
	}
	fixed, ok := contract["fixed_arguments_if_invocation_condition_met"].(map[string]any)
	if !ok || !reflect.DeepEqual(fixed, map[string]any{
		"search_id":                         "nhs_sr_AAAAAAAAAAAAAAAA",
		"caller_attests_principal_interest": true,
		"confirmation_version":              models.ActionInterestConfirmationV1,
	}) {
		t.Fatalf("selected fixed arguments = %#v", contract["fixed_arguments_if_invocation_condition_met"])
	}
	if !reflect.DeepEqual(contract["domain_must_be_one_of"], []string{"example.com"}) ||
		!reflect.DeepEqual(contract["action_type_must_be_one_of"], models.ActionInterestTypes()) ||
		!reflect.DeepEqual(contract["arguments_must_contain_only"], actionInterestArgumentNames) {
		t.Fatalf("selected choices = %#v", contract)
	}

	unavailable := selectedActionInterestOpportunity("https://nothumansearch.ai", "nhs_sr_AAAAAAAAAAAAAAAA", nil)
	if available, _ := unavailable["available"].(bool); available || unavailable["search_id"] != "" {
		t.Fatalf("nil-site opportunity did not fail closed: %#v", unavailable)
	}
	unavailableContract, _ := unavailable["call_contract"].(map[string]any)
	if unavailableContract["available"] != false ||
		len(unavailableContract["fixed_arguments_if_invocation_condition_met"].(map[string]any)) != 0 ||
		len(unavailableContract["domain_must_be_one_of"].([]string)) != 0 ||
		len(unavailableContract["action_type_must_be_one_of"].([]string)) != 0 {
		t.Fatalf("unavailable call contract retained capability: %#v", unavailableContract)
	}
}

func TestMCPInitializeExplainsSelectionToInterestBoundary(t *testing.T) {
	handler := NewMCPHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
	))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("initialize status = %d; body=%s", rr.Code, rr.Body.String())
	}
	var response struct {
		Result struct {
			Instructions string `json:"instructions"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"record selection only", "If and only if the principal explicitly currently wants",
		"action_interest.call_contract", "never infer interest", "contacts no provider",
	} {
		if !strings.Contains(response.Result.Instructions, required) {
			t.Fatalf("initialize instructions missing %q: %s", required, response.Result.Instructions)
		}
	}
}

func TestGetSiteDetailsDescribesPostSelectionAttestationBoundary(t *testing.T) {
	handler := NewMCPHandler(nil, "https://nothumansearch.ai")
	for _, definition := range handler.toolDefinitions() {
		if definition["name"] != "get_site_details" {
			continue
		}
		description, _ := definition["description"].(string)
		for _, required := range []string{"record_action_interest", "Selection alone is never interest", "never contacts the provider"} {
			if !strings.Contains(description, required) {
				t.Fatalf("get_site_details description missing %q: %s", required, description)
			}
		}
		schema, _ := definition["inputSchema"].(map[string]any)
		properties, _ := schema["properties"].(map[string]any)
		searchID, _ := properties["search_id"].(map[string]any)
		searchDescription, _ := searchID["description"].(string)
		if !strings.Contains(searchDescription, "explicit current-principal attestation") {
			t.Fatalf("search_id description lost separate-attestation requirement: %s", searchDescription)
		}
		return
	}
	t.Fatal("get_site_details tool missing")
}

func TestActionInterestRESTAndMCPShareFreeAbuseGuard(t *testing.T) {
	shared := NewActionInterestHandler(nil, "https://nothumansearch.ai")
	shared.limiter = newMCPDiscoveryRateLimiter(1, time.Hour)

	first := httptest.NewRecorder()
	shared.Record(first, newActionInterestRequest(validActionInterestJSON))
	if first.Code != http.StatusServiceUnavailable {
		t.Fatalf("first REST status = %d, want store unavailable", first.Code)
	}

	mcp := NewMCPHandler(nil, "https://nothumansearch.ai")
	mcp.ActionInterests = shared
	second := httptest.NewRecorder()
	mcp.toolRecordActionInterest(second, json.RawMessage(`1`), map[string]any{
		"search_id":                         "nhs_sr_AAAAAAAAAAAAAAAA",
		"domain":                            "example.com",
		"action_type":                       "quote",
		"caller_attests_principal_interest": true,
		"confirmation_version":              models.ActionInterestConfirmationV1,
	}, newActionInterestRequest(validActionInterestJSON))
	if !strings.Contains(second.Body.String(), "safety limit exceeded") {
		t.Fatalf("MCP surface bypassed shared limiter: %s", second.Body.String())
	}
}

func TestInvalidActionInterestCallsConsumeTransientLimit(t *testing.T) {
	rest := NewActionInterestHandler(nil, "https://nothumansearch.ai")
	rest.limiter = newMCPDiscoveryRateLimiter(1, time.Hour)
	invalidREST := httptest.NewRecorder()
	rest.Record(invalidREST, newActionInterestRequest(`{"notes":"large invalid body"}`))
	if invalidREST.Code != http.StatusBadRequest {
		t.Fatalf("invalid REST status = %d, want 400", invalidREST.Code)
	}
	limitedREST := httptest.NewRecorder()
	rest.Record(limitedREST, newActionInterestRequest(validActionInterestJSON))
	if limitedREST.Code != http.StatusTooManyRequests {
		t.Fatalf("REST request after invalid flood status = %d, want 429; body=%s", limitedREST.Code, limitedREST.Body.String())
	}

	shared := NewActionInterestHandler(nil, "https://nothumansearch.ai")
	shared.limiter = newMCPDiscoveryRateLimiter(1, time.Hour)
	mcp := NewMCPHandler(nil, "https://nothumansearch.ai")
	mcp.ActionInterests = shared
	invalidMCP := httptest.NewRecorder()
	mcp.toolRecordActionInterest(invalidMCP, json.RawMessage(`1`), map[string]any{
		"notes": "large invalid body",
	}, newActionInterestRequest(validActionInterestJSON))
	if !strings.Contains(invalidMCP.Body.String(), "unsupported field") {
		t.Fatalf("invalid MCP result = %s", invalidMCP.Body.String())
	}
	limitedMCP := httptest.NewRecorder()
	mcp.toolRecordActionInterest(limitedMCP, json.RawMessage(`2`), map[string]any{
		"search_id":                         "nhs_sr_AAAAAAAAAAAAAAAA",
		"domain":                            "example.com",
		"action_type":                       "quote",
		"caller_attests_principal_interest": true,
		"confirmation_version":              models.ActionInterestConfirmationV1,
	}, newActionInterestRequest(validActionInterestJSON))
	if !strings.Contains(limitedMCP.Body.String(), "safety limit exceeded") {
		t.Fatalf("MCP request after invalid flood bypassed limiter: %s", limitedMCP.Body.String())
	}

	fullPathShared := NewActionInterestHandler(nil, "https://nothumansearch.ai")
	fullPathShared.limiter = newMCPDiscoveryRateLimiter(1, time.Hour)
	fullPathMCP := NewMCPHandler(nil, "https://nothumansearch.ai")
	fullPathMCP.ActionInterests = fullPathShared
	postToolCall := func(id int, arguments string) *httptest.ResponseRecorder {
		t.Helper()
		body := `{"jsonrpc":"2.0","id":` + strconv.Itoa(id) + `,"method":"tools/call","params":{"name":"record_action_interest","arguments":` + arguments + `}}`
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
		req.RemoteAddr = "203.0.113.81:4321"
		rr := httptest.NewRecorder()
		fullPathMCP.ServeHTTP(rr, req)
		return rr
	}
	invalidFullPath := postToolCall(1, `[]`)
	if !strings.Contains(invalidFullPath.Body.String(), "arguments must be an object") {
		t.Fatalf("full MCP invalid-arguments result = %s", invalidFullPath.Body.String())
	}
	limitedFullPath := postToolCall(2, validActionInterestJSON)
	if !strings.Contains(limitedFullPath.Body.String(), "safety limit exceeded") {
		t.Fatalf("full MCP invalid arguments bypassed action-interest limiter: %s", limitedFullPath.Body.String())
	}
}

func TestRecordActionInterestMCPToolHasExactPrivacySchema(t *testing.T) {
	handler := NewMCPHandler(nil, "https://nothumansearch.ai")
	var definition map[string]any
	for _, candidate := range handler.toolDefinitions() {
		if candidate["name"] == "record_action_interest" {
			definition = candidate
			break
		}
	}
	if definition == nil {
		t.Fatal("record_action_interest tool missing")
	}
	schema, ok := definition["inputSchema"].(map[string]any)
	if !ok || schema["additionalProperties"] != false {
		t.Fatalf("action-interest schema = %#v", schema)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok || len(properties) != 5 {
		t.Fatalf("action-interest properties = %#v", properties)
	}
	for _, key := range []string{"search_id", "domain", "action_type", "caller_attests_principal_interest", "confirmation_version"} {
		if _, ok := properties[key]; !ok {
			t.Fatalf("action-interest schema missing %q", key)
		}
	}
	for _, forbidden := range []string{"query", "email", "contact", "notes", "identity", "budget", "region"} {
		if _, ok := properties[forbidden]; ok {
			t.Fatalf("action-interest schema exposes forbidden property %q", forbidden)
		}
	}
	if err := validateRecordActionInterestArguments(map[string]any{
		"search_id": "nhs_sr_AAAAAAAAAAAAAAAA", "domain": "example.com",
		"action_type": "quote", "caller_attests_principal_interest": true,
		"confirmation_version": models.ActionInterestConfirmationV1,
		"notes":                "private",
	}); err == nil {
		t.Fatal("MCP runtime validator accepted an unknown free-form field")
	}
}

func TestMCPReceiptBoundDetailActionsCarryExactSelectionWithoutInterest(t *testing.T) {
	sites := []models.Site{{Domain: "Example.DEV"}, {Domain: "second.example"}}
	if actions := mcpReceiptBoundDetailActions("", sites); len(actions) != 0 {
		t.Fatalf("unscoped detail actions = %#v, want none", actions)
	}

	const searchID = "nhs_sr_AAAAAAAAAAAAAAAA"
	actions := mcpReceiptBoundDetailActions(searchID, sites)
	if len(actions) != len(sites) {
		t.Fatalf("detail action count = %d, want %d", len(actions), len(sites))
	}
	for index, action := range actions {
		arguments, ok := action["arguments"].(map[string]any)
		if !ok {
			t.Fatalf("detail action %d arguments = %#v", index, action["arguments"])
		}
		if action["tool"] != "get_site_details" ||
			arguments["search_id"] != searchID ||
			arguments["domain"] != strings.ToLower(sites[index].Domain) ||
			action["records_result_selection"] != true ||
			action["action_interest_inferred"] != false ||
			action["provider_contacted"] != false ||
			action["organic_rank_affected"] != false {
			t.Fatalf("detail action %d boundary = %#v", index, action)
		}
		for _, forbidden := range []string{"query", "prompt", "contact", "identity", "provider_offer"} {
			if _, exists := action[forbidden]; exists {
				t.Fatalf("detail action exposed %q: %#v", forbidden, action)
			}
			if _, exists := arguments[forbidden]; exists {
				t.Fatalf("detail arguments exposed %q: %#v", forbidden, arguments)
			}
		}
	}
}

func TestMCPReceiptBoundDetailActionTextCarriesExactCallWithoutCommercialClaim(t *testing.T) {
	var builder strings.Builder
	appendMCPReceiptBoundDetailActionText(&builder, "", "example.dev")
	appendMCPReceiptBoundDetailActionText(&builder, "nhs_sr_AAAAAAAAAAAAAAAA", "")
	if builder.Len() != 0 {
		t.Fatalf("unscoped detail action text = %q, want empty", builder.String())
	}

	appendMCPReceiptBoundDetailActionText(
		&builder,
		"nhs_sr_AAAAAAAAAAAAAAAA",
		"Example.DEV",
	)
	text := builder.String()
	for _, required := range []string{
		`get_site_details {"domain":"example.dev","search_id":"nhs_sr_AAAAAAAAAAAAAAAA"}`,
		"records selection only",
		"does not infer interest",
		"contact a provider",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("detail action text missing %q: %s", required, text)
		}
	}
	for _, forbidden := range []string{"query", "prompt", "identity", "provider_offer", "paid"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("detail action text exposed %q: %s", forbidden, text)
		}
	}
}

func TestActionInterestOpportunityIsExplicitBoundedAndNonCommercial(t *testing.T) {
	sites := []models.Site{
		{Domain: "Example.COM"},
		{Domain: "second.example"},
		{Domain: "example.com"},
	}
	opportunity := publicActionInterestOpportunity(
		"https://nothumansearch.ai/",
		"nhs_sr_AAAAAAAAAAAAAAAA",
		sites,
		true,
	)
	if opportunity["available"] != true || opportunity["search_id"] != "nhs_sr_AAAAAAAAAAAAAAAA" {
		t.Fatalf("available opportunity binding = %#v", opportunity)
	}
	domains, ok := opportunity["eligible_domains"].([]string)
	if !ok || len(domains) != 2 || domains[0] != "example.com" || domains[1] != "second.example" {
		t.Fatalf("eligible domains = %#v", opportunity["eligible_domains"])
	}
	if opportunity["caller_attestation_required"] != true || opportunity["invocation_condition"] != actionInterestInvocationCondition {
		t.Fatalf("explicit invocation contract missing: %#v", opportunity)
	}
	if opportunity["confirmation_url"] != "https://nothumansearch.ai/privacy#action-interest-v1" {
		t.Fatalf("confirmation url = %#v", opportunity["confirmation_url"])
	}
	for _, key := range []string{"provider_contacted", "commercial_proof", "organic_rank_affected"} {
		if opportunity[key] != false {
			t.Fatalf("truth field %s = %#v", key, opportunity[key])
		}
	}
	for _, forbidden := range []string{"query", "prompt", "identity", "contact", "email", "paid_rank"} {
		if _, exists := opportunity[forbidden]; exists {
			t.Fatalf("opportunity exposes forbidden field %q", forbidden)
		}
	}

	unavailable := publicActionInterestOpportunity(
		"https://nothumansearch.ai",
		"nhs_sr_AAAAAAAAAAAAAAAA",
		sites,
		false,
	)
	if unavailable["available"] != false || unavailable["search_id"] != "" {
		t.Fatalf("unavailable opportunity retained a receipt: %#v", unavailable)
	}
	if got := unavailable["eligible_domains"].([]string); len(got) != 0 {
		t.Fatalf("unavailable opportunity retained domains: %#v", got)
	}
}

func TestMCPActionInterestOpportunityCarriesReadyInputsWithoutInferringIntent(t *testing.T) {
	context := mcpDiscoveryExchange{
		searchID:                "nhs_sr_AAAAAAAAAAAAAAAA",
		sites:                   []models.Site{{Domain: "example.com"}},
		actionInterestAvailable: true,
	}
	opportunity := mcpDiscoveryActionInterest("https://nothumansearch.ai", context)
	if opportunity["tool"] != "record_action_interest" || opportunity["search_id"] != context.searchID {
		t.Fatalf("MCP opportunity is not ready to bind: %#v", opportunity)
	}
	if opportunity["invocation_condition"] != actionInterestInvocationCondition {
		t.Fatalf("MCP opportunity can infer intent: %#v", opportunity)
	}
}

func TestMCPActionInterestGuidanceUsesConditionalCallContract(t *testing.T) {
	var unavailable strings.Builder
	appendMCPActionInterestGuidance(&unavailable, mcpDiscoveryExchange{})
	if unavailable.Len() != 0 {
		t.Fatalf("unavailable guidance = %q", unavailable.String())
	}

	var available strings.Builder
	appendMCPActionInterestGuidance(&available, mcpDiscoveryExchange{actionInterestAvailable: true})
	text := available.String()
	for _, required := range []string{
		"action_interest.call_contract", "fixed receipt and consent-version fields",
		"choose only that returned domain", "Do not call it otherwise",
		"does not contact the provider", "no later than 30 days",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("action-interest guidance missing %q: %s", required, text)
		}
	}
}

func TestRecordActionInterestBypassesMCPIdentityAndPaidUsageTelemetry(t *testing.T) {
	source, err := os.ReadFile("mcp.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "func (h *MCPHandler) handleToolCall")
	end := strings.Index(text[start:], "func isNHSActiveProbeTool")
	if start < 0 || end < 0 {
		t.Fatal("could not isolate handleToolCall")
	}
	handle := text[start : start+end]
	branch := strings.Index(handle, `if route.Name == "record_action_interest"`)
	paidAccess := strings.Index(handle, "resolveRequestRateAccess")
	if branch < 0 || paidAccess < 0 || branch > paidAccess {
		t.Fatalf("action-interest privacy branch is not before paid/identity access resolution")
	}
	between := handle[branch:paidAccess]
	for _, required := range []string{"toolRecordActionInterestRaw", "flushTo(w)", "return"} {
		if !strings.Contains(between, required) {
			t.Fatalf("action-interest early branch missing %q", required)
		}
	}
	for _, forbidden := range []string{"reservePriorityUnit", "LogMCPRequest", "LogIntentEvent", "mcpAnalyticsArguments"} {
		if strings.Contains(between, forbidden) {
			t.Fatalf("action-interest early branch contains forbidden telemetry/quota path %q", forbidden)
		}
	}

	serverSource, err := os.ReadFile("../../cmd/server/main.go")
	if err != nil {
		t.Fatal(err)
	}
	server := string(serverSource)
	if !strings.Contains(server, `"/api/v1/action-interests", // receipt table only; never bind interest invocation to IP/UA telemetry`) {
		t.Fatal("REST action-interest route is not excluded from page-view identity telemetry")
	}
	for _, required := range []string{
		`func suppressRequestLineIdentityTelemetry(p string) bool`,
		`"/api/v1/action-interests"`,
		`"/mcp"`,
		`if !suppressRequestLineIdentityTelemetry(r.URL.Path)`,
	} {
		if !strings.Contains(server, required) {
			t.Fatalf("REST and MCP action-interest surfaces are not excluded from outer request-line user-agent logging; missing %q", required)
		}
	}
}

func TestActionInterestRejectsCrossOriginBrowserMutation(t *testing.T) {
	handler := NewActionInterestHandler(nil, "https://nothumansearch.ai")
	for name, mutate := range map[string]func(*http.Request){
		"foreign origin":   func(r *http.Request) { r.Header.Set("Origin", "https://attacker.example") },
		"cross-site fetch": func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") },
	} {
		t.Run(name, func(t *testing.T) {
			req := newActionInterestRequest(validActionInterestJSON)
			mutate(req)
			rr := httptest.NewRecorder()
			rr.Header().Set("Access-Control-Allow-Origin", "*")
			handler.Record(rr, req)
			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403; body=%s", rr.Code, rr.Body.String())
			}
			if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Fatalf("forbidden mutation retained CORS read permission %q", got)
			}
		})
	}

	sameOrigin := newActionInterestRequest(validActionInterestJSON)
	sameOrigin.Header.Set("Origin", "https://nothumansearch.ai")
	rr := httptest.NewRecorder()
	handler.Record(rr, sameOrigin)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("same-origin status = %d, want store unavailable after origin gate; body=%s", rr.Code, rr.Body.String())
	}

	mcp := NewMCPHandler(nil, "https://nothumansearch.ai")
	mcp.ActionInterests = handler
	mcpReq := newActionInterestRequest(validActionInterestJSON)
	mcpReq.Header.Set("Origin", "https://attacker.example")
	mcpResponse := httptest.NewRecorder()
	mcpResponse.Header().Set("Access-Control-Allow-Origin", "*")
	mcp.toolRecordActionInterest(mcpResponse, json.RawMessage(`1`), map[string]any{
		"search_id":                         "nhs_sr_AAAAAAAAAAAAAAAA",
		"domain":                            "example.com",
		"action_type":                       "quote",
		"caller_attests_principal_interest": true,
		"confirmation_version":              models.ActionInterestConfirmationV1,
	}, mcpReq)
	if !strings.Contains(mcpResponse.Body.String(), "cross-origin action-interest mutation rejected") {
		t.Fatalf("MCP cross-origin result = %s", mcpResponse.Body.String())
	}
	if got := mcpResponse.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("MCP forbidden mutation retained CORS read permission %q", got)
	}
}

func TestStage1DemandAdminRequiresConfiguredConstantTimeKey(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "")
	handler := NewActionInterestHandler(nil, "https://nothumansearch.ai")
	rr := httptest.NewRecorder()
	handler.Stage1DemandProof(rr, httptest.NewRequest(http.MethodGet, "/api/v1/admin/demand-stage1", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured admin status = %d, want 503; body=%s", rr.Code, rr.Body.String())
	}
}
