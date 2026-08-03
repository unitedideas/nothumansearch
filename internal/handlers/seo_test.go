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
		"Tools (14):",
		"record_action_interest",
		"handoff_provider_action",
		"handoff_endpoint",
		"nhs-provider-handoff-consent-v1",
		"Handoff consent wording: /privacy#handoff-consent-v1",
		"it does not\nreturn the provider action URL",
		"provider-authenticated pilot-company, exact-terms acceptance, or",
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
		"/provider/commercial-acceptances:",
		"/action-interests:",
		"/action-tickets:",
		"/action-tickets/handoff:",
		"/provider/outcomes:",
		"/action-receipts/verify:",
		"signature_valid",
		"within_validity_window",
		"nhs-principal-consent-v1",
		"nhs-provider-handoff-consent-v1",
		"nhs-action-handoff-v1",
		"nhs-action-interest-v1",
		"ActionInterestRequest:",
		"ActionInterestResponse:",
		"It expires with the source search, no later than 30 days after that search.",
		"caller_attests_principal_interest",
		`schema: { $ref: "#/components/schemas/ActionInterestResponse" }`,
		"commercial proof",
		"organic_rank_paid",
		"provider_acknowledges_merchant_of_record",
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

func TestOpenAPIDescribesVerifiedAcceptancesAndSeparateObservedHandoff(t *testing.T) {
	t.Parallel()
	seo := NewSEOHandler(nil, "https://nothumansearch.ai")
	rr := httptest.NewRecorder()
	seo.OpenAPISpec(rr, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	body := rr.Body.String()

	for _, required := range []string{
		"/provider/commercial-acceptances:",
		"security: [{ ProviderKey: [] }]",
		"maxLength: 128",
		"ProviderCommercialAcceptanceRequest:",
		"enum: [pilot_company]",
		"enum: [terms_acceptance]",
		"enum: [terms_renewal]",
		"required: [event_type, offer_id, offer_version, exact_terms_sha256, provider_acceptance_reference]",
		"Exact version reviewed by the provider",
		"Exact commercial terms hash reviewed by the provider",
		"provider_acceptance_reference",
		"owner_verification_required",
		"commercial_proof_created",
		"/provider/action-tickets/resolve:",
		"ProviderControlledIntentResolveRequest:",
		"ProviderControlledIntentResolution:",
		"enum: [nhs-provider-controlled-intent-resolver-v1]",
		"enum: [nhs-provider-controlled-intent-disclosure-consent-v1]",
		"/action-tickets/handoff:",
		"ActionTicketPreparationResponse:",
		"preparation_contract_version:",
		"enum: [nhs-action-ticket-preparation-v2]",
		"PublicActionTicket:",
		"attribution_token:",
		"handoff_endpoint:",
		"ActionTicketHandoffRequest:",
		"principal_handoff_consent:",
		"handoff_consent_version:",
		"enum: [nhs-provider-handoff-consent-v1]",
		"principal_controlled_intent_disclosure_consent:",
		"controlled_intent_disclosure_consent_version:",
		"Declining it does not block handoff.",
		"ProviderActionHandoffReceipt:",
		"event_contract_version:",
		"private, no-store",
		"principal_charged: { type: boolean, enum: [false] }",
		"provider_charged: { type: boolean, enum: [false] }",
		"offer_version:",
		"commercial_terms_contract_version:",
		"commercial_terms_sha256:",
		"credit_rule:",
		"response_expectation:",
		"terms_period_anchor_rule:",
		"provider_acknowledges_merchant_of_record:",
		`"410": { description: Ticket attribution expired; neither party charged }`,
		"Verified commercial evidence unavailable",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("OpenAPI verified acceptance/handoff contract missing %q", required)
		}
	}
	if strings.Contains(body, "provider_is_merchant_of_record") {
		t.Fatal("OpenAPI presents provider Merchant-of-Record acknowledgement as an NHS-verified fact")
	}
	handoffStart := strings.Index(body, "  /action-tickets/handoff:")
	handoffEnd := strings.Index(body, "  /provider/outcomes:")
	if handoffStart < 0 || handoffEnd <= handoffStart {
		t.Fatalf("OpenAPI handoff path bounds not found: start=%d end=%d", handoffStart, handoffEnd)
	}
	if strings.Contains(body[handoffStart:handoffEnd], `"402":`) {
		t.Fatal("OpenAPI incorrectly maps missing handoff commercial evidence to payment-required instead of conflict")
	}
	ticketStart := strings.Index(body, "  /action-tickets:")
	if ticketStart < 0 || handoffStart <= ticketStart {
		t.Fatalf("OpenAPI ticket-preparation path bounds not found: start=%d end=%d", ticketStart, handoffStart)
	}
	ticketPath := body[ticketStart:handoffStart]
	for _, required := range []string{
		`"400": { description: Invalid JSON, unknown field, ticket input, or consent attestation }`,
		`"404": { description: Exact public offer or returned-offer evidence unavailable; intentionally indistinguishable }`,
		`"409": { description: Provider claim, authorization, commercial evidence, or provider-funded capacity unavailable; or request conflicts with a prior ticket. The principal is not charged. }`,
		`"410": { description: Exact replay refers to an expired ticket authorization }`,
		`"429": { description: Temporary action-ticket safety limit exceeded }`,
		`"503": { description: Signed provider actions are not configured }`,
	} {
		if !strings.Contains(ticketPath, required) {
			t.Fatalf("OpenAPI ticket preparation path missing runtime response %q: %s", required, ticketPath)
		}
	}
	if strings.Contains(ticketPath, `"402":`) {
		t.Fatalf("ticket preparation incorrectly presents provider capacity as a caller payment challenge: %s", ticketPath)
	}

	start := strings.Index(body, "    ActionTicketPreparationResponse:")
	end := strings.Index(body, "    ActionTicketHandoffRequest:")
	if start < 0 || end <= start {
		t.Fatalf("OpenAPI ticket preparation schema bounds not found: start=%d end=%d", start, end)
	}
	creationSchema := body[start:end]
	if !strings.Contains(creationSchema, "preparation_contract_version") ||
		!strings.Contains(creationSchema, "nhs-action-ticket-preparation-v2") {
		t.Fatalf("ticket creation schema omits explicit breaking preparation contract: %s", creationSchema)
	}
	if strings.Contains(creationSchema, "\n        action_url:") {
		t.Fatalf("ticket creation schema exposes provider action_url before observed handoff: %s", creationSchema)
	}
}

