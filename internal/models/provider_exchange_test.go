package models

import (
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func int64Pointer(value int64) *int64 { return &value }
func intPointer(value int) *int       { return &value }

func validOfferFixture() ProviderOfferInput {
	return ProviderOfferInput{
		OfferName:           "Start an API sandbox",
		OfferSummary:        "Create a sandbox workspace with machine-readable setup instructions.",
		ActionType:          "trial",
		ActionURL:           "https://provider.example/start",
		ChargeEvent:         "activated",
		BountyCents:         2500,
		Currency:            "usd",
		PrincipalPriceMode:  "free",
		PrincipalPriceCents: int64Pointer(0),
		PrincipalCurrency:   "usd",
		BillingMode:         "prepaid",
	}
}

func TestProviderSecretsAndIDsAreOpaqueAndNonDeterministic(t *testing.T) {
	first, _, err := newProviderSecret("nhs_provider")
	if err != nil {
		t.Fatalf("newProviderSecret first: %v", err)
	}
	second, _, err := newProviderSecret("nhs_provider")
	if err != nil {
		t.Fatalf("newProviderSecret second: %v", err)
	}
	if first == second {
		t.Fatal("provider secret generator returned a duplicate")
	}
	if hash := HashProviderSecret(first); !providerHashPattern.MatchString(hash) || strings.Contains(hash, first) {
		t.Fatalf("provider secret hash has invalid persisted form %q", hash)
	}
	firstID, err := newProviderUUID()
	if err != nil {
		t.Fatalf("newProviderUUID first: %v", err)
	}
	secondID, err := newProviderUUID()
	if err != nil {
		t.Fatalf("newProviderUUID second: %v", err)
	}
	if !validProviderUUID(firstID) || firstID == secondID {
		t.Fatalf("opaque UUIDs = %q and %q", firstID, secondID)
	}
}

func TestProviderClaimVerificationFreshnessHasAClosedSevenDayBoundary(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	if !providerClaimVerificationFresh(now.Add(-ProviderClaimVerificationFreshness+time.Nanosecond), now) {
		t.Fatal("proof just inside the seven-day boundary was treated as stale")
	}
	if providerClaimVerificationFresh(now.Add(-ProviderClaimVerificationFreshness), now) {
		t.Fatal("proof exactly seven days old remained fresh")
	}
	if providerClaimVerificationFresh(time.Time{}, now) || providerClaimVerificationFresh(now, time.Time{}) {
		t.Fatal("zero timestamps were accepted as fresh proof")
	}
}

func TestProviderDNSTokenMatchRequiresThePersistedExactProof(t *testing.T) {
	raw := "nhs_claim_abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	if !providerDNSTokenMatches(HashProviderSecret(raw), []string{"unrelated-record", raw}) {
		t.Fatal("exact DNS proof did not match its persisted hash")
	}
	if providerDNSTokenMatches(HashProviderSecret(raw), []string{raw + "-different"}) {
		t.Fatal("different DNS proof matched")
	}
	if providerDNSTokenMatches("not-a-sha256-hash", []string{raw}) {
		t.Fatal("invalid persisted proof hash matched")
	}
	if providerDNSTokenMatches(HashProviderSecret(raw), []string{strings.Repeat("x", 129)}) {
		t.Fatal("oversized DNS proof was accepted")
	}
}

func TestProviderOfferValidationSeparatesBountyFromPrincipalPrice(t *testing.T) {
	valid := validOfferFixture()
	got, err := normalizeProviderOfferInput(valid)
	if err != nil {
		t.Fatalf("normalizeProviderOfferInput valid fixture: %v", err)
	}
	terms := validOfferFixture()
	terms.BillingMode = "terms"
	terms.TermsCreditLimitCents = int64Pointer(10000)
	terms.TermsPeriodDays = intPointer(30)
	if _, err := normalizeProviderOfferInput(terms); err != nil {
		t.Fatalf("bounded terms offer validation: %v", err)
	}
	terms.TermsCreditLimitCents = nil
	if _, err := normalizeProviderOfferInput(terms); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("unbounded terms offer error = %v", err)
	}
	if got.BountyCents != 2500 || got.PrincipalPriceCents == nil || *got.PrincipalPriceCents != 0 {
		t.Fatalf("commercial semantics changed: %+v", got)
	}

	tests := []struct {
		name   string
		mutate func(*ProviderOfferInput)
	}{
		{"HTTP action URL", func(in *ProviderOfferInput) { in.ActionURL = "http://provider.example/start" }},
		{"credentialed action URL", func(in *ProviderOfferInput) { in.ActionURL = "https://user:pass@provider.example/start" }},
		{"query-bearing action URL", func(in *ProviderOfferInput) { in.ActionURL = "https://provider.example/start?token=secret" }},
		{"normalized action URL expansion", func(in *ProviderOfferInput) { in.ActionURL = "https://provider.example/" + strings.Repeat("é", 400) }},
		{"fixed price without amount", func(in *ProviderOfferInput) { in.PrincipalPriceMode, in.PrincipalPriceCents = "fixed", nil }},
		{"non USD bounty", func(in *ProviderOfferInput) { in.Currency = "eur" }},
		{"non USD principal price", func(in *ProviderOfferInput) { in.PrincipalCurrency = "eur" }},
		{"oversized bounty", func(in *ProviderOfferInput) { in.BountyCents = ProviderBountyMaximumCents + 1 }},
		{"unknown charge event", func(in *ProviderOfferInput) { in.ChargeEvent = "clicked" }},
		{"newline in machine name", func(in *ProviderOfferInput) { in.OfferName = "Start\nnow" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validOfferFixture()
			test.mutate(&input)
			if _, err := normalizeProviderOfferInput(input); !errors.Is(err, ErrInvalidProviderExchange) {
				t.Fatalf("validation error = %v, want ErrInvalidProviderExchange", err)
			}
		})
	}
}

