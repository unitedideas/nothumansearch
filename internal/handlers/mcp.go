package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/unitedideas/nothumansearch/internal/crawler"
	"github.com/unitedideas/nothumansearch/internal/models"
)

const (
	mcpMaxRequestBodyBytes                int64 = 64 << 10
	mcpDiscoveryRateLimiterMaximumBuckets       = 10_000
)

// MCPHandler exposes Not Human Search as a remote MCP (Model Context Protocol) server.
// Agents can register this server via:
//
//	claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp
//
// Protocol: JSON-RPC 2.0 over Streamable HTTP (POST requests, JSON responses).
// Spec: https://modelcontextprotocol.io/specification/2025-06-18/basic/transports
type MCPHandler struct {
	DB                      *sql.DB
	BaseURL                 string
	ProviderExchange        *ProviderExchangeHandler
	discoveryRateLimiter    *mcpDiscoveryRateLimiter
	toolRateLimiter         *mcpDiscoveryRateLimiter
	probeRateLimiter        *mcpDiscoveryRateLimiter
	priorityToolRateLimiter *mcpDiscoveryRateLimiter
	priorityProbeLimiter    *mcpDiscoveryRateLimiter
}

func NewMCPHandler(db *sql.DB, baseURL string) *MCPHandler {
	return &MCPHandler{
		DB:                      db,
		BaseURL:                 baseURL,
		discoveryRateLimiter:    newMCPDiscoveryRateLimiter(90, time.Hour),
		toolRateLimiter:         newMCPDiscoveryRateLimiter(freeSearchHourlyLimit, time.Hour),
		probeRateLimiter:        newMCPDiscoveryRateLimiter(freeActiveProbeHourlyLimit, time.Hour),
		priorityToolRateLimiter: newMCPDiscoveryRateLimiter(prioritySearchHourlyLimit, time.Hour),
		priorityProbeLimiter:    newMCPDiscoveryRateLimiter(priorityActiveProbeLimit, time.Hour),
	}
}

type mcpDiscoveryRateLimiter struct {
	mu         sync.Mutex
	limit      int
	window     time.Duration
	buckets    map[string]mcpDiscoveryBucket
	maxBuckets int
	nextPrune  time.Time
}

type mcpDiscoveryBucket struct {
	count   int
	resetAt time.Time
}

func newMCPDiscoveryRateLimiter(limit int, window time.Duration) *mcpDiscoveryRateLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Hour
	}
	return &mcpDiscoveryRateLimiter{
		limit:      limit,
		window:     window,
		buckets:    map[string]mcpDiscoveryBucket{},
		maxBuckets: mcpDiscoveryRateLimiterMaximumBuckets,
	}
}

