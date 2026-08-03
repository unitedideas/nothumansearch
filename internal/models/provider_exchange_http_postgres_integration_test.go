package models_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/handlers"
	"github.com/unitedideas/nothumansearch/internal/models"
)

type loopbackTXTResolver struct {
	records map[string][]string
}

func (resolver *loopbackTXTResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	values := resolver.records[name]
	if len(values) == 0 {
		return nil, fmt.Errorf("fixture TXT record unavailable")
	}
	return append([]string(nil), values...), nil
}

// exerciseProviderExchangeHTTPLoopback proves the value-bearing pilot path at
// the actual JSON/HTTP boundary while reusing the one disposable PostgreSQL
// database owned by TestProviderExchangePostgresReleaseRegressions. It never
// contacts production and never prints returned-once credentials or bearers.
func exerciseProviderExchangeHTTPLoopback(t *testing.T, db *sql.DB) {
	t.Helper()
	t.Run("http_loopback_terms_handoff_outcome", func(t *testing.T) {
		const (
			signingKeyID = "pg-release-v1"
			adminKey     = "loopback-admin-key-not-production"
		)
		t.Setenv("NHS_PROVIDER_EXCHANGE_SIGNING_KEY_ID", signingKeyID)
		t.Setenv("NHS_PROVIDER_EXCHANGE_SIGNING_KEY", strings.Repeat("s", 32))
		t.Setenv("NHS_PROVIDER_EXCHANGE_PREVIOUS_SIGNING_KEYS_JSON", "")
		t.Setenv("ADMIN_API_KEY", adminKey)

		// The Stage 2 cohort is frozen at activation. Consume the loopback
		// provider that the parent PostgreSQL fixture enrolled beforehand, then
		// exercise every value-bearing step from offer creation onward over HTTP.
		provider := createPostgresCommercialProvider(t, db, "http-loopback")
		domain := provider.site.Domain
		if _, err := db.Exec(`
			UPDATE sites
			SET name='Loopbackneedle Provider',
			    description='Disposable HTTP provider exchange fixture.',
			    has_mcp_server=true,
			    mcp_endpoint='https://http-loopback.example/mcp',
			    agentic_score=100,
			    created_at=clock_timestamp()
			WHERE id=$1::uuid`, provider.site.ID); err != nil {
			t.Fatalf("prepare loopback provider search fixture: %v", err)
		}
		session, err := models.CreateSession(db, provider.accountID)
		if err != nil {
			t.Fatalf("create loopback provider session: %v", err)
		}

		auth := handlers.NewAuthService(db, "http://loopback.invalid")
		templatesDir, err := filepath.Abs(filepath.Join("..", "..", "templates"))
		if err != nil {
			t.Fatalf("resolve loopback templates: %v", err)
		}
		exchange, err := handlers.NewProviderExchangeHandler(db, "http://loopback.invalid", auth, templatesDir)
		if err != nil {
			t.Fatalf("construct loopback provider exchange: %v", err)
		}
		resolver := &loopbackTXTResolver{records: map[string][]string{}}
		exchange.TXTResolver = resolver
		api := handlers.NewAPIHandler(db)
		api.Auth = auth
		api.ProviderExchangeEnabled = true
		mcp := handlers.NewMCPHandler(db, "http://loopback.invalid")
		mcp.ProviderExchange = exchange
		mcp.ActionInterests = handlers.NewActionInterestHandler(db, "http://loopback.invalid")
		mcp.ProviderExchangeEnabled = true

		mux := http.NewServeMux()
		mux.Handle("/mcp", mcp)
		mux.HandleFunc("/api/v1/search", api.Search)
		mux.HandleFunc("/api/v1/provider/claims", exchange.Claims)
		mux.HandleFunc("/api/v1/provider/claims/", exchange.ClaimAction)
		mux.HandleFunc("/api/v1/provider/offers", exchange.Offers)
		mux.HandleFunc("/api/v1/provider/offers/", exchange.OfferAction)
		mux.HandleFunc("/api/v1/provider/commercial-acceptances", exchange.ProviderCommercialAcceptances)
		mux.HandleFunc("/api/v1/provider/pilot-status", exchange.ProviderPilotStatus)
		mux.HandleFunc("/api/v1/provider/demand", exchange.ProviderDemand)
		mux.HandleFunc("/api/v1/provider/action-tickets/resolve", exchange.ResolveProviderControlledIntent)
		mux.HandleFunc("/api/v1/provider/outcomes", exchange.ProviderOutcomes)
		mux.HandleFunc("/api/v1/provider/receipts/", exchange.ProviderReceipt)
		mux.HandleFunc("/api/v1/action-tickets", exchange.ActionTickets)
		mux.HandleFunc("/api/v1/action-tickets/handoff", exchange.ActionTicketHandoff)
		mux.HandleFunc("/api/v1/action-receipts/verify", exchange.VerifyOutcomeReceipt)
		mux.HandleFunc("/api/v1/admin/provider-offers/action", exchange.AdminOfferAction)
		mux.HandleFunc("/api/v1/admin/provider-commercial/action", exchange.AdminCommercialAction)
		mux.HandleFunc("/api/v1/admin/provider-pilot-queue", exchange.AdminProviderPilotQueue)
		mux.HandleFunc("/api/v1/admin/provider-pilot-review", exchange.AdminProviderPilotReview)
		server := httptest.NewServer(mux)
		defer server.Close()
		exchange.BaseURL = server.URL
		auth.BaseURL = server.URL
		api.BaseURL = server.URL
		mcp.BaseURL = server.URL
		mcp.ActionInterests.BaseURL = server.URL
		client := server.Client()
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}

		sessionHeaders := map[string]string{
			"Cookie": "nhs_session=" + session,
			"Origin": server.URL,
		}
		claimID := provider.claim.ID
		providerKey := provider.rawKey
		if providerKey == "" {
			t.Fatal("pre-enrolled loopback provider key fixture is unavailable")
		}
		providerHeaders := map[string]string{"X-NHS-Provider-Key": providerKey}
		adminHeaders := map[string]string{"Authorization": "Bearer " + adminKey}
		loopbackRecordPilotReview(
			t, client, server.URL, adminHeaders, "provider", claimID, true,
		)

		termsCap := int64(25_000)
		termsDays := 30
		offerCreated := loopbackExpectJSON(t, client, "create terms offer", http.MethodPost,
			server.URL+"/api/v1/provider/offers",
			map[string]any{
				"claim_id": claimID, "name": "Start a loopback API trial",
				"summary":     "A bounded disposable action used only by the PostgreSQL loopback test.",
				"action_type": "trial", "action_url": "https://" + domain + "/start",
				"charge_event": "accepted", "bounty_cents": 2500, "currency": "usd",
				"principal_price_mode": "free", "principal_price_cents": 0,
				"principal_currency": "usd", "billing_mode": "terms",
				"terms_credit_limit_cents": termsCap, "terms_period_days": termsDays,
			}, sessionHeaders, http.StatusCreated)
		offer := loopbackObject(t, offerCreated, "offer")
		offerID := loopbackString(t, offer, "id")
		offerVersion := loopbackInt(t, offer, "version")
		exactTermsHash := loopbackString(t, offer, "commercial_terms_sha256")
		if loopbackBool(t, offer, "provider_acknowledges_merchant_of_record") {
			t.Fatal("draft offer overstated provider Merchant-of-Record acceptance")
		}
		if !loopbackBool(t, offer, "provider_mor_acknowledgement_required") {
			t.Fatal("draft offer omitted required Merchant-of-Record acknowledgement")
		}

		termsAccepted := loopbackExpectJSON(t, client, "accept exact terms", http.MethodPost,
			server.URL+"/api/v1/provider/commercial-acceptances",
			map[string]any{
				"event_type": "terms_acceptance", "offer_id": offerID,
				"offer_version": offerVersion, "exact_terms_sha256": exactTermsHash,
				"provider_acceptance_reference": "loopback-terms-acceptance-0001",
			}, loopbackMergeHeaders(providerHeaders, map[string]string{
				"Idempotency-Key": "loopback-terms-idempotency-0001",
			}), http.StatusCreated)
		termsAcceptance := loopbackObject(t, termsAccepted, "acceptance")
		termsAcceptanceID := loopbackString(t, termsAcceptance, "id")
		acceptedAt := loopbackString(t, termsAcceptance, "provider_accepted_at")
		pendingTerms := loopbackExpectJSON(t, client, "read pending terms queue", http.MethodGet,
			server.URL+"/api/v1/admin/provider-pilot-queue?state=pending_terms",
			nil, adminHeaders, http.StatusOK)
		loopbackRequireQueueItem(t, pendingTerms, "pending_terms", "acceptance_event_id", termsAcceptanceID)
		verifiedTerms := loopbackExpectJSON(t, client, "verify exact terms", http.MethodPost,
			server.URL+"/api/v1/admin/provider-commercial/action",
			map[string]any{
				"action": "verify_terms", "offer_id": offerID,
				"provider_acceptance_event_id": termsAcceptanceID,
				"source_system":                "loopback_test", "source_event_id": "loopback-terms-source-0001",
				"source_effective_at":      acceptedAt,
				"operator_reference":       "operator:loopback-terms-0001",
				"owner_evidence_reference": "evidence:loopback-terms-0001",
			}, adminHeaders, http.StatusCreated)
		initialCommitmentID := loopbackString(t, loopbackObject(t, verifiedTerms, "commitment"), "id")
		loopbackRecordPilotReview(
			t, client, server.URL, adminHeaders, "offer", offerID, false,
		)
		activationQueue := loopbackExpectJSON(t, client, "read activation review queue", http.MethodGet,
			server.URL+"/api/v1/admin/provider-pilot-queue?state=activation_review",
			nil, adminHeaders, http.StatusOK)
		loopbackRequireQueueItem(t, activationQueue, "activation_review", "offer_id", offerID)
		activated := loopbackExpectJSON(t, client, "activate terms offer", http.MethodPost,
			server.URL+"/api/v1/admin/provider-offers/action",
			map[string]any{
				"action": "activate", "offer_id": offerID,
				"operator_reference": "operator:loopback-activate-0001",
				"evidence_reference": "evidence:loopback-activate-0001",
			}, adminHeaders, http.StatusOK)
		if loopbackString(t, loopbackObject(t, activated, "offer"), "status") != "active" {
			t.Fatal("loopback offer did not activate")
		}

		exerciseProviderExchangeMCPDiscoverySurfaces(
			t, db, client, server.URL, domain, offerID,
		)

		search := loopbackExpectJSON(t, client, "free search with paid sidecar", http.MethodGet,
			server.URL+"/api/v1/search?q=loopbackneedle&category=developer", nil, nil, http.StatusOK)
		if loopbackString(t, search, "access") != "free" {
			t.Fatal("loopback search was not free")
		}
		results := loopbackArray(t, search, "results")
		if len(results) != 1 || loopbackString(t, loopbackObjectValue(t, results[0]), "domain") != domain {
			t.Fatal("loopback search did not return the exact organic provider")
		}
		paidOffers := loopbackArray(t, search, "paid_offers")
		if len(paidOffers) != 1 {
			t.Fatalf("loopback paid sidecar count = %d, want 1", len(paidOffers))
		}
		publicOffer := loopbackObjectValue(t, paidOffers[0])
		if loopbackString(t, publicOffer, "id") != offerID || loopbackInt(t, publicOffer, "organic_position") != 1 ||
			!loopbackBool(t, publicOffer, "provider_acknowledges_merchant_of_record") {
			t.Fatal("loopback paid sidecar lost exact active-offer evidence")
		}
		searchID := loopbackString(t, search, "search_id")
		providerDemand := loopbackExpectJSON(t, client, "read claim-scoped provider demand", http.MethodGet,
			server.URL+"/api/v1/provider/demand?days=30", nil, providerHeaders, http.StatusOK)
		demand := loopbackObject(t, providerDemand, "demand")
		if loopbackString(t, demand, "domain") != domain {
			t.Fatal("provider demand was not derived from the key claim")
		}
		demandSummary := loopbackObject(t, demand, "summary")
		if !loopbackBool(t, demandSummary, "result_selection_suppressed") ||
			demandSummary["result_selections"] != nil || demandSummary["result_selection_rate"] != nil {
			t.Fatal("thin provider selection cohort was not suppressed")
		}
		pilotStatus := loopbackExpectJSON(t, client, "read provider pilot status", http.MethodGet,
			server.URL+"/api/v1/provider/pilot-status", nil, providerHeaders, http.StatusOK)
		status := loopbackObject(t, pilotStatus, "pilot_status")
		if loopbackString(t, status, "domain") != domain ||
			!loopbackBool(t, status, "company_owner_verified") {
			t.Fatal("provider pilot status lost claim/company verification state")
		}
		statusOffers := loopbackArray(t, status, "offers")
		if len(statusOffers) != 1 {
			t.Fatalf("provider pilot status offer count = %d, want 1", len(statusOffers))
		}
		statusOffer := loopbackObjectValue(t, statusOffers[0])
		if loopbackString(t, statusOffer, "offer_id") != offerID ||
			loopbackString(t, statusOffer, "status") != "active" ||
			!loopbackBool(t, statusOffer, "latest_acceptance_owner_verified") ||
			!loopbackBool(t, statusOffer, "current_terms_owner_verified") {
			t.Fatal("provider pilot status lost exact accepted and verified terms")
		}

		prepared := loopbackExpectJSON(t, client, "prepare action ticket", http.MethodPost,
			server.URL+"/api/v1/action-tickets",
			map[string]any{
				"offer_id": offerID, "search_id": searchID, "demand_topic": "developer-tools",
				"principal_consent": true, "consent_version": models.ProviderPrincipalConsentV1,
			}, nil, http.StatusCreated)
		if _, leaked := prepared["action_url"]; leaked {
			t.Fatal("ticket preparation leaked provider action URL")
		}
		ticket := loopbackObject(t, prepared, "ticket")
		if _, leaked := ticket["action_url"]; leaked {
			t.Fatal("ticket JSON leaked provider action URL")
		}
		ticketID := loopbackString(t, ticket, "id")
		attributionToken := loopbackString(t, prepared, "attribution_token")
		loopbackRecordPilotReview(
			t, client, server.URL, adminHeaders, "ticket", ticketID, false,
		)

		handoff := loopbackExpectJSON(t, client, "handoff without optional intent", http.MethodPost,
			server.URL+"/api/v1/action-tickets/handoff",
			map[string]any{
				"ticket_id": ticketID, "attribution_token": attributionToken,
				"principal_handoff_consent": true,
				"handoff_consent_version":   models.ProviderActionHandoffConsentV1,
			}, nil, http.StatusCreated)
		handoffReceipt := loopbackObject(t, handoff, "handoff_receipt")
		handoffReceiptID := loopbackString(t, handoffReceipt, "id")
		loopbackRecordPilotReview(
			t, client, server.URL, adminHeaders, "handoff", handoffReceiptID, false,
		)
		if loopbackBool(t, handoffReceipt, "principal_controlled_intent_disclosure_consent") {
			t.Fatal("omitted optional controlled-intent consent was recorded as granted")
		}
		actionURL := loopbackString(t, handoff, "action_url")
		parsedActionURL, err := url.Parse(actionURL)
		if err != nil || parsedActionURL.Query().Get("nhs_attribution") != attributionToken {
			t.Fatal("handoff did not return the exact attributed provider URL")
		}
		awaitingCallback := loopbackExpectJSON(t, client, "read handoff callback queue", http.MethodGet,
			server.URL+"/api/v1/admin/provider-pilot-queue?state=handoff_awaiting_callback",
			nil, adminHeaders, http.StatusOK)
		loopbackRequireQueueItem(t, awaitingCallback, "handoff_awaiting_callback", "ticket_id", ticketID)

		loopbackExpectJSON(t, client, "reject unconsented intent resolution", http.MethodPost,
			server.URL+"/api/v1/provider/action-tickets/resolve",
			map[string]any{"attribution_token": attributionToken}, providerHeaders, http.StatusNotFound)

		outcomeHeaders := loopbackMergeHeaders(providerHeaders, map[string]string{
			"Idempotency-Key": "loopback-outcome-idempotency-0001",
		})
		outcomeBody := map[string]any{"attribution_token": attributionToken, "outcome": "accepted"}
		outcome := loopbackExpectJSON(t, client, "token-only accepted outcome", http.MethodPost,
			server.URL+"/api/v1/provider/outcomes", outcomeBody, outcomeHeaders, http.StatusCreated)
		receipt := loopbackObject(t, outcome, "receipt")
		receiptID := loopbackString(t, receipt, "id")
		loopbackRecordPilotReview(
			t, client, server.URL, adminHeaders, "callback", receiptID, false,
		)
		if loopbackString(t, receipt, "action_ticket_id") != ticketID ||
			loopbackString(t, receipt, "outcome") != "accepted" ||
			loopbackString(t, receipt, "charge_status") != "charged" ||
			loopbackInt64(t, receipt, "billed_cents") != 2500 ||
			loopbackBool(t, outcome, "principal_charged") {
			t.Fatal("token-only outcome did not create the expected provider-funded receipt")
		}
		replayed := loopbackExpectJSON(t, client, "replay token-only accepted outcome", http.MethodPost,
			server.URL+"/api/v1/provider/outcomes", outcomeBody, outcomeHeaders, http.StatusOK)
		if loopbackString(t, loopbackObject(t, replayed, "receipt"), "id") != receiptID ||
			!loopbackBool(t, replayed, "idempotent_replay") {
			t.Fatal("token-only outcome replay did not return the original receipt")
		}
		recentCallback := loopbackExpectJSON(t, client, "read recent callback queue", http.MethodGet,
			server.URL+"/api/v1/admin/provider-pilot-queue?state=recent_callback",
			nil, adminHeaders, http.StatusOK)
		loopbackRequireQueueItem(t, recentCallback, "recent_callback", "outcome_receipt_id", receiptID)
		pilotStatus = loopbackExpectJSON(t, client, "read provider handoff status", http.MethodGet,
			server.URL+"/api/v1/provider/pilot-status", nil, providerHeaders, http.StatusOK)
		status = loopbackObject(t, pilotStatus, "pilot_status")
		recentHandoffs := loopbackArray(t, status, "recent_observed_handoffs")
		if len(recentHandoffs) != 1 ||
			loopbackString(t, loopbackObjectValue(t, recentHandoffs[0]), "outcome_receipt_id") != receiptID {
			t.Fatal("provider pilot status did not expose the observed handoff outcome")
		}

		stored := loopbackExpectJSON(t, client, "read provider outcome receipt", http.MethodGet,
			server.URL+"/api/v1/provider/receipts/"+url.PathEscape(receiptID), nil,
			providerHeaders, http.StatusOK)
		storedReceipt := loopbackObject(t, stored, "receipt")
		if loopbackString(t, storedReceipt, "id") != receiptID {
			t.Fatal("provider receipt lookup returned a different receipt")
		}
		verifiedReceipt := loopbackExpectJSON(t, client, "verify signed outcome receipt", http.MethodPost,
			server.URL+"/api/v1/action-receipts/verify",
			map[string]any{
				"signed_receipt": loopbackString(t, receipt, "signed_receipt"),
				"signature":      loopbackString(t, receipt, "signature"),
			}, nil, http.StatusOK)
		if !loopbackBool(t, verifiedReceipt, "signature_valid") ||
			!loopbackBool(t, verifiedReceipt, "within_validity_window") ||
			!loopbackBool(t, verifiedReceipt, "current_state_available") {
			t.Fatal("signed outcome receipt did not verify against current state")
		}
		currentState := loopbackObject(t, verifiedReceipt, "current_state")
		if loopbackString(t, currentState, "current_ticket_status") != "accepted" ||
			loopbackInt64(t, currentState, "net_commercial_effect_cents") != 2500 {
			t.Fatal("verified outcome receipt reported the wrong current commercial state")
		}

		renewedTerms := loopbackExpectJSON(t, client, "extend exact terms", http.MethodPost,
			server.URL+"/api/v1/provider/commercial-acceptances",
			map[string]any{
				"event_type": "terms_renewal", "offer_id": offerID,
				"offer_version": offerVersion, "exact_terms_sha256": exactTermsHash,
				"related_acceptance_event_id":   termsAcceptanceID,
				"provider_acceptance_reference": "loopback-terms-renewal-0001",
			}, loopbackMergeHeaders(providerHeaders, map[string]string{
				"Idempotency-Key": "loopback-terms-renewal-idempotency-0001",
			}), http.StatusCreated)
		renewalAcceptance := loopbackObject(t, renewedTerms, "acceptance")
		renewalAcceptanceID := loopbackString(t, renewalAcceptance, "id")
		renewalAcceptedAt := loopbackString(t, renewalAcceptance, "provider_accepted_at")
		pendingRenewal := loopbackExpectJSON(t, client, "read pending terms renewal queue", http.MethodGet,
			server.URL+"/api/v1/admin/provider-pilot-queue?state=pending_terms",
			nil, adminHeaders, http.StatusOK)
		loopbackRequireQueueItem(t, pendingRenewal, "pending_terms", "acceptance_event_id", renewalAcceptanceID)
		loopbackRequireQueueItem(t, pendingRenewal, "pending_terms", "related_commitment_event_id", initialCommitmentID)
		verifiedRenewal := loopbackExpectJSON(t, client, "verify exact terms renewal", http.MethodPost,
			server.URL+"/api/v1/admin/provider-commercial/action",
			map[string]any{
				"action": "verify_terms", "offer_id": offerID,
				"provider_acceptance_event_id": renewalAcceptanceID,
				"related_commitment_event_id":  initialCommitmentID,
				"source_system":                "loopback_test", "source_event_id": "loopback-renewal-source-0001",
				"source_effective_at":      renewalAcceptedAt,
				"operator_reference":       "operator:loopback-renewal-0001",
				"owner_evidence_reference": "evidence:loopback-renewal-0001",
			}, adminHeaders, http.StatusCreated)
		renewalCommitment := loopbackObject(t, verifiedRenewal, "commitment")
		if loopbackString(t, renewalCommitment, "event_type") != "terms_renewal" ||
			loopbackString(t, renewalCommitment, "related_event_id") != initialCommitmentID {
			t.Fatal("verified renewal lost the exact preceding owner commitment")
		}
	})
}

