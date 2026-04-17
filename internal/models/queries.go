package models

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/lib/pq"
)

func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "")
}

func UpsertSite(db *sql.DB, s *Site) error {
	s.Name = sanitizeUTF8(s.Name)
	s.Description = sanitizeUTF8(s.Description)
	s.LLMsTxtContent = sanitizeUTF8(s.LLMsTxtContent)
	s.OpenAPISummary = sanitizeUTF8(s.OpenAPISummary)
	s.CrawlError = sanitizeUTF8(s.CrawlError)
	if s.Tags == nil {
		s.Tags = pq.StringArray{}
	}
	_, err := db.Exec(`
		INSERT INTO sites (domain, url, name, description,
			has_llms_txt, has_ai_plugin, has_openapi, has_robots_ai,
			has_structured_api, has_mcp_server, has_schema_org,
			llms_txt_content, openapi_summary, mcp_endpoint,
			agentic_score, category, tags,
			is_featured, last_crawled_at, crawl_status, crawl_error,
			has_favicon, favicon_url,
			search_vector)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,
			setweight(to_tsvector('english', COALESCE($3, '')), 'A') ||
			setweight(to_tsvector('english', COALESCE($1, '')), 'A') ||
			setweight(to_tsvector('english', COALESCE($4, '')), 'B') ||
			setweight(to_tsvector('english', COALESCE($16, '')), 'C') ||
			setweight(to_tsvector('english', COALESCE(array_to_string($17::text[], ' '), '')), 'C'))
		ON CONFLICT (domain) DO UPDATE SET
			url=EXCLUDED.url, name=EXCLUDED.name, description=EXCLUDED.description,
			has_llms_txt=EXCLUDED.has_llms_txt, has_ai_plugin=EXCLUDED.has_ai_plugin,
			has_openapi=EXCLUDED.has_openapi, has_robots_ai=EXCLUDED.has_robots_ai,
			has_structured_api=EXCLUDED.has_structured_api, has_mcp_server=EXCLUDED.has_mcp_server,
			has_schema_org=EXCLUDED.has_schema_org,
			llms_txt_content=EXCLUDED.llms_txt_content, openapi_summary=EXCLUDED.openapi_summary,
			mcp_endpoint=EXCLUDED.mcp_endpoint,
			agentic_score=EXCLUDED.agentic_score, category=EXCLUDED.category, tags=EXCLUDED.tags,
			is_featured=(sites.is_featured OR EXCLUDED.is_featured),
			last_crawled_at=EXCLUDED.last_crawled_at, crawl_status=EXCLUDED.crawl_status,
			crawl_error=EXCLUDED.crawl_error,
			has_favicon=EXCLUDED.has_favicon, favicon_url=EXCLUDED.favicon_url,
			updated_at=NOW(),
			search_vector=EXCLUDED.search_vector`,
		s.Domain, s.URL, s.Name, s.Description,
		s.HasLLMsTxt, s.HasAIPlugin, s.HasOpenAPI, s.HasRobotsAI,
		s.HasStructuredAPI, s.HasMCPServer, s.HasSchemaOrg,
		s.LLMsTxtContent, s.OpenAPISummary, s.MCPEndpoint,
		s.AgenticScore, s.Category, pq.Array(s.Tags),
		s.IsFeatured, s.LastCrawledAt, s.CrawlStatus, s.CrawlError,
		s.HasFavicon, s.FaviconURL,
	)
	return err
}

type SearchParams struct {
	Query       string
	Category    string
	Tag         string // exact tag match against tags[] column
	MinScore    int
	HasAPI      bool
	HasMCP      bool
	HasOpenAPI  bool
	HasLLMsTxt  bool
	Limit       int
	Page        int
}