func TestProviderCommercialFieldsUseDistinctPilotCaps(t *testing.T) {
	if ProviderBountyMaximumCents != 1_000_000 ||
		ProviderPrincipalPriceMaximumCents != 100_000_000 ||
		ProviderTermsCreditMaximumCents != 10_000_000 ||
		ProviderMoneyMaximumCents != 100_000_000 {
		t.Fatal("provider pilot caps drifted from the reviewed commercial contract")
	}

	bounty := validOfferFixture()
	bounty.BountyCents = ProviderBountyMaximumCents
	if _, err := normalizeProviderOfferInput(bounty); err != nil {
		t.Fatalf("maximum bounty rejected: %v", err)
	}
	bounty.BountyCents++
	if _, err := normalizeProviderOfferInput(bounty); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("oversized bounty error = %v", err)
	}

	principal := validOfferFixture()
	principal.PrincipalPriceMode = "fixed"
	principal.PrincipalPriceCents = int64Pointer(ProviderPrincipalPriceMaximumCents)
	if _, err := normalizeProviderOfferInput(principal); err != nil {
		t.Fatalf("maximum principal fixed price rejected: %v", err)
	}
	principal.PrincipalPriceCents = int64Pointer(ProviderPrincipalPriceMaximumCents + 1)
	if _, err := normalizeProviderOfferInput(principal); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("oversized principal fixed price error = %v", err)
	}

	terms := validOfferFixture()
	terms.BillingMode = "terms"
	terms.TermsCreditLimitCents = int64Pointer(ProviderTermsCreditMaximumCents)
	terms.TermsPeriodDays = intPointer(30)
	if _, err := normalizeProviderOfferInput(terms); err != nil {
		t.Fatalf("maximum terms credit rejected: %v", err)
	}
	terms.TermsCreditLimitCents = int64Pointer(ProviderTermsCreditMaximumCents + 1)
	if _, err := normalizeProviderOfferInput(terms); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("oversized terms credit error = %v", err)
	}

	if value, err := parseProviderMoney("100000001"); err != nil || value != ProviderMoneyMaximumCents+1 {
		t.Fatalf("platform aggregate parser applied per-offer cap: value=%d err=%v", value, err)
	}
	if _, err := parseBoundedProviderMoney("100000001"); !errors.Is(err, ErrProviderBudgetLimit) {
		t.Fatalf("per-offer ledger bound error = %v", err)
	}
}

func TestActionTicketValidationAllowsOnlyControlledConsentFields(t *testing.T) {
	input := ActionTicketInput{
		ProviderOfferID:       "123e4567-e89b-42d3-a456-426614174000",
		SearchReceiptPublicID: "nhs_sr_opaque-receipt",
		DemandTopic:           "developer-tools",
		RegionCode:            "us-ca",
		BudgetBand:            "500_1999",
		Urgency:               "30_days",
		RequirementFlags:      []string{"mcp", "api_access", "mcp"},
		PrincipalConsent:      true,
		ConsentVersion:        ProviderPrincipalConsentV1,
	}
	got, err := normalizeActionTicketInput(input)
	if err != nil {
		t.Fatalf("normalizeActionTicketInput valid fixture: %v", err)
	}
	if got.RegionCode != "US-CA" || strings.Join(got.RequirementFlags, ",") != "api_access,mcp" {
		t.Fatalf("normalized controlled fields = %+v", got)
	}
	if got.TTL != ActionTicketDefaultTTL || got.TTL < 29*24*time.Hour {
		t.Fatalf("default attribution TTL = %v, want approximately 30 days", got.TTL)
	}

	input.DemandTopic = "buyer@example.com wants confidential zephyr service"
	if _, err := normalizeActionTicketInput(input); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("caller text demand topic error = %v", err)
	}
	input.DemandTopic = "developer-tools"
	input.PrincipalConsent = false
	if _, err := normalizeActionTicketInput(input); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("missing principal consent error = %v", err)
	}
}