func (l *mcpDiscoveryRateLimiter) allow(key string, now time.Time) (remaining int, retryAfter time.Duration, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Expired buckets are opportunistically evicted so attacker-controlled keys
	// cannot make this process-local abuse guard grow without bound.
	if l.nextPrune.IsZero() || !now.Before(l.nextPrune) {
		for bucketKey, bucket := range l.buckets {
			if !now.Before(bucket.resetAt) {
				delete(l.buckets, bucketKey)
			}
		}
		pruneEvery := l.window
		if pruneEvery > 5*time.Minute {
			pruneEvery = 5 * time.Minute
		}
		l.nextPrune = now.Add(pruneEvery)
	}
	if key == "" {
		key = "unknown"
	}
	b, exists := l.buckets[key]
	maxBuckets := l.maxBuckets
	if maxBuckets < 1 {
		maxBuckets = mcpDiscoveryRateLimiterMaximumBuckets
	}
	if !exists && len(l.buckets) >= maxBuckets {
		// Never evict an active bucket to admit attacker-controlled churn. A new
		// identity fails closed until normal expiry pruning creates capacity.
		return 0, l.window, false
	}
	if b.resetAt.IsZero() || !now.Before(b.resetAt) {
		b = mcpDiscoveryBucket{resetAt: now.Add(l.window)}
	}
	if b.count >= l.limit {
		l.buckets[key] = b
		return 0, b.resetAt.Sub(now), false
	}
	b.count++
	l.buckets[key] = b
	return l.limit - b.count, b.resetAt.Sub(now), true
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
			"tools":       h.toolNames(),
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
	// MCP tool responses can mint or consume bearer search receipts and signed
	// provider action URLs. Treat every POST response as private capability data
	// so future tools cannot accidentally reopen a cache or referrer leak.
	protectReceiptBearingResponse(w)

	r.Body = http.MaxBytesReader(w, r.Body, mcpMaxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	var req rpcRequest
	if err := decoder.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32000, Message: "request body exceeds 64 KiB"},
			})
			return
		}
		h.writeError(w, nil, -32700, "parse error")
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32000, Message: "request body exceeds 64 KiB"},
			})
			return
		}
		h.writeError(w, req.ID, -32700, "parse error: request must contain exactly one JSON value")
		return
	}

	start := time.Now()
	ua := r.UserAgent()
	ipHash := submitHashIP(r)

	// Notifications (no id) expect no response body, just 202 Accepted.
	if len(req.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if req.Method == "initialize" || req.Method == "tools/list" {
		if h.discoveryRateLimiter == nil {
			h.discoveryRateLimiter = newMCPDiscoveryRateLimiter(90, time.Hour)
		}
		remaining, retryAfter, ok := h.discoveryRateLimiter.allow(ipHash+":"+req.Method, time.Now())
		resetSeconds := int(retryAfter.Seconds())
		if resetSeconds < 1 {
			resetSeconds = 1
		}
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", h.discoveryRateLimiter.limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(retryAfter).Unix()))
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", fmt.Sprintf("%d", resetSeconds))
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &rpcError{
					Code:    -32029,
					Message: "rate limit exceeded for MCP discovery calls; retry after the indicated number of seconds",
					Data: map[string]any{
						"limit":       h.discoveryRateLimiter.limit,
						"window":      "1h",
						"retry_after": resetSeconds,
					},
				},
			})
			return
		}
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
		go models.LogMCPRequest(h.DB, "initialize", "", nil, -1, ua, ipHash, int(time.Since(start).Milliseconds()))

	case "ping":
		h.writeResult(w, req.ID, map[string]any{})

	case "tools/list":
		h.writeResult(w, req.ID, map[string]any{
			"tools": h.toolDefinitions(),
		})
		go models.LogMCPRequest(h.DB, "tools/list", "", nil, -1, ua, ipHash, int(time.Since(start).Milliseconds()))

	case "tools/call":
		h.handleToolCall(w, r, req, start, ua, ipHash)

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
						"description": searchCategoryDescription(),
						"enum":        publicSearchCategories,
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
					"has_mcp": map[string]any{
						"type":        "boolean",
						"description": "Only return sites that expose an MCP server",
					},
					"has_openapi": map[string]any{
						"type":        "boolean",
						"description": "Only return sites with a published OpenAPI / Swagger spec",
					},
					"has_llms_txt": map[string]any{
						"type":        "boolean",
						"description": "Only return sites that publish an llms.txt file (LLM-first site summary)",
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
					"search_id": map[string]any{
						"type":        "string",
						"description": "Optional receipt returned by search_agents. If this domain was returned, NHS records a detail selection.",
					},
				},
				"required": []string{"domain"},
			},
		},
		{
			"name":        "get_stats",
			"title":       "Get Index Stats",
			"description": "Current Not Human Search index stats: total sites, average agentic score, top category, sites added in the last 7 days, count of sites exposing an MCP server, and count scoring a perfect 100/100.",
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
			"name":        "check_url",
			"title":       "Check Any URL's Agentic Readiness",
			"description": "On-demand agentic-readiness check for any URL. Runs the NHS 7-signal crawler live (llms.txt, ai-plugin.json, OpenAPI, structured API, MCP server, robots.txt AI rules, Schema.org) and returns a score 0-100 with per-signal breakdown. Use before calling an unfamiliar API to confirm it's agent-usable. Re-runnable without the submissions-table side-effect of submit_site — ideal for verify-before-use workflows.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "Full URL or bare domain (e.g. 'stripe.com' or 'https://stripe.com'). Homepage works best — NHS probes the /.well-known/ paths, /robots.txt, /llms.txt relative to the site root.",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			"name":        "verify_mcp",
			"title":       "Verify MCP Endpoint",
			"description": "Actively probe any URL to check if it is a live, spec-compliant MCP server. Sends a JSON-RPC tools/list request and verifies a valid response. Use this before depending on a third-party MCP endpoint — manifests and documentation can claim MCP support without actually serving it. Returns {verified: true/false, endpoint, note}.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "Full URL of the MCP endpoint to probe (include scheme, e.g. 'https://example.com/mcp').",
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
		{
			"name":        "list_categories",
			"title":       "List Index Categories",
			"description": "List all categories in the Not Human Search index with site counts and average agentic scores. Use this to understand what kinds of agent-ready services exist before searching — counts are live, so the distribution shifts as the index grows.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "get_top_sites",
			"title":       "Get Top Scored Sites",
			"description": "Get the highest-scored agent-ready sites in the index, optionally filtered by category. Returns sites ranked by agentic readiness score (100 = perfect agent support). Use this to discover the most agent-ready services overall or in a specific domain like 'finance' or 'developer'.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{
						"type":        "string",
						"description": searchCategoryDescription() + " Omit for all categories.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Max results (default 10, max 50)",
						"minimum":     1,
						"maximum":     50,
					},
				},
			},
		},
		{
			"name":        "recent_additions",
			"title":       "Recently Indexed Agent-First Sites",
			"description": "List agent-ready sites newly added to the Not Human Search index, sorted newest first. Use this to discover what's just landed on the agentic web — new MCP servers, fresh llms.txt adopters, new OpenAPI publishers. Good for weekly agent digests or tracking ecosystem momentum.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"days": map[string]any{
						"type":        "integer",
						"description": "Look back window in days (default 7, max 90)",
						"minimum":     1,
						"maximum":     90,
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Max results (default 10, max 50)",
						"minimum":     1,
						"maximum":     50,
					},
				},
			},
		},
		{
			"name":        "find_mcp_servers",
			"title":       "Find MCP Servers",
			"description": "List sites in the index that expose a live MCP server, ranked by agentic readiness. Use this when your agent needs to discover callable MCP endpoints for a domain ('payments', 'jobs', 'search') or overall. Pairs naturally with verify_mcp for a probe-before-use workflow.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Optional keyword to narrow results (e.g. 'payments', 'jobs', 'weather')",
					},
					"category": map[string]any{
						"type":        "string",
						"description": searchCategoryDescription() + " Omit for all categories.",
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
			"name":        "prepare_provider_action",
			"title":       "Prepare an Authorization-Attested Provider Action",
			"description": "Create a signed action handoff for a separately disclosed provider-funded offer returned by search_agents. Requires the search receipt and the caller's explicit principal-authorization attestation. Accepts controlled constraints only—no name, email, contact detail, raw prompt, agent identity, or principal identity fields. Search and direct provider access remain free; creating a ticket charges neither party. Exact v1 wording: " + h.BaseURL + "/privacy#consent-v1",
			"inputSchema": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"offer_id":  map[string]any{"type": "string", "description": "Provider-funded offer ID returned separately by search_agents."},
					"search_id": map[string]any{"type": "string", "description": "Search receipt proving the provider appeared organically."},
					"demand_topic": map[string]any{
						"type":        "string",
						"enum":        []string{"payments", "commerce", "jobs", "data", "search", "weather", "maps", "email", "messaging", "image", "video", "audio", "documents", "security", "finance", "health", "education", "news", "analytics", "automation", "productivity", "identity", "storage", "ai-tools", "developer-tools", "other"},
						"description": "One controlled topic already present on the referenced search receipt.",
					},
					"region_code": map[string]any{"type": "string", "description": "Optional uppercase ISO country or country-region code."},
					"budget_band": map[string]any{"type": "string", "enum": []string{"unspecified", "under_100", "100_499", "500_1999", "2000_plus"}},
					"urgency":     map[string]any{"type": "string", "enum": []string{"unspecified", "now", "7_days", "30_days", "researching"}},
					"requirement_flags": map[string]any{
						"type": "array", "maxItems": 8, "uniqueItems": true,
						"items": map[string]any{"type": "string", "enum": []string{"api_access", "mcp", "sandbox", "self_serve", "enterprise", "compliance", "multilingual", "human_support"}},
					},
					"principal_consent": map[string]any{"type": "boolean", "const": true, "description": "Caller attests that the human/company principal authorized this handoff under the published versioned wording."},
					"consent_version":   map[string]any{"type": "string", "enum": []string{models.ProviderPrincipalConsentV1}, "description": h.BaseURL + "/privacy#consent-v1"},
				},
				"required": []string{"offer_id", "search_id", "demand_topic", "principal_consent", "consent_version"},
			},
		},
	}
}

