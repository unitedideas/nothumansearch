package models

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
)

func UpsertSite(db *sql.DB, s *Site) error {
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
	Query    string
	Category string
	MinScore int
	HasAPI   bool
	Limit    int
	Page     int
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
				tsQueryStr := strings.Join(tsTerms, " & ")
				// Match: FTS OR ILIKE (catches things FTS misses like domain substrings)
				conditions = append(conditions, fmt.Sprintf(
					"(search_vector @@ to_tsquery('english', $%d) OR name ILIKE $%d OR domain ILIKE $%d OR array_to_string(tags, ' ') ILIKE $%d)",
					argN, argN+1, argN+1, argN+1))
				args = append(args, tsQueryStr, "%"+p.Query+"%")
				tsQueryArg = argN
				useFTS = true
				argN += 2
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
		// Rank by: FTS relevance * agentic_score weighting
		orderBy = fmt.Sprintf(
			"ORDER BY ts_rank(search_vector, to_tsquery('english', $%d)) * (1 + agentic_score::float/100) DESC, is_featured DESC, agentic_score DESC",
			tsQueryArg)
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
		SELECT category, count(*) as cnt
		FROM sites WHERE ` + AgentFirstFilter + `
		GROUP BY category ORDER BY cnt DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []CategoryCount
	for rows.Next() {
		var c CategoryCount
		if err := rows.Scan(&c.Name, &c.Count); err != nil {
			continue
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

type CategoryCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