func exerciseProviderExchangeMCPDiscoverySurfaces(
	t *testing.T,
	db *sql.DB,
	client *http.Client,
	baseURL, providerDomain, offerID string,
) {
	t.Helper()

	topExpected, err := models.GetTopSites(db, "developer", 1)
	if err != nil {
		t.Fatalf("read expected category-bound top sites: %v", err)
	}
	requireLoopbackProviderFirst(t, "category-bound top sites", topExpected, providerDomain)
	top := loopbackMCPCall(t, client, baseURL, "get_top_sites", map[string]any{
		"category": "developer",
		"limit":    1,
	})
	loopbackRequireMCPDiscoveryExchange(t, "category-bound top sites", top, topExpected, providerDomain, offerID)

	recentExpected, err := models.GetRecentSites(db, 90, 1, "developer")
	if err != nil {
		t.Fatalf("read expected category-bound recent sites: %v", err)
	}
	requireLoopbackProviderFirst(t, "category-bound recent sites", recentExpected, providerDomain)
	recent := loopbackMCPCall(t, client, baseURL, "recent_additions", map[string]any{
		"category": "developer",
		"days":     90,
		"limit":    1,
	})
	loopbackRequireMCPDiscoveryExchange(t, "category-bound recent sites", recent, recentExpected, providerDomain, offerID)

	mcpParams := models.SearchParams{
		Query:    "loopbackneedle",
		Category: "developer",
		HasMCP:   true,
		Limit:    20,
		Page:     1,
	}
	mcpExpected, _, err := models.SearchSites(db, mcpParams)
	if err != nil {
		t.Fatalf("read expected MCP-server discovery: %v", err)
	}
	requireLoopbackProviderFirst(t, "MCP-server discovery", mcpExpected, providerDomain)
	mcpServers := loopbackMCPCall(t, client, baseURL, "find_mcp_servers", map[string]any{
		"query":    mcpParams.Query,
		"category": mcpParams.Category,
		"limit":    mcpParams.Limit,
	})
	loopbackRequireMCPDiscoveryExchange(t, "MCP-server discovery", mcpServers, mcpExpected, providerDomain, offerID)

	unfilteredTopExpected, err := models.GetTopSites(db, "", 1)
	if err != nil {
		t.Fatalf("read expected unfiltered top sites: %v", err)
	}
	unfilteredTop := loopbackMCPCall(t, client, baseURL, "get_top_sites", map[string]any{"limit": 1})
	loopbackRequireMCPUnscopedBrowse(t, "unfiltered top sites", unfilteredTop, unfilteredTopExpected)

	unfilteredRecentExpected, err := models.GetRecentSites(db, 90, 1, "")
	if err != nil {
		t.Fatalf("read expected unfiltered recent sites: %v", err)
	}
	unfilteredRecent := loopbackMCPCall(t, client, baseURL, "recent_additions", map[string]any{
		"days":  90,
		"limit": 1,
	})
	loopbackRequireMCPUnscopedBrowse(t, "unfiltered recent sites", unfilteredRecent, unfilteredRecentExpected)
}

