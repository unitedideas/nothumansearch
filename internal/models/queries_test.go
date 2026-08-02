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
	for _, required := range []string{`"demand_topics"`, `"distinct_client_buckets"`} {
		if !strings.Contains(analyticsSource, required) {
			t.Fatalf("MCP analytics missing controlled metric %s", required)
		}
	}
}