func (h *MCPHandler) toolNames() []string {
	tools := h.toolDefinitions()
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		name, ok := tool["name"].(string)
		if ok && name != "" {
			names = append(names, name)
		}
	}
	return names
}

func (h *MCPHandler) handleToolCall(w http.ResponseWriter, r *http.Request, req rpcRequest, start time.Time, ua, ipHash string) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		h.writeError(w, req.ID, -32602, "invalid params")
		return
	}
	knownTool := isKnownMCPTool(params.Name)

	access := resolveRequestRateAccess(h.DB, r)
	activeTool := isNHSActiveProbeTool(params.Name)
	limiter := h.toolRateLimiter
	if activeTool {
		limiter = h.probeRateLimiter
	}
	if access.tier == priorityRateLimitTier {
		limiter = h.priorityToolRateLimiter
		if activeTool {
			limiter = h.priorityProbeLimiter
		}
	}
	if limiter == nil {
		limit := freeSearchHourlyLimit
		if activeTool {
			limit = freeActiveProbeHourlyLimit
		}
		if access.tier == priorityRateLimitTier {
			limit = prioritySearchHourlyLimit
			if activeTool {
				limit = priorityActiveProbeLimit
			}
		}
		limiter = newMCPDiscoveryRateLimiter(limit, time.Hour)
		if access.tier == priorityRateLimitTier && activeTool {
			h.priorityProbeLimiter = limiter
		} else if access.tier == priorityRateLimitTier {
			h.priorityToolRateLimiter = limiter
		} else if activeTool {
			h.probeRateLimiter = limiter
		} else {
			h.toolRateLimiter = limiter
		}
	}
	rateBucket := access.bucket + ":mcp-tools"
	if activeTool {
		rateBucket = access.bucket + ":mcp-active"
	}
	remaining, retryAfter, ok := limiter.allow(rateBucket, time.Now())
	if ok && access.key != nil && knownTool {
		var reserved bool
		access, reserved = reservePriorityUnit(h.DB, access, "mcp", "tools/call", "/mcp", params.Name)
		if !reserved {
			access = freeRequestRateAccess(r)
			if activeTool {
				if h.probeRateLimiter == nil {
					h.probeRateLimiter = newMCPDiscoveryRateLimiter(freeActiveProbeHourlyLimit, time.Hour)
				}
				limiter = h.probeRateLimiter
				rateBucket = access.bucket + ":mcp-active"
			} else {
				if h.toolRateLimiter == nil {
					h.toolRateLimiter = newMCPDiscoveryRateLimiter(freeSearchHourlyLimit, time.Hour)
				}
				limiter = h.toolRateLimiter
				rateBucket = access.bucket + ":mcp-tools"
			}
			remaining, retryAfter, ok = limiter.allow(rateBucket, time.Now())
		}
	}
	access.setHeaders(w)
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.limit))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(retryAfter).Unix()))
	if !ok {
		retrySeconds := max(1, int(retryAfter.Seconds()))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retrySeconds))
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &rpcError{
				Code:    -32029,
				Message: "MCP safety limit exceeded; retry after the indicated interval. Discovery access is not paywalled",
				Data: map[string]any{
					"limit":       limiter.limit,
					"window":      "1h",
					"retry_after": retrySeconds,
				},
			},
		})
		return
	}

	toolResponse := &mcpToolResponseBuffer{header: w.Header()}
	switch params.Name {
	case "search_agents", "search":
		// "search" is an alias; some agents guess this simpler name when
		// surveying the tool list. Route both to the same handler.
		h.toolSearchAgents(toolResponse, req.ID, params.Arguments, demandRequestIsSynthetic(r))
	case "get_site_details":
		h.toolGetSiteDetails(toolResponse, req.ID, params.Arguments)
	case "get_stats":
		h.toolGetStats(toolResponse, req.ID)
	case "submit_site":
		h.toolSubmitSite(toolResponse, req.ID, params.Arguments)
	case "register_monitor":
		h.toolRegisterMonitor(toolResponse, req.ID, params.Arguments)
	case "check_url":
		h.toolCheckURL(toolResponse, req.ID, params.Arguments)
	case "verify_mcp":
		h.toolVerifyMCP(toolResponse, req.ID, params.Arguments)
	case "list_categories":
		h.toolListCategories(toolResponse, req.ID)
	case "get_top_sites":
		h.toolGetTopSites(toolResponse, req.ID, params.Arguments)
	case "find_mcp_servers":
		h.toolFindMCPServers(toolResponse, req.ID, params.Arguments, demandRequestIsSynthetic(r))
	case "recent_additions":
		h.toolRecentAdditions(toolResponse, req.ID, params.Arguments)
	case "prepare_provider_action":
		h.toolPrepareProviderAction(toolResponse, req.ID, params.Arguments, r)
	default:
		h.writeToolError(toolResponse, req.ID, "unknown tool: "+params.Name)
	}
	if toolResponse.toolError {
		access = releasePriorityUnit(h.DB, access)
		access.setHeaders(w)
	}
	if err := toolResponse.flushTo(w); err != nil {
		log.Printf("MCP response write %s: %v", params.Name, err)
	}

	safeArgs := mcpAnalyticsArguments(params.Name, params.Arguments)
	argsJSON, _ := json.Marshal(safeArgs)
	go models.LogMCPRequest(h.DB, "tools/call", params.Name, argsJSON, -1, ua, ipHash, int(time.Since(start).Milliseconds()))
	go models.LogIntentEvent(h.DB, models.IntentEvent{
		EventName:  "mcp_tool_call",
		EntityType: "mcp_tool",
		EntityID:   params.Name,
		UserAgent:  ua,
		IPHash:     ipHash,
		IsBot:      true,
		Metadata: map[string]any{
			"arguments": safeArgs,
		},
	})
}