func requireLoopbackProviderFirst(t *testing.T, label string, sites []models.Site, providerDomain string) {
	t.Helper()
	if len(sites) == 0 || sites[0].Domain != providerDomain {
		t.Fatalf("%s fixture does not put %q first: %#v", label, providerDomain, sites)
	}
}

func loopbackMCPCall(
	t *testing.T,
	client *http.Client,
	baseURL, tool string,
	arguments map[string]any,
) map[string]any {
	t.Helper()
	response := loopbackExpectJSON(t, client, "MCP "+tool, http.MethodPost,
		baseURL+"/mcp",
		map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      tool,
				"arguments": arguments,
			},
		}, nil, http.StatusOK)
	if rpcError, exists := response["error"]; exists && rpcError != nil {
		t.Fatalf("MCP %s returned JSON-RPC error: %#v", tool, rpcError)
	}
	result := loopbackObject(t, response, "result")
	if isError, exists := result["isError"].(bool); exists && isError {
		t.Fatalf("MCP %s returned a tool error: %#v", tool, result)
	}
	return loopbackObject(t, result, "structuredContent")
}

func loopbackRequireMCPDiscoveryExchange(
	t *testing.T,
	label string,
	structured map[string]any,
	expected []models.Site,
	providerDomain, offerID string,
) {
	t.Helper()
	if loopbackString(t, structured, "access") != "free" {
		t.Fatalf("%s was not free", label)
	}
	if !loopbackBool(t, structured, "receipt_recorded") {
		t.Fatalf("%s did not record an eligible discovery receipt", label)
	}
	_ = loopbackString(t, structured, "search_id")
	loopbackRequireMCPOrganicOrder(t, label, structured, expected)

	actionInterest := loopbackObject(t, structured, "action_interest")
	if !loopbackBool(t, actionInterest, "available") ||
		loopbackBool(t, actionInterest, "provider_contacted") ||
		loopbackBool(t, actionInterest, "commercial_proof") ||
		loopbackBool(t, actionInterest, "organic_rank_affected") {
		t.Fatalf("%s action-interest guidance overstated or suppressed value: %#v", label, actionInterest)
	}
	if !loopbackBool(t, structured, "paid_offers_available") {
		t.Fatalf("%s suppressed the active provider-funded sidecar", label)
	}
	paidOffers := loopbackArray(t, structured, "paid_offers")
	if len(paidOffers) != 1 {
		t.Fatalf("%s paid sidecar count = %d, want 1", label, len(paidOffers))
	}
	paidOffer := loopbackObjectValue(t, paidOffers[0])
	position := 0
	for index, site := range expected {
		if site.Domain == providerDomain {
			position = index + 1
			break
		}
	}
	if position == 0 ||
		loopbackString(t, paidOffer, "id") != offerID ||
		loopbackString(t, paidOffer, "provider_domain") != providerDomain ||
		loopbackInt(t, paidOffer, "organic_position") != position {
		t.Fatalf("%s sidecar was not tied to the unchanged organic position: %#v", label, paidOffer)
	}
}

