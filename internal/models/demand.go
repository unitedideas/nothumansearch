package models

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/lib/pq"
)

var readDemandSearchEntropy = rand.Read

var ErrDemandStoreUnavailable = errors.New("demand receipt store unavailable")
var ErrDemandEntropyUnavailable = errors.New("demand receipt entropy unavailable")

// ProviderDemandPrivacyThreshold is a release gate for segmented reporting.
// It counts persisted search receipts, not unique agents, people, or sessions.
const ProviderDemandPrivacyThreshold = 20

type DemandSearchReceipt struct {
	PublicID    string
	Surface     string
	Query       string // classified in memory; never persisted
	Category    string
	HasAPI      bool
	HasMCP      bool
	HasOpenAPI  bool
	HasLLMsTxt  bool
	ResultCount int
	Page        int
	PageSize    int
	Synthetic   bool
}

type demandTopicRule struct {
	name  string
	terms []string
}

// Topics are deliberately coarse and allowlisted. Search text, query-derived
// labels, fingerprints, IP hashes, user agents, and alleged agent identities do
// not enter the discovery-receipt tables or provider reports.
var demandTopicRules = []demandTopicRule{
	{name: "payments", terms: []string{"payment", "payments", "billing", "invoice", "invoicing", "checkout", "subscription"}},
	{name: "commerce", terms: []string{"commerce", "ecommerce", "store", "shop", "catalog", "inventory"}},
	{name: "jobs", terms: []string{"job", "jobs", "hiring", "recruiting", "candidate", "career"}},
	{name: "data", terms: []string{"data", "dataset", "datasets", "scraping", "enrichment"}},
	{name: "search", terms: []string{"search", "discovery", "index", "directory"}},
	{name: "weather", terms: []string{"weather", "forecast", "climate"}},
	{name: "maps", terms: []string{"map", "maps", "geocode", "geocoding", "location", "routing"}},
	{name: "email", terms: []string{"email", "inbox", "smtp", "newsletter"}},
	{name: "messaging", terms: []string{"message", "messaging", "sms", "chat", "communication", "voice"}},
	{name: "image", terms: []string{"image", "images", "photo", "photos", "ocr"}},
	{name: "video", terms: []string{"video", "videos", "render", "rendering"}},
	{name: "audio", terms: []string{"audio", "speech", "transcription", "music"}},
	{name: "documents", terms: []string{"document", "documents", "pdf", "spreadsheet", "file", "files"}},
	{name: "security", terms: []string{"security", "fraud", "threat", "vulnerability", "compliance"}},
	{name: "finance", terms: []string{"finance", "financial", "banking", "market", "markets", "stock", "stocks"}},
	{name: "health", terms: []string{"health", "healthcare", "medical", "clinical", "fitness"}},
	{name: "education", terms: []string{"education", "learning", "course", "courses", "school"}},
	{name: "news", terms: []string{"news", "article", "articles", "journalism", "headlines"}},
	{name: "analytics", terms: []string{"analytics", "metrics", "telemetry", "tracking", "measurement"}},
	{name: "automation", terms: []string{"automation", "workflow", "scheduler", "cron", "integration"}},
	{name: "productivity", terms: []string{"productivity", "task", "tasks", "calendar", "notes"}},
	{name: "identity", terms: []string{"identity", "authentication", "oauth", "login", "verification"}},
	{name: "storage", terms: []string{"storage", "database", "hosting", "cloud", "backup"}},
	{name: "ai-tools", terms: []string{"ai", "agent", "agents", "llm", "model", "models", "mcp"}},
	{name: "developer-tools", terms: []string{"api", "openapi", "sdk", "developer", "code", "testing"}},
}

var demandCategoryTopics = map[string]string{
	"ai-tools":      "ai-tools",
	"communication": "messaging",
	"data":          "data",
	"developer":     "developer-tools",
	"ecommerce":     "commerce",
	"finance":       "finance",
	"health":        "health",
	"education":     "education",
	"jobs":          "jobs",
	"news":          "news",
	"productivity":  "productivity",
	"security":      "security",
}

