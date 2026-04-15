package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/crawler"
	"github.com/unitedideas/nothumansearch/internal/models"
)

// MCPHandler exposes Not Human Search as a remote MCP (Model Context Protocol) server.
// Agents can register this server via:
//   claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp
//
// Protocol: JSON-RPC 2.0 over Streamable HTTP (POST requests, JSON responses).
// Spec: https://modelcontextprotocol.io/specification/2025-06-18/basic/transports
type MCPHandler struct {
	DB      *sql.DB
	BaseURL string
}

func NewMCPHandler(db *sql.DB, baseURL string) *MCPHandler {
	return &MCPHandler{DB: db, BaseURL: baseURL}
}

// JSON-RPC 2.0 envelope types
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ServeHTTP handles the /mcp endpoint. Accepts POST with JSON-RPC 2.0 requests.
// GET returns a simple info blurb for humans poking at the URL.
func (h *MCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":        "nothumansearch",
			"description": "MCP server for Not Human Search — search the agentic web.",
			"transport":   "streamable-http",
			"endpoint":    h.BaseURL + "/mcp",
			"tools":       []string{"search_agents", "get_site_details", "get_stats", "submit_site"},
			"setup": map[string]string{
				"claude_code": "claude mcp add --transport http nothumansearch " + h.BaseURL + "/mcp",
			},
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "POST or GET only", http.StatusMethodNotAllowed)
		return
	}

	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, nil, -32700, "parse error")
		return
	}

	// Notifications (no id) expect no response body, just 202 Accepted.
	if len(req.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	switch req.Method {
	case "initialize":
		h.writeResult(w, req.ID, map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]any{
				"name":    "nothumansearch",
				"title":   "Not Human Search",
				"version": "1.0.0",
			},
			"instructions": "Search engine for AI agents. Use search_agents to find agent-ready tools, APIs, and services ranked by agentic readiness score (0-100). Use get_site_details for a full readiness report on a specific domain.",
		})

	case "ping":
		h.writeResult(w, req.ID, map[string]any{})

	case "tools/list":
		h.writeResult(w, req.ID, map[string]any{
			"tools": h.toolDefinitions(),
		})

	case "tools/call":
		h.handleToolCall(w, req)

	default:
		h.writeError(w, req.ID, -32601, "method not found: "+req.Method)
	}
}