func isNHSActiveProbeTool(name string) bool {
	switch name {
	case "check_url", "verify_mcp", "submit_site", "register_monitor":
		return true
	default:
		return false
	}
}

func isKnownMCPTool(name string) bool {
	switch name {
	case "search_agents", "search", "get_site_details", "get_stats", "submit_site",
		"register_monitor", "check_url", "verify_mcp", "list_categories",
		"get_top_sites", "find_mcp_servers", "recent_additions", "prepare_provider_action":
		return true
	default:
		return false
	}
}

func (h *MCPHandler) toolPrepareProviderAction(w http.ResponseWriter, id json.RawMessage, args map[string]any, r *http.Request) {
	if h.ProviderExchange == nil || h.ProviderExchange.Signer == nil {
		h.writeToolError(w, id, "signed provider actions are not configured")
		return
	}
	if retry, ok := h.ProviderExchange.consumeActionTicketLimit(r, time.Now()); !ok {
		retrySeconds := max(1, int(retry.Seconds()))
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retrySeconds))
		h.writeToolError(w, id, "provider action ticket safety limit exceeded; retry after the indicated interval")
		return
	}
	if err := validatePrepareProviderActionArguments(args); err != nil {
		h.writeToolError(w, id, err.Error())
		return
	}
	prepared, err := h.ProviderExchange.prepareActionTicket(actionTicketRequest{
		OfferID:          asString(args["offer_id"]),
		SearchID:         asString(args["search_id"]),
		DemandTopic:      asString(args["demand_topic"]),
		RegionCode:       asString(args["region_code"]),
		BudgetBand:       asString(args["budget_band"]),
		Urgency:          asString(args["urgency"]),
		RequirementFlags: asStringList(args["requirement_flags"]),
		PrincipalConsent: asBool(args["principal_consent"]),
		ConsentVersion:   asString(args["consent_version"]),
	})
	if err != nil {
		_, message := providerExchangeStatus(err)
		h.writeToolError(w, id, message)
		return
	}
	ticket, _ := prepared["ticket"].(*models.ActionTicket)
	actionURL, _ := prepared["action_url"].(string)
	text := "Prepared a consent-attested provider action. Creating this ticket charged neither the principal nor the provider."
	if ticket != nil {
		text += " Ticket: " + ticket.ID + "."
		if ticket.Replayed {
			text += " This was an exact idempotent replay; NHS reconstructed the same token from its non-secret nonce and retained signing key ID."
		}
	}
	if actionURL != "" {
		text += " Provider action URL: " + actionURL
	}
	h.writeResult(w, id, map[string]any{
		"content":           []map[string]any{{"type": "text", "text": text}},
		"structuredContent": prepared,
	})
}

