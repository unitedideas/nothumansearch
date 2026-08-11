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
			healthHandler(healthPingerStub{err: test.pingError}, providerExchangeModeDisabled).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
			if rr.Code != test.wantStatus || rr.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("health status=%d cache=%q", rr.Code, rr.Header().Get("Cache-Control"))
			}
			var payload healthPayload
			if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.ReleaseRevision != releaseRevision || payload.Database != test.wantDB || payload.ProviderExchange != providerExchangeModeDisabled {
				t.Fatalf("health payload = %+v", payload)
			}
		})
	}
}

func TestConfiguredProviderExchangeModeFailsClosed(t *testing.T) {
	for _, test := range []struct {
		name      string
		value     string
		want      string
		wantError bool
	}{
		{name: "missing", wantError: true},
		{name: "pilot", value: " PILOT ", want: providerExchangeModePilot},
		{name: "disabled recovery", value: "disabled", want: providerExchangeModeDisabled},
		{name: "unknown", value: "observe", wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(providerExchangeModeEnv, test.value)
			got, err := configuredProviderExchangeMode()
			if (err != nil) != test.wantError || got != test.want {
				t.Fatalf("configured mode = %q, err=%v", got, err)
			}
		})
	}
}

func TestConfiguredListenAddressSupportsExactLoopbackRecovery(t *testing.T) {
	for _, test := range []struct {
		name      string
		host      string
		want      string
		wantError bool
	}{
		{name: "production all interfaces", want: ":8091"},
		{name: "ipv4 loopback", host: "127.0.0.1", want: "127.0.0.1:8091"},
		{name: "ipv6 loopback", host: "::1", want: "[::1]:8091"},
		{name: "hostname rejected", host: "localhost", wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("NHS_LISTEN_HOST", test.host)
			got, err := configuredListenAddress("8091")
			if (err != nil) != test.wantError || got != test.want {
				t.Fatalf("listen address = %q, err=%v", got, err)
			}
		})
	}
}

func TestProviderExchangeDisabledHandlerIsPrivateAndRetryable(t *testing.T) {
	rr := httptest.NewRecorder()
	providerExchangeDisabledHandler(rr, httptest.NewRequest(http.MethodPost, "/api/v1/action-tickets", strings.NewReader("{}")))
	if rr.Code != http.StatusServiceUnavailable || rr.Header().Get("Cache-Control") != "private, no-store" || rr.Header().Get("Retry-After") == "" {
		t.Fatalf("disabled response status=%d cache=%q retry=%q", rr.Code, rr.Header().Get("Cache-Control"), rr.Header().Get("Retry-After"))
	}
	if !strings.Contains(rr.Body.String(), "free discovery and action-interest observation remain available") {
		t.Fatalf("disabled response body = %q", rr.Body.String())
	}
}

func TestPrivacySensitiveAgentActionPathsBypassIdentityTelemetry(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		"/api/v1/action-interests",
		"/api/v1/action-tickets",
		"/api/v1/action-tickets/handoff",
		"/api/v1/provider/action-tickets/resolve",
		"/mcp",
	} {
		if shouldLogPageView(path) {
			t.Errorf("privacy-sensitive path %q would write page_views", path)
		}
		if !suppressRequestLineIdentityTelemetry(path) {
			t.Errorf("privacy-sensitive path %q would emit request-line UA telemetry", path)
		}
	}
	if !shouldLogPageView("/api/v1/search") {
		t.Fatal("ordinary aggregate discovery traffic was accidentally excluded")
	}
}

func TestPageViewAnalyticsDropCallerControlledReferer(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "https://nothumansearch.ai/api/v1/search", nil)
	req.Header.Set("Referer", "https://example.com/private?token=secret&query=person@example.com#fragment")
	if got := pageViewReferer(req); got != "" {
		t.Fatalf("page-view referer = %q, want omitted", got)
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
	connect := strings.Index(text, "database.ConnectWithReleaseRevisionContext(connectContext, releaseRevision)")
	migrate := strings.Index(text, "database.RunMigrationsWithPreflight(")
	retention := strings.Index(text, "handlers.ProviderExchangeProtectedMigrationPreflight")
	if preflight < 0 || connect < 0 || migrate < 0 || retention < 0 ||
		preflight > connect || connect > migrate || retention < migrate {
		t.Fatalf("signing preflight order config=%d connect=%d migrate=%d retention_hook=%d", preflight, connect, migrate, retention)
	}
}