func (h *MCPHandler) toolDefinitions() []map[string]any {
	return []map[string]any{
		{
			"name":        "search_agents",
			"title":       "Search the Agentic Web",
			"description": "Search for websites, APIs, and services that AI agents can actually use. Results are ranked by agentic readiness score (0-100) based on llms.txt, OpenAPI specs, ai-plugin.json, structured APIs, and MCP server availability. Use this to discover payment APIs, job boards, data sources, or any web service your agent needs to call.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Keyword query (e.g. 'payment API', 'weather data', 'job board')",
					},
					"category": map[string]any{
						"type":        "string",
						"description": "Filter by category",
						"enum":        []string{"ai-tools", "developer", "data", "jobs", "finance", "ecommerce", "health", "education", "security", "communication", "productivity", "news"},
					},
					"min_score": map[string]any{
						"type":        "integer",
						"description": "Minimum agentic readiness score 0-100 (higher = more agent-ready)",
						"minimum":     0,
						"maximum":     100,
					},
					"has_api": map[string]any{
						"type":        "boolean",
						"description": "Only return sites with a documented structured API",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Max results (default 10, max 20)",
						"minimum":     1,
						"maximum":     20,
					},
				},
			},
		},
		{
			"name":        "get_site_details",
			"title":       "Get Site Agentic Readiness Report",
			"description": "Get the full agentic readiness report for a specific domain: score, category, all 7 signal checks (llms.txt, ai-plugin.json, OpenAPI, structured API, MCP server, robots.txt AI rules, Schema.org), plus any cached llms.txt content and OpenAPI summary.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"domain": map[string]any{
						"type":        "string",
						"description": "Domain to look up (e.g. 'stripe.com'). Do not include scheme or path.",
					},
				},
				"required": []string{"domain"},
			},
		},
		{
			"name":        "get_stats",
			"title":       "Get Index Stats",
			"description": "Get current statistics for the Not Human Search index: total sites, average agentic score, top category.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "submit_site",
			"title":       "Submit a Site for Indexing",
			"description": "Submit a URL for NHS to crawl and score. Use when you discover an agent-first tool, API, or service that isn't in the index yet. NHS will fetch the site, check its 7 agentic signals (llms.txt, ai-plugin.json, OpenAPI, structured API, MCP server, robots.txt AI rules, Schema.org), compute a score, and add it to the index. The site becomes searchable within a few seconds if the crawl succeeds.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "Full URL to submit (include scheme, e.g. 'https://example.com'). Homepage is best — NHS will check /.well-known/ paths, /robots.txt, /llms.txt, etc. relative to the site root.",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			"name":        "register_monitor",
			"title":       "Monitor a Site's Agentic Readiness",
			"description": "Register an email to get alerted when the indicated domain's agentic readiness score drops. Useful for agents tracking a dependency's agent-readiness health — e.g. an agent that relies on stripe.com's MCP surface wants to know the moment it regresses. Returns an unsubscribe URL. Multiple monitors per email allowed, one per domain.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"email":  map[string]any{"type": "string", "description": "Email address to receive alert"},
					"domain": map[string]any{"type": "string", "description": "Domain to monitor (no scheme, e.g. 'stripe.com')"},
				},
				"required": []string{"email", "domain"},
			},
		},
	}
}

func (h *MCPHandler) handleToolCall(w http.ResponseWriter, req rpcRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		h.writeError(w, req.ID, -32602, "invalid params")
		return
	}

	switch params.Name {
	case "search_agents":
		h.toolSearchAgents(w, req.ID, params.Arguments)
	case "get_site_details":
		h.toolGetSiteDetails(w, req.ID, params.Arguments)
	case "get_stats":
		h.toolGetStats(w, req.ID)
	case "submit_site":
		h.toolSubmitSite(w, req.ID, params.Arguments)
	case "register_monitor":
		h.toolRegisterMonitor(w, req.ID, params.Arguments)
	default:
		h.writeToolError(w, req.ID, "unknown tool: "+params.Name)
	}
}

