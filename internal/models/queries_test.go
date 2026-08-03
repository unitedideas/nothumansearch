package models

import (
	"os"
	"strings"
	"testing"
)

func TestSearchOrderingDoesNotUseCommercialFlagsOrOwnedPins(t *testing.T) {
	source, err := os.ReadFile("queries.go")
	if err != nil {
		t.Fatalf("read queries.go: %v", err)
	}
	text := string(source)
	for _, forbidden := range []string{
		"is_featured DESC",
		"CASE WHEN lower(domain)",
		"PinDomain",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("organic search ordering contains commercial/pinned term %q", forbidden)
		}
	}
}

func TestMCPAnalyticsUsesControlledDemandAndClientBucketTerms(t *testing.T) {
	source, err := os.ReadFile("queries.go")
	if err != nil {
		t.Fatalf("read queries.go: %v", err)
	}
	text := string(source)
	start := strings.Index(text, "func GetMCPAnalytics")
	end := strings.Index(text, "func GetTrafficAnalytics")
	if start < 0 || end <= start {
		t.Fatal("could not isolate GetMCPAnalytics source")
	}
	analyticsSource := text[start:end]
	for _, forbidden := range []string{`"top_queries"`, `"unique_agents"`} {
		if strings.Contains(analyticsSource, forbidden) {
			t.Fatalf("legacy MCP analytics contains forbidden metric %s", forbidden)
		}
	}
	for _, required := range []string{
		`"demand_topics"`, `"distinct_client_buckets"`,
		`'get_top_sites'`, `'recent_additions'`,
	} {
		if !strings.Contains(analyticsSource, required) {
			t.Fatalf("MCP analytics missing controlled metric %s", required)
		}
	}
}

func TestMCPAnalyticsToolNameStorageIsAllowlisted(t *testing.T) {
	if got := normalizeMCPAnalyticsToolName("search_agents"); got != "search_agents" {
		t.Fatalf("known tool normalized to %q", got)
	}
	for _, hostile := range []string{
		"person@example.com",
		"private-token-value",
		strings.Repeat("x", 64<<10),
	} {
		if got := normalizeMCPAnalyticsToolName(hostile); got != "unknown_tool" {
			t.Fatalf("hostile tool name normalized to %q", got)
		}
	}
}
