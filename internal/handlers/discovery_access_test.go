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

func TestReceiptBearingResponseHeadersPreventCapabilityLeakage(t *testing.T) {
	rr := httptest.NewRecorder()
	protectReceiptBearingResponse(rr)
	for name, want := range map[string]string{
		"Cache-Control":   "private, no-store",
		"Pragma":          "no-cache",
		"Referrer-Policy": "no-referrer",
	} {
		if got := rr.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
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
	if got := rr.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("web search Cache-Control = %q, want private, no-store", got)
	}
	if got := rr.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("web search Referrer-Policy = %q, want no-referrer", got)
	}
}

func TestRESTSearchHeadersProtectReceiptCapabilityOnEveryOutcome(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=payment", nil)
	limiter := newMCPDiscoveryRateLimiter(1, time.Hour)
	limiter.allow(submitHashIP(req)+":rest-search", time.Now())
	h := &APIHandler{searchRateLimiter: limiter}
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rr.Code)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("REST search Cache-Control = %q, want private, no-store", got)
	}
	if got := rr.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("REST search Referrer-Policy = %q, want no-referrer", got)
	}
}

func TestReceiptBearingSitePageHeadersPrecedeLookup(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/site/?search_id=nhs_sr_example", nil)
	rr := httptest.NewRecorder()
	h := &WebHandler{}

	h.SitePage(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("site detail Cache-Control = %q, want private, no-store", got)
	}
	if got := rr.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("site detail Referrer-Policy = %q, want no-referrer", got)
	}
}

func TestReceiptBearingRESTSiteHeadersPrecedeLookup(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/site/?search_id=nhs_sr_example", nil)
	rr := httptest.NewRecorder()
	h := &APIHandler{}

	h.GetSite(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	for name, want := range map[string]string{
		"Cache-Control":   "private, no-store",
		"Pragma":          "no-cache",
		"Referrer-Policy": "no-referrer",
	} {
		if got := rr.Header().Get(name); got != want {
			t.Fatalf("REST site detail %s = %q, want %q", name, got, want)
		}
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

func TestDiscoveryRateLimiterFailsClosedAtBucketCapacity(t *testing.T) {
	start := time.Unix(2_000, 0)
	limiter := newMCPDiscoveryRateLimiter(2, time.Minute)
	limiter.maxBuckets = 2
	if _, _, ok := limiter.allow("first", start); !ok {
		t.Fatal("first bucket was rejected")
	}
	if _, _, ok := limiter.allow("second", start); !ok {
		t.Fatal("second bucket was rejected")
	}
	if _, retry, ok := limiter.allow("attacker-churn", start); ok || retry <= 0 {
		t.Fatalf("new bucket at capacity ok=%t retry=%s, want fail-closed", ok, retry)
	}
	if len(limiter.buckets) != 2 {
		t.Fatalf("bucket count = %d, want hard cap 2", len(limiter.buckets))
	}
	if _, _, ok := limiter.allow("after-expiry", start.Add(2*time.Minute)); !ok {
		t.Fatal("expired buckets did not restore bounded capacity")
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

func TestMCPAnalyticsCategoryBrowseRetainsOnlyControlledTopics(t *testing.T) {
	for _, tool := range []string{"get_top_sites", "recent_additions"} {
		t.Run(tool+"/recognized_category", func(t *testing.T) {
			got := mcpAnalyticsArguments(tool, map[string]any{
				"query":    "private payment query for buyer@example.com",
				"category": "developer",
			})
			if _, exists := got["query"]; exists {
				t.Fatalf("analytics arguments retained query text: %#v", got)
			}
			if _, exists := got["category"]; exists {
				t.Fatalf("analytics arguments retained raw category: %#v", got)
			}
			if want := []string{"developer-tools"}; !reflect.DeepEqual(got["demand_topics"], want) {
				t.Fatalf("demand_topics = %#v, want %#v", got["demand_topics"], want)
			}
		})

		t.Run(tool+"/unfiltered", func(t *testing.T) {
			got := mcpAnalyticsArguments(tool, map[string]any{
				"query": "private payment query for buyer@example.com",
			})
			if _, exists := got["query"]; exists {
				t.Fatalf("analytics arguments retained query text: %#v", got)
			}
			if _, exists := got["category"]; exists {
				t.Fatalf("analytics arguments retained raw category: %#v", got)
			}
			if want := []string{"other"}; !reflect.DeepEqual(got["demand_topics"], want) {
				t.Fatalf("demand_topics = %#v, want %#v", got["demand_topics"], want)
			}
		})
	}
}

func TestMCPPublicCategoryAcceptsOnlyThePublicTaxonomy(t *testing.T) {
	for _, tc := range []struct {
		name         string
		raw          string
		wantCategory string
		wantBound    bool
	}{
		{name: "trim_and_normalize", raw: " Developer ", wantCategory: "developer", wantBound: true},
		{name: "empty_is_unbound", raw: "  ", wantCategory: "", wantBound: false},
		{name: "audit_other_rejected", raw: "other", wantCategory: "", wantBound: false},
		{name: "audit_spam_rejected", raw: "spam", wantCategory: "", wantBound: false},
		{name: "unknown_rejected", raw: "private-category", wantCategory: "", wantBound: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotCategory, gotBound := mcpPublicCategory(tc.raw)
			if gotCategory != tc.wantCategory || gotBound != tc.wantBound {
				t.Fatalf("mcpPublicCategory(%q) = (%q, %t), want (%q, %t)", tc.raw, gotCategory, gotBound, tc.wantCategory, tc.wantBound)
			}
		})
	}
}

func TestFindMCPServersRejectsAuditCategoryBeforeQuery(t *testing.T) {
	handler := NewMCPHandler(nil, "https://nothumansearch.ai")
	for _, category := range []string{"other", "spam", "private-category"} {
		t.Run(category, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.toolFindMCPServers(recorder, json.RawMessage(`1`), map[string]any{
				"query":    "payments",
				"category": category,
			}, false)
			if !strings.Contains(recorder.Body.String(), "unsupported public category") {
				t.Fatalf("audit category %q reached the database path: %s", category, recorder.Body.String())
			}
		})
	}
}

func TestMCPDiscoveryCategoryTypesFailClosedBeforeQuery(t *testing.T) {
	handler := NewMCPHandler(nil, "https://nothumansearch.ai")
	calls := []struct {
		name string
		call func(http.ResponseWriter, map[string]any)
	}{
		{name: "get_top_sites", call: func(w http.ResponseWriter, args map[string]any) {
			handler.toolGetTopSites(w, json.RawMessage(`1`), args, false)
		}},
		{name: "recent_additions", call: func(w http.ResponseWriter, args map[string]any) {
			handler.toolRecentAdditions(w, json.RawMessage(`1`), args, false)
		}},
		{name: "find_mcp_servers", call: func(w http.ResponseWriter, args map[string]any) {
			handler.toolFindMCPServers(w, json.RawMessage(`1`), args, false)
		}},
	}
	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			for _, category := range []any{float64(1), true, nil, []any{"developer"}} {
				recorder := httptest.NewRecorder()
				call.call(recorder, map[string]any{"category": category})
				if !strings.Contains(recorder.Body.String(), "category must be a string") {
					t.Fatalf("category type %T reached the database path: %s", category, recorder.Body.String())
				}
			}
		})
	}
}