func TestActionTicketStatusDoesNotRegressAndInvalidationWins(t *testing.T) {
	for _, test := range []struct {
		current string
		outcome string
		want    string
	}{
		{"accepted", "activated", "activated"},
		{"activated", "accepted", "activated"},
		{"converted", "activated", "converted"},
		{"converted", "invalid", "invalid"},
		{"duplicate", "converted", "duplicate"},
	} {
		if got := nextActionTicketStatus(test.current, test.outcome); got != test.want {
			t.Fatalf("nextActionTicketStatus(%q,%q) = %q, want %q", test.current, test.outcome, got, test.want)
		}
	}
}

func TestProviderChargeResolutionIsTheOnlyLateOutcome(t *testing.T) {
	for _, test := range []struct {
		outcome      string
		chargedCents int64
		want         bool
	}{
		{"invalid", 2500, true},
		{"duplicate", 2500, true},
		{"invalid", 0, false},
		{"accepted", 2500, false},
		{"converted", 2500, false},
	} {
		if got := providerChargeResolutionAllowed(test.outcome, test.chargedCents); got != test.want {
			t.Fatalf("providerChargeResolutionAllowed(%q,%d) = %t, want %t", test.outcome, test.chargedCents, got, test.want)
		}
	}
}

func TestProviderExchangeNilStoresFailClosed(t *testing.T) {
	if _, _, err := CreateProviderClaim(nil, 1, "123e4567-e89b-42d3-a456-426614174000"); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("CreateProviderClaim nil DB error = %v", err)
	}
	if _, err := ListProviderOffers(nil, 1, "123e4567-e89b-42d3-a456-426614174000"); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("ListProviderOffers nil DB error = %v", err)
	}
	if _, err := ResolveProviderAPIKeyForChargeResolution(nil, "", ""); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("ResolveProviderAPIKeyForChargeResolution nil inputs error = %v", err)
	}
	if _, _, _, err := CreateActionTicket(nil, ActionTicketInput{}, nil); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("CreateActionTicket nil DB/signer error = %v", err)
	}
	if _, _, err := RecordProviderOutcome(nil, nil, ProviderOutcomeInput{}, nil); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("RecordProviderOutcome nil inputs error = %v", err)
	}
	if _, err := ResolveActionTicketForChargeResolution(nil, "", ""); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("ResolveActionTicketForChargeResolution nil inputs error = %v", err)
	}
	if _, err := GetPublicOutcomeReceiptState(nil, "", ""); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("GetPublicOutcomeReceiptState nil inputs error = %v", err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	if _, err := LeaseDueProviderClaimDNSChecks(nil, now, 1); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("LeaseDueProviderClaimDNSChecks nil DB error = %v", err)
	}
	if _, err := CompleteProviderClaimDNSCheck(nil,
		"123e4567-e89b-42d3-a456-426614174000",
		"223e4567-e89b-42d3-a456-426614174000", nil, now,
	); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("CompleteProviderClaimDNSCheck nil DB error = %v", err)
	}
	if _, err := RecordProviderClaimDNSFailure(nil, 1,
		"123e4567-e89b-42d3-a456-426614174000", now,
	); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("RecordProviderClaimDNSFailure nil DB error = %v", err)
	}
}

