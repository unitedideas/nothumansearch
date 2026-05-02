package handlers

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// DigestHandler renders /digest — a weekly public snapshot of the MCP /
// agent-ready ecosystem. Shareable URL class + newsletter fuel. Query-once,
// cache-1h. Three surfaces: HTML at /digest, JSON at /digest.json, RSS at
// /digest.rss.
type DigestHandler struct {
	DB      *sql.DB
	BaseURL string
	tmpl    *template.Template
}

func NewDigestHandler(db *sql.DB, baseURL, templatesDir string) (*DigestHandler, error) {
	// Re-parse just digest.html here so the handler owns its template lifecycle
	// and doesn't depend on WebHandler's glob. Funcs mirror the minimum needed.
	funcMap := template.FuncMap{
		"pct": func(part, total int) string {
			if total == 0 {
				return "0.0"
			}
			return fmt.Sprintf("%.1f", float64(part)*100.0/float64(total))
		},
		"add": func(a, b int) int { return a + b },
		"displayText": func(s string) string {
			for i := 0; i < 3; i++ {
				next := html.UnescapeString(s)
				if next == s {
					return next
				}
				s = next
			}
			return s
		},
		"scoreClass": func(score int) string {
			if score >= 70 {
				return "high"
			}
			if score >= 40 {
				return "medium"
			}
			return "low"
		},
	}
	t, err := template.New("digest.html").Funcs(funcMap).ParseFiles(filepath.Join(templatesDir, "digest.html"))
	if err != nil {
		return nil, err
	}
	return &DigestHandler{DB: db, BaseURL: baseURL, tmpl: t}, nil
}

// DigestServer holds a compact site row surfaced in the digest.
type DigestSite struct {
	Domain      string   `json:"domain"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Score       int      `json:"agentic_score"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	HasMCP      bool     `json:"has_mcp_server"`
	HasLlmsTxt  bool     `json:"has_llms_txt"`
	HasOpenAPI  bool     `json:"has_openapi"`
}

type digestCategoryCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type digestData struct {
	GeneratedAt     time.Time
	WeekStart       time.Time
	NewMCP          []DigestSite
	TopMCP          []DigestSite
	Categories      []digestCategoryCount
	SubmissionsWeek int
	TotalSites      int
	MCPVerified     int
	LlmsTxtCount    int
	OpenAPICount    int
	PctMCP          string
	PctLlmsTxt      string
	PctOpenAPI      string
	BaseURL         string
	Canonical       string
}