func GenerateDemandSearchID() (string, error) {
	var raw [12]byte
	if _, err := readDemandSearchEntropy(raw[:]); err != nil {
		return "", fmt.Errorf("%w: %v", ErrDemandEntropyUnavailable, err)
	}
	return "nhs_sr_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func normalizeDemandSurface(surface string) string {
	switch strings.ToLower(strings.TrimSpace(surface)) {
	case "web", "rest", "mcp":
		return strings.ToLower(strings.TrimSpace(surface))
	default:
		return "unknown"
	}
}

func normalizeDemandCategory(category string) string {
	category = strings.ToLower(strings.TrimSpace(category))
	if _, ok := demandCategoryTopics[category]; ok {
		return category
	}
	return ""
}

// ClassifyDemandTopics reduces a query to at most three controlled topics.
// It never returns any caller-provided text.
func ClassifyDemandTopics(query, category string) []string {
	topics := []string{}
	seen := map[string]bool{}
	add := func(topic string) {
		if topic != "" && !seen[topic] && len(topics) < 3 {
			seen[topic] = true
			topics = append(topics, topic)
		}
	}
	if topic := demandCategoryTopics[normalizeDemandCategory(category)]; topic != "" {
		add(topic)
	}

	tokens := map[string]bool{}
	for _, token := range strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if token != "" {
			tokens[token] = true
		}
	}
	for _, rule := range demandTopicRules {
		for _, term := range rule.terms {
			if tokens[term] {
				add(rule.name)
				break
			}
		}
	}
	if len(topics) == 0 {
		return []string{"other"}
	}
	return topics
}