func SearchSites(db *sql.DB, p SearchParams) ([]Site, int, error) {
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Page <= 0 {
		p.Page = 1
	}

	var conditions []string
	var args []interface{}
	argN := 1

	conditions = append(conditions, "crawl_status = 'success'")
	// Only show sites with at least one HARD agent signal — something an agent can
	// programmatically interact with. llms.txt alone is passive content (markdown index)
	// and does not qualify. Must match AgentFirstFilter const below.
	conditions = append(conditions, "(has_structured_api = true OR has_openapi = true OR has_ai_plugin = true OR has_mcp_server = true)")

	useFTS := false
	var tsQueryArg int

	if p.Query != "" {
		// Build a tsquery from the user's input for full-text search
		// Convert "payment api" → "payment & api" for AND matching,
		// but also add an ILIKE fallback for short/partial terms
		words := strings.Fields(p.Query)
		if len(words) == 1 && len(words[0]) <= 2 {
			// Very short query: ILIKE only (FTS won't match partial 1-2 char terms)
			conditions = append(conditions, fmt.Sprintf(
				"(name ILIKE $%d OR description ILIKE $%d OR domain ILIKE $%d OR category ILIKE $%d OR array_to_string(tags, ' ') ILIKE $%d)",
				argN, argN, argN, argN, argN))
			args = append(args, "%"+p.Query+"%")
			argN++
		} else {
			// Full-text search with ts_rank + ILIKE fallback for partial matches
			// Build tsquery: each word joined with & (AND), with :* prefix matching
			var tsTerms []string
			for _, w := range words {
				// Sanitize: remove non-alphanumeric except hyphens
				clean := strings.Map(func(r rune) rune {
					if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
						return r
					}
					return -1
				}, w)
				if clean != "" {
					tsTerms = append(tsTerms, clean+":*")
				}
			}
			if len(tsTerms) > 0 {
				// Use OR across terms so multi-word queries surface partial matches.
				// But ALSO pass an AND query so sites matching every term can be boosted
				// in the ORDER BY — keeps precision for queries like "SMS text messaging"
				// where messagebird (all-terms) should beat agentsafe (one-term, higher score).
				tsQueryOr := strings.Join(tsTerms, " | ")
				tsQueryAnd := strings.Join(tsTerms, " & ")
				// Reference $argN+1 (AND tsquery) in WHERE as a harmless tautology so
				// Postgres can infer its type even on the count query where $argN+1 is
				// otherwise only used in ORDER BY. Without this, pq errors with
				// "could not determine data type of parameter".
				conditions = append(conditions, fmt.Sprintf(
					"(search_vector @@ to_tsquery('english', $%d) OR name ILIKE $%d OR domain ILIKE $%d OR description ILIKE $%d OR array_to_string(tags, ' ') ILIKE $%d) AND ($%d::text IS NOT NULL)",
					argN, argN+2, argN+2, argN+2, argN+2, argN+1))
				args = append(args, tsQueryOr, tsQueryAnd, "%"+p.Query+"%")
				tsQueryArg = argN
				useFTS = true
				argN += 3
			}
		}
	}
	if p.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argN))
		args = append(args, p.Category)
		argN++
	}
	if p.MinScore > 0 {
		conditions = append(conditions, fmt.Sprintf("agentic_score >= $%d", argN))
		args = append(args, p.MinScore)
		argN++
	}
	if p.HasAPI {
		conditions = append(conditions, "has_structured_api = true")
	}
	if p.HasMCP {
		conditions = append(conditions, "has_mcp_server = true")
	}
	if p.HasOpenAPI {
		conditions = append(conditions, "has_openapi = true")
	}
	if p.HasLLMsTxt {
		conditions = append(conditions, "has_llms_txt = true")
	}
	if p.Tag != "" {
		conditions = append(conditions, fmt.Sprintf("$%d = ANY(tags)", argN))
		args = append(args, p.Tag)
		argN++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count
	var total int
	countQ := "SELECT count(*) FROM sites " + where
	if err := db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count query: %w", err)
	}

	// Fetch with relevance ranking
	offset := (p.Page - 1) * p.Limit
	var orderBy string
	if useFTS {
		// Rank by: all-terms-match boost + OR-relevance * score-multiplier + additive
		// score floor. The additive term (score/75) gives high-score sites a baseline
		// advantage of up to ~1.33 even when ts_rank is weak — prevents score-20 sites
		// with better text match from outranking score-100 agent-ready sites on common
		// queries like "payment api", "cron monitor", "job board". Baseline captured at
		// project_nhs_ranking_improvement.md (2026-04-15). Initial /150 floor was too
		// weak for multi-term queries; bumped to /75 after A/B on 10 baseline queries.
		orderBy = fmt.Sprintf(
			"ORDER BY ts_rank(search_vector, to_tsquery('english', $%d)) * 3 + ts_rank(search_vector, to_tsquery('english', $%d)) * (1 + agentic_score::float/100) + agentic_score::float/75 DESC, is_featured DESC, agentic_score DESC",
			tsQueryArg+1, tsQueryArg)
	} else {
		orderBy = "ORDER BY is_featured DESC, agentic_score DESC, updated_at DESC"
	}

	query := fmt.Sprintf(`
		SELECT id, domain, url, name, description,
			has_llms_txt, has_ai_plugin, has_openapi, has_robots_ai,
			has_structured_api, has_mcp_server, has_schema_org,
			agentic_score, category, tags,
			is_verified, is_featured, last_crawled_at, crawl_status,
			COALESCE(has_favicon, false), COALESCE(favicon_url, ''),
			created_at, updated_at
		FROM sites %s
		%s
		LIMIT $%d OFFSET $%d`, where, orderBy, argN, argN+1)
	args = append(args, p.Limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var s Site
		var tags pq.StringArray
		err := rows.Scan(
			&s.ID, &s.Domain, &s.URL, &s.Name, &s.Description,
			&s.HasLLMsTxt, &s.HasAIPlugin, &s.HasOpenAPI, &s.HasRobotsAI,
			&s.HasStructuredAPI, &s.HasMCPServer, &s.HasSchemaOrg,
			&s.AgenticScore, &s.Category, &tags,
			&s.IsVerified, &s.IsFeatured, &s.LastCrawledAt, &s.CrawlStatus,
			&s.HasFavicon, &s.FaviconURL,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			log.Printf("scan error: %v", err)
			continue
		}
		s.Tags = tags
		sites = append(sites, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("search rows: %w", err)
	}
	return sites, total, nil
}