// toolRegisterMonitor wraps the /api/v1/monitor/register REST handler so
// agents can subscribe to drop alerts via MCP. Mirrors the email+domain
// flow exactly; returns the unsubscribe URL in the response text.
func (h *MCPHandler) toolRegisterMonitor(w http.ResponseWriter, id json.RawMessage, args map[string]any) {
	email := strings.TrimSpace(asString(args["email"]))
	domain := strings.TrimSpace(asString(args["domain"]))
	if email == "" || domain == "" {
		h.writeToolError(w, id, "email and domain both required")
		return
	}
	m, err := models.RegisterMonitor(h.DB, email, domain)
	if err != nil {
		switch err {
		case models.ErrInvalidEmail:
			h.writeToolError(w, id, "invalid email")
		case models.ErrInvalidDomain:
			h.writeToolError(w, id, "invalid or unsupported domain")
		case models.ErrTooManyMonitors:
			h.writeToolError(w, id, "too many monitors for this email")
		default:
			h.writeToolError(w, id, "registration failed: "+err.Error())
		}
		return
	}
	unsub := h.BaseURL + "/monitor/unsubscribe/" + m.Token
	text := fmt.Sprintf("Monitor registered for %s — alert will fire if score drops. Unsubscribe: %s", m.Domain, unsub)
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"content": []map[string]any{{"type": "text", "text": text}},
			"structuredContent": map[string]any{
				"ok":              true,
				"domain":          m.Domain,
				"unsubscribe_url": unsub,
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// toolSubmitSite queues a URL for crawling and tries an inline crawl if
// concurrency allows. Mirrors the /api/v1/submit handler behavior so agents
// get identical semantics regardless of transport.
func (h *MCPHandler) toolSubmitSite(w http.ResponseWriter, id json.RawMessage, args map[string]any) {
	rawURL := strings.TrimSpace(asString(args["url"]))
	if rawURL == "" {
		h.writeToolError(w, id, "url required")
		return
	}
	// Normalize — accept domains without scheme, reject obvious garbage.
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	_, err := h.DB.Exec(`
		INSERT INTO submissions (url, status) VALUES ($1, 'pending')
		ON CONFLICT DO NOTHING`, rawURL)
	if err != nil {
		h.writeToolError(w, id, "submission failed: "+err.Error())
		return
	}

	// Try an inline crawl if the global submit-crawl semaphore has room. If
	// not, fall back to queued status and let the scheduled recrawl pick it
	// up. The semaphore lives in api.go to prevent OOM during bulk submissions.
	crawled := false
	var crawlText string
	select {
	case submitCrawlSem <- struct{}{}:
		site, err := crawler.CrawlSite(rawURL)
		<-submitCrawlSem
		if err != nil {
			log.Printf("mcp submit crawl failed for %s: %v", rawURL, err)
			h.DB.Exec("UPDATE submissions SET status='failed' WHERE url=$1", rawURL)
			crawlText = fmt.Sprintf("Queued %s, but inline crawl failed: %v. Will retry on next scheduled recrawl.", rawURL, err)
		} else {
			if err := models.UpsertSite(h.DB, site); err != nil {
				log.Printf("mcp submit upsert failed for %s: %v", rawURL, err)
				crawlText = fmt.Sprintf("Crawled %s (score %d/100) but index write failed; will retry.", site.Domain, site.AgenticScore)
			} else {
				h.DB.Exec("UPDATE submissions SET status='crawled' WHERE url=$1", rawURL)
				crawled = true
				crawlText = fmt.Sprintf("Indexed %s — agentic score %d/100, category %s. Live at %s/site/%s.", site.Domain, site.AgenticScore, site.Category, h.BaseURL, site.Domain)
			}
		}
	default:
		crawlText = fmt.Sprintf("Queued %s for crawl (index busy — scheduled recrawl will pick it up within the hour).", rawURL)
	}

	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": crawlText},
		},
		"structuredContent": map[string]any{
			"url":     rawURL,
			"crawled": crawled,
			"queued":  !crawled,
		},
	})
}

func (h *MCPHandler) toolSearchAgents(w http.ResponseWriter, id json.RawMessage, args map[string]any) {
	p := models.SearchParams{
		Query:    asString(args["query"]),
		Category: asString(args["category"]),
		MinScore: asInt(args["min_score"]),
		HasAPI:   asBool(args["has_api"]),
		Limit:    asInt(args["limit"]),
		Page:     1,
	}
	if p.Limit <= 0 || p.Limit > 20 {
		p.Limit = 10
	}

	sites, total, err := models.SearchSites(h.DB, p)
	if err != nil {
		h.writeToolError(w, id, "search failed: "+err.Error())
		return
	}

	// Compact text view for agents (cheap tokens, still readable).
	var b strings.Builder
	fmt.Fprintf(&b, "Found %d total results (showing %d).\n\n", total, len(sites))
	for i, s := range sites {
		name := s.Name
		if name == "" {
			name = s.Domain
		}
		fmt.Fprintf(&b, "%d. %s [%d/100] — %s (%s)\n", i+1, name, s.AgenticScore, s.Domain, s.Category)
		if s.Description != "" {
			fmt.Fprintf(&b, "   %s\n", s.Description)
		}
		var signals []string
		if s.HasLLMsTxt {
			signals = append(signals, "llms.txt")
		}
		if s.HasAIPlugin {
			signals = append(signals, "ai-plugin")
		}
		if s.HasOpenAPI {
			signals = append(signals, "openapi")
		}
		if s.HasStructuredAPI {
			signals = append(signals, "api")
		}
		if s.HasMCPServer {
			signals = append(signals, "mcp")
		}
		if len(signals) > 0 {
			fmt.Fprintf(&b, "   Signals: %s\n", strings.Join(signals, ", "))
		}
		fmt.Fprintf(&b, "   URL: %s\n   Report: %s/site/%s\n\n", s.URL, h.BaseURL, s.Domain)
	}

	// Return both human-readable text (content) and structured JSON (structuredContent).
	// Per MCP spec, structuredContent lets agents parse without string-munging.
	structured := map[string]any{
		"total":   total,
		"results": sites,
	}
	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
		"structuredContent": structured,
	})
}