func TestProviderExchangeMigrationEnforcesPrivacyOwnershipAndAccounting(t *testing.T) {
	source, err := os.ReadFile("../../migrations/019_provider_exchange.sql")
	if err != nil {
		t.Fatalf("read provider exchange migration: %v", err)
	}
	text := string(source)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS provider_claims",
		"verification_token_hash",
		"challenge_expires_at",
		"verification_last_succeeded_at",
		"verification_last_attempted_at",
		"verification_consecutive_failures",
		"verification_next_check_at",
		"verification_lease_id",
		"verification_lease_until",
		"idx_provider_claims_one_verified_site",
		"idx_provider_claims_one_pending_account_site",
		"idx_provider_claims_verification_due",
		"idx_provider_claims_verification_freshness",
		"CREATE TABLE IF NOT EXISTS provider_api_keys",
		"idx_provider_api_keys_one_active_claim",
		"CREATE TABLE IF NOT EXISTS provider_offers",
		"disclosure_label = 'Provider-funded action'",
		"principal_price_mode",
		"terms_credit_limit_cents",
		"terms_period_days",
		"attribution_key_id_snapshot",
		"octet_length(action_url) <= 1536",
		"octet_length(action_url_snapshot) <= 1536",
		"authorization_revoked_at",
		"CREATE TABLE IF NOT EXISTS action_tickets",
		"search_receipt_id     UUID REFERENCES search_receipts(id) ON DELETE SET NULL",
		"creation_request_hash",
		"offer_version_snapshot",
		"bounty_cents_snapshot",
		"principal_consent     BOOLEAN NOT NULL CHECK (principal_consent)",
		"CREATE TABLE IF NOT EXISTS provider_budget_ledger",
		"CREATE TABLE IF NOT EXISTS provider_offers_returned",
		"provider_offers_returned_no_update",
		"idx_provider_budget_one_charge_per_ticket",
		"idx_provider_budget_one_credit_per_ticket",
		"idx_provider_budget_unique_funding_reference",
		"provider_budget_ledger_no_update",
		"provider_budget_ledger_no_delete",
		"entry_type IN ('fund', 'adjustment') AND action_ticket_id IS NULL",
		"entry_type IN ('charge', 'credit') AND action_ticket_id IS NOT NULL",
		"CREATE TABLE IF NOT EXISTS outcome_receipts",
		"nhs_event_id",
		"idempotency_key_hash",
		"payload_hash",
		"charge_status IN ('charged', 'credited', 'none')",
		"charge_status = 'none' AND billed_cents = 0",
		"charge_status = 'charged' AND billed_cents > 0",
		"charge_status = 'credited' AND billed_cents > 0",
		"outcome_receipts_no_update",
		"outcome_receipts_no_delete",
		"signature ~ '^[A-Za-z0-9_-]{43}$'",
		"redact_action_ticket_intent_on_receipt_delete",
		"CREATE TABLE IF NOT EXISTS provider_admin_audit_events",
		"provider_admin_audit_no_update",
		"provider_admin_audit_no_delete",
		"currency                 TEXT NOT NULL CHECK (currency = 'usd')",
		"bounty_cents BETWEEN 1 AND 1000000",
		"principal_price_cents BETWEEN 0 AND 100000000",
		"terms_credit_limit_cents BETWEEN 1 AND 10000000",
		"bounty_cents_snapshot BETWEEN 1 AND 1000000",
		"principal_price_cents_snapshot BETWEEN 0 AND 100000000",
		"terms_credit_limit_cents_snapshot BETWEEN 1 AND 10000000",
		"amount_cents BETWEEN -100000000 AND 100000000",
		"billed_cents BETWEEN 0 AND 1000000",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("provider exchange migration missing %q", required)
		}
	}
	for _, forbiddenColumn := range []string{
		"raw_query TEXT", "query_text TEXT", "user_agent TEXT", "ip_hash TEXT",
		"contact_email TEXT", "contact_name TEXT", "principal_id TEXT",
		"agent_id TEXT", "idempotency_key       TEXT",
	} {
		if strings.Contains(text, forbiddenColumn) {
			t.Fatalf("provider exchange migration contains forbidden privacy field %q", forbiddenColumn)
		}
	}
	if strings.Contains(text, "$$") {
		t.Fatal("provider migration uses a procedural body incompatible with the repository migration splitter")
	}
	if strings.Contains(text, "100000000000") {
		t.Fatal("provider migration retains the superseded generic one-billion-dollar cap")
	}
}

func TestProviderDNSFreshnessSourceContracts(t *testing.T) {
	source, err := os.ReadFile("provider_exchange.go")
	if err != nil {
		t.Fatalf("read provider exchange model: %v", err)
	}
	text := string(source)
	section := func(startMarker, endMarker string) string {
		t.Helper()
		start := strings.Index(text, startMarker)
		if start < 0 {
			t.Fatalf("source marker not found: %s", startMarker)
		}
		endOffset := strings.Index(text[start+len(startMarker):], endMarker)
		if endOffset < 0 {
			t.Fatalf("source marker not found after %s: %s", startMarker, endMarker)
		}
		return text[start : start+len(startMarker)+endOffset]
	}
	requireAll := func(name, got string, required ...string) {
		t.Helper()
		for _, item := range required {
			if !strings.Contains(got, item) {
				t.Fatalf("%s missing %q", name, item)
			}
		}
	}

	verify := section("func VerifyProviderClaim", "func RevokeProviderClaim")
	if strings.Count(verify, "verification_last_succeeded_at=$1") < 2 ||
		strings.Count(verify, "providerDNSTokenMatches") < 2 {
		t.Fatal("initial and repeat DNS matches do not both refresh persisted proof")
	}
	requireAll("claim verification", verify,
		"verification_last_attempted_at=$1",
		"verification_consecutive_failures=0",
		"ProviderClaimDNSRecheckInterval",
	)

	lease := section("func LeaseDueProviderClaimDNSChecks", "func CompleteProviderClaimDNSCheck")
	requireAll("DNS lease acquisition", lease,
		"ProviderClaimDNSMaximumBatch",
		"verification_next_check_at <= $1",
		"verification_lease_until IS NULL OR verification_lease_until <= $1",
		"FOR UPDATE SKIP LOCKED",
		"verification_lease_id=uuid_generate_v4()",
		"ProviderClaimDNSLeaseDuration",
	)

	complete := section("func CompleteProviderClaimDNSCheck", "func RecordProviderClaimDNSFailure")
	requireAll("DNS lease completion", complete,
		"verification_lease_id=$2::uuid",
		"verification_lease_until > $3",
		"providerDNSTokenMatches(tokenHash, observedDNSTokens)",
		"verification_consecutive_failures=0",
		"recordProviderClaimDNSFailureTx",
	)

	failure := section("func recordProviderClaimDNSFailureTx", "func scanProviderAPIKey")
	requireAll("DNS proof failure", failure,
		"consecutiveFailures >= ProviderClaimDNSFailureLimit",
		"!providerClaimVerificationFresh(lastSucceededAt, checkedAt)",
		"ProviderClaimDNSFailureRetry",
		"verification_lease_id=NULL",
		"revokeProviderClaimTx(tx, claimID, checkedAt)",
	)

	for _, gate := range []struct {
		name, start, end string
		required         []string
	}{
		{"new callback key", "func CreateProviderAPIKey", "func RotateProviderAPIKey", []string{"providerClaimVerificationFresh", "ErrProviderClaimVerificationStale"}},
		{"rotated callback key", "func RotateProviderAPIKey", "func ResolveProviderAPIKey", []string{"providerClaimVerificationFresh", "ErrProviderClaimVerificationStale"}},
		{"callback-key use", "func ResolveProviderAPIKey", "func ResolveProviderAPIKeyForChargeResolution", []string{"verification_last_succeeded_at", "ProviderClaimVerificationFreshness"}},
		{"new offer", "func CreateProviderOffer", "func UpdateProviderOffer", []string{"providerClaimVerificationFresh", "ErrProviderClaimVerificationStale"}},
		{"offer mutation", "func lockVerifiedProviderOfferClaim", "func providerOfferBalance", []string{"providerClaimVerificationFresh", "ErrProviderClaimVerificationStale"}},
		{"public offer", "func ListPublicProviderOffersForOrganicResults", "func scanProviderOfferWithPosition", []string{"verification_last_succeeded_at", "ProviderClaimVerificationFreshness"}},
		{"new action ticket", "func CreateActionTicket", "func ResolveActionTicket", []string{"verification_last_succeeded_at", "ProviderClaimVerificationFreshness"}},
		{"action-token use", "func ResolveActionTicket", "func ResolveActionTicketForChargeResolution", []string{"verification_last_succeeded_at", "ProviderClaimVerificationFreshness"}},
		{"redirect", "func MarkActionTicketRedirected", "func RedactExpiredActionTicketIntent", []string{"verification_last_succeeded_at", "ProviderClaimVerificationFreshness"}},
		{"positive provider outcome", "func RecordProviderOutcome", "func GetOutcomeReceipt", []string{"providerClaimVerificationFresh", "staleClaimResolution", "providerChargeResolutionAllowed"}},
	} {
		requireAll(gate.name, section(gate.start, gate.end), gate.required...)
	}
}

