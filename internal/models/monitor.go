package models

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

type Monitor struct {
	ID               int64      `json:"id"`
	Email            string     `json:"email"`
	Domain           string     `json:"domain"`
	Token            string     `json:"token"`
	LastScore        *int       `json:"last_score,omitempty"`
	LastSignalsHash  *string    `json:"last_signals_hash,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"`
	LastNotifiedAt   *time.Time `json:"last_notified_at,omitempty"`
	Status           string     `json:"status"`
	QuarantineReason *string    `json:"quarantine_reason,omitempty"`
	QuarantinedAt    *time.Time `json:"quarantined_at,omitempty"`
}

type MonitorAdminActionCount struct {
	Day    time.Time `json:"day"`
	Action string    `json:"action"`
	Count  int       `json:"count"`
}

var (
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidDomain   = errors.New("invalid domain")
	ErrTooManyMonitors = errors.New("too many monitors for this email")
)

// MaxMonitorsPerEmail caps how many domain subscriptions a single email
// address can hold. Prevents a script from filling the monitors table with
// arbitrary domains and turning the weekly worker into a crawl amplifier.
const MaxMonitorsPerEmail = 20

const (
	MonitorStatusActive      = "active"
	MonitorStatusQuarantined = "quarantined"
)

const (
	MonitorAdminActionApproveMonitoring  = "approve_monitoring"
	MonitorAdminActionKeepQuarantined    = "keep_quarantined"
	MonitorAdminActionRequestScoreRerun  = "request_score_rerun"
	MonitorAdminActionRemediationOffered = "remediation_offered"
	activeMonitorDuePredicate            = "status = 'active' AND (last_checked_at IS NULL OR last_checked_at < $1)"
	monitorAdminActionCountsQuery        = `
		SELECT date_trunc('day', created_at)::date AS day, action, COUNT(*)::int
		FROM monitor_admin_actions
		WHERE created_at >= NOW() - ($1::int * INTERVAL '1 day')
		GROUP BY day, action
		ORDER BY day DESC, action ASC`
)

var sharedHostApexDomains = map[string]bool{
	"carrd.co":      true,
	"github.io":     true,
	"gitlab.io":     true,
	"glitch.me":     true,
	"herokuapp.com": true,
	"netlify.app":   true,
	"pages.dev":     true,
	"repl.co":       true,
	"vercel.app":    true,
	"webflow.io":    true,
	"wixsite.com":   true,
}

func MonitorInitialStatus(domain string) (string, *string) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "www.")
	if sharedHostApexDomains[domain] {
		reason := "shared-host apex domain requires admin review"
		return MonitorStatusQuarantined, &reason
	}
	return MonitorStatusActive, nil
}

func RedactEmail(email string) (domain string, hash string) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if at := strings.LastIndex(normalized, "@"); at >= 0 && at < len(normalized)-1 {
		domain = normalized[at+1:]
	}
	sum := sha256.Sum256([]byte(normalized))
	return domain, hex.EncodeToString(sum[:])[:16]
}

// privateHostPrefixes lists hostname shapes that must never be monitored —
// SSRF from the worker crawling arbitrary internal IPs / metadata endpoints.
// Also blocks the literal hostnames that resolve to them.
var privateHostPrefixes = []string{
	"localhost",
	"127.",
	"10.",
	"192.168.",
	"169.254.", // link-local incl. AWS/GCP metadata 169.254.169.254
	"0.",
	"::1",
	"fc00:",
	"fd",
}

// is172Private catches 172.16.0.0/12 which isn't a simple prefix match.
func is172Private(host string) bool {
	if !strings.HasPrefix(host, "172.") {
		return false
	}
	rest := strings.TrimPrefix(host, "172.")
	dot := strings.Index(rest, ".")
	if dot < 0 {
		return false
	}
	octet := rest[:dot]
	// 172.16 — 172.31 are private.
	if len(octet) != 2 {
		return false
	}
	if octet[0] != '1' && octet[0] != '2' && octet[0] != '3' {
		return false
	}
	// 16-31: second digit 6-9 with '1', 0-9 with '2', 0-1 with '3'.
	switch octet[0] {
	case '1':
		return octet[1] >= '6' && octet[1] <= '9'
	case '2':
		return octet[1] >= '0' && octet[1] <= '9'
	case '3':
		return octet[1] == '0' || octet[1] == '1'
	}
	return false
}