// validatePrepareProviderActionArguments makes the MCP privacy boundary
// enforceable at runtime. Tool schemas are advisory to clients, and JSON-RPC
// arguments arrive as a map, so a caller could otherwise add free-form contact,
// prompt, or notes fields that the HTTP strict decoder would never see.
func validatePrepareProviderActionArguments(args map[string]any) error {
	allowed := map[string]struct{}{
		"offer_id": {}, "search_id": {}, "demand_topic": {}, "region_code": {},
		"budget_band": {}, "urgency": {}, "requirement_flags": {},
		"principal_consent": {}, "consent_version": {},
	}
	for key := range args {
		if _, ok := allowed[key]; !ok {
			return errors.New("provider action arguments contain an unsupported field; contact details, raw prompts, notes, and identities are not accepted")
		}
	}
	for _, key := range []string{"offer_id", "search_id", "demand_topic", "region_code", "budget_band", "urgency", "consent_version"} {
		if value, exists := args[key]; exists {
			if _, ok := value.(string); !ok {
				return errors.New("provider action arguments have an invalid field type")
			}
		}
	}
	if value, exists := args["principal_consent"]; exists {
		if _, ok := value.(bool); !ok {
			return errors.New("provider action arguments have an invalid field type")
		}
	}
	if value, exists := args["requirement_flags"]; exists {
		items, ok := value.([]any)
		if !ok {
			return errors.New("provider action arguments have an invalid field type")
		}
		for _, item := range items {
			if _, ok := item.(string); !ok {
				return errors.New("provider action arguments have an invalid field type")
			}
		}
	}
	return nil
}

func asStringList(value any) []string {
	items, ok := value.([]any)
	if !ok {
		if strings, ok := value.([]string); ok {
			return strings
		}
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			result = append(result, value)
		}
	}
	return result
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
	// Case-insensitive check: without ToLower, "HTTPS://x.com" would be
	// re-prefixed to "https://HTTPS://x.com".
	lower := strings.ToLower(rawURL)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		rawURL = "https://" + rawURL
	}
	if err := crawler.ValidatePublicURL(rawURL); err != nil {
		h.writeToolError(w, id, "url must resolve to a public HTTP(S) address")
		return
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

func (h *MCPHandler) toolSearchAgents(w http.ResponseWriter, id json.RawMessage, args map[string]any, synthetic bool) {
	p := models.SearchParams{
		Query:      asString(args["query"]),
		Category:   asString(args["category"]),
		MinScore:   asInt(args["min_score"]),
		HasAPI:     asBool(args["has_api"]),
		HasMCP:     asBool(args["has_mcp"]),
		HasOpenAPI: asBool(args["has_openapi"]),
		HasLLMsTxt: asBool(args["has_llms_txt"]),
		Limit:      asInt(args["limit"]),
		Page:       1,
	}
	if p.Limit <= 0 || p.Limit > 20 {
		p.Limit = 10
	}

	sites, total, err := models.SearchSites(h.DB, p)
	if err != nil {
		h.writeToolError(w, id, "search failed: "+err.Error())
		return
	}

	searchID := h.recordMCPSearchReceipt(p, total, sites, synthetic)
	paidOffers := []publicProviderOffer{}
	paidOffersAvailable := false
	if searchID != "" {
		modelOffers, paidErr := models.ListPublicProviderOffersForOrganicResults(h.DB, sites)
		if paidErr != nil {
			log.Printf("provider offers MCP search: %v", paidErr)
		} else if returnedErr := models.RecordProviderOffersReturned(h.DB, searchID, modelOffers); returnedErr != nil {
			log.Printf("provider offers MCP return evidence: %v", returnedErr)
		} else {
			paidOffers = publicProviderOfferModelViews(modelOffers, h.BaseURL)
			paidOffersAvailable = len(modelOffers) > 0
		}
	}

	// Compact text view for agents (cheap tokens, still readable).
	var b strings.Builder
	fmt.Fprintf(&b, "Found %d total results (showing %d).", total, len(sites))
	if searchID != "" {
		fmt.Fprintf(&b, " Search receipt: %s. Pass this search_id to get_site_details to record a result selection.", searchID)
	} else {
		fmt.Fprint(&b, " Search receipt unavailable; no selection attribution will be claimed for this response.")
	}
	b.WriteString("\n\n")
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
	if len(paidOffers) > 0 {
		b.WriteString("Provider-funded actions (separate from organic order; optional):\n")
		for _, offer := range paidOffers {
			fmt.Fprintf(&b, "- Organic result %d, %s: %s (%s). NHS may receive %d %s minor units if the provider reports %s. Use prepare_provider_action with offer_id %s and this search receipt.\n",
				offer.OrganicPosition, offer.ProviderDomain, offer.Name, offer.ActionType,
				offer.NHSCompensation.AmountMinor, offer.NHSCompensation.Currency,
				offer.NHSCompensation.Event, offer.ID)
		}
		b.WriteString("\n")
	}

	// Return both human-readable text (content) and structured JSON (structuredContent).
	// Per MCP spec, structuredContent lets agents parse without string-munging.
	structured := map[string]any{
		"access":                "free",
		"receipt_recorded":      searchID != "",
		"total":                 total,
		"results":               sites,
		"paid_offers":           paidOffers,
		"paid_offers_available": paidOffersAvailable,
	}
	if searchID != "" {
		structured["search_id"] = searchID
	}
	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
		"structuredContent": structured,
	})
}

