package handlers

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func renderProviderExchangePage(t *testing.T, name string, data providerPageData) string {
	t.Helper()
	path := filepath.Join("..", "..", "templates", name)
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, name, data); err != nil {
		t.Fatalf("render %s: %v", name, err)
	}
	return rendered.String()
}

func TestProviderPageExplainsPersistentDNSOwnershipAndSafeStatus(t *testing.T) {
	t.Parallel()
	data := newProviderPageData("https://nothumansearch.ai")
	data.SignedIn = true
	data.Email = "provider@example.com"
	body := renderProviderExchangePage(t, "providers.html", data)

	for _, required := range []string{
		"keep it in place",
		"stores only a SHA-256 hash of the challenge token",
		"stops paid actions after 3 consecutive failed checks or 7 days without a successful check",
		"Last successful check:",
		"Next automatic check:",
		"Keep the record published after verification so NHS can recheck ownership.",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("provider page missing persistent DNS copy %q", required)
		}
	}
}

func TestProviderPageExplainsSeparateObservedHandoffBeforePositiveOutcome(t *testing.T) {
	t.Parallel()
	body := renderProviderExchangePage(t, "providers.html", newProviderPageData("https://nothumansearch.ai"))

	for _, required := range []string{
		"separate versioned attestations for ticket preparation and provider handoff",
		"returns no provider action URL",
		"nhs-provider-handoff-consent-v1",
		"Only then does NHS return the attributed provider action URL",
		"This handoff charges neither party",
		"positive outcome only after that observed handoff",
		"principal_handoff_consent",
		"handoff-consent wording",
		"principal-controlled routing context",
		"nhs-provider-controlled-intent-disclosure-consent-v1",
		"nhs-provider-controlled-intent-resolver-v1",
		"Declining the optional disclosure does not block handoff or free direct provider access",
		"current offer version and exact commercial-terms SHA-256",
		"Authentication is not commercial proof",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("provider page missing observed-handoff contract copy %q", required)
		}
	}
}

func TestPrivacyPagePublishesExactOptionalControlledIntentDisclosureBoundary(t *testing.T) {
	t.Parallel()
	body := renderProviderExchangePage(t, "privacy.html", newProviderPageData("https://nothumansearch.ai"))
	for _, required := range []string{
		`id="controlled-intent-disclosure-consent-v1"`,
		"nhs-provider-controlled-intent-disclosure-consent-v1",
		"nhs-provider-controlled-intent-resolver-v1",
		"separate from both ticket authorization and handoff consent",
		"declining it does not block the observed handoff or free direct provider access",
		"controlled demand topic, optional region code, USD budget band, urgency, and allowlisted requirement flags",
		"shorter of the ticket authorization window and NHS’s 30-day controlled-intent retention period",
		"will not disclose the search query or query hash, prompt, free-form text, name, contact data",
		"does not authorize provider outreach, verify identity or agency, prove uniqueness or an outcome, change organic rank, or charge either party",
		"writes no page-view, request-line identity, MCP, intent-event, durable quota, receipt, outcome, provider-budget entry, charge, or commercial-proof record",
		"64-bit truncated SHA-256 hash of the network address",
		"provider-key record ID plus that same hash",
		"one-hour counting windows",
		"Expired entries are evicted opportunistically on later resolver-limiter use or at process restart",
		"the buckets are never logged or persisted",
		"Consent is recorded only in the append-only handoff receipt and cannot be added later by replay",
		"physical redaction on the next successful boot or hourly cleanup",
		"downtime or cleanup failure can delay that redaction without extending resolver availability",
		"NHS exposes no provider resolver MCP tool",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("privacy page missing optional controlled-intent boundary %q", required)
		}
	}
	for _, forbidden := range []string{
		"No action URL, price, bounty, provider-account, budget, charge",
		"resolver changes organic rank",
		"resolver fee",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("privacy page contains contradictory controlled-intent copy %q", forbidden)
		}
	}
}

func TestPrivacyPageExplainsDNSHashRechecksAndFreshnessRevocation(t *testing.T) {
	t.Parallel()
	body := renderProviderExchangePage(t, "privacy.html", newProviderPageData("https://nothumansearch.ai"))

	for _, required := range []string{
		"must keep its TXT value published",
		"does not retain the raw answers or challenge token",
		"stores only the SHA-256 token hash, check timestamps, and failure count",
		"stops after 3 consecutive failed checks",
		"last successful check reaches 7 days old",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("privacy page missing DNS privacy copy %q", required)
		}
	}
}

func TestPrivacyPageDisclosesExactReturnedOfferEvidence(t *testing.T) {
	t.Parallel()
	body := renderProviderExchangePage(t, "privacy.html", newProviderPageData("https://nothumansearch.ai"))
	for _, required := range []string{
		"Any paid offer shown:", "its ID, offer version, name, action type",
		"disclosed bounty/currency, charge event, and organic-result binding",
		"exact commercial-terms contract version and SHA-256",
		"Merchant-of-Record acknowledgement",
		"30-day eligibility · hourly physical deletion",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("privacy page missing exact returned-offer disclosure %q", required)
		}
	}
}

func TestPrivacyPagePublishesSeparateObservedHandoffAndCommercialEvidenceBoundary(t *testing.T) {
	t.Parallel()
	body := renderProviderExchangePage(t, "privacy.html", newProviderPageData("https://nothumansearch.ai"))
	for _, required := range []string{
		`id="handoff-consent-v1"`,
		"nhs-provider-handoff-consent-v1",
		"principal_handoff_consent=true",
		"one-way SHA-256 hash of the presented ticket bearer",
		"retained ticket nonce/key metadata and signing material can reconstruct it for an exact replay",
		"will then disclose an attributed provider action URL containing the opaque bearer",
		"charges neither the principal nor the provider",
		"No raw query, query hash, prompt, topic, region, budget band, urgency",
		"No IP address, IP hash, network identifier, referrer, referral trail, user agent",
		"keyed company digest",
		"provider-key-authenticated pilot-company or exact-terms acceptance",
		"separate owner-verified append-only events",
		"no automatic deletion interval",
		"Effective August 2, 2026",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("privacy page missing observed-handoff/commercial boundary %q", required)
		}
	}
}

func TestPrivacyPagePublishesExactNonContactingActionInterestBoundary(t *testing.T) {
	t.Parallel()
	body := renderProviderExchangePage(t, "privacy.html", newProviderPageData("https://nothumansearch.ai"))
	for _, required := range []string{
		`id="action-interest-v1"`, "nhs-action-interest-v1",
		"Record interest without contacting a provider",
		"caller can attest that its principal currently wants a controlled next step",
		"does not contact or identify the caller to the provider",
		"does not create an action ticket, redirect, charge, lead, activation, or conversion",
		"creates no persisted IP/user-agent, page-view, MCP-request, intent-event, or API-key quota row",
		"cannot count toward NHS commercial proof",
		"downtime or cleanup failure can delay physical deletion but not replay or reporting eligibility",
		"Aggregate counts are receipts, not unique people, principals, or agents",
		"requires a separate action", "nhs-principal-consent-v1",
		"nhs-provider-handoff-consent-v1", "ticket authorization",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("privacy page missing action-interest boundary %q", required)
		}
	}
	if strings.Contains(body, "for an NHS provider-funded handoff, the separate <code>nhs-principal-consent-v1</code> authorization") {
		t.Fatal("privacy page conflates ticket authorization with handoff consent")
	}
}
