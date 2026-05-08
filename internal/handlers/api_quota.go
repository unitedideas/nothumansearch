package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/models"
)

type UsageGate struct {
	DB      *sql.DB
	BaseURL string
}

func NewUsageGate(db *sql.DB, baseURL string) *UsageGate {
	return &UsageGate{DB: db, BaseURL: baseURL}
}

func extractAPIKey(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-API-Key")); v != "" {
		return v
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return strings.TrimSpace(r.URL.Query().Get("api_key"))
}

func clientIP(r *http.Request) string {
	for _, h := range []string{"Fly-Client-IP", "X-Forwarded-For", "X-Real-IP"} {
		if v := strings.TrimSpace(r.Header.Get(h)); v != "" {
			return strings.TrimSpace(strings.Split(v, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

func (g *UsageGate) Billable(surface, label string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, anonHash, limit, used, ok := g.consume(w, r, surface, r.Method, label, "", 1)
		if !ok {
			return
		}
		w.Header().Set("X-Quota-Limit", intString(limit))
		w.Header().Set("X-Quota-Used", intString(used+1))
		if key != nil {
			w.Header().Set("X-API-Key-Plan", key.Plan)
		} else {
			w.Header().Set("X-Anonymous-ID", anonHash[:16])
		}
		next.ServeHTTP(w, r)
	})
}

func (g *UsageGate) consume(w http.ResponseWriter, r *http.Request, surface, method, path, tool string, units int) (*models.APIKey, string, int, int, bool) {
	rawKey := extractAPIKey(r)
	anonHash := models.HashAnonymousID(clientIP(r))
	var key *models.APIKey
	var err error
	if rawKey != "" {
		key, err = models.ResolveAPIKey(g.DB, rawKey)
		if err != nil {
			writeQuotaJSON(w, http.StatusUnauthorized, map[string]any{
				"error":   "invalid_api_key",
				"message": "API key was not found or is inactive",
			})
			_ = models.RecordUsageEvent(g.DB, nil, anonHash, surface, method, path, tool, 0, http.StatusUnauthorized, r.UserAgent())
			return nil, anonHash, 0, 0, false
		}
	}
	limit := models.AnonymousMonthlyQuota
	if key != nil {
		limit = key.MonthlyLimit
	}
	used, err := models.CurrentMonthUsage(g.DB, key, anonHash)
	if err != nil {
		writeQuotaJSON(w, http.StatusInternalServerError, map[string]any{"error": "quota_check_failed"})
		return key, anonHash, limit, used, false
	}
	if used+units > limit {
		status := http.StatusPaymentRequired
		_ = models.RecordUsageEvent(g.DB, key, anonHash, surface, method, path, tool, 0, status, r.UserAgent())
		writeQuotaJSON(w, status, map[string]any{
			"error":         "quota_exceeded",
			"message":       models.QuotaErrorMessage(limit),
			"limit":         limit,
			"used":          used,
			"reset_at_unix": models.QuotaResetUnix(),
			"subscribe_url": g.BaseURL + "/api/v1/api-keys/subscribe",
		})
		return key, anonHash, limit, used, false
	}
	if err := models.RecordUsageEvent(g.DB, key, anonHash, surface, method, path, tool, units, http.StatusOK, r.UserAgent()); err != nil {
		writeQuotaJSON(w, http.StatusInternalServerError, map[string]any{"error": "usage_record_failed"})
		return key, anonHash, limit, used, false
	}
	return key, anonHash, limit, used, true
}

func (g *UsageGate) ConsumeMCP(r *http.Request, tool string) (*models.APIKey, string, int, int, error) {
	keyRaw := extractAPIKey(r)
	anonHash := models.HashAnonymousID(clientIP(r))
	var key *models.APIKey
	if keyRaw != "" {
		var err error
		key, err = models.ResolveAPIKey(g.DB, keyRaw)
		if err != nil {
			_ = models.RecordUsageEvent(g.DB, nil, anonHash, "mcp", "tools/call", "/mcp", tool, 0, http.StatusUnauthorized, r.UserAgent())
			return nil, anonHash, 0, 0, errors.New("invalid_api_key")
		}
	}
	limit := models.AnonymousMonthlyQuota
	if key != nil {
		limit = key.MonthlyLimit
	}
	used, err := models.CurrentMonthUsage(g.DB, key, anonHash)
	if err != nil {
		return key, anonHash, limit, used, err
	}
	if used+1 > limit {
		_ = models.RecordUsageEvent(g.DB, key, anonHash, "mcp", "tools/call", "/mcp", tool, 0, http.StatusPaymentRequired, r.UserAgent())
		return key, anonHash, limit, used, errors.New("quota_exceeded")
	}
	return key, anonHash, limit, used, models.RecordUsageEvent(g.DB, key, anonHash, "mcp", "tools/call", "/mcp", tool, 1, http.StatusOK, r.UserAgent())
}

func writeQuotaJSON(w http.ResponseWriter, status int, data map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func intString(v int) string {
	return strconv.Itoa(v)
}