func loopbackRequireMCPUnscopedBrowse(
	t *testing.T,
	label string,
	structured map[string]any,
	expected []models.Site,
) {
	t.Helper()
	if loopbackString(t, structured, "access") != "free" {
		t.Fatalf("%s was not free", label)
	}
	if loopbackBool(t, structured, "receipt_recorded") {
		t.Fatalf("%s incorrectly recorded a Stage 1 receipt", label)
	}
	if _, exists := structured["search_id"]; exists {
		t.Fatalf("%s exposed a search_id without an eligible receipt", label)
	}
	if loopbackBool(t, structured, "paid_offers_available") || len(loopbackArray(t, structured, "paid_offers")) != 0 {
		t.Fatalf("%s disclosed provider-funded sidecars without scoped discovery", label)
	}
	actionInterest := loopbackObject(t, structured, "action_interest")
	if loopbackBool(t, actionInterest, "available") {
		t.Fatalf("%s exposed action interest without scoped discovery", label)
	}
	loopbackRequireMCPOrganicOrder(t, label, structured, expected)
}

func loopbackRequireMCPOrganicOrder(
	t *testing.T,
	label string,
	structured map[string]any,
	expected []models.Site,
) {
	t.Helper()
	results := loopbackArray(t, structured, "results")
	if len(results) != len(expected) {
		t.Fatalf("%s organic result count = %d, want %d", label, len(results), len(expected))
	}
	for index, site := range expected {
		result := loopbackObjectValue(t, results[index])
		if loopbackString(t, result, "domain") != site.Domain {
			t.Fatalf("%s organic position %d changed: got %q want %q",
				label, index+1, result["domain"], site.Domain)
		}
	}
}

