package crawler

import (
	"strings"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

// TestExtractTitle covers the HTML <title> extractor. Pure string op; a
// regression here silently blanks site names across the index. Called
// once per crawl per site.
func TestExtractTitle(t *testing.T) {
	cases := map[string]string{
		"<html><head><title>Stripe | Payments</title></head></html>": "Stripe | Payments",
		// Title with whitespace is trimmed
		"<title>  Anthropic  </title>": "Anthropic",
		// Attributes on the title tag (rare but legal)
		`<title dir="ltr">OpenAI</title>`: "OpenAI",
		// Nested structure; we just grab first <title>...</title>
		"<head><meta><title>NHS</title></head>": "NHS",
		// Missing end tag → empty
		"<title>no closer":             "",
		// Missing start tag → empty
		"<html>no title here</html>":   "",
		// Empty string
		"":                            "",
	}
	for input, want := range cases {
		if got := extractTitle(input); got != want {
			t.Errorf("extractTitle(%q) = %q, want %q", input, got, want)
		}
	}

	// 200-char truncation
	long := "<title>" + strings.Repeat("x", 300) + "</title>"
	got := extractTitle(long)
	if len(got) != 200 {
		t.Errorf("extractTitle truncation: got len %d, want 200", len(got))
	}
}

// TestExtractMetaDescription covers the <meta name="description" content="...">
// extractor. Supports both single-quote and double-quote variants.
func TestExtractMetaDescription(t *testing.T) {
	cases := map[string]string{
		// Standard double-quote
		`<html><head><meta name="description" content="Stripe is a payment platform"></head></html>`: "Stripe is a payment platform",
		// Single-quote attribute values
		`<head><meta name='description' content='LLM inference'></head>`: "LLM inference",
		// Different attribute order
		`<meta content="Agent-ready API platform" name="description">`: "Agent-ready API platform",
		// Uppercase attribute values (case-insensitive name match)
		`<META NAME="description" CONTENT="Uppercase meta">`: "Uppercase meta",
		// No description tag
		`<html><head></head></html>`: "",
		// Empty content
		`<meta name="description" content="">`: "",
	}
	for input, want := range cases {
		if got := extractMetaDescription(input); got != want {
			t.Errorf("extractMetaDescription(%q) = %q, want %q", input, got, want)
		}
	}

	// 500-char truncation
	long := `<meta name="description" content="` + strings.Repeat("x", 600) + `">`
	got := extractMetaDescription(long)
	if len(got) != 500 {
		t.Errorf("extractMetaDescription truncation: got len %d, want 500", len(got))
	}
}

// TestCategorize covers the domain → category classifier for NHS's
// landing-page system. Getting this wrong sends sites to the wrong
// /category/{x}-apis landing page and diffuses SEO signal.
func TestCategorize(t *testing.T) {
	mk := func(domain, name, desc string, tags ...string) *models.Site {
		return &models.Site{
			Domain:      domain,
			Name:        name,
			Description: desc,
			Tags:        tags,
		}
	}

	cases := []struct {
		name string
		site *models.Site
		want string
	}{
		// Domain rules (highest confidence) — case-insensitive
		{"aidevboard→jobs", mk("aidevboard.com", "AI Dev Jobs", "jobs"), "jobs"},
		{"greenhouse→jobs", mk("jobs.greenhouse.io", "Greenhouse", "hire"), "jobs"},
		{"stripe→finance", mk("stripe.com", "Stripe", "payments"), "finance"},
		{"coinbase→finance", mk("coinbase.com", "Coinbase", "crypto"), "finance"},
		{"openai→ai-tools", mk("openai.com", "OpenAI", "models"), "ai-tools"},
		{"anthropic→ai-tools", mk("anthropic.com", "Anthropic", "models"), "ai-tools"},
		{"shopify→ecommerce", mk("shopify.com", "Shopify", "stores"), "ecommerce"},

		// TLD fallbacks — when no domain rule matches
		{"any-.ai→ai-tools", mk("random-startup.ai", "Random", ""), "ai-tools"},
		{"any-.ml→ai-tools", mk("example.ml", "Example", ""), "ai-tools"},
		{"any-.dev→developer", mk("tooling.dev", "Tooling", ""), "developer"},
		{"any-.sh→developer", mk("utils.sh", "Utils", ""), "developer"},

		// Fall-through → "other" when no rule matches
		{"no-match→other", mk("randomsite.xyz", "Random", "Generic content"), "other"},
	}
	for _, tc := range cases {
		if got := categorize(tc.site); got != tc.want {
			t.Errorf("categorize(%s) = %q, want %q", tc.name, got, tc.want)
		}
	}
}