func TestPrivacyCleanupRunsOnBootAndThenHourly(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if !strings.Contains(text, "prune()\n\t\tticker := time.NewTicker(1 * time.Hour)") ||
		strings.Contains(text, "time.Sleep(2 * time.Minute)") {
		t.Fatal("privacy cleanup must run immediately on boot before its hourly ticker")
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
		`org.opencontainers.image.source_archive_sha256=$SOURCE_ARCHIVE_SHA256`,
		`grep -Eq '^[0-9a-f]{64}$'`,
		`go build -trimpath -ldflags "-X main.releaseRevision=${RELEASE_REVISION}" -o provider-cutover-preflight ./cmd/provider-cutover-preflight/`,
		`COPY --from=builder /app/provider-cutover-preflight .`,
	} {
		if !strings.Contains(dockerfile, required) {
			t.Fatalf("Dockerfile is missing release binding %q", required)
		}
	}
	preparer := read(filepath.Join("tools", "prepare-exact-release.sh"))
	for _, required := range []string{
		"contract=nhs-exact-release-verification-v2",
		"git -C \"$REPOSITORY\" archive",
		"RELEASE_REVISION=$COMMIT",
		"migration_022_sha256=$MIGRATION_022_SHA",
		"migration_023_sha256=$MIGRATION_023_SHA",
		"migration_024_sha256=$MIGRATION_024_SHA",
		"migration_025_sha256=$MIGRATION_025_SHA",
		"migration_026_sha256=$MIGRATION_026_SHA",
		"migration_027_sha256=$MIGRATION_027_SHA",
		"migration_028_sha256=$MIGRATION_028_SHA",
		"migration_029_sha256=$MIGRATION_029_SHA",
		"migration_030_sha256=$MIGRATION_030_SHA",
		"TestProviderExchangePostgresReleaseRegressions",
		"TestProtectedMigrationLedgerPostgres",
		"preflight_binary_revision_bound=true",
		"candidate_revision_mismatch",
		"test -race ./internal/database",
		"run ./cmd/openapi-dump",
		"provider-exchange design audit is not passing",
		"export GOFLAGS=''",
		"export GOWORK=off",
		"codex-secret scan \"${CHANGED_PATHS[@]}\"",
		"oci_image_digest_verified=false",
		"target_cutover_preflight_verified=false",
		"restore_drill_verified=false",
		"deployment_ready=false",
		"deployment_command_emitted=false",
		"No deploy command was emitted",
		"No deployment was performed",
		"Migrations 019-028 require an owner-authorized",
	} {
		if !strings.Contains(preparer, required) {
			t.Fatalf("exact release preparer is missing %q", required)
		}
	}
	imageBuilder := read(filepath.Join("tools", "build-exact-provider-image.sh"))
	for _, required := range []string{
		`git -C "$CANDIDATE_REPOSITORY" archive`,
		`<"$ARCHIVE"`,
		`SOURCE_ARCHIVE_SHA256=$ARCHIVE_SHA256`,
		`org.opencontainers.image.source_archive_sha256`,
		`registry_digest_verified=false`,
		`deployment_ready=false`,
		`push_command_emitted=false`,
		`deployment_command_emitted=false`,
	} {
		if !strings.Contains(imageBuilder, required) {
			t.Fatalf("exact image builder is missing %q", required)
		}
	}
	smoke := read(filepath.Join("tools", "smoke-test.sh"))
	for _, required := range []string{
		`"method":"initialize"`,
		`"method":"tools/list"`,
		`"method":"tools/call"`,
		`"name":"search_agents"`,
		`.result.structuredContent.access == "free"`,
		`.result.structuredContent.paid_offers_available == false`,
	} {
		if !strings.Contains(smoke, required) {
			t.Fatalf("live smoke is missing runtime MCP assertion %q", required)
		}
	}
	disabledSmoke := read(filepath.Join("tools", "disabled-recovery-smoke.py"))
	for _, required := range []string{
		`/api/v1/admin/provider-pilot/action`,
		`/api/v1/admin/provider-pilot-review`,
		`/api/v1/admin/provider-proof-manifest`,
		`/api/v1/provider/pilot-status`,
		`/api/v1/provider/demand`,
		`/api/v1/provider/receipts/00000000-0000-4000-8000-000000000000`,
		`"private" not in cache_control`,
		`"no-store" not in cache_control`,
	} {
		if !strings.Contains(disabledSmoke, required) {
			t.Fatalf("disabled recovery smoke is missing %q", required)
		}
	}
}