func TestProviderExchangeSourceKeepsOrganicRankAndMoneyAtomic(t *testing.T) {
	source, err := os.ReadFile("provider_exchange.go")
	if err != nil {
		t.Fatalf("read provider exchange model: %v", err)
	}
	text := string(source)
	publicStart := strings.Index(text, "func ListPublicProviderOffersForOrganicResults")
	publicEnd := strings.Index(text[publicStart:], "func scanProviderOfferWithPosition")
	if publicStart < 0 || publicEnd < 0 {
		t.Fatal("could not isolate public paid-offer lookup")
	}
	publicSource := text[publicStart : publicStart+publicEnd]
	for _, forbidden := range []string{"agentic_score", "is_featured", "bounty_cents DESC", "ActionURL:"} {
		if strings.Contains(publicSource, forbidden) {
			t.Fatalf("paid offer lookup affects organic/commercial rank via %q", forbidden)
		}
	}
	for _, required := range []string{
		"unnest($1::text[], $2::text[]) WITH ORDINALITY",
		"claim.domain_snapshot=organic.domain", "organic.organic_position",
		"claim.status='verified'", "offer.status='active'", "ChargeEvent:",
		"claim.verification_last_succeeded_at", "ProviderClaimVerificationFreshness",
		"offer.terms_credit_limit_cents >= offer.bounty_cents",
		"offer.terms_credit_limit_cents-offer.bounty_cents",
		"ledger.entry_type IN ('charge','credit')", "offer.terms_period_anchor_at",
		"offer.terms_period_days*86400", "make_interval(", "secs =>",
	} {
		if !strings.Contains(publicSource, required) {
			t.Fatalf("public paid-offer lookup missing eligibility contract %q", required)
		}
	}

	returnedStart := strings.Index(text, "func RecordProviderOffersReturned")
	returnedEnd := strings.Index(text[returnedStart:], "func scanProviderOfferWithPosition")
	if returnedStart < 0 || returnedEnd < 0 {
		t.Fatal("could not isolate exact paid-offer return evidence")
	}
	returnedSource := text[returnedStart : returnedStart+returnedEnd]
	for _, required := range []string{
		"NOT receipt.is_synthetic", "organic.site_id=claim.site_id",
		"organic.site_domain_snapshot=claim.domain_snapshot",
		"offer.version=$3", "offer.status='active'",
		"pg_advisory_xact_lock", "NOT EXISTS (",
		"offer_version_snapshot", "tx.Commit()",
	} {
		if !strings.Contains(returnedSource, required) {
			t.Fatalf("paid-offer return evidence missing %q", required)
		}
	}

	outcomeStart := strings.Index(text, "func RecordProviderOutcome")
	outcomeEnd := strings.Index(text[outcomeStart:], "func GetOutcomeReceipt")
	if outcomeStart < 0 || outcomeEnd < 0 {
		t.Fatal("could not isolate atomic outcome recorder")
	}
	outcomeSource := text[outcomeStart : outcomeStart+outcomeEnd]
	for _, required := range []string{
		"tx.Begin", // deliberately checked below through the actual call spelling
		"pg_advisory_xact_lock", "FOR UPDATE OF ticket, offer",
		"entry_type, amount_cents", "'charge'", "'credit'",
		"ticket.charge_event_snapshot", "ticket.bounty_cents_snapshot",
		"ProviderBountyMaximumCents", "ensureTermsCreditCapacity",
		"balance < -ProviderMoneyMaximumCents+bountyCents",
		"SignOutcomeReceipt", "tx.Commit()",
	} {
		if required == "tx.Begin" {
			if !strings.Contains(outcomeSource, "db.Begin()") {
				t.Fatal("outcome recorder missing transaction begin")
			}
			continue
		}
		if !strings.Contains(outcomeSource, required) {
			t.Fatalf("outcome recorder missing atomic contract %q", required)
		}
	}
	if strings.Contains(outcomeSource, "IdempotencyKey,") {
		t.Fatal("outcome recorder appears to persist a raw idempotency key")
	}
}

