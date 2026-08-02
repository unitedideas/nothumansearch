package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDomainRedirectFailsClosedForCapabilityURLs(t *testing.T) {
	t.Parallel()
	for _, target := range []string{
		"https://nothumansearch.com/auth/verify?token=credential-value",
		"https://www.nothumansearch.ai/site/example.com?search_id=nhs_sr_example",
		"https://nothumansearch.com/monitor/unsubscribe/opaque-capability",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rr := httptest.NewRecorder()
		domainRedirectMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("capability request reached canonical handler")
		})).ServeHTTP(rr, req)

		if rr.Code != http.StatusMisdirectedRequest {
			t.Fatalf("%s status = %d, want 421", target, rr.Code)
		}
		if location := rr.Header().Get("Location"); location != "" {
			t.Fatalf("%s leaked capability into Location %q", target, location)
		}
		for name, want := range map[string]string{
			"Cache-Control": "private, no-store", "Pragma": "no-cache", "Referrer-Policy": "no-referrer",
		} {
			if got := rr.Header().Get(name); got != want {
				t.Fatalf("%s %s = %q, want %q", target, name, got, want)
			}
		}
	}
}

func TestDomainRedirectUsesBoundedRedirectSemantics(t *testing.T) {
	t.Parallel()
	clean := httptest.NewRequest(http.MethodGet, "https://nothumansearch.com/privacy", nil)
	cleanRR := httptest.NewRecorder()
	domainRedirectMiddleware(http.NotFoundHandler()).ServeHTTP(cleanRR, clean)
	if cleanRR.Code != http.StatusPermanentRedirect || cleanRR.Header().Get("Location") != "https://nothumansearch.ai/privacy" {
		t.Fatalf("clean redirect status=%d location=%q", cleanRR.Code, cleanRR.Header().Get("Location"))
	}

	query := httptest.NewRequest(http.MethodGet, "https://nothumansearch.com/?utm_source=test", nil)
	queryRR := httptest.NewRecorder()
	domainRedirectMiddleware(http.NotFoundHandler()).ServeHTTP(queryRR, query)
	if queryRR.Code != http.StatusTemporaryRedirect || queryRR.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("query redirect status=%d cache=%q", queryRR.Code, queryRR.Header().Get("Cache-Control"))
	}
}

func TestSigningPreflightRunsBeforeDatabaseMutation(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	preflight := strings.Index(text, "handlers.ValidateProviderExchangeSigningConfiguration()")
	connect := strings.Index(text, "database.Connect()")
	migrate := strings.Index(text, "database.RunMigrations(")
	if preflight < 0 || connect < 0 || migrate < 0 || preflight > connect || preflight > migrate {
		t.Fatalf("signing preflight order preflight=%d connect=%d migrate=%d", preflight, connect, migrate)
	}
}