func RecordDemandSearch(db *sql.DB, receipt DemandSearchReceipt, sites []Site) error {
	if db == nil {
		return ErrDemandStoreUnavailable
	}
	if receipt.PublicID == "" {
		publicID, err := GenerateDemandSearchID()
		if err != nil {
			return err
		}
		receipt.PublicID = publicID
	}
	if receipt.ResultCount < 0 {
		receipt.ResultCount = 0
	}
	if receipt.Page < 1 {
		receipt.Page = 1
	}
	if receipt.PageSize < 1 {
		receipt.PageSize = len(sites)
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var receiptID string
	err = tx.QueryRow(`
		INSERT INTO search_receipts (
			public_id, surface, explicit_category, demand_topics,
			has_api, has_mcp, has_openapi, has_llms_txt, result_count,
			page_number, page_size, is_synthetic
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id::text`,
		receipt.PublicID,
		normalizeDemandSurface(receipt.Surface),
		normalizeDemandCategory(receipt.Category),
		pq.Array(ClassifyDemandTopics(receipt.Query, receipt.Category)),
		receipt.HasAPI,
		receipt.HasMCP,
		receipt.HasOpenAPI,
		receipt.HasLLMsTxt,
		receipt.ResultCount,
		receipt.Page,
		receipt.PageSize,
		receipt.Synthetic,
	).Scan(&receiptID)
	if err != nil {
		return err
	}

	if len(sites) > 0 {
		siteIDs := make([]string, 0, len(sites))
		domains := make([]string, 0, len(sites))
		positions := make([]int64, 0, len(sites))
		scores := make([]int64, 0, len(sites))
		for position, site := range sites {
			siteIDs = append(siteIDs, strings.TrimSpace(site.ID))
			domains = append(domains, NormalizeProviderDomain(site.Domain))
			positions = append(positions, int64(((receipt.Page-1)*receipt.PageSize)+position+1))
			scores = append(scores, int64(site.AgenticScore))
		}
		// One set-based insert avoids up to 50 sequential database round trips per
		// search while preserving the receipt/result transaction boundary.
		_, err = tx.Exec(`
			INSERT INTO organic_results_returned (
				search_receipt_id, site_id, site_domain_snapshot,
				organic_position, score_snapshot
			)
			SELECT $1::uuid,
			       NULLIF(result.site_id, '')::uuid,
			       result.domain,
			       result.position::integer,
			       result.score::integer
			FROM unnest($2::text[], $3::text[], $4::bigint[], $5::bigint[])
			     AS result(site_id, domain, position, score)
			ON CONFLICT (search_receipt_id, site_domain_snapshot) DO NOTHING`,
			receiptID,
			pq.Array(siteIDs),
			pq.Array(domains),
			pq.Array(positions),
			pq.Array(scores),
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RecordDemandSelection records a successful detail request only when the
// domain was actually returned in the referenced organic result set.
func RecordDemandSelection(db *sql.DB, searchPublicID, domain, surface string) (bool, error) {
	if db == nil {
		return false, nil
	}
	searchPublicID = strings.TrimSpace(searchPublicID)
	domain = NormalizeProviderDomain(domain)
	if !strings.HasPrefix(searchPublicID, "nhs_sr_") || domain == "" {
		return false, nil
	}
	var matched bool
	err := db.QueryRow(`
		WITH candidate AS (
			SELECT sr.id AS search_receipt_id, returned.site_id, returned.site_domain_snapshot
			FROM search_receipts sr
			JOIN organic_results_returned returned
			  ON returned.search_receipt_id = sr.id
			WHERE sr.public_id = $1 AND returned.site_domain_snapshot = $2
		), inserted AS (
			INSERT INTO result_selections (
				search_receipt_id, site_id, site_domain_snapshot, surface
			)
			SELECT search_receipt_id, site_id, site_domain_snapshot, $3
			FROM candidate
			ON CONFLICT (search_receipt_id, site_domain_snapshot) DO NOTHING
			RETURNING 1
		)
		SELECT EXISTS(SELECT 1 FROM inserted)`,
		searchPublicID, domain, normalizeDemandSurface(surface)).Scan(&matched)
	return matched, err
}

func NormalizeProviderDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "www.")
	if i := strings.IndexAny(domain, "/?#"); i >= 0 {
		domain = domain[:i]
	}
	return strings.TrimSuffix(domain, ".")
}

// GetProviderDemandAnalytics exposes only provider-scoped aggregate facts and
// controlled topics. It never returns raw searches, fingerprints, network
// identifiers, user agents, individual receipts, or alleged agent identities.
func GetProviderDemandAnalytics(db *sql.DB, domain string, days int) (map[string]any, error) {
	if db == nil {
		return nil, ErrDemandStoreUnavailable
	}
	domain = NormalizeProviderDomain(domain)
	if days <= 0 || days > 30 {
		days = 30
	}
	// Keep every metric on one repeatable-read snapshot and one database wall-
	// clock boundary. Both are required: the timestamp excludes later writes,
	// while the snapshot prevents cleanup/cascade deletes between statements from
	// changing membership in the same report.
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var cohortAsOf time.Time
	if err := tx.QueryRow(`SELECT clock_timestamp()`).Scan(&cohortAsOf); err != nil {
		return nil, err
	}
	result := map[string]any{
		"domain":                             domain,
		"days":                               days,
		"retention_days":                     30,
		"action_interest_cohort":             "organic_result_returned_at",
		"topic_receipt_threshold":            ProviderDemandPrivacyThreshold,
		"result_selection_receipt_threshold": ProviderDemandPrivacyThreshold,
		"action_interest_receipt_threshold":  ProviderDemandPrivacyThreshold,
		"synthetic_excluded":                 true,
	}

	var resultsReturned, searchReceipts, resultSelections, actionInterests int
	var averagePosition float64
	err = tx.QueryRow(`
		SELECT COUNT(*)::int,
		       COUNT(DISTINCT sr.id)::int,
		       COUNT(DISTINCT selected.search_receipt_id)::int,
		       COUNT(DISTINCT interest.search_receipt_id)::int,
		       COALESCE(AVG(returned.organic_position), 0)::float
		FROM organic_results_returned returned
		JOIN search_receipts sr ON sr.id = returned.search_receipt_id
		LEFT JOIN result_selections selected
		  ON selected.search_receipt_id = returned.search_receipt_id
		 AND selected.site_domain_snapshot = returned.site_domain_snapshot
		 AND selected.selected_at <= $3::timestamptz
		LEFT JOIN action_interest_receipts interest
		  ON interest.search_receipt_id = returned.search_receipt_id
		 AND interest.site_domain_snapshot = returned.site_domain_snapshot
		 AND interest.created_at <= $3::timestamptz
		 AND interest.expires_at > $3::timestamptz
		WHERE returned.site_domain_snapshot = $1
		  AND NOT sr.is_synthetic
		  AND returned.returned_at >= $3::timestamptz - $2::int * INTERVAL '1 day'
		  AND returned.returned_at <= $3::timestamptz`, domain, days, cohortAsOf).
		Scan(&resultsReturned, &searchReceipts, &resultSelections, &actionInterests, &averagePosition)
	if err != nil {
		return nil, err
	}
	selectionRate := 0.0
	actionInterestRate := 0.0
	if searchReceipts > 0 {
		selectionRate = float64(resultSelections) / float64(searchReceipts)
		actionInterestRate = float64(actionInterests) / float64(searchReceipts)
	}
	summary := map[string]any{
		"organic_results_returned": resultsReturned,
		"search_receipts":          searchReceipts,
		"average_organic_position": averagePosition,
	}
	applyProviderDemandSelectionPrivacy(summary, resultSelections, &selectionRate)
	if actionInterests >= ProviderDemandPrivacyThreshold {
		summary["action_interest_receipts"] = actionInterests
		summary["action_interest_rate"] = actionInterestRate
		summary["action_interest_suppressed"] = false
	} else {
		summary["action_interest_receipts"] = nil
		summary["action_interest_rate"] = nil
		summary["action_interest_suppressed"] = true
	}
	result["summary"] = summary

	rows, err := tx.Query(`
		SELECT sr.surface,
		       COUNT(*)::int,
		       COUNT(DISTINCT selected.search_receipt_id)::int,
		       COUNT(DISTINCT interest.search_receipt_id)::int
		FROM organic_results_returned returned
		JOIN search_receipts sr ON sr.id = returned.search_receipt_id
		LEFT JOIN result_selections selected
		  ON selected.search_receipt_id = returned.search_receipt_id
		 AND selected.site_domain_snapshot = returned.site_domain_snapshot
		 AND selected.selected_at <= $3::timestamptz
		LEFT JOIN action_interest_receipts interest
		  ON interest.search_receipt_id = returned.search_receipt_id
		 AND interest.site_domain_snapshot = returned.site_domain_snapshot
		 AND interest.created_at <= $3::timestamptz
		 AND interest.expires_at > $3::timestamptz
		WHERE returned.site_domain_snapshot = $1
		  AND NOT sr.is_synthetic
		  AND returned.returned_at >= $3::timestamptz - $2::int * INTERVAL '1 day'
		  AND returned.returned_at <= $3::timestamptz
		GROUP BY sr.surface
		ORDER BY COUNT(*) DESC`, domain, days, cohortAsOf)
	if err != nil {
		return nil, err
	}
	surfaces := []map[string]any{}
	for rows.Next() {
		var surface string
		var returned, selections, interests int
		if err := rows.Scan(&surface, &returned, &selections, &interests); err != nil {
			return nil, err
		}
		entry := map[string]any{
			"surface":                  surface,
			"organic_results_returned": returned,
		}
		applyProviderDemandSelectionPrivacy(entry, selections, nil)
		if interests >= ProviderDemandPrivacyThreshold {
			entry["action_interest_receipts"] = interests
			entry["action_interest_suppressed"] = false
		} else {
			entry["action_interest_receipts"] = nil
			entry["action_interest_suppressed"] = true
		}
		surfaces = append(surfaces, entry)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	result["surfaces"] = surfaces

	rows2, err := tx.Query(`
		SELECT topic,
		       COUNT(DISTINCT sr.id)::int,
		       COUNT(DISTINCT selected.search_receipt_id)::int,
		       COUNT(DISTINCT interest.search_receipt_id)::int,
		       COALESCE(AVG(returned.organic_position), 0)::float
		FROM organic_results_returned returned
		JOIN search_receipts sr ON sr.id = returned.search_receipt_id
		CROSS JOIN LATERAL unnest(sr.demand_topics) AS topic
		LEFT JOIN result_selections selected
		  ON selected.search_receipt_id = returned.search_receipt_id
		 AND selected.site_domain_snapshot = returned.site_domain_snapshot
		 AND selected.selected_at <= $3::timestamptz
		LEFT JOIN action_interest_receipts interest
		  ON interest.search_receipt_id = returned.search_receipt_id
		 AND interest.site_domain_snapshot = returned.site_domain_snapshot
		 AND interest.created_at <= $3::timestamptz
		 AND interest.expires_at > $3::timestamptz
		WHERE returned.site_domain_snapshot = $1
		  AND NOT sr.is_synthetic
		  AND returned.returned_at >= $3::timestamptz - $2::int * INTERVAL '1 day'
		  AND returned.returned_at <= $3::timestamptz
		GROUP BY topic
		HAVING COUNT(DISTINCT sr.id) >= $4
		ORDER BY COUNT(DISTINCT sr.id) DESC, topic ASC
		LIMIT 20`, domain, days, cohortAsOf, ProviderDemandPrivacyThreshold)
	if err != nil {
		return nil, err
	}
	topics := []map[string]any{}
	for rows2.Next() {
		var topic string
		var receipts, selections, interests int
		var avgPosition float64
		if err := rows2.Scan(&topic, &receipts, &selections, &interests, &avgPosition); err != nil {
			return nil, err
		}
		entry := map[string]any{
			"topic":                    topic,
			"search_receipts":          receipts,
			"average_organic_position": avgPosition,
		}
		applyProviderDemandSelectionPrivacy(entry, selections, nil)
		if interests >= ProviderDemandPrivacyThreshold {
			entry["action_interest_receipts"] = interests
			entry["action_interest_suppressed"] = false
		} else {
			entry["action_interest_receipts"] = nil
			entry["action_interest_suppressed"] = true
		}
		topics = append(topics, entry)
	}
	if err := rows2.Err(); err != nil {
		_ = rows2.Close()
		return nil, err
	}
	if err := rows2.Close(); err != nil {
		return nil, err
	}
	result["demand_topics"] = topics

	rows3, err := tx.Query(`
		SELECT interest.action_type,
		       COUNT(DISTINCT interest.search_receipt_id)::int
		FROM action_interest_receipts interest
		JOIN search_receipts sr ON sr.id=interest.search_receipt_id
		JOIN organic_results_returned returned
		  ON returned.search_receipt_id=interest.search_receipt_id
		 AND returned.site_domain_snapshot=interest.site_domain_snapshot
		WHERE interest.site_domain_snapshot=$1
		  AND NOT sr.is_synthetic
		  AND returned.returned_at >= $3::timestamptz - $2::int * INTERVAL '1 day'
		  AND returned.returned_at <= $3::timestamptz
		  AND interest.created_at <= $3::timestamptz
		  AND interest.expires_at > $3::timestamptz
		GROUP BY interest.action_type
		HAVING COUNT(DISTINCT interest.search_receipt_id) >= $4
		ORDER BY COUNT(DISTINCT interest.search_receipt_id) DESC,
		         interest.action_type`, domain, days, cohortAsOf, ProviderDemandPrivacyThreshold)
	if err != nil {
		return nil, err
	}
	actionTypes := []map[string]any{}
	for rows3.Next() {
		var actionType string
		var receipts int
		if err := rows3.Scan(&actionType, &receipts); err != nil {
			return nil, err
		}
		actionTypes = append(actionTypes, map[string]any{
			"action_type":   actionType,
			"receipt_count": receipts,
		})
	}
	if err := rows3.Err(); err != nil {
		_ = rows3.Close()
		return nil, err
	}
	if err := rows3.Close(); err != nil {
		return nil, err
	}
	result["action_types"] = actionTypes
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

// applyProviderDemandSelectionPrivacy prevents a provider from learning that
// one thin cohort selected its result. Search-return counts remain useful for
// availability, but behavior counts and rates appear only at the same minimum
// aggregate threshold used by the rest of the provider demand report.
func applyProviderDemandSelectionPrivacy(entry map[string]any, selections int, rate *float64) {
	if selections >= ProviderDemandPrivacyThreshold {
		entry["result_selections"] = selections
		entry["result_selection_suppressed"] = false
		if rate != nil {
			entry["result_selection_rate"] = *rate
		}
		return
	}
	entry["result_selections"] = nil
	entry["result_selection_suppressed"] = true
	if rate != nil {
		entry["result_selection_rate"] = nil
	}
}
