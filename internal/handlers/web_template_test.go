package handlers

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestHomeTemplateDisplaysScoreReasonsAndDecodedText(t *testing.T) {
	h, err := NewWebHandler(nil, "../../templates")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	data := map[string]interface{}{
		"Sites": []models.Site{{
			Domain:           "example.com",
			URL:              "https://example.com",
			Name:             "AT&amp;amp;T Agent API",
			Description:      "Tools &amp;amp; APIs for agents",
			AgenticScore:     45,
			Category:         "developer",
			HasLLMsTxt:       true,
			HasStructuredAPI: true,
		}},
		"Total":      1,
		"TotalSites": 1,
		"AvgScore":   45,
	}

	var out bytes.Buffer
	if err := h.tmpl.ExecuteTemplate(&out, "home.html", data); err != nil {
		t.Fatalf("execute home template: %v", err)
	}
	html := out.String()
	if strings.Contains(html, "amp;amp") {
		t.Fatalf("expected double-escaped entities to be decoded before render: %s", html)
	}
	for _, want := range []string{"AT&amp;T Agent API", "+25 llms.txt", "+15 structured API", "missing +20 ai-plugin"} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in rendered home template", want)
		}
	}

	var siteOut bytes.Buffer
	site := &models.Site{
		Domain:       "example.com",
		URL:          "https://example.com",
		Name:         "AT&amp;amp;T Agent API",
		AgenticScore: 25,
		HasLLMsTxt:   true,
		Category:     "developer",
	}
	if err := h.tmpl.ExecuteTemplate(&siteOut, "site.html", site); err != nil {
		t.Fatalf("execute site template: %v", err)
	}
	if !strings.Contains(siteOut.String(), "+25 llms.txt") {
		t.Fatalf("expected pointer site render to include score reasons")
	}
	if strings.Contains(siteOut.String(), "Fix this for $199") {
		t.Fatalf("expected passive llms-only site to omit score-fix CTA")
	}

	site.HasStructuredAPI = true
	var hardSignalOut bytes.Buffer
	if err := h.tmpl.ExecuteTemplate(&hardSignalOut, "site.html", site); err != nil {
		t.Fatalf("execute hard-signal site template: %v", err)
	}
	if !strings.Contains(hardSignalOut.String(), "Fix this for $199") {
		t.Fatalf("expected low-score hard-signal site to include score-fix CTA")
	}

	site.AgenticScore = 95
	var highScoreOut bytes.Buffer
	if err := h.tmpl.ExecuteTemplate(&highScoreOut, "site.html", site); err != nil {
		t.Fatalf("execute high-score hard-signal site template: %v", err)
	}
	if strings.Contains(highScoreOut.String(), "Fix this for $199") {
		t.Fatalf("expected high-score hard-signal site to omit score-fix CTA")
	}
}

func TestReportPageUsesAgentFirstCorpus(t *testing.T) {
	source, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatalf("read web.go: %v", err)
	}
	text := string(source)
	start := strings.Index(text, "func (h *WebHandler) ReportPage")
	if start < 0 {
		t.Fatal("ReportPage not found")
	}
	end := strings.Index(text[start:], "\n}\n")
	if end < 0 {
		t.Fatal("ReportPage end not found")
	}
	reportPage := text[start : start+end]
	if got := strings.Count(reportPage, "models.AgentFirstFilter"); got < 3 {
		t.Fatalf("ReportPage should filter summary, category, and top-site queries with AgentFirstFilter; got %d uses", got)
	}

	h, err := NewWebHandler(nil, "../../templates")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	var out bytes.Buffer
	data := ReportData{
		Total:      4240,
		HighScore:  219,
		AvgScore:   35,
		LlmsTxt:    2000,
		OpenAPI:    300,
		AIPlugin:   100,
		API:        500,
		MCP:        120,
		SchemaOrg:  1500,
		RobotsAI:   600,
		LlmsMCP:    90,
		Categories: []CategoryStat{{Name: "developer", Count: 1300, AvgScore: 34}},
		TopSites:   []TopSite{{Domain: "example.com", Score: 95, Category: "developer"}},
	}
	if err := h.tmpl.ExecuteTemplate(&out, "report.html", data); err != nil {
		t.Fatalf("execute report template: %v", err)
	}
	html := out.String()
	for _, want := range []string{"agent-first indexed domains", "Agent-First Sites", "hard signal", "excluded from this report"} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected report template to contain %q", want)
		}
	}
	if strings.Contains(html, "Only 219") {
		t.Fatalf("report template should not imply the filtered agent-first corpus is the whole crawl corpus")
	}
}

func TestScoreTemplateIncludesOwnerHandoffWithoutPaidRanking(t *testing.T) {
	h, err := NewWebHandler(nil, "../../templates")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	var out bytes.Buffer
	if err := h.tmpl.ExecuteTemplate(&out, "score.html", nil); err != nil {
		t.Fatalf("execute score template: %v", err)
	}
	html := out.String()
	for _, want := range []string{
		"Site owner next step",
		"Monitor this score",
		"Get implementation help",
		"Improve public signals",
		"score < 70 && hasHardSignal",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected score template to contain %q", want)
		}
	}
	for _, banned := range []string{"paid ranking", "paid placement", "score bypass"} {
		if strings.Contains(strings.ToLower(html), banned) {
			t.Fatalf("score template contains banned paid-ranking language %q", banned)
		}
	}
}