func (h *MCPHandler) recordMCPSearchReceipt(p models.SearchParams, total int, sites []models.Site, synthetic bool) string {
	searchID, err := recordDemandSearchReceipt(h.DB, models.DemandSearchReceipt{
		Surface:     "mcp",
		Query:       p.Query,
		Category:    p.Category,
		HasAPI:      p.HasAPI,
		HasMCP:      p.HasMCP,
		HasOpenAPI:  p.HasOpenAPI,
		HasLLMsTxt:  p.HasLLMsTxt,
		ResultCount: total,
		Page:        max(1, p.Page),
		PageSize:    p.Limit,
		Synthetic:   synthetic,
	}, sites)
	if err != nil {
		log.Printf("demand receipt MCP search: %v", err)
		return ""
	}
	return searchID
}

func mcpAnalyticsArguments(tool string, args map[string]any) map[string]any {
	safe := map[string]any{}
	copyBool := func(key string) {
		if value, ok := args[key].(bool); ok {
			safe[key] = value
		}
	}
	copyBoundedString := func(key string) {
		if value := strings.TrimSpace(asString(args[key])); value != "" {
			if len(value) > 120 {
				value = value[:120]
			}
			safe[key] = value
		}
	}

	switch tool {
	case "search_agents", "search":
		safe["demand_topics"] = models.ClassifyDemandTopics(asString(args["query"]), asString(args["category"]))
		for _, key := range []string{"has_api", "has_mcp", "has_openapi", "has_llms_txt"} {
			copyBool(key)
		}
	case "get_site_details", "register_monitor":
		copyBoundedString("domain")
	case "submit_site", "check_url", "verify_mcp":
		// URLs can contain credentials or private paths. Retain only a coarse
		// redacted label for operational counts, never the raw value.
		safe["target_supplied"] = strings.TrimSpace(asString(args["url"])) != ""
	case "find_mcp_servers":
		safe["demand_topics"] = models.ClassifyDemandTopics(asString(args["query"]), asString(args["category"]))
	case "prepare_provider_action":
		if value := asString(args["demand_topic"]); stringInSet(value, []string{"payments", "commerce", "jobs", "data", "search", "weather", "maps", "email", "messaging", "image", "video", "audio", "documents", "security", "finance", "health", "education", "news", "analytics", "automation", "productivity", "identity", "storage", "ai-tools", "developer-tools", "other"}) {
			safe["demand_topic"] = value
		}
		if value := asString(args["budget_band"]); stringInSet(value, []string{"unspecified", "under_100", "100_499", "500_1999", "2000_plus"}) {
			safe["budget_band"] = value
		}
		if value := asString(args["urgency"]); stringInSet(value, []string{"unspecified", "now", "7_days", "30_days", "researching"}) {
			safe["urgency"] = value
		}
		copyBool("principal_consent")
		flags := []string{}
		for _, value := range asStringList(args["requirement_flags"]) {
			if stringInSet(value, []string{"api_access", "mcp", "sandbox", "self_serve", "enterprise", "compliance", "multilingual", "human_support"}) {
				flags = append(flags, value)
			}
			if len(flags) == 8 {
				break
			}
		}
		safe["requirement_flags"] = flags
	}
	return safe
}

func stringInSet(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func (h *MCPHandler) toolGetSiteDetails(w http.ResponseWriter, id json.RawMessage, args map[string]any) {
	domain := asString(args["domain"])
	if domain == "" {
		h.writeToolError(w, id, "domain required")
		return
	}
	// Normalize: lowercase FIRST so scheme/www prefixes strip regardless
	// of casing, then strip. The previous order (TrimPrefix then ToLower)
	// let an uppercase-scheme input like "HTTPS://Stripe.COM" bypass the
	// prefix strip — ToLower'd to "https://stripe.com" but GetSiteByDomain
	// expects bare "stripe.com" and silently returned not-found.
	domain = strings.ToLower(domain)
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "www.")
	domain = strings.TrimSuffix(domain, "/")

	site, err := models.GetSiteByDomain(h.DB, domain)
	if err != nil {
		h.writeToolError(w, id, fmt.Sprintf("site not found: %s (try search_agents first)", domain))
		return
	}
	if searchID := strings.TrimSpace(asString(args["search_id"])); searchID != "" {
		recorded, selectionErr := models.RecordDemandSelection(h.DB, searchID, site.Domain, "mcp")
		if selectionErr != nil {
			log.Printf("demand selection MCP detail: %v", selectionErr)
		} else {
			w.Header().Set("NHS-Selection-Recorded", strconv.FormatBool(recorded))
		}
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

	// Cheap signal breakouts — three single COUNTs on indexed columns.
	var addedThisWeek, mcpSites, perfectScore int
	_ = h.DB.QueryRow(`SELECT count(*) FROM sites WHERE ` + models.AgentFirstFilter + ` AND created_at >= NOW() - INTERVAL '7 days'`).Scan(&addedThisWeek)
	_ = h.DB.QueryRow(`SELECT count(*) FROM sites WHERE ` + models.AgentFirstFilter + ` AND has_mcp_server = true`).Scan(&mcpSites)
	_ = h.DB.QueryRow(`SELECT count(*) FROM sites WHERE ` + models.AgentFirstFilter + ` AND agentic_score >= 100`).Scan(&perfectScore)

	text := fmt.Sprintf("Not Human Search index: %d agent-ready sites, average agentic score %d/100, top category %q. %d sites added this week. %d expose an MCP server. %d scored 100/100.", total, avg, top, addedThisWeek, mcpSites, perfectScore)
	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"structuredContent": map[string]any{
			"total_sites":     total,
			"avg_score":       avg,
			"top_category":    top,
			"added_this_week": addedThisWeek,
			"mcp_sites":       mcpSites,
			"perfect_score":   perfectScore,
		},
	})
}