func TestMCPRecentWindowDaysMatchesTheReportedQueryWindow(t *testing.T) {
	for _, tc := range []struct {
		raw  any
		want int
	}{
		{raw: nil, want: 7},
		{raw: float64(-1), want: 7},
		{raw: float64(30), want: 30},
		{raw: float64(365), want: 90},
	} {
		if got := mcpRecentWindowDays(tc.raw); got != tc.want {
			t.Fatalf("mcpRecentWindowDays(%#v) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestMCPExchangeSchemasReferenceEveryReceiptBearingDiscoveryTool(t *testing.T) {
	definitions := NewMCPHandler(nil, "https://nothumansearch.ai").toolDefinitions()
	byName := map[string]map[string]any{}
	for _, definition := range definitions {
		name, _ := definition["name"].(string)
		byName[name] = definition
	}
	property := func(tool, name string) map[string]any {
		t.Helper()
		schema, ok := byName[tool]["inputSchema"].(map[string]any)
		if !ok {
			t.Fatalf("%s input schema unavailable", tool)
		}
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s properties unavailable", tool)
		}
		value, ok := properties[name].(map[string]any)
		if !ok {
			t.Fatalf("%s.%s definition unavailable", tool, name)
		}
		return value
	}

	interestDescription, _ := property("record_action_interest", "search_id")["description"].(string)
	if !strings.Contains(interestDescription, "receipt-bearing NHS discovery tool") {
		t.Fatalf("record_action_interest search_id description is surface-specific: %q", interestDescription)
	}
	offerDescription, _ := property("prepare_provider_action", "offer_id")["description"].(string)
	if !strings.Contains(offerDescription, "receipt-bearing NHS discovery result") {
		t.Fatalf("prepare_provider_action offer_id description is surface-specific: %q", offerDescription)
	}
	if got := property("find_mcp_servers", "category")["enum"]; !reflect.DeepEqual(got, publicSearchCategories) {
		t.Fatalf("find_mcp_servers category enum = %#v, want public categories", got)
	}
}

func TestMCPAnalyticsCanonicalizeUnknownToolNames(t *testing.T) {
	hostile := strings.Repeat("private-token-", 10_000)
	if got := mcpAnalyticsToolName(hostile); got != "unknown_tool" {
		t.Fatalf("unknown analytics tool name = %q", got)
	}
	if got := mcpAnalyticsToolName("search_agents"); got != "search_agents" {
		t.Fatalf("known analytics tool name = %q", got)
	}
	if got := mcpAnalyticsArguments(mcpAnalyticsToolName(hostile), map[string]any{
		"query": hostile,
	}); len(got) != 0 {
		t.Fatalf("unknown tool retained arguments: %#v", got)
	}
}

func TestMCPDetailAnalyticsRetainsOnlyFollowthroughBooleans(t *testing.T) {
	args := map[string]any{
		"domain":    "example.dev",
		"search_id": "nhs_sr_private_receipt",
	}
	safe := mcpAnalyticsArguments("get_site_details", args)
	safe["search_receipt_supplied"] = strings.TrimSpace(asString(args["search_id"])) != ""
	safe["selection_recorded"] = true
	safe["synthetic_test"] = false

	if _, retained := safe["search_id"]; retained {
		t.Fatalf("detail analytics retained a search receipt: %#v", safe)
	}
	if safe["search_receipt_supplied"] != true || safe["selection_recorded"] != true {
		t.Fatalf("detail followthrough booleans unavailable: %#v", safe)
	}
	if safe["synthetic_test"] != false {
		t.Fatalf("ordinary detail call marked synthetic: %#v", safe)
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
