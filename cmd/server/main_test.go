package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type healthPingerStub struct {
	err error
}

func (p healthPingerStub) Ping() error { return p.err }

func TestHealthReportsCompiledReleaseRevision(t *testing.T) {
	previous := releaseRevision
	releaseRevision = "0123456789abcdef0123456789abcdef01234567"
	t.Cleanup(func() { releaseRevision = previous })

	for _, test := range []struct {
		name       string
		pingError  error
		wantStatus int
		wantDB     string
	}{
		{name: "healthy", wantStatus: http.StatusOK, wantDB: "ok"},
		{name: "degraded", pingError: errors.New("unreachable"), wantStatus: http.StatusServiceUnavailable, wantDB: "unreachable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			healthHandler(healthPingerStub{err: test.pingError}).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
			if rr.Code != test.wantStatus || rr.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("health status=%d cache=%q", rr.Code, rr.Header().Get("Cache-Control"))
			}
			var payload healthPayload
			if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.ReleaseRevision != releaseRevision || payload.Database != test.wantDB {
				t.Fatalf("health payload = %+v", payload)
			}
		})
	}
}

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

func TestReleaseBuildRequiresArchiveCommitIdentity(t *testing.T) {
	repositoryRoot := filepath.Join("..", "..")
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repositoryRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	marker := strings.TrimSpace(read("release-source-revision"))
	_, markerHexError := hex.DecodeString(marker)
	if marker != "$Format:%H$" && (len(marker) != 40 || markerHexError != nil) {
		t.Fatalf("release source marker is neither the checkout placeholder nor an archived commit: %q", marker)
	}
	if attributes := read(".gitattributes"); !strings.Contains(attributes, "release-source-revision export-subst") {
		t.Fatal("release source marker is not expanded by git archive")
	}
	dockerfile := read("Dockerfile")
	for _, required := range []string{
		`grep -Fxq "$RELEASE_REVISION" release-source-revision`,
		`-X main.releaseRevision=${RELEASE_REVISION}`,
		`org.opencontainers.image.revision=$RELEASE_REVISION`,
	} {
		if !strings.Contains(dockerfile, required) {
			t.Fatalf("Dockerfile is missing release binding %q", required)
		}
	}
	preparer := read(filepath.Join("tools", "prepare-exact-release.sh"))
	for _, required := range []string{"git -C \"$REPOSITORY\" archive", "RELEASE_REVISION=$COMMIT", "No deployment was performed"} {
		if !strings.Contains(preparer, required) {
			t.Fatalf("exact release preparer is missing %q", required)
		}
	}
}
