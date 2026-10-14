package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestNormalizeProviderActionURLRequiresClaimedHTTPSOrigin(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		raw  string
		want string
	}{
		{name: "apex", raw: "https://example.com/start", want: "https://example.com/start"},
		{name: "subdomain", raw: "https://buy.example.com/", want: "https://buy.example.com/"},
		{name: "normalizes", raw: "HTTPS://EXAMPLE.COM", want: "https://example.com/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeProviderActionURL(test.raw, "example.com")
			if err != nil || got != test.want {
				t.Fatalf("normalizeProviderActionURL(%q) = %q, %v; want %q", test.raw, got, err, test.want)
			}
		})
	}

	for _, raw := range []string{
		"http://example.com/start",
		"https://example.net/start",
		"https://notexample.com/start",
		"https://user:secret@example.com/start",
		"https://example.com:8443/start",
		"https://example.com/start#fragment",
		"https://example.com/start?plan=pro",
		"https://example.com/start?nhs_attribution=forged",
		"https://127.0.0.1/start",
	} {
		if got, err := normalizeProviderActionURL(raw, "example.com"); err == nil {
			t.Fatalf("normalizeProviderActionURL(%q) = %q, want rejection", raw, got)
		}
	}
}

func TestProviderExchangeStatusMapsBoundedCommercialFailures(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		err  error
		want int
	}{
		{err: models.ErrProviderOfferLimit, want: http.StatusConflict},
		{err: models.ErrProviderOfferRevoked, want: http.StatusConflict},
		{err: models.ErrProviderBudgetLimit, want: http.StatusUnprocessableEntity},
	} {
		got, _ := providerExchangeStatus(test.err)
		if got != test.want {
			t.Fatalf("providerExchangeStatus(%v) = %d, want %d", test.err, got, test.want)
		}
	}
}

func TestActionURLWithAttributionPreservesProviderQuery(t *testing.T) {
	t.Parallel()
	got, err := actionURLWithAttribution("https://example.com/start?plan=pro", "signed.token")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"plan=pro", "nhs_attribution=signed.token"} {
		if !strings.Contains(got, want) {
			t.Fatalf("action URL %q missing %q", got, want)
		}
	}
}

func TestActionURLLengthReservesAndEnforcesAttributionSpace(t *testing.T) {
	t.Parallel()
	tooLongBase := "https://example.com/" + strings.Repeat("a", providerActionURLBaseMaximumBytes)
	if _, err := normalizeProviderActionURL(tooLongBase, "example.com"); err == nil {
		t.Fatal("offer accepted an action URL with no bounded attribution reserve")
	}

	nearLimitBase := "https://example.com/" + strings.Repeat("a", providerActionURLMaximumBytes-32)
	if _, err := actionURLWithAttribution(nearLimitBase, strings.Repeat("b", 64)); err == nil {
		t.Fatal("ticket returned an attributed action URL over the final length limit")
	}

	unicodeExpansion := "https://example.com/" + strings.Repeat("é", 400)
	if len(unicodeExpansion) >= providerActionURLBaseMaximumBytes {
		t.Fatal("fixture must fit the pre-normalization byte cap")
	}
	if _, err := normalizeProviderActionURL(unicodeExpansion, "example.com"); err == nil {
		t.Fatal("offer accepted a URL whose normalized encoding exceeds the reserved base cap")
	}
}

func TestDecodeProviderJSONIsStrictAndBounded(t *testing.T) {
	t.Parallel()
	type payload struct {
		OfferID string `json:"offer_id"`
	}

	good := httptest.NewRequest(http.MethodPost, "/api/v1/action-tickets", bytes.NewBufferString(`{"offer_id":"offer-1"}`))
	good.Header.Set("Content-Type", "application/json")
	var parsed payload
	if err := decodeProviderJSON(httptest.NewRecorder(), good, &parsed); err != nil || parsed.OfferID != "offer-1" {
		t.Fatalf("strict decode = %#v, %v", parsed, err)
	}

	for name, body := range map[string]string{
		"unknown":  `{"offer_id":"offer-1","contact":"private@example.com"}`,
		"multiple": `{"offer_id":"offer-1"}{"offer_id":"offer-2"}`,
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/action-tickets", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			if err := decodeProviderJSON(httptest.NewRecorder(), req, &payload{}); err == nil {
				t.Fatalf("decode %s body unexpectedly succeeded", name)
			}
		})
	}
}

func TestProviderMutationOriginGate(t *testing.T) {
	t.Parallel()
	base := "https://nothumansearch.ai"
	for _, origin := range []string{"", base} {
		req := httptest.NewRequest(http.MethodPost, base+"/api/v1/provider/claims", nil)
		req.Header.Set("Origin", origin)
		if !requestOriginAllowed(req, base) {
			t.Fatalf("same-origin request with Origin %q rejected", origin)
		}
	}
	cross := httptest.NewRequest(http.MethodPost, base+"/api/v1/provider/claims", nil)
	cross.Header.Set("Origin", "https://attacker.example")
	if requestOriginAllowed(cross, base) {
		t.Fatal("cross-origin provider mutation accepted")
	}
}

func TestProviderEvidenceReferenceRejectsFreeFormOrSecrets(t *testing.T) {
	t.Parallel()
	if !validProviderEvidenceReference("contract:provider-42:2026-08-01") {
		t.Fatal("bounded opaque evidence reference rejected")
	}
	for _, value := range []string{"short", "contains spaces and notes", "provider-ref?token=secret", "email@example.com"} {
		if validProviderEvidenceReference(value) {
			t.Fatalf("evidence reference %q accepted", value)
		}
	}
}