func (h *DigestHandler) gather(r *http.Request) (*digestData, error) {
	ctx := r.Context()
	d := &digestData{
		GeneratedAt: time.Now().UTC(),
		WeekStart:   time.Now().UTC().AddDate(0, 0, -7),
		BaseURL:     h.BaseURL,
		Canonical:   h.BaseURL + "/digest",
	}

	// Top 10 new MCP servers added in the past 7 days.
	newRows, err := h.DB.QueryContext(ctx, `
		SELECT domain, name, description, agentic_score, category, tags,
		       has_mcp_server, has_llms_txt, has_openapi, created_at
		  FROM sites
		 WHERE created_at > NOW() - INTERVAL '7 days'
		   AND has_mcp_server = true
		   AND crawl_status = 'success'
		 ORDER BY agentic_score DESC, created_at DESC
		 LIMIT 10`)
	if err != nil {
		return nil, fmt.Errorf("new mcp query: %w", err)
	}
	for newRows.Next() {
		var s DigestSite
		var tags []byte
		var created time.Time
		if err := newRows.Scan(&s.Domain, &s.Name, &s.Description, &s.Score, &s.Category, &tags,
			&s.HasMCP, &s.HasLlmsTxt, &s.HasOpenAPI, &created); err != nil {
			continue
		}
		s.Tags = parsePGArray(tags)
		s.CreatedAt = created.UTC().Format(time.RFC3339)
		if s.Name == "" {
			s.Name = s.Domain
		}
		d.NewMCP = append(d.NewMCP, s)
	}
	newRows.Close()

	// Top 10 verified MCP servers overall. is_verified is only true for
	// domain-verified claims (rare), so we fall back to has_mcp_server=true
	// which is the operational verification signal (crawler confirmed endpoint).
	topRows, err := h.DB.QueryContext(ctx, `
		SELECT domain, name, description, agentic_score, category, tags,
		       has_mcp_server, has_llms_txt, has_openapi
		  FROM sites
		 WHERE has_mcp_server = true
		   AND crawl_status = 'success'
		 ORDER BY agentic_score DESC, is_verified DESC, updated_at DESC
		 LIMIT 10`)
	if err != nil {
		return nil, fmt.Errorf("top mcp query: %w", err)
	}
	for topRows.Next() {
		var s DigestSite
		var tags []byte
		if err := topRows.Scan(&s.Domain, &s.Name, &s.Description, &s.Score, &s.Category, &tags,
			&s.HasMCP, &s.HasLlmsTxt, &s.HasOpenAPI); err != nil {
			continue
		}
		s.Tags = parsePGArray(tags)
		if s.Name == "" {
			s.Name = s.Domain
		}
		d.TopMCP = append(d.TopMCP, s)
	}
	topRows.Close()

	// Category distribution — agent-first sites only.
	catRows, err := h.DB.QueryContext(ctx, `
		SELECT category, COUNT(*) AS n
		  FROM sites
		 WHERE crawl_status = 'success'
		   AND (has_structured_api = true OR has_llms_txt = true OR has_openapi = true
		        OR has_ai_plugin = true OR has_mcp_server = true)
		 GROUP BY category
		 ORDER BY n DESC
		 LIMIT 20`)
	if err != nil {
		return nil, fmt.Errorf("category query: %w", err)
	}
	for catRows.Next() {
		var c digestCategoryCount
		if err := catRows.Scan(&c.Name, &c.Count); err != nil {
			continue
		}
		d.Categories = append(d.Categories, c)
	}
	catRows.Close()

	// Weekly submission activity.
	if err := h.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM submissions WHERE created_at > NOW() - INTERVAL '7 days'`,
	).Scan(&d.SubmissionsWeek); err != nil {
		log.Printf("digest submissions count: %v", err)
	}

	// Ecosystem health — totals across agent-first filter.
	if err := h.DB.QueryRowContext(ctx, `
		SELECT
		  COUNT(*),
		  COUNT(*) FILTER (WHERE has_mcp_server = true),
		  COUNT(*) FILTER (WHERE has_llms_txt = true),
		  COUNT(*) FILTER (WHERE has_openapi = true)
		FROM sites
		WHERE crawl_status = 'success'
		  AND (has_structured_api = true OR has_llms_txt = true OR has_openapi = true
		       OR has_ai_plugin = true OR has_mcp_server = true)`,
	).Scan(&d.TotalSites, &d.MCPVerified, &d.LlmsTxtCount, &d.OpenAPICount); err != nil {
		log.Printf("digest health counts: %v", err)
	}

	if d.TotalSites > 0 {
		d.PctMCP = fmt.Sprintf("%.1f", float64(d.MCPVerified)*100.0/float64(d.TotalSites))
		d.PctLlmsTxt = fmt.Sprintf("%.1f", float64(d.LlmsTxtCount)*100.0/float64(d.TotalSites))
		d.PctOpenAPI = fmt.Sprintf("%.1f", float64(d.OpenAPICount)*100.0/float64(d.TotalSites))
	}

	return d, nil
}

// HTMLHandler serves /digest as rendered HTML.
func (h *DigestHandler) HTMLHandler(w http.ResponseWriter, r *http.Request) {
	d, err := h.gather(r)
	if err != nil {
		log.Printf("digest gather: %v", err)
		http.Error(w, "digest error", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if err := h.tmpl.ExecuteTemplate(w, "digest.html", d); err != nil {
		log.Printf("digest template: %v", err)
		http.Error(w, "template error", 500)
	}
}

// JSONHandler serves /digest.json.
func (h *DigestHandler) JSONHandler(w http.ResponseWriter, r *http.Request) {
	d, err := h.gather(r)
	if err != nil {
		log.Printf("digest json gather: %v", err)
		http.Error(w, `{"error":"digest error"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"generated_at":     d.GeneratedAt.Format(time.RFC3339),
		"week_start":       d.WeekStart.Format(time.RFC3339),
		"new_mcp_servers":  d.NewMCP,
		"top_mcp_servers":  d.TopMCP,
		"categories":       d.Categories,
		"submissions_week": d.SubmissionsWeek,
		"total_sites":      d.TotalSites,
		"mcp_verified":     d.MCPVerified,
		"llms_txt_count":   d.LlmsTxtCount,
		"openapi_count":    d.OpenAPICount,
		"pct_mcp":          d.PctMCP,
		"pct_llms_txt":     d.PctLlmsTxt,
		"pct_openapi":      d.PctOpenAPI,
		"canonical_url":    d.Canonical,
	})
}