func TestActionTicketCreationSnapshotsTermsAndSupportsSafeRetry(t *testing.T) {
	source, err := os.ReadFile("provider_exchange.go")
	if err != nil {
		t.Fatalf("read provider exchange model: %v", err)
	}
	text := string(source)
	start := strings.Index(text, "func CreateActionTicket")
	end := strings.Index(text, "func ResolveActionTicket")
	if start < 0 || end <= start {
		t.Fatal("could not isolate action ticket creation")
	}
	createSource := text[start:end]
	for _, required := range []string{
		"organic.site_id=claim.site_id",
		"organic.site_domain_snapshot=claim.domain_snapshot",
		"claim.verification_last_succeeded_at",
		"ProviderClaimVerificationFreshness",
		"$3=ANY(receipt.demand_topics)",
		"JOIN provider_offers_returned returned",
		"returned.offer_version_snapshot=offer.version",
		"existing.CreationRequestHash != requestHash",
		"Nonce: existing.TokenNonce",
		"existing.AttributionKeyIDSnapshot",
		"providerexchange.ActionURLWithAttribution",
		"attributionKeyID := signer.ActiveKeyID()",
		"subtle.ConstantTimeCompare",
		"offer_version_snapshot",
		"action_url_snapshot",
		"terms_credit_limit_cents_snapshot",
		"terms_period_anchor_at_snapshot",
		"attribution_key_id_snapshot",
	} {
		if !strings.Contains(createSource, required) {
			t.Fatalf("action ticket creation missing %q", required)
		}
	}
	if strings.Contains(createSource, "organic.site_id=claim.site_id OR") {
		t.Fatal("action ticket accepts an organic match on ID without matching domain")
	}

	redactStart := strings.Index(text, "func RedactExpiredActionTicketIntent")
	if redactStart < 0 || !strings.Contains(text[redactStart:], "ActionTicketIntentRetention") {
		t.Fatal("action ticket intent has no explicit bounded redaction method")
	}
}

func TestProviderExchangeProofExcludesSyntheticAndUsesPerCurrencyMoney(t *testing.T) {
	source, err := os.ReadFile("provider_exchange.go")
	if err != nil {
		t.Fatalf("read provider exchange model: %v", err)
	}
	text := string(source)
	start := strings.Index(text, "func GetProviderExchangeProof")
	if start < 0 {
		t.Fatal("GetProviderExchangeProof not found")
	}
	proofSource := text[start:]
	for _, required := range []string{
		"NOT ticket.source_is_synthetic", "GROUP BY ledger.currency",
		"ticket.authorization_revoked_at IS NULL", "SELECT DISTINCT claim.account_id",
		"operator_recorded_budgets", "terminal.outcome IN ('duplicate','invalid')",
		"charged.created_at, charged.id", "funding.created_at, funding.id",
		"funding.amount_cents >= -charged.amount_cents",
		"PrepaidNetDebitedByCurrency", "TermsNetReceivableByCurrency",
		"OperatorRecordedCollectedByCurrency", "PilotThresholdsMet",
	} {
		if !strings.Contains(proofSource, required) {
			t.Fatalf("provider proof missing truthful aggregate contract %q", required)
		}
	}
	if strings.Contains(proofSource, "NetBilledCents") || strings.Contains(proofSource, "NetBilledByCurrency") {
		t.Fatal("provider proof sums unlike currencies into one money total")
	}
}