func TestOpenAPIControlledIntentResolverIsStrictNarrowAndProviderScoped(t *testing.T) {
	t.Parallel()
	seo := NewSEOHandler(nil, "https://nothumansearch.ai")
	rr := httptest.NewRecorder()
	seo.OpenAPISpec(rr, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	body := rr.Body.String()

	start := strings.Index(body, "  /provider/action-tickets/resolve:")
	end := strings.Index(body, "  /action-interests:")
	if start < 0 || end <= start {
		t.Fatalf("controlled-intent resolver path bounds not found: start=%d end=%d", start, end)
	}
	path := body[start:end]
	for _, required := range []string{
		"security: [{ ProviderKey: [] }]",
		`schema: { $ref: "#/components/schemas/ProviderControlledIntentResolveRequest" }`,
		`schema: { $ref: "#/components/schemas/ProviderControlledIntentResolution" }`,
		`"200":`, `"400":`, `"401":`, `"404":`, `"410":`, `"429":`, `"503":`,
		"no query, search receipt, identity, contact, network, action URL, price, accounting data, charge, outcome, or commercial proof",
	} {
		if !strings.Contains(path, required) {
			t.Fatalf("controlled-intent resolver path missing %q: %s", required, path)
		}
	}

	requestStart := strings.Index(body, "    ProviderControlledIntentResolveRequest:")
	responseEnd := strings.Index(body, "    PublicProviderOffer:")
	if requestStart < 0 || responseEnd <= requestStart {
		t.Fatalf("controlled-intent schema bounds not found: start=%d end=%d", requestStart, responseEnd)
	}
	schemas := body[requestStart:responseEnd]
	for _, required := range []string{
		"additionalProperties: false",
		"required: [attribution_token]",
		"resolver_contract_version:",
		"controlled_intent:",
		"intent_available_until:",
	} {
		if !strings.Contains(schemas, required) {
			t.Fatalf("controlled-intent schemas missing %q: %s", required, schemas)
		}
	}
	for _, forbidden := range []string{
		"search_receipt_id:", "query:", "contact:", "action_url:", "bounty_cents:",
		"currency:", "charge_status:", "provider_claim_id:",
	} {
		if strings.Contains(schemas, forbidden) {
			t.Fatalf("controlled-intent schemas expose forbidden field %q: %s", forbidden, schemas)
		}
	}
}

func TestOpenAPIProviderStatusAndDemandAreStrictClaimScopedReads(t *testing.T) {
	t.Parallel()
	seo := NewSEOHandler(nil, "https://nothumansearch.ai")
	rr := httptest.NewRecorder()
	seo.OpenAPISpec(rr, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	body := rr.Body.String()

	statusStart := strings.Index(body, "  /provider/pilot-status:")
	demandStart := strings.Index(body, "  /provider/demand:")
	resolverStart := strings.Index(body, "  /provider/action-tickets/resolve:")
	if statusStart < 0 || demandStart <= statusStart || resolverStart <= demandStart {
		t.Fatalf("provider read path bounds not found: status=%d demand=%d resolver=%d", statusStart, demandStart, resolverStart)
	}
	statusPath := body[statusStart:demandStart]
	demandPath := body[demandStart:resolverStart]

	for name, contract := range map[string]struct {
		path     string
		response string
		param    string
	}{
		"status": {path: statusPath, response: "ProviderPilotStatusResponse", param: "limit"},
		"demand": {path: demandPath, response: "ProviderDemandResponse", param: "days"},
	} {
		for _, required := range []string{
			"    get:",
			"security: [{ ProviderKey: [] }]",
			"- name: " + contract.param,
			`schema: { $ref: "#/components/schemas/` + contract.response + `" }`,
			`"200":`, `"400":`, `"401":`, `"404":`, `"429":`, `"500":`, `"503":`,
			`enum: ["private, no-store"]`,
		} {
			if !strings.Contains(contract.path, required) {
				t.Fatalf("provider %s OpenAPI path missing %q: %s", name, required, contract.path)
			}
		}
		if got := strings.Count(contract.path, "security: [{ ProviderKey: [] }]"); got != 1 {
			t.Fatalf("provider %s path has %d ProviderKey declarations, want exactly 1: %s", name, got, contract.path)
		}
	}
	if strings.Contains(demandPath, "name: domain") {
		t.Fatalf("claim-scoped provider demand accepts an arbitrary domain: %s", demandPath)
	}
	for _, required := range []string{
		"domain is derived from the authenticated claim and cannot be selected by the caller",
		"not unique agents or principals",
		"Result-selection and action-interest counts and rates are suppressed",
	} {
		if !strings.Contains(demandPath, required) {
			t.Fatalf("provider demand privacy/scope contract missing %q: %s", required, demandPath)
		}
	}

	schemaBounds := []struct {
		name string
		next string
	}{
		{name: "ProviderPilotOfferStatus", next: "ProviderPilotRecentEvent"},
		{name: "ProviderPilotRecentEvent", next: "ProviderPilotStatus"},
		{name: "ProviderPilotStatus", next: "ProviderPilotStatusResponse"},
		{name: "ProviderPilotStatusResponse", next: "ProviderDemandSummary"},
		{name: "ProviderDemandSummary", next: "ProviderDemandSurface"},
		{name: "ProviderDemandSurface", next: "ProviderDemandTopic"},
		{name: "ProviderDemandTopic", next: "ProviderDemandActionType"},
		{name: "ProviderDemandActionType", next: "ProviderDemandAnalytics"},
		{name: "ProviderDemandAnalytics", next: "ProviderDemandResponse"},
		{name: "ProviderDemandResponse", next: "ProviderControlledIntentResolveRequest"},
	}
	providerSchemasStart := strings.Index(body, "    ProviderPilotOfferStatus:")
	providerSchemasEnd := strings.Index(body, "    ProviderControlledIntentResolveRequest:")
	if providerSchemasStart < 0 || providerSchemasEnd <= providerSchemasStart {
		t.Fatalf("provider status/demand schema bounds not found: start=%d end=%d", providerSchemasStart, providerSchemasEnd)
	}
	providerSchemas := body[providerSchemasStart:providerSchemasEnd]
	for _, bound := range schemaBounds {
		start := strings.Index(body, "    "+bound.name+":")
		end := strings.Index(body, "    "+bound.next+":")
		if start < 0 || end <= start {
			t.Fatalf("schema %s bounds not found: start=%d end=%d", bound.name, start, end)
		}
		schema := body[start:end]
		if !strings.Contains(schema, "additionalProperties: false") || !strings.Contains(schema, "required:") {
			t.Fatalf("schema %s is not strict and required-field-bearing: %s", bound.name, schema)
		}
	}
	for _, required := range []string{
		"recent_observed_handoffs:",
		"provider_mor_acknowledgement_required:",
		"current_terms_owner_verified:",
		"result_selection_suppressed:",
		"action_interest_suppressed:",
		fmt.Sprintf("topic_receipt_threshold: { type: integer, enum: [%d] }", models.ProviderDemandPrivacyThreshold),
		fmt.Sprintf("result_selection_receipt_threshold: { type: integer, enum: [%d] }", models.ProviderDemandPrivacyThreshold),
		fmt.Sprintf("action_interest_receipt_threshold: { type: integer, enum: [%d] }", models.ProviderDemandPrivacyThreshold),
	} {
		if !strings.Contains(providerSchemas, required) {
			t.Fatalf("provider status/demand schemas missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"\n        attribution_token:",
		"\n        search_receipt_id:",
		"\n        query:",
		"\n        contact:",
		"\n        action_url:",
		"\n        provider_api_key_id:",
		"\n        company_hash:",
	} {
		if strings.Contains(providerSchemas, forbidden) {
			t.Fatalf("provider status/demand schemas expose forbidden field %q: %s", forbidden, providerSchemas)
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
