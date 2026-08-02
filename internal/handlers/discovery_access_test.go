package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestSubmitHashIPIgnoresSourcePortAndHandlesForwardedIPv6(t *testing.T) {
	first := httptest.NewRequest("GET", "/", nil)
	first.RemoteAddr = "203.0.113.9:4000"
	second := httptest.NewRequest("GET", "/", nil)
	second.RemoteAddr = "203.0.113.9:5999"
	if submitHashIP(first) != submitHashIP(second) {
		t.Fatal("same client IP with different source ports must share an abuse bucket")
	}

	v6a := httptest.NewRequest("GET", "/", nil)
	v6a.Header.Set("X-Forwarded-For", "2001:db8::42, 198.51.100.1")
	v6b := httptest.NewRequest("GET", "/", nil)
	v6b.RemoteAddr = "[2001:db8::42]:443"
	if submitHashIP(v6a) != submitHashIP(v6b) {
		t.Fatal("forwarded and socket forms of the same IPv6 address must share a bucket")
	}
}

func TestSubmitHashIPPrefersFlyClientIPOverForwardedHeaders(t *testing.T) {
	first := httptest.NewRequest(http.MethodGet, "/", nil)
	first.RemoteAddr = "192.0.2.10:1000"
	first.Header.Set("Fly-Client-IP", "203.0.113.42")
	first.Header.Set("X-Forwarded-For", "198.51.100.1")

	second := httptest.NewRequest(http.MethodGet, "/", nil)
	second.RemoteAddr = "192.0.2.11:2000"
	second.Header.Set("Fly-Client-IP", "203.0.113.42")
	second.Header.Set("X-Forwarded-For", "198.51.100.99")

	if submitHashIP(first) != submitHashIP(second) {
		t.Fatal("Fly-Client-IP must determine the abuse bucket ahead of X-Forwarded-For")
	}
}

func TestDemandRequestIsSyntheticOnlyForDeploySmokeMarker(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if demandRequestIsSynthetic(req) {
		t.Fatal("ordinary request marked synthetic")
	}
	req.Header.Set("NHS-Synthetic-Test", "deploy-smoke")
	if !demandRequestIsSynthetic(req) {
		t.Fatal("deploy smoke marker not recognized")
	}
	req.Header.Set("NHS-Synthetic-Test", "provider-pilot")
	if demandRequestIsSynthetic(req) {
		t.Fatal("unknown marker must not silently exclude real demand")
	}
}

func TestDemandReceiptIDFailsClosedWhenStoreIsUnavailable(t *testing.T) {
	searchID, err := recordDemandSearchReceipt(nil, models.DemandSearchReceipt{}, nil)
	if err == nil {
		t.Fatal("nil demand store unexpectedly succeeded")
	}
	if searchID != "" {
		t.Fatalf("failed demand write advertised search ID %q", searchID)
	}
}

func TestWebSearchRateLimitRunsBeforeDatabaseSearch(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?q=payment", nil)
	limiter := newMCPDiscoveryRateLimiter(1, time.Hour)
	limiter.allow(submitHashIP(req), time.Now())
	h := &WebHandler{searchRateLimiter: limiter}
	rr := httptest.NewRecorder()

	h.HomePage(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Fatal("web search 429 omitted Retry-After")
	}
}

func TestDiscoveryRateLimiterEvictsExpiredBuckets(t *testing.T) {
	start := time.Unix(1_000, 0)
	limiter := newMCPDiscoveryRateLimiter(2, time.Minute)
	limiter.allow("old", start)
	if len(limiter.buckets) != 1 {
		t.Fatalf("bucket count = %d, want 1", len(limiter.buckets))
	}
	limiter.allow("new", start.Add(6*time.Minute))
	if _, exists := limiter.buckets["old"]; exists {
		t.Fatal("expired abuse bucket was not evicted")
	}
}

func TestParsePositivePageNormalizesInvalidValues(t *testing.T) {
	for raw, want := range map[string]int{
		"":        1,
		"0":       1,
		"-3":      1,
		"invalid": 1,
		" 7 ":     7,
	} {
		if got := parsePositivePage(raw); got != want {
			t.Errorf("parsePositivePage(%q) = %d, want %d", raw, got, want)
		}
	}
}

func TestMCPAnalyticsArgumentsNeverRetainSearchText(t *testing.T) {
	got := mcpAnalyticsArguments("search_agents", map[string]any{
		"query":    "payment API for buyer@example.com",
		"category": "developer",
		"has_mcp":  true,
	})
	if _, exists := got["query"]; exists {
		t.Fatalf("analytics arguments retained query text: %#v", got)
	}
	wantTopics := []string{"developer-tools", "payments"}
	if !reflect.DeepEqual(got["demand_topics"], wantTopics) {
		t.Fatalf("demand_topics = %#v, want %#v", got["demand_topics"], wantTopics)
	}
}