func GetSiteByDomain(db *sql.DB, domain string) (*Site, error) {
	var s Site
	var tags pq.StringArray
	err := db.QueryRow(`
		SELECT id, domain, url, name, description,
			has_llms_txt, has_ai_plugin, has_openapi, has_robots_ai,
			has_structured_api, has_mcp_server, has_schema_org,
			llms_txt_content, openapi_summary,
			agentic_score, category, tags,
			is_verified, is_featured, last_crawled_at, crawl_status,
			COALESCE(has_favicon, false), COALESCE(favicon_url, ''),
			created_at, updated_at
		FROM sites WHERE domain = $1`, domain).Scan(
		&s.ID, &s.Domain, &s.URL, &s.Name, &s.Description,
		&s.HasLLMsTxt, &s.HasAIPlugin, &s.HasOpenAPI, &s.HasRobotsAI,
		&s.HasStructuredAPI, &s.HasMCPServer, &s.HasSchemaOrg,
		&s.LLMsTxtContent, &s.OpenAPISummary,
		&s.AgenticScore, &s.Category, &tags,
		&s.IsVerified, &s.IsFeatured, &s.LastCrawledAt, &s.CrawlStatus,
		&s.HasFavicon, &s.FaviconURL,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Tags = tags
	return &s, nil
}

// AgentFirstFilter: a site only qualifies if it exposes at least one HARD agent signal —
// something an agent can programmatically interact with. llms.txt alone is passive content
// and doesn't qualify. Schema.org and robots.txt never qualified.
// Hard signals: structured API, OpenAPI spec, MCP server, ai-plugin manifest.
// llms.txt counts ONLY when paired with one of the above (enforced via score-agnostic OR).
const AgentFirstFilter = "crawl_status='success' AND (has_structured_api = true OR has_openapi = true OR has_ai_plugin = true OR has_mcp_server = true)"

func GetStats(db *sql.DB) (totalSites, avgScore int, topCategory string) {
	db.QueryRow("SELECT count(*), COALESCE(AVG(agentic_score), 0)::int FROM sites WHERE " + AgentFirstFilter).Scan(&totalSites, &avgScore)
	db.QueryRow("SELECT category FROM sites WHERE " + AgentFirstFilter + " GROUP BY category ORDER BY count(*) DESC LIMIT 1").Scan(&topCategory)
	return
}

func LogSearch(db *sql.DB, query string, resultsCount int, userAgent, ipHash string) {
	_, _ = db.Exec(`INSERT INTO search_queries (query, results_count, user_agent, ip_hash) VALUES ($1, $2, $3, $4)`,
		query, resultsCount, userAgent, ipHash)
}

// GetCategories returns all categories with their site counts, ordered by count desc.
func GetCategories(db *sql.DB) ([]CategoryCount, error) {
	rows, err := db.Query(`
		SELECT category, count(*) as cnt, COALESCE(AVG(agentic_score)::int, 0) as avg
		FROM sites WHERE ` + AgentFirstFilter + `
		GROUP BY category ORDER BY cnt DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []CategoryCount
	for rows.Next() {
		var c CategoryCount
		if err := rows.Scan(&c.Name, &c.Count, &c.AvgScore); err != nil {
			continue
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

type CategoryCount struct {
	Name     string `json:"name"`
	Count    int    `json:"count"`
	AvgScore int    `json:"avg_score"`
}

func GetTopSites(db *sql.DB, category string, limit int) ([]Site, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	q := `SELECT id, domain, url, name, description,
	             has_llms_txt, has_ai_plugin, has_openapi, has_robots_ai,
	             has_structured_api, has_mcp_server, has_schema_org,
	             agentic_score, category, tags
	      FROM sites WHERE ` + AgentFirstFilter
	var args []any
	if category != "" {
		q += ` AND category = $1`
		args = append(args, category)
	}
	q += ` ORDER BY agentic_score DESC, domain ASC LIMIT ` + fmt.Sprintf("%d", limit)

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var s Site
		err := rows.Scan(&s.ID, &s.Domain, &s.URL, &s.Name, &s.Description,
			&s.HasLLMsTxt, &s.HasAIPlugin, &s.HasOpenAPI, &s.HasRobotsAI,
			&s.HasStructuredAPI, &s.HasMCPServer, &s.HasSchemaOrg,
			&s.AgenticScore, &s.Category, &s.Tags)
		if err != nil {
			continue
		}
		sites = append(sites, s)
	}
	return sites, rows.Err()
}

// TagCount is a single tag and the number of agent-first sites carrying it.
type TagCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// TopTags returns the most popular tags across the index, filtered to
// slug-safe tags with at least 2 sites (matches /tag/{name} landing gate).
func TopTags(db *sql.DB, limit int) ([]TagCount, error) {
	if limit <= 0 {
		limit = 12
	}
	rows, err := db.Query(`
		SELECT tag, COUNT(*) AS n
		  FROM (SELECT unnest(tags) AS tag FROM sites
		        WHERE `+AgentFirstFilter+`) t
		 WHERE tag ~ '^[a-z0-9-]+$'
		 GROUP BY tag HAVING COUNT(*) >= 2
		 ORDER BY n DESC, tag ASC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []TagCount
	for rows.Next() {
		var t TagCount
		if err := rows.Scan(&t.Name, &t.Count); err != nil {
			continue
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// LogMCPRequest records an MCP JSON-RPC request for analytics. Called as a
// goroutine so it never blocks the handler response.
func LogMCPRequest(db *sql.DB, method, toolName string, arguments []byte, resultCount int, userAgent, ipHash string, durationMs int) {
	var args *string
	if len(arguments) > 0 {
		s := string(arguments)
		args = &s
	}
	var rc *int
	if resultCount >= 0 {
		rc = &resultCount
	}
	db.Exec(`INSERT INTO mcp_requests (method, tool_name, arguments, result_count, user_agent, ip_hash, duration_ms)
		VALUES ($1, $2, $3::jsonb, $4, $5, $6, $7)`,
		method, toolName, args, rc, userAgent, ipHash, durationMs)
}

// GetMCPAnalytics returns aggregated MCP request data: tool breakdown, method
// breakdown, top agent user-agents, and top search queries.
func GetMCPAnalytics(db *sql.DB, days int) (map[string]any, error) {
	result := map[string]any{}

	rows, err := db.Query(`
		SELECT tool_name, COUNT(*) as calls
		FROM mcp_requests
		WHERE tool_name IS NOT NULL AND tool_name != '' AND created_at > NOW() - make_interval(days => $1)
		GROUP BY tool_name ORDER BY calls DESC`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var toolBreakdown []map[string]any
	for rows.Next() {
		var name string
		var count int
		rows.Scan(&name, &count)
		toolBreakdown = append(toolBreakdown, map[string]any{"tool": name, "calls": count})
	}
	result["tools"] = toolBreakdown

	rows2, err := db.Query(`
		SELECT method, COUNT(*) as calls
		FROM mcp_requests
		WHERE created_at > NOW() - make_interval(days => $1)
		GROUP BY method ORDER BY calls DESC`, days)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	var methodBreakdown []map[string]any
	for rows2.Next() {
		var m string
		var c int
		rows2.Scan(&m, &c)
		methodBreakdown = append(methodBreakdown, map[string]any{"method": m, "calls": c})
	}
	result["methods"] = methodBreakdown

	rows3, err := db.Query(`
		SELECT user_agent, COUNT(*) as calls
		FROM mcp_requests
		WHERE created_at > NOW() - make_interval(days => $1)
		GROUP BY user_agent ORDER BY calls DESC LIMIT 20`, days)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()
	var agentBreakdown []map[string]any
	for rows3.Next() {
		var ua sql.NullString
		var c int
		rows3.Scan(&ua, &c)
		agentBreakdown = append(agentBreakdown, map[string]any{"user_agent": ua.String, "calls": c})
	}
	result["agents"] = agentBreakdown

	// Top search queries via MCP (search_agents tool)
	rows4, err := db.Query(`
		SELECT arguments->>'query' as q, COUNT(*) as cnt
		FROM mcp_requests
		WHERE tool_name = 'search_agents' AND arguments->>'query' IS NOT NULL AND arguments->>'query' != ''
			AND created_at > NOW() - make_interval(days => $1)
		GROUP BY q ORDER BY cnt DESC LIMIT 30`, days)
	if err != nil {
		return nil, err
	}
	defer rows4.Close()
	var topQueries []map[string]any
	for rows4.Next() {
		var q string
		var c int
		rows4.Scan(&q, &c)
		topQueries = append(topQueries, map[string]any{"query": q, "count": c})
	}
	result["top_queries"] = topQueries

	var totalReqs, uniqueAgents int
	db.QueryRow(`SELECT COUNT(*) FROM mcp_requests WHERE created_at > NOW() - make_interval(days => $1)`, days).Scan(&totalReqs)
	db.QueryRow(`SELECT COUNT(DISTINCT ip_hash) FROM mcp_requests WHERE created_at > NOW() - make_interval(days => $1)`, days).Scan(&uniqueAgents)
	result["total_requests"] = totalReqs
	result["unique_agents"] = uniqueAgents

	return result, nil
}

func GetTrafficAnalytics(db *sql.DB, days int) (map[string]interface{}, error) {
	if days <= 0 {
		days = 14
	}
	result := map[string]interface{}{}

	type dayRow struct {
		Day       string `json:"day"`
		Total     int    `json:"total"`
		Humans    int    `json:"humans"`
		Bots      int    `json:"bots"`
		UniqueIPs int    `json:"unique_ips"`
	}
	daily := []dayRow{}
	rows, err := db.Query(`
		SELECT date_trunc('day', created_at)::date::text AS day,
			count(*) AS total,
			count(*) FILTER (WHERE NOT is_bot) AS humans,
			count(*) FILTER (WHERE is_bot) AS bots,
			count(DISTINCT ip_hash) AS unique_ips
		FROM page_views
		WHERE created_at >= NOW() - $1::int * INTERVAL '1 day'
		GROUP BY 1 ORDER BY 1`, days)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d dayRow
			rows.Scan(&d.Day, &d.Total, &d.Humans, &d.Bots, &d.UniqueIPs)
			daily = append(daily, d)
		}
	}
	result["daily"] = daily

	type pageRow struct {
		Path  string `json:"path"`
		Count int    `json:"count"`
	}
	pages := []pageRow{}
	rows2, err := db.Query(`
		SELECT path, count(*) AS cnt FROM page_views
		WHERE NOT is_bot AND created_at >= NOW() - $1::int * INTERVAL '1 day'
		GROUP BY path ORDER BY cnt DESC LIMIT 25`, days)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var p pageRow
			rows2.Scan(&p.Path, &p.Count)
			pages = append(pages, p)
		}
	}
	result["top_pages"] = pages

	type refRow struct {
		Referer string `json:"referer"`
		Count   int    `json:"count"`
	}
	refs := []refRow{}
	rows3, err := db.Query(`
		SELECT referer, count(*) AS cnt FROM page_views
		WHERE NOT is_bot AND referer != '' AND created_at >= NOW() - $1::int * INTERVAL '1 day'
		GROUP BY referer ORDER BY cnt DESC LIMIT 25`, days)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var r refRow
			rows3.Scan(&r.Referer, &r.Count)
			refs = append(refs, r)
		}
	}
	result["top_referrers"] = refs

	// Error rates (5xx responses)
	type errRow struct {
		Status int `json:"status"`
		Count  int `json:"count"`
	}
	errors := []errRow{}
	rows4, err := db.Query(`
		SELECT status, count(*) AS cnt FROM page_views
		WHERE status >= 500 AND created_at >= NOW() - $1::int * INTERVAL '1 day'
		GROUP BY status ORDER BY cnt DESC`, days)
	if err == nil {
		defer rows4.Close()
		for rows4.Next() {
			var e errRow
			rows4.Scan(&e.Status, &e.Count)
			errors = append(errors, e)
		}
	}
	result["errors"] = errors

	var errorsLastHour int
	db.QueryRow(`SELECT count(*) FROM page_views WHERE status >= 500 AND created_at >= NOW() - INTERVAL '1 hour'`).Scan(&errorsLastHour)
	result["errors_last_hour"] = errorsLastHour

	return result, nil
}