func loopbackRecordPilotReview(
	t *testing.T,
	client *http.Client,
	baseURL string,
	adminHeaders map[string]string,
	reviewType, subjectID string,
	replay bool,
) {
	t.Helper()
	query := url.Values{
		"pilot_id":    {postgresProviderPilotEpochID},
		"review_type": {reviewType},
		"subject_id":  {subjectID},
	}.Encode()
	candidateResponse := loopbackExpectJSON(
		t, client, "preview "+reviewType+" pilot review", http.MethodGet,
		baseURL+"/api/v1/admin/provider-pilot-review?"+query,
		nil, adminHeaders, http.StatusOK,
	)
	candidate := loopbackObject(t, candidateResponse, "review_candidate")
	if loopbackString(t, candidate, "review_contract_version") != models.ProviderPilotReviewContractV1 ||
		loopbackString(t, candidate, "provider_pilot_epoch_id") != postgresProviderPilotEpochID ||
		loopbackString(t, candidate, "provider_pilot_contract_version") != models.ProviderPilotEpochContractV1 ||
		loopbackString(t, candidate, "pilot_demand_topic") != "developer-tools" ||
		loopbackString(t, candidate, "review_type") != reviewType ||
		loopbackString(t, candidate, "subject_id") != subjectID {
		t.Fatalf("%s loopback review candidate lost exact identity", reviewType)
	}
	digest := loopbackString(t, candidate, "subject_snapshot_sha256")
	if !providerHashPatternForPostgresTest(digest) {
		t.Fatalf("%s loopback review digest=%q", reviewType, digest)
	}
	encoded, err := json.Marshal(candidate)
	if err != nil {
		t.Fatalf("encode %s loopback review candidate: %v", reviewType, err)
	}
	for _, forbidden := range []string{
		`"search_receipt`, `"query`, `"attribution_token`, `"token_hash`,
		`"company_key_hash`, `"signed_receipt`, `"signature`,
	} {
		if strings.Contains(strings.ToLower(string(encoded)), forbidden) {
			t.Fatalf("%s loopback review candidate exposed %q", reviewType, forbidden)
		}
	}

	payload := map[string]any{
		"provider_pilot_epoch_id":  postgresProviderPilotEpochID,
		"review_type":              reviewType,
		"subject_id":               subjectID,
		"expected_snapshot_sha256": digest,
		"owner_reference":          "owner:loopback-review:" + reviewType,
		"evidence_reference":       "evidence:loopback-review:" + reviewType,
	}
	wantStatus := http.StatusCreated
	wantCreated := true
	wantReplay := false
	if replay {
		if loopbackString(t, candidate, "existing_review_id") == "" {
			t.Fatalf("%s loopback review candidate omitted existing review", reviewType)
		}
		wantStatus = http.StatusOK
		wantCreated = false
		wantReplay = true
	}
	recorded := loopbackExpectJSON(
		t, client, "record "+reviewType+" pilot review", http.MethodPost,
		baseURL+"/api/v1/admin/provider-pilot-review",
		payload, adminHeaders, wantStatus,
	)
	if loopbackBool(t, recorded, "created") != wantCreated ||
		loopbackBool(t, recorded, "idempotent_replay") != wantReplay ||
		loopbackBool(t, recorded, "commercial_proof_created") {
		t.Fatalf("%s loopback review overstated creation or commercial proof", reviewType)
	}
	review := loopbackObject(t, recorded, "review")
	if loopbackString(t, review, "provider_pilot_epoch_id") != postgresProviderPilotEpochID ||
		loopbackString(t, review, "review_type") != reviewType ||
		loopbackString(t, review, "subject_id") != subjectID ||
		loopbackString(t, review, "subject_snapshot_sha256") != digest {
		t.Fatalf("%s loopback review receipt lost exact snapshot identity", reviewType)
	}
	if replay {
		replayed := loopbackExpectJSON(
			t, client, "replay "+reviewType+" pilot review", http.MethodPost,
			baseURL+"/api/v1/admin/provider-pilot-review",
			payload, adminHeaders, http.StatusOK,
		)
		if loopbackBool(t, replayed, "created") ||
			!loopbackBool(t, replayed, "idempotent_replay") ||
			loopbackString(t, loopbackObject(t, replayed, "review"), "id") != loopbackString(t, review, "id") {
			t.Fatalf("%s loopback review replay was not exact", reviewType)
		}
	}
}

