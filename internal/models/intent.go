package models

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

type IntentEvent struct {
	VisitID    string
	EventName  string
	EntityType string
	EntityID   string
	Path       string
	Referrer   string
	UserAgent  string
	IPHash     string
	IsBot      bool
	Metadata   map[string]any
}

func LogIntentEvent(db *sql.DB, ev IntentEvent) {
	if db == nil {
		return
	}
	ev.EventName = strings.TrimSpace(ev.EventName)
	if ev.EventName == "" {
		return
	}
	if ev.Metadata == nil {
		ev.Metadata = map[string]any{}
	}
	body, err := json.Marshal(ev.Metadata)
	if err != nil {
		body = []byte(`{}`)
	}
	_, _ = db.Exec(`
		INSERT INTO intent_events
			(visit_id, event_name, entity_type, entity_id, path, referrer, user_agent, ip_hash, is_bot, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb)`,
		limitText(ev.VisitID, 128),
		limitText(ev.EventName, 80),
		limitText(ev.EntityType, 80),
		limitText(ev.EntityID, 256),
		limitText(ev.Path, 1024),
		limitText(ev.Referrer, 2048),
		limitText(ev.UserAgent, 512),
		limitText(ev.IPHash, 128),
		ev.IsBot,
		string(body),
	)
}

func LogIntentFromRequest(db *sql.DB, r *http.Request, eventName, entityType, entityID string, metadata map[string]any) {
	if r == nil {
		LogIntentEvent(db, IntentEvent{EventName: eventName, EntityType: entityType, EntityID: entityID, Metadata: metadata})
		return
	}
	ip := requestIP(r)
	LogIntentEvent(db, IntentEvent{
		VisitID:    requestVisitID(r),
		EventName:  eventName,
		EntityType: entityType,
		EntityID:   entityID,
		Path:       r.URL.Path,
		Referrer:   r.Referer(),
		UserAgent:  r.UserAgent(),
		IPHash:     hashIntentIP(ip),
		IsBot:      isIntentBot(r.UserAgent()),
		Metadata:   metadata,
	})
}

func GetIntentAnalytics(db *sql.DB, days int) (map[string]any, error) {
	if days <= 0 || days > 90 {
		days = 14
	}
	result := map[string]any{}

	rows, err := db.Query(`
		SELECT event_name,
		       COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE NOT is_bot) AS humans,
		       COUNT(*) FILTER (WHERE is_bot) AS bots,
		       COUNT(DISTINCT NULLIF(ip_hash, '')) AS unique_visitors
		  FROM intent_events
		 WHERE created_at >= NOW() - $1::int * INTERVAL '1 day'
		 GROUP BY event_name
		 ORDER BY total DESC`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []map[string]any{}
	for rows.Next() {
		var event string
		var total, humans, bots, unique int
		if err := rows.Scan(&event, &total, &humans, &bots, &unique); err != nil {
			return nil, err
		}
		events = append(events, map[string]any{
			"event":           event,
			"total":           total,
			"humans":          humans,
			"bots":            bots,
			"unique_visitors": unique,
		})
	}
	result["events"] = events

	rows2, err := db.Query(`
		SELECT event_name, entity_type, entity_id, COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE NOT is_bot) AS humans
		  FROM intent_events
		 WHERE created_at >= NOW() - $1::int * INTERVAL '1 day'
		   AND entity_id != ''
		 GROUP BY event_name, entity_type, entity_id
		 ORDER BY humans DESC, total DESC
		 LIMIT 50`, days)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	entities := []map[string]any{}
	for rows2.Next() {
		var event, entityType, entityID string
		var total, humans int
		if err := rows2.Scan(&event, &entityType, &entityID, &total, &humans); err != nil {
			return nil, err
		}
		entities = append(entities, map[string]any{
			"event":       event,
			"entity_type": entityType,
			"entity_id":   entityID,
			"total":       total,
			"humans":      humans,
		})
	}
	result["top_entities"] = entities

	rows3, err := db.Query(`
		SELECT created_at::text, event_name, entity_type, entity_id, path,
		       COALESCE(metadata::text, '{}') AS metadata
		  FROM intent_events
		 WHERE created_at >= NOW() - $1::int * INTERVAL '1 day'
		 ORDER BY created_at DESC
		 LIMIT 100`, days)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()
	recent := []map[string]any{}
	for rows3.Next() {
		var createdAt, event, entityType, entityID, path, metadata string
		if err := rows3.Scan(&createdAt, &event, &entityType, &entityID, &path, &metadata); err != nil {
			return nil, err
		}
		recent = append(recent, map[string]any{
			"created_at":  createdAt,
			"event":       event,
			"entity_type": entityType,
			"entity_id":   entityID,
			"path":        path,
			"metadata":    json.RawMessage(metadata),
		})
	}
	result["recent"] = recent

	var total, human, bot int
	_ = db.QueryRow(`
		SELECT COUNT(*), COUNT(*) FILTER (WHERE NOT is_bot), COUNT(*) FILTER (WHERE is_bot)
		  FROM intent_events
		 WHERE created_at >= NOW() - $1::int * INTERVAL '1 day'`, days).Scan(&total, &human, &bot)
	result["totals"] = map[string]int{"total": total, "human": human, "bot": bot}
	return result, nil
}

func requestIP(r *http.Request) string {
	for _, h := range []string{"Fly-Client-IP", "X-Forwarded-For", "X-Real-IP"} {
		if v := strings.TrimSpace(r.Header.Get(h)); v != "" {
			return strings.TrimSpace(strings.Split(v, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func requestVisitID(r *http.Request) string {
	for _, h := range []string{"X-Foundry-Visit-ID", "X-Visit-ID"} {
		if v := strings.TrimSpace(r.Header.Get(h)); v != "" {
			return v
		}
	}
	for _, name := range []string{"foundry_visit_id", "nhs_visit_id", "visit_id"} {
		if c, err := r.Cookie(name); err == nil && strings.TrimSpace(c.Value) != "" {
			return strings.TrimSpace(c.Value)
		}
	}
	return ""
}

func hashIntentIP(ip string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(ip)))
	return hex.EncodeToString(sum[:8])
}

func isIntentBot(ua string) bool {
	lower := strings.ToLower(ua)
	if lower == "" || lower == "node" {
		return true
	}
	for _, marker := range []string{
		"bot", "crawl", "spider", "curl/", "wget", "python-requests", "python-urllib",
		"go-http-client", "httpx", "scrapy", "fetch", "claude-code/", "mcp-client",
		"gptbot", "chatgpt", "claudebot", "anthropic", "oai-searchbot", "firecrawl",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func limitText(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max]
}