func TestActiveProbeToolsUseTheStrictBucket(t *testing.T) {
	for _, tool := range []string{"check_url", "verify_mcp", "submit_site", "register_monitor"} {
		if !isNHSActiveProbeTool(tool) {
			t.Fatalf("%s should use the strict active-probe bucket", tool)
		}
	}
	if isNHSActiveProbeTool("search_agents") {
		t.Fatal("cached search should not use the active-probe bucket")
	}
}

func TestMissingOrUnresolvableAPIKeyFallsBackToFreeAccess(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	req.Header.Set("X-API-Key", "not-a-real-key")
	access := resolveRequestRateAccess(nil, req)
	if access.tier != freeRateLimitTier || access.key != nil {
		t.Fatalf("access = %#v, want free fallback without a resolved key", access)
	}
	rr := httptest.NewRecorder()
	access.setHeaders(rr)
	if got := rr.Header().Get("X-RateLimit-Tier"); got != freeRateLimitTier {
		t.Fatalf("X-RateLimit-Tier = %q, want %q", got, freeRateLimitTier)
	}
}

func TestPriorityThroughputContractsRemainAboveFreeSafetyLimits(t *testing.T) {
	api := NewAPIHandler(nil)
	if api.searchRateLimiter.limit != freeSearchHourlyLimit || api.prioritySearchRateLimiter.limit != prioritySearchHourlyLimit {
		t.Fatalf("REST search limits = free %d priority %d", api.searchRateLimiter.limit, api.prioritySearchRateLimiter.limit)
	}
	if api.probeRateLimiter.limit != freeActiveProbeHourlyLimit || api.priorityProbeRateLimiter.limit != priorityActiveProbeLimit {
		t.Fatalf("REST probe limits = free %d priority %d", api.probeRateLimiter.limit, api.priorityProbeRateLimiter.limit)
	}
	mcp := NewMCPHandler(nil, "https://nothumansearch.ai")
	if mcp.toolRateLimiter.limit != freeSearchHourlyLimit || mcp.priorityToolRateLimiter.limit != prioritySearchHourlyLimit {
		t.Fatalf("MCP tool limits = free %d priority %d", mcp.toolRateLimiter.limit, mcp.priorityToolRateLimiter.limit)
	}
	if checkPriorityLimit <= checkFreeLimit {
		t.Fatalf("check priority limit %d must exceed free limit %d", checkPriorityLimit, checkFreeLimit)
	}
}

func TestPriorityCheckoutNeverClaimsSearchRequiresPayment(t *testing.T) {
	auth := NewAuthService(nil, "https://nothumansearch.ai")
	rr := httptest.NewRecorder()
	auth.SubscribePage(rr, httptest.NewRequest(http.MethodGet, "/subscribe", nil))
	body := rr.Body.String()
	for _, forbidden := range []string{"Unlimited agent-ready search", "subscription-only"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("priority checkout retained paywall promise %q", forbidden)
		}
	}
	for _, required := range []string{"Search and organic results are already free", "50,000 priority-throughput"} {
		if !strings.Contains(body, required) {
			t.Fatalf("priority checkout missing truthful copy %q", required)
		}
	}

	apiKeys := NewAPIKeyHandler(nil, "https://nothumansearch.ai")
	rr = httptest.NewRecorder()
	apiKeys.subscribeDocument(rr)
	var doc struct {
		Description string `json:"description"`
		Plans       []struct {
			Name    string `json:"name"`
			Benefit string `json:"benefit"`
		} `json:"plans"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("decode priority API document: %v", err)
	}
	if !strings.Contains(doc.Description, "baseline REST/MCP discovery remain free") || len(doc.Plans) != 1 || doc.Plans[0].Name != "Not Human Search Priority API" {
		t.Fatalf("priority API document is not a truthful free-fallback contract: %#v", doc)
	}
}

func TestStripeCheckoutLineItemDescribesPriorityThroughputTruthfully(t *testing.T) {
	source, err := os.ReadFile("api_keys.go")
	if err != nil {
		t.Fatalf("read api_keys.go: %v", err)
	}
	text := string(source)
	for _, required := range []string{
		`gostripe.String("Not Human Search Priority API")`,
		`priority-throughput REST/MCP calls per month; baseline discovery remains free afterward`,
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("Stripe line item missing truthful copy %q", required)
		}
	}
	for _, forbidden := range []string{`Not Human Search API " + strings.Title(plan.Name)`, `billable API/MCP calls per month`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Stripe line item retained misleading copy %q", forbidden)
		}
	}
}