func loopbackExpectJSON(
	t *testing.T,
	client *http.Client,
	label, method, endpoint string,
	body any,
	headers map[string]string,
	wantStatus int,
) map[string]any {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("%s encode fixture request: %v", label, err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, endpoint, reader)
	if err != nil {
		t.Fatalf("%s construct request: %v", label, err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s execute request: %v", label, err)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 1<<20)
	decoder := json.NewDecoder(limited)
	decoder.UseNumber()
	result := map[string]any{}
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("%s decode response status %d: %v", label, response.StatusCode, err)
	}
	if response.StatusCode != wantStatus {
		errorMessage, _ := result["error"].(string)
		t.Fatalf("%s status=%d want=%d error=%q", label, response.StatusCode, wantStatus, errorMessage)
	}
	return result
}

func loopbackMergeHeaders(left, right map[string]string) map[string]string {
	merged := make(map[string]string, len(left)+len(right))
	for name, value := range left {
		merged[name] = value
	}
	for name, value := range right {
		merged[name] = value
	}
	return merged
}

func loopbackRequireQueueItem(
	t *testing.T,
	response map[string]any,
	state, field, want string,
) {
	t.Helper()
	queue := loopbackObject(t, response, "queue")
	for _, raw := range loopbackArray(t, queue, "items") {
		item := loopbackObjectValue(t, raw)
		gotState, _ := item["state"].(string)
		gotValue, _ := item[field].(string)
		if gotState == state && gotValue == want {
			return
		}
	}
	t.Fatalf("queue state %q did not contain expected opaque %s", state, field)
}

func loopbackObject(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := object[key]
	if !ok {
		t.Fatalf("response missing object field %q", key)
	}
	return loopbackObjectValue(t, value)
}

func loopbackObjectValue(t *testing.T, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("response field has type %T, want object", value)
	}
	return object
}

func loopbackArray(t *testing.T, object map[string]any, key string) []any {
	t.Helper()
	value, ok := object[key].([]any)
	if !ok {
		t.Fatalf("response field %q has type %T, want array", key, object[key])
	}
	return value
}

func loopbackString(t *testing.T, object map[string]any, key string) string {
	t.Helper()
	value, ok := object[key].(string)
	if !ok || value == "" {
		t.Fatalf("response field %q has type %T or is empty", key, object[key])
	}
	return value
}

func loopbackBool(t *testing.T, object map[string]any, key string) bool {
	t.Helper()
	value, ok := object[key].(bool)
	if !ok {
		t.Fatalf("response field %q has type %T, want boolean", key, object[key])
	}
	return value
}

func loopbackInt(t *testing.T, object map[string]any, key string) int {
	t.Helper()
	return int(loopbackInt64(t, object, key))
}

func loopbackInt64(t *testing.T, object map[string]any, key string) int64 {
	t.Helper()
	number, ok := object[key].(json.Number)
	if !ok {
		t.Fatalf("response field %q has type %T, want number", key, object[key])
	}
	value, err := number.Int64()
	if err != nil {
		t.Fatalf("response field %q is not an integer: %v", key, err)
	}
	return value
}
