package handlers

import (
	"bytes"
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
}