// toolCheckURL runs a live 7-signal crawl on the provided URL and returns
// the agentic-readiness score + per-signal breakdown. Unlike submit_site,
// no record is written to the submissions table — this is for
// verify-before-use flows where the agent just wants the score.
// The crawl is capped at 25s so a slow target can't pin the request.
func (h *MCPHandler) toolCheckURL(w http.ResponseWriter, id json.RawMessage, args map[string]any) {
	raw, _ := args["url"].(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		h.writeToolError(w, id, "url is required")
		return
	}
	// Case-insensitive scheme check (see companion fix in api.go + check.go)
	lowerRaw := strings.ToLower(raw)
	if !strings.HasPrefix(lowerRaw, "http://") && !strings.HasPrefix(lowerRaw, "https://") {
		raw = "https://" + raw
	}
	if err := crawler.ValidatePublicURL(raw); err != nil {
		h.writeToolError(w, id, "url must resolve to a public HTTP(S) address")
		return
	}
	select {
	case submitCrawlSem <- struct{}{}:
		// The crawl goroutine owns the slot until it exits, including after a
		// handler timeout, so distributed callers cannot accumulate background
		// crawl work faster than it completes.
	default:
		h.writeToolError(w, id, "live check capacity is busy; retry shortly")
		return
	}

	done := make(chan struct{})
	var site *models.Site
	var crawlErr error
	go func() {
		defer func() { <-submitCrawlSem }()
		site, crawlErr = crawler.CrawlSite(raw)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(25 * time.Second):
		h.writeToolError(w, id, "target site took too long to respond (25s timeout)")
		return
	}
	if crawlErr != nil {
		h.writeToolError(w, id, "crawl failed: "+crawlErr.Error())
		return
	}

	// Upsert as a fire-and-forget side-effect so repeat checks improve the index
	// over time. Failure here doesn't affect the caller's response — same
	// pattern as /api/v1/check.
	if h.DB != nil {
		go func(s *models.Site) {
			_ = models.UpsertSite(h.DB, s)
		}(site)
	}

	var signals []string
	if site.HasLLMsTxt {
		signals = append(signals, "llms.txt")
	}
	if site.HasAIPlugin {
		signals = append(signals, "ai-plugin")
	}
	if site.HasOpenAPI {
		signals = append(signals, "openapi")
	}
	if site.HasStructuredAPI {
		signals = append(signals, "api")
	}
	if site.HasMCPServer {
		signals = append(signals, "mcp")
	}
	if site.HasRobotsAI {
		signals = append(signals, "robots-ai")
	}
	if site.HasSchemaOrg {
		signals = append(signals, "schema.org")
	}
	sigList := "none detected"
	if len(signals) > 0 {
		sigList = strings.Join(signals, ", ")
	}
	text := fmt.Sprintf("check_url %s\n  score: %d/100\n  category: %s\n  signals: %s\n  report: %s/site/%s",
		site.Domain, site.AgenticScore, site.Category, sigList, h.BaseURL, site.Domain)

	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"structuredContent": map[string]any{
			"domain":        site.Domain,
			"url":           site.URL,
			"agentic_score": site.AgenticScore,
			"category":      site.Category,
			"signals": map[string]bool{
				"llms_txt":       site.HasLLMsTxt,
				"ai_plugin":      site.HasAIPlugin,
				"openapi":        site.HasOpenAPI,
				"structured_api": site.HasStructuredAPI,
				"mcp_server":     site.HasMCPServer,
				"robots_ai":      site.HasRobotsAI,
				"schema_org":     site.HasSchemaOrg,
			},
			"report_url": h.BaseURL + "/site/" + site.Domain,
		},
	})
}

func (h *MCPHandler) toolVerifyMCP(w http.ResponseWriter, id json.RawMessage, args map[string]any) {
	raw, _ := args["url"].(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		h.writeToolError(w, id, "url is required")
		return
	}
	// Case-insensitive scheme check (see companion fix in api.go + check.go)
	lowerRaw := strings.ToLower(raw)
	if !strings.HasPrefix(lowerRaw, "http://") && !strings.HasPrefix(lowerRaw, "https://") {
		raw = "https://" + raw
	}
	if err := crawler.ValidatePublicURL(raw); err != nil {
		h.writeToolError(w, id, "url must resolve to a public HTTP(S) address")
		return
	}
	verified := crawler.ProbeMCPJSONRPC(raw)
	note := "Endpoint responded with valid JSON-RPC 2.0 — server is live and MCP-compliant."
	if !verified {
		note = "Endpoint did not respond with valid JSON-RPC 2.0. Could be down, not an MCP server, or requires an initialize() handshake this probe does not send."
	}
	text := fmt.Sprintf("verify_mcp %s\n  verified: %v\n  %s", raw, verified, note)
	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"structuredContent": map[string]any{
			"verified": verified,
			"endpoint": raw,
			"note":     note,
		},
	})
}

