package crawler

import "testing"

// TestIsValidOpenAPI covers the operational bar stated in the repo CLAUDE.md:
// "OpenAPI requires real parseable 3.x/Swagger 2.x spec with non-empty paths,
// not HTML containing the word 'openapi'." Regressing this to a naive
// substring check was the specific failure mode tightened at v0.14-2026-04-14;
// these cases lock in the current strictness.
func TestIsValidOpenAPI(t *testing.T) {
	// Cases that MUST return true (valid OpenAPI/Swagger specs).
	valid := []struct {
		name string
		body string
	}{
		{"openapi-3-json-with-paths", `{"openapi":"3.0.3","paths":{"/users":{"get":{}}}}`},
		{"swagger-2-json-with-paths", `{"swagger":"2.0","paths":{"/v1/jobs":{"get":{}}}}`},
		{"openapi-3-yaml-with-paths", "openapi: 3.0.3\npaths:\n  /users:\n    get:\n      summary: list"},
		{"swagger-2-yaml-with-paths", "swagger: '2.0'\npaths:\n  /v1/jobs:\n    get:\n      summary: list"},
		{"openapi-paths-quoted-slash", "openapi: 3.0.3\npaths:\n '/users':\n    get:\n      summary: list"},
		{"openapi-with-leading-whitespace", "  \n\n  {\"openapi\":\"3.0.3\",\"paths\":{\"/x\":{\"get\":{}}}}  "},
	}
	for _, tc := range valid {
		if !isValidOpenAPI(tc.body) {
			t.Errorf("isValidOpenAPI(%s) = false, want true", tc.name)
		}
	}

	// Cases that MUST return false (the specific bypasses we tightened against).
	invalid := []struct {
		name string
		body string
	}{
		{"empty", ""},
		{"whitespace-only", "   \n  \t  "},
		{"html-doctype", "<!DOCTYPE html><html><body>openapi page</body></html>"},
		{"html-starts-html", "<html><head><title>openapi doc</title></head></html>"},
		{"xml-not-openapi", "<?xml version='1.0'?><foo>openapi</foo>"},
		{"json-without-openapi-key", `{"paths":{"/x":{"get":{}}}}`},
		{"json-openapi-but-no-paths", `{"openapi":"3.0.3","info":{"title":"x"}}`},
		{"json-openapi-empty-paths", `{"openapi":"3.0.3","paths":{}}`},
		{"json-paths-but-not-object", `{"openapi":"3.0.3","paths":"oops"}`},
		{"malformed-json", `{"openapi":"3.0.3","paths":{`},
		{"yaml-without-version-key", "paths:\n  /x:\n    get:\n      summary: foo"},
		{"yaml-version-but-no-paths", "openapi: 3.0.3\ninfo:\n  title: x"},
		{"yaml-paths-header-no-routes", "openapi: 3.0.3\npaths:\n  # empty block"},
		{"plain-text-mentioning-openapi", "This page documents our openapi spec somewhere."},
		{"html-with-openapi-in-body", "<html>openapi: 3.0.3<br>paths:<br>/users</html>"},
	}
	for _, tc := range invalid {
		if isValidOpenAPI(tc.body) {
			t.Errorf("isValidOpenAPI(%s) = true, want false", tc.name)
		}
	}
}

// TestResolveURL covers the scheme/host/relative-path expansion logic used
// by the crawler when extracting links from HTML (favicons, sitemap entries,
// doc links). Each branch of the resolver has an edge case.
func TestResolveURL(t *testing.T) {
	cases := []struct {
		base, href, want string
	}{
		// Already-absolute URLs pass through unchanged.
		{"https://example.com", "https://stripe.com/api", "https://stripe.com/api"},
		{"https://example.com", "http://legacy.example.com/x", "http://legacy.example.com/x"},

		// Scheme-relative URLs get forced to https (never http).
		{"https://example.com", "//cdn.example.com/asset.js", "https://cdn.example.com/asset.js"},
		{"http://example.com", "//cdn.example.com/asset.js", "https://cdn.example.com/asset.js"},

		// Root-relative paths concat to base host.
		{"https://example.com", "/api/v1/jobs", "https://example.com/api/v1/jobs"},
		{"https://example.com/", "/api/v1/jobs", "https://example.com/api/v1/jobs"},
		{"https://example.com/nested/page", "/api/v1/jobs", "https://example.com/nested/page/api/v1/jobs"},

		// Path-relative hrefs get a / separator inserted.
		{"https://example.com", "favicon.ico", "https://example.com/favicon.ico"},
		{"https://example.com/", "favicon.ico", "https://example.com/favicon.ico"},
		{"https://example.com/docs", "api.yaml", "https://example.com/docs/api.yaml"},
	}
	for _, tc := range cases {
		got := resolveURL(tc.base, tc.href)
		if got != tc.want {
			t.Errorf("resolveURL(%q, %q) = %q, want %q", tc.base, tc.href, got, tc.want)
		}
	}
}