func TestProviderEmergencyPauseRevokesLiveAuthorizationAndPreservesCreditPath(t *testing.T) {
	source, err := os.ReadFile("provider_exchange.go")
	if err != nil {
		t.Fatalf("read provider exchange model: %v", err)
	}
	text := string(source)
	pauseStart := strings.Index(text, "func AdminPauseProviderOffer")
	pauseEnd := strings.Index(text[pauseStart:], "func ListProviderAdminAuditEvents")
	if pauseStart < 0 || pauseEnd < 0 {
		t.Fatal("could not isolate emergency pause")
	}
	pauseSource := text[pauseStart : pauseStart+pauseEnd]
	for _, required := range []string{
		"authorization_revoked_at=COALESCE", "expires_at > NOW()",
		"event_type='emergency_pause'", "status IN ('active','paused')",
	} {
		if !strings.Contains(pauseSource, required) {
			t.Fatalf("emergency pause missing %q", required)
		}
	}
	outcomeStart := strings.Index(text, "func RecordProviderOutcome")
	outcomeEnd := strings.Index(text[outcomeStart:], "func GetOutcomeReceipt")
	outcomeSource := text[outcomeStart : outcomeStart+outcomeEnd]
	for _, required := range []string{
		"chargeResolutionAllowed := providerChargeResolutionAllowed",
		"!expiresAt.After(recordedAt) && !chargeResolutionAllowed",
		"authorizationRevokedAt.Valid && !chargeResolutionAllowed",
		"ErrProviderOfferRevoked", "ErrActionTicketExpired",
	} {
		if !strings.Contains(outcomeSource, required) {
			t.Fatalf("outcome revocation guard missing %q", required)
		}
	}
	resolveStart := strings.Index(text, "func ResolveActionTicket")
	resolveEnd := strings.Index(text[resolveStart:], "func RedactExpiredActionTicketIntent")
	if resolveStart < 0 || resolveEnd < 0 {
		t.Fatal("could not isolate action authorization resolution")
	}
	resolveSource := text[resolveStart : resolveStart+resolveEnd]
	if strings.Count(resolveSource, "expires_at > NOW()") < 2 ||
		strings.Count(resolveSource, "authorization_revoked_at IS NULL") < 2 {
		t.Fatal("late credit exception weakened redirect or action-token authorization")
	}
}

