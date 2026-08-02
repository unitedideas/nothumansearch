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
		"Any paid offer shown:", "its ID, version, name, action type",
		"disclosed bounty/currency, charge event, and organic-result binding",
		"30-day retention",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("privacy page missing exact returned-offer disclosure %q", required)
		}
	}
}
