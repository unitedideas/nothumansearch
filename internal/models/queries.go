package models

import (
	"database/sql"
	"fmt"
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
			llms_txt_content, openapi_summary,
			agentic_score, category, tags,
			is_featured, last_crawled_at, crawl_status, crawl_error)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		ON CONFLICT (domain) DO UPDATE SET
			url=EXCLUDED.url, name=EXCLUDED.name, description=EXCLUDED.description,
			has_llms_txt=EXCLUDED.has_llms_txt, has_ai_plugin=EXCLUDED.has_ai_plugin,
			has_openapi=EXCLUDED.has_openapi, has_robots_ai=EXCLUDED.has_robots_ai,
			has_structured_api=EXCLUDED.has_structured_api, has_mcp_server=EXCLUDED.has_mcp_server,
			has_schema_org=EXCLUDED.has_schema_org,
			llms_txt_content=EXCLUDED.llms_txt_content, openapi_summary=EXCLUDED.openapi_summary,
			agentic_score=EXCLUDED.agentic_score, category=EXCLUDED.category, tags=EXCLUDED.tags,
			last_crawled_at=EXCLUDED.last_crawled_at, crawl_status=EXCLUDED.crawl_status,
			crawl_error=EXCLUDED.crawl_error, updated_at=NOW()`,
		s.Domain, s.URL, s.Name, s.Description,
		s.HasLLMsTxt, s.HasAIPlugin, s.HasOpenAPI, s.HasRobotsAI,
		s.HasStructuredAPI, s.HasMCPServer, s.HasSchemaOrg,
		s.LLMsTxtContent, s.OpenAPISummary,
		s.AgenticScore, s.Category, pq.Array(s.Tags),
		s.IsFeatured, s.LastCrawledAt, s.CrawlStatus, s.CrawlError,
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

	// Track if we have a multi-word query for relevance ordering
	hasMultiWordQuery := false
	var relevanceExpr string

	if p.Query != "" {
		// Split query into words for broader matching (agent-style queries)
		words := strings.Fields(p.Query)
		if len(words) <= 1 {
			// Single word: match name, description, domain, category, or tags
			conditions = append(conditions, fmt.Sprintf(
				"(name ILIKE $%d OR description ILIKE $%d OR domain ILIKE $%d OR category ILIKE $%d OR array_to_string(tags, ' ') ILIKE $%d)",
				argN, argN, argN, argN, argN))
			args = append(args, "%"+p.Query+"%")
			argN++
		} else {
			// Multi-word: match sites containing ANY word (OR)
			var wordConditions []string
			var relevanceParts []string
			for _, word := range words {
				wordConditions = append(wordConditions, fmt.Sprintf(
					"(name ILIKE $%d OR description ILIKE $%d OR domain ILIKE $%d OR category ILIKE $%d OR array_to_string(tags, ' ') ILIKE $%d)",
					argN, argN, argN, argN, argN))
				// Count how many words match for relevance ranking (include tags)
				relevanceParts = append(relevanceParts, fmt.Sprintf(
					"CASE WHEN (name ILIKE $%d OR description ILIKE $%d OR domain ILIKE $%d OR category ILIKE $%d OR array_to_string(tags, ' ') ILIKE $%d) THEN 1 ELSE 0 END",
					argN, argN, argN, argN, argN))
				args = append(args, "%"+word+"%")
				argN++
			}
			conditions = append(conditions, "("+strings.Join(wordConditions, " OR ")+")")
			relevanceExpr = "(" + strings.Join(relevanceParts, " + ") + ")"
			hasMultiWordQuery = true
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

	// Fetch
	offset := (p.Page - 1) * p.Limit
	orderBy := "ORDER BY is_featured DESC, agentic_score DESC, updated_at DESC"
	if hasMultiWordQuery {
		// Rank by: relevance (word match count) first, then featured, then score
		orderBy = fmt.Sprintf("ORDER BY %s DESC, is_featured DESC, agentic_score DESC", relevanceExpr)
	}
	query := fmt.Sprintf(`
		SELECT id, domain, url, name, description,
			has_llms_txt, has_ai_plugin, has_openapi, has_robots_ai,
			has_structured_api, has_mcp_server, has_schema_org,
			agentic_score, category, tags,
			is_verified, is_featured, last_crawled_at, crawl_status,
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
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
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
			created_at, updated_at
		FROM sites WHERE domain = $1`, domain).Scan(
		&s.ID, &s.Domain, &s.URL, &s.Name, &s.Description,
		&s.HasLLMsTxt, &s.HasAIPlugin, &s.HasOpenAPI, &s.HasRobotsAI,
		&s.HasStructuredAPI, &s.HasMCPServer, &s.HasSchemaOrg,
		&s.LLMsTxtContent, &s.OpenAPISummary,
		&s.AgenticScore, &s.Category, &tags,
		&s.IsVerified, &s.IsFeatured, &s.LastCrawledAt, &s.CrawlStatus,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Tags = tags
	return &s, nil
}

func GetStats(db *sql.DB) (totalSites, avgScore int, topCategory string) {
	db.QueryRow("SELECT count(*), COALESCE(AVG(agentic_score), 0)::int FROM sites WHERE crawl_status='success'").Scan(&totalSites, &avgScore)
	db.QueryRow("SELECT category FROM sites WHERE crawl_status='success' GROUP BY category ORDER BY count(*) DESC LIMIT 1").Scan(&topCategory)
	return
}