func TestChargeResolutionResolverOnlyProvesChargedTicketTokenBinding(t *testing.T) {
	source, err := os.ReadFile("provider_exchange.go")
	if err != nil {
		t.Fatalf("read provider exchange model: %v", err)
	}
	text := string(source)
	start := strings.Index(text, "func ResolveActionTicketForChargeResolution")
	end := strings.Index(text[start:], "func MarkActionTicketRedirected")
	if start < 0 || end < 0 {
		t.Fatal("could not isolate charge-resolution resolver")
	}
	resolverSource := text[start : start+end]
	for _, required := range []string{
		"ticket.id=$1::uuid AND ticket.token_hash=$2",
		"charge.action_ticket_id=ticket.id AND charge.entry_type='charge'",
		"HashProviderSecret(rawToken)",
	} {
		if !strings.Contains(resolverSource, required) {
			t.Fatalf("charge-resolution resolver missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"expires_at > NOW()", "authorization_revoked_at IS NULL",
		"UPDATE action_tickets", "RecordProviderOutcome(",
	} {
		if strings.Contains(resolverSource, forbidden) {
			t.Fatalf("charge-resolution resolver crosses its authentication-only boundary via %q", forbidden)
		}
	}
}

func TestClaimRevocationAtomicallyRevokesOutstandingTicketAuthorization(t *testing.T) {
	source, err := os.ReadFile("provider_exchange.go")
	if err != nil {
		t.Fatalf("read provider exchange model: %v", err)
	}
	text := string(source)
	start := strings.Index(text, "func RevokeProviderClaim")
	end := strings.Index(text[start:], "func scanProviderAPIKey")
	if start < 0 || end < 0 {
		t.Fatal("could not isolate provider claim revocation")
	}
	revokeSource := text[start : start+end]
	for _, required := range []string{
		"tx, err := db.Begin()", "UPDATE provider_claims", "UPDATE provider_api_keys",
		"UPDATE provider_offers", "UPDATE action_tickets",
		"authorization_revoked_at=COALESCE", "expires_at > $2",
		"status IN ('created','redirected','accepted','activated','converted')",
		"return tx.Commit()",
	} {
		if !strings.Contains(revokeSource, required) {
			t.Fatalf("claim revocation transaction missing %q", required)
		}
	}
}

func TestRevokedClaimChargeResolutionRequiresKeyRevokedWithClaim(t *testing.T) {
	source, err := os.ReadFile("provider_exchange.go")
	if err != nil {
		t.Fatalf("read provider exchange model: %v", err)
	}
	text := string(source)
	resolveStart := strings.Index(text, "func ResolveProviderAPIKeyForChargeResolution")
	resolveEnd := strings.Index(text[resolveStart:], "func RevokeProviderAPIKey")
	if resolveStart < 0 || resolveEnd < 0 {
		t.Fatal("could not isolate callback-key charge-resolution recovery")
	}
	resolveSource := text[resolveStart : resolveStart+resolveEnd]
	for _, required := range []string{
		"ticket.id=$2::uuid", "ticket.provider_claim_id=key.provider_claim_id",
		"charged.charge_status='charged'", "key.status='revoked' AND claim.status='revoked'",
		"key.revoked_at=claim.revoked_at", "HashProviderSecret(raw)",
	} {
		if !strings.Contains(resolveSource, required) {
			t.Fatalf("revoked callback-key recovery missing %q", required)
		}
	}
	outcomeStart := strings.Index(text, "func RecordProviderOutcome")
	outcomeEnd := strings.Index(text[outcomeStart:], "func GetOutcomeReceipt")
	outcomeSource := text[outcomeStart : outcomeStart+outcomeEnd]
	for _, required := range []string{
		"revokedClaimResolution := claimStatus == \"revoked\"",
		"staleClaimResolution := claimStatus == \"verified\"",
		"providerChargeResolutionOutcome(input.Outcome)",
		"$4::boolean AND key.status='active'", "$5::boolean AND key.status='revoked'",
		"charged.provider_claim_id=key.provider_claim_id",
		"charged.charge_status='charged'", "key.revoked_at=(",
		"providerChargeResolutionAllowed",
	} {
		if !strings.Contains(outcomeSource, required) {
			t.Fatalf("outcome recorder revoked-claim recovery missing %q", required)
		}
	}
}

func TestProviderBudgetMutationsAreBoundedAndPrepaidCannotGoNegative(t *testing.T) {
	source, err := os.ReadFile("provider_exchange.go")
	if err != nil {
		t.Fatalf("read provider exchange model: %v", err)
	}
	text := string(source)
	start := strings.Index(text, "func recordProviderBudgetEntry")
	end := strings.Index(text[start:], "func FundProviderOffer")
	if start < 0 || end < 0 {
		t.Fatal("could not isolate provider budget mutation")
	}
	budgetSource := text[start : start+end]
	for _, required := range []string{
		"ProviderMoneyMaximumCents", "proposedBalance := balance + amountCents",
		"providerOutstandingCreditExposure", "creditExposure > ProviderMoneyMaximumCents-proposedBalance",
		"billingMode == \"prepaid\" && proposedBalance < 0", "ErrInsufficientProviderFunds",
		"proposedBalance < -termsCreditLimit.Int64", "ErrProviderTermsCreditLimit",
		"entry.Replayed = true",
	} {
		if !strings.Contains(budgetSource, required) {
			t.Fatalf("budget mutation missing %q", required)
		}
	}
}

func TestProviderOfferCreationAndActivationHaveClaimScopedCaps(t *testing.T) {
	source, err := os.ReadFile("provider_exchange.go")
	if err != nil {
		t.Fatalf("read provider exchange model: %v", err)
	}
	text := string(source)
	createStart := strings.Index(text, "func CreateProviderOffer")
	createEnd := strings.Index(text[createStart:], "func UpdateProviderOffer")
	activateStart := strings.Index(text, "func ActivateProviderOffer")
	activateEnd := strings.Index(text[activateStart:], "func recordProviderBudgetEntry")
	if createStart < 0 || createEnd < 0 || activateStart < 0 || activateEnd < 0 {
		t.Fatal("could not isolate provider offer limit boundaries")
	}
	createSource := text[createStart : createStart+createEnd]
	activateSource := text[activateStart : activateStart+activateEnd]
	for _, required := range []string{"FOR UPDATE", "status IN ('draft','active')", "ProviderOfferMaximumPerClaim", "ErrProviderOfferLimit"} {
		if !strings.Contains(createSource, required) {
			t.Fatalf("offer creation cap missing %q", required)
		}
	}
	for _, required := range []string{"action_type=$2", "status='active'", "ProviderActiveOfferMaximumPerAction", "ErrProviderOfferLimit"} {
		if !strings.Contains(activateSource, required) {
			t.Fatalf("offer activation cap missing %q", required)
		}
	}
}

func TestPublicOutcomeStateIsStrictAndProviderFree(t *testing.T) {
	source, err := os.ReadFile("provider_exchange.go")
	if err != nil {
		t.Fatalf("read provider exchange model: %v", err)
	}
	text := string(source)
	start := strings.Index(text, "func GetPublicOutcomeReceiptState")
	end := strings.Index(text[start:], "func GetProviderExchangeProof")
	if start < 0 || end < 0 {
		t.Fatal("could not isolate public outcome state lookup")
	}
	stateSource := text[start : start+end]
	for _, required := range []string{
		"receipt.id=$1::uuid AND ticket.id=$2::uuid", "current_ticket_status",
		"original_charge_credited", "superseded_by_later_state",
		"net_commercial_effect_cents", "authorization_revoked_at",
	} {
		if !strings.Contains(stateSource, required) {
			t.Fatalf("public outcome state missing %q", required)
		}
	}
	for _, forbidden := range []string{"provider_claims", "provider_api_keys", "account_id", "action_url", "demand_topic"} {
		if strings.Contains(stateSource, forbidden) {
			t.Fatalf("public outcome state exposes provider/principal field %q", forbidden)
		}
	}
}