// RSS types for the digest feed. One <item> per "new MCP server of the week".
type digestRSS struct {
	XMLName xml.Name         `xml:"rss"`
	Version string           `xml:"version,attr"`
	Atom    string           `xml:"xmlns:atom,attr"`
	Channel digestRSSChannel `xml:"channel"`
}

type digestRSSChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	AtomLink      atomLink  `xml:"atom:link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Items         []rssItem `xml:"item"`
}

// RSSHandler serves /digest.rss.
func (h *DigestHandler) RSSHandler(w http.ResponseWriter, r *http.Request) {
	d, err := h.gather(r)
	if err != nil {
		log.Printf("digest rss gather: %v", err)
		http.Error(w, "digest rss error", 500)
		return
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	feed := digestRSS{
		Version: "2.0",
		Atom:    "http://www.w3.org/2005/Atom",
		Channel: digestRSSChannel{
			Title:         "Not Human Search — Weekly MCP Ecosystem Digest",
			Link:          h.BaseURL + "/digest",
			AtomLink:      atomLink{Href: h.BaseURL + "/digest.rss", Rel: "self", Type: "application/rss+xml"},
			Description:   "A weekly snapshot of the MCP ecosystem — top new servers, ecosystem health, and category distribution.",
			Language:      "en-us",
			LastBuildDate: d.GeneratedAt.Format(time.RFC1123Z),
		},
	}
	for _, s := range d.NewMCP {
		body := s.Description
		if body == "" {
			body = fmt.Sprintf("New MCP server: %s. Agentic readiness score %d/100. Category: %s.", s.Domain, s.Score, s.Category)
		}
		pub := d.GeneratedAt
		if s.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, s.CreatedAt); err == nil {
				pub = t
			}
		}
		feed.Channel.Items = append(feed.Channel.Items, rssItem{
			Title:       fmt.Sprintf("%s — MCP server (score %d/100)", s.Domain, s.Score),
			Link:        h.BaseURL + "/site/" + s.Domain,
			GUID:        h.BaseURL + "/site/" + s.Domain + "#digest-" + d.GeneratedAt.Format("20060102"),
			Category:    s.Category,
			PubDate:     pub.UTC().Format(time.RFC1123Z),
			Description: body,
		})
	}

	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(feed); err != nil {
		log.Printf("digest rss encode: %v", err)
	}
}

// parsePGArray decodes Postgres text-array wire format (e.g. `{foo,bar}`) into
// []string. Safer than pulling lib/pq.Array into scan targets for a list we
// only need to display. Returns nil for empty/malformed input.
func parsePGArray(b []byte) []string {
	if len(b) < 2 {
		return nil
	}
	s := string(b)
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, `"`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