func isPrivateHost(host string) bool {
	for _, p := range privateHostPrefixes {
		if strings.HasPrefix(host, p) {
			return true
		}
	}
	return is172Private(host)
}

// NormalizeDomain strips scheme, path, port, and lowercases. Returns
// ErrInvalidDomain for empty/junk input or private-address shapes.
func NormalizeDomain(raw string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(raw))
	d = strings.TrimPrefix(d, "http://")
	d = strings.TrimPrefix(d, "https://")
	if i := strings.Index(d, "/"); i >= 0 {
		d = d[:i]
	}
	if i := strings.Index(d, ":"); i >= 0 {
		d = d[:i]
	}
	d = strings.TrimPrefix(d, "www.")
	if d == "" || !strings.Contains(d, ".") {
		return "", ErrInvalidDomain
	}
	if isPrivateHost(d) {
		return "", ErrInvalidDomain
	}
	return d, nil
}

// ValidateEmail is a cheap shape check — not full RFC 5322.
func ValidateEmail(raw string) (string, error) {
	e := strings.TrimSpace(strings.ToLower(raw))
	if !strings.Contains(e, "@") || len(e) < 5 || len(e) > 254 {
		return "", ErrInvalidEmail
	}
	at := strings.LastIndex(e, "@")
	if at == 0 || at == len(e)-1 {
		return "", ErrInvalidEmail
	}
	if !strings.Contains(e[at+1:], ".") {
		return "", ErrInvalidEmail
	}
	return e, nil
}

func newToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// RegisterMonitor inserts a new monitor row or refreshes created_at on an
// existing (email, domain) pair. Returns the monitor row with token.
// The token for an existing pair is preserved — RETURNING gives back the
// already-stored value, not the newly generated one. Unsubscribe links
// from a prior registration continue to work after re-registration.
func RegisterMonitor(db *sql.DB, email, domain string) (*Monitor, error) {
	email, err := ValidateEmail(email)
	if err != nil {
		return nil, err
	}
	domain, err = NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}

	// Rate limit: cap the number of domains any one email can watch to
	// prevent a script from filling the table + turning the weekly worker
	// into a large-scale third-party-domain fetcher.
	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM monitors WHERE email = $1 AND domain != $2`,
		email, domain,
	).Scan(&count); err != nil {
		return nil, err
	}
	if count >= MaxMonitorsPerEmail {
		return nil, ErrTooManyMonitors
	}

	token, err := newToken()
	if err != nil {
		return nil, err
	}
	status, quarantineReason := MonitorInitialStatus(domain)

	row := db.QueryRow(`
		INSERT INTO monitors (email, domain, token, status, quarantine_reason, quarantined_at)
		VALUES ($1, $2, $3, $4, $5, CASE WHEN $4 = 'quarantined' THEN NOW() ELSE NULL END)
		ON CONFLICT (email, domain) DO UPDATE SET created_at = NOW()
		RETURNING id, email, domain, token, status, quarantine_reason, quarantined_at, created_at
	`, email, domain, token, status, quarantineReason)

	m := &Monitor{}
	var reason sql.NullString
	var quarantinedAt sql.NullTime
	if err := row.Scan(&m.ID, &m.Email, &m.Domain, &m.Token, &m.Status, &reason, &quarantinedAt, &m.CreatedAt); err != nil {
		return nil, err
	}
	if reason.Valid {
		m.QuarantineReason = &reason.String
	}
	if quarantinedAt.Valid {
		m.QuarantinedAt = &quarantinedAt.Time
	}
	return m, nil
}

func GetMonitorByToken(db *sql.DB, token string) (*Monitor, error) {
	row := db.QueryRow(`
		SELECT id, email, domain, token, last_score, last_signals_hash,
		       created_at, last_checked_at, last_notified_at,
		       status, quarantine_reason, quarantined_at
		FROM monitors WHERE token = $1
	`, token)
	m := &Monitor{}
	var reason sql.NullString
	var quarantinedAt sql.NullTime
	if err := row.Scan(&m.ID, &m.Email, &m.Domain, &m.Token,
		&m.LastScore, &m.LastSignalsHash, &m.CreatedAt,
		&m.LastCheckedAt, &m.LastNotifiedAt, &m.Status,
		&reason, &quarantinedAt); err != nil {
		return nil, err
	}
	if reason.Valid {
		m.QuarantineReason = &reason.String
	}
	if quarantinedAt.Valid {
		m.QuarantinedAt = &quarantinedAt.Time
	}
	return m, nil
}

func DeleteMonitorByToken(db *sql.DB, token string) error {
	_, err := db.Exec(`DELETE FROM monitors WHERE token = $1`, token)
	return err
}

// ListDueMonitors returns monitors whose last_checked_at is older than the
// given cutoff (or NULL). Used by the weekly check job.
func ListDueMonitors(db *sql.DB, cutoff time.Time, limit int) ([]Monitor, error) {
	rows, err := db.Query(`
		SELECT id, email, domain, token, last_score, last_signals_hash,
		       created_at, last_checked_at, last_notified_at,
		       status, quarantine_reason, quarantined_at
		FROM monitors
		WHERE `+activeMonitorDuePredicate+`
		ORDER BY last_checked_at NULLS FIRST
		LIMIT $2
	`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Monitor
	for rows.Next() {
		m := Monitor{}
		var reason sql.NullString
		var quarantinedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.Email, &m.Domain, &m.Token,
			&m.LastScore, &m.LastSignalsHash, &m.CreatedAt,
			&m.LastCheckedAt, &m.LastNotifiedAt, &m.Status,
			&reason, &quarantinedAt); err != nil {
			return nil, err
		}
		if reason.Valid {
			m.QuarantineReason = &reason.String
		}
		if quarantinedAt.Valid {
			m.QuarantinedAt = &quarantinedAt.Time
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func ActiveMonitorDuePredicateForTest() string {
	return activeMonitorDuePredicate
}

func MonitorAdminActionCountsQueryForTest() string {
	return monitorAdminActionCountsQuery
}

// UpdateMonitorCheck records a check result. notified=true also bumps
// last_notified_at so we can rate-limit alerts.
func UpdateMonitorCheck(db *sql.DB, id int64, score int, signalsHash string, notified bool) error {
	if notified {
		_, err := db.Exec(`
			UPDATE monitors SET last_score=$2, last_signals_hash=$3,
			                     last_checked_at=NOW(), last_notified_at=NOW()
			WHERE id=$1
		`, id, score, signalsHash)
		return err
	}
	_, err := db.Exec(`
		UPDATE monitors SET last_score=$2, last_signals_hash=$3,
		                     last_checked_at=NOW()
		WHERE id=$1
	`, id, score, signalsHash)
	return err
}

// QuarantineMonitorCheck records a check result while removing the monitor
// from the weekly active-check queue. Used for first-check junk rows that
// should remain auditable but should not keep generating crawl work.
func QuarantineMonitorCheck(db *sql.DB, id int64, score int, signalsHash string, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "monitor requires admin review"
	}
	_, err := db.Exec(`
		UPDATE monitors SET status=$2, quarantine_reason=$3,
		                     quarantined_at=COALESCE(quarantined_at, NOW()),
		                     last_score=$4, last_signals_hash=$5,
		                     last_checked_at=NOW()
		WHERE id=$1
	`, id, MonitorStatusQuarantined, reason, score, signalsHash)
	return err
}

// ListRecentMonitors returns the most recently created/updated monitor rows.
func ListRecentMonitors(db *sql.DB, limit int) ([]Monitor, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := db.Query(`
		SELECT id, email, domain, token, last_score, last_signals_hash,
		       created_at, last_checked_at, last_notified_at,
		       status, quarantine_reason, quarantined_at
		FROM monitors
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Monitor
	for rows.Next() {
		m := Monitor{}
		var reason sql.NullString
		var quarantinedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.Email, &m.Domain, &m.Token,
			&m.LastScore, &m.LastSignalsHash, &m.CreatedAt,
			&m.LastCheckedAt, &m.LastNotifiedAt, &m.Status,
			&reason, &quarantinedAt); err != nil {
			return nil, err
		}
		if reason.Valid {
			m.QuarantineReason = &reason.String
		}
		if quarantinedAt.Valid {
			m.QuarantinedAt = &quarantinedAt.Time
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func ValidMonitorAdminAction(action string) bool {
	switch strings.TrimSpace(action) {
	case MonitorAdminActionApproveMonitoring,
		MonitorAdminActionKeepQuarantined,
		MonitorAdminActionRequestScoreRerun,
		MonitorAdminActionRemediationOffered:
		return true
	default:
		return false
	}
}

func ApplyMonitorAdminAction(db *sql.DB, monitorID int64, action, operator, source, notes string) error {
	action = strings.TrimSpace(action)
	operator = strings.TrimSpace(operator)
	source = strings.TrimSpace(source)
	notes = strings.TrimSpace(notes)
	if !ValidMonitorAdminAction(action) {
		return errors.New("invalid monitor admin action")
	}
	if operator == "" {
		return errors.New("monitor admin operator required")
	}
	if source == "" {
		source = "admin_api"
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		INSERT INTO monitor_admin_actions (monitor_id, action, operator, source, notes)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
	`, monitorID, action, operator, source, notes); err != nil {
		return err
	}

	var update string
	switch action {
	case MonitorAdminActionApproveMonitoring:
		update = `
			UPDATE monitors
			SET status='active',
			    quarantine_reason=NULL,
			    last_admin_action=$2,
			    last_admin_action_at=NOW(),
			    last_admin_operator=$3,
			    last_admin_source=$4,
			    private_review_notes=COALESCE(NULLIF($5, ''), private_review_notes)
			WHERE id=$1`
	case MonitorAdminActionKeepQuarantined:
		update = `
			UPDATE monitors
			SET status='quarantined',
			    quarantine_reason=COALESCE(NULLIF($5, ''), quarantine_reason, 'kept quarantined by admin review'),
			    quarantined_at=COALESCE(quarantined_at, NOW()),
			    last_admin_action=$2,
			    last_admin_action_at=NOW(),
			    last_admin_operator=$3,
			    last_admin_source=$4,
			    private_review_notes=COALESCE(NULLIF($5, ''), private_review_notes)
			WHERE id=$1`
	case MonitorAdminActionRequestScoreRerun:
		update = `
			UPDATE monitors
			SET score_rerun_requested_at=NOW(),
			    last_admin_action=$2,
			    last_admin_action_at=NOW(),
			    last_admin_operator=$3,
			    last_admin_source=$4,
			    private_review_notes=COALESCE(NULLIF($5, ''), private_review_notes)
			WHERE id=$1`
	case MonitorAdminActionRemediationOffered:
		update = `
			UPDATE monitors
			SET remediation_offered_at=NOW(),
			    last_admin_action=$2,
			    last_admin_action_at=NOW(),
			    last_admin_operator=$3,
			    last_admin_source=$4,
			    private_review_notes=COALESCE(NULLIF($5, ''), private_review_notes)
			WHERE id=$1`
	}

	res, err := tx.Exec(update, monitorID, action, operator, source, notes)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func ListMonitorAdminActionCounts(db *sql.DB, days int) ([]MonitorAdminActionCount, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	rows, err := db.Query(monitorAdminActionCountsQuery, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MonitorAdminActionCount
	for rows.Next() {
		var item MonitorAdminActionCount
		if err := rows.Scan(&item.Day, &item.Action, &item.Count); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