func (h *MCPHandler) toolGetSiteDetails(w http.ResponseWriter, id json.RawMessage, args map[string]any) {
	domain := asString(args["domain"])
	if domain == "" {
		h.writeToolError(w, id, "domain required")
		return
	}
	// Normalize: strip scheme and trailing slashes if agent passed a URL.
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "www.")
	domain = strings.TrimSuffix(domain, "/")
	domain = strings.ToLower(domain)

	site, err := models.GetSiteByDomain(h.DB, domain)
	if err != nil {
		h.writeToolError(w, id, fmt.Sprintf("site not found: %s (try search_agents first)", domain))
		return
	}

	var b strings.Builder
	name := site.Name
	if name == "" {
		name = site.Domain
	}
	fmt.Fprintf(&b, "%s — Agentic Readiness %d/100\n", name, site.AgenticScore)
	fmt.Fprintf(&b, "Domain: %s  Category: %s\n", site.Domain, site.Category)
	if site.Description != "" {
		fmt.Fprintf(&b, "%s\n", site.Description)
	}
	b.WriteString("\nSignals:\n")
	fmt.Fprintf(&b, "  llms.txt:          %s\n", yesNo(site.HasLLMsTxt))
	fmt.Fprintf(&b, "  ai-plugin.json:    %s\n", yesNo(site.HasAIPlugin))
	fmt.Fprintf(&b, "  OpenAPI spec:      %s\n", yesNo(site.HasOpenAPI))
	fmt.Fprintf(&b, "  Structured API:    %s\n", yesNo(site.HasStructuredAPI))
	fmt.Fprintf(&b, "  MCP server:        %s\n", yesNo(site.HasMCPServer))
	fmt.Fprintf(&b, "  robots.txt AI:     %s\n", yesNo(site.HasRobotsAI))
	fmt.Fprintf(&b, "  Schema.org:        %s\n", yesNo(site.HasSchemaOrg))
	fmt.Fprintf(&b, "\nFull report: %s/site/%s\n", h.BaseURL, site.Domain)

	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
		"structuredContent": site,
	})
}

func (h *MCPHandler) toolGetStats(w http.ResponseWriter, id json.RawMessage) {
	total, avg, top := models.GetStats(h.DB)
	text := fmt.Sprintf("Not Human Search index: %d agent-ready sites, average agentic score %d/100, top category %q.", total, avg, top)
	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"structuredContent": map[string]any{
			"total_sites":  total,
			"avg_score":    avg,
			"top_category": top,
		},
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (h *MCPHandler) writeResult(w http.ResponseWriter, id json.RawMessage, result any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (h *MCPHandler) writeError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message},
	})
}

// writeToolError surfaces tool-level errors as MCP spec recommends:
// a normal result with isError=true rather than a JSON-RPC error, so the
// agent can still reason about what went wrong.
func (h *MCPHandler) writeToolError(w http.ResponseWriter, id json.RawMessage, message string) {
	h.writeResult(w, id, map[string]any{
		"isError": true,
		"content": []map[string]any{
			{"type": "text", "text": message},
		},
	})
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		return 0
	}
	return 0
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