func (h *MCPHandler) toolListCategories(w http.ResponseWriter, id json.RawMessage) {
	cats, err := models.GetCategories(h.DB)
	if err != nil {
		h.writeToolError(w, id, "failed to list categories: "+err.Error())
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d categories in the index:\n\n", len(cats))
	for _, c := range cats {
		fmt.Fprintf(&b, "  %-20s %4d sites  (avg score %d)\n", c.Name, c.Count, c.AvgScore)
	}

	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
		"structuredContent": map[string]any{
			"categories": cats,
		},
	})
}

func (h *MCPHandler) toolGetTopSites(w http.ResponseWriter, id json.RawMessage, args map[string]any) {
	category := asString(args["category"])
	limit := asInt(args["limit"])
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	sites, err := models.GetTopSites(h.DB, category, limit)
	if err != nil {
		h.writeToolError(w, id, "query failed: "+err.Error())
		return
	}

	var b strings.Builder
	label := "all categories"
	if category != "" {
		label = category
	}
	fmt.Fprintf(&b, "Top %d agent-ready sites (%s):\n\n", len(sites), label)
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
		fmt.Fprintf(&b, "   URL: %s\n\n", s.URL)
	}

	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
		"structuredContent": map[string]any{
			"category": category,
			"results":  sites,
		},
	})
}

// toolRecentAdditions surfaces newly-indexed agent-first sites. Pairs
// with get_stats — agents checking "what's new" can sample fresh sites
// without a full crawl of the index.
func (h *MCPHandler) toolRecentAdditions(w http.ResponseWriter, id json.RawMessage, args map[string]any) {
	days := asInt(args["days"])
	if days <= 0 {
		days = 7
	}
	limit := asInt(args["limit"])
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	sites, err := models.GetRecentSites(h.DB, days, limit)
	if err != nil {
		h.writeToolError(w, id, "query failed: "+err.Error())
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d sites added in the last %d days:\n\n", len(sites), days)
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
		fmt.Fprintf(&b, "   Added: %s   URL: %s\n\n", s.CreatedAt.Format("2006-01-02"), s.URL)
	}

	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
		"structuredContent": map[string]any{
			"days":    days,
			"results": sites,
		},
	})
}

// toolFindMCPServers is a convenience wrapper over SearchSites that pins
// HasMCP=true. Agents looking specifically for MCP endpoints don't have
// to know the has_mcp filter exists on search_agents — they can discover
// this tool by name. Pairs with verify_mcp for probe-before-use flows.
func (h *MCPHandler) toolFindMCPServers(w http.ResponseWriter, id json.RawMessage, args map[string]any, synthetic bool) {
	p := models.SearchParams{
		Query:    asString(args["query"]),
		Category: asString(args["category"]),
		HasMCP:   true,
		Limit:    asInt(args["limit"]),
		Page:     1,
	}
	if p.Limit <= 0 || p.Limit > 20 {
		p.Limit = 10
	}

	sites, total, err := models.SearchSites(h.DB, p)
	if err != nil {
		h.writeToolError(w, id, "query failed: "+err.Error())
		return
	}
	searchID := h.recordMCPSearchReceipt(p, total, sites, synthetic)

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d MCP-exposing sites (showing %d).", total, len(sites))
	if searchID != "" {
		fmt.Fprintf(&b, " Search receipt: %s. Pass this search_id to get_site_details to record a result selection.", searchID)
	} else {
		fmt.Fprint(&b, " Search receipt unavailable; no selection attribution will be claimed for this response.")
	}
	b.WriteString("\n\n")
	for i, s := range sites {
		name := s.Name
		if name == "" {
			name = s.Domain
		}
		fmt.Fprintf(&b, "%d. %s [%d/100] — %s (%s)\n", i+1, name, s.AgenticScore, s.Domain, s.Category)
		if s.Description != "" {
			fmt.Fprintf(&b, "   %s\n", s.Description)
		}
		fmt.Fprintf(&b, "   URL: %s\n   Report: %s/site/%s\n\n", s.URL, h.BaseURL, s.Domain)
	}

	structured := map[string]any{
		"receipt_recorded": searchID != "",
		"total":            total,
		"results":          sites,
	}
	if searchID != "" {
		structured["search_id"] = searchID
	}
	h.writeResult(w, id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": b.String()},
		},
		"structuredContent": structured,
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// mcpToolResponseBuffer delays committing a tool response until quota
// accounting knows whether the tool succeeded. This lets NHS refund failed or
// capacity-rejected priority calls and publish the corrected remaining quota in
// the same HTTP response.
type mcpToolResponseBuffer struct {
	header    http.Header
	body      bytes.Buffer
	status    int
	toolError bool
}

func (w *mcpToolResponseBuffer) Header() http.Header { return w.header }

func (w *mcpToolResponseBuffer) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *mcpToolResponseBuffer) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}

func (w *mcpToolResponseBuffer) flushTo(dst http.ResponseWriter) error {
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	dst.WriteHeader(status)
	_, err := dst.Write(w.body.Bytes())
	return err
}

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
	if buffered, ok := w.(*mcpToolResponseBuffer); ok {
		buffered.toolError = true
	}
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
