package models

import (
	"crypto/rand"
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
}

var (
	ErrInvalidEmail  = errors.New("invalid email")
	ErrInvalidDomain = errors.New("invalid domain")
)

// NormalizeDomain strips scheme, path, and lowercases. Returns ErrInvalidDomain
// for empty/junk input.
func NormalizeDomain(raw string) (string, error) {
	d := strings.TrimSpace(raw)
	d = strings.TrimPrefix(d, "http://")
	d = strings.TrimPrefix(d, "https://")
	if i := strings.Index(d, "/"); i >= 0 {
		d = d[:i]
	}
	d = strings.TrimPrefix(d, "www.")
	d = strings.ToLower(d)
	if d == "" || !strings.Contains(d, ".") {
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

// RegisterMonitor inserts a new monitor row or refreshes the created_at on
// an existing (email, domain) pair. Returns the monitor row with token.
func RegisterMonitor(db *sql.DB, email, domain string) (*Monitor, error) {
	email, err := ValidateEmail(email)
	if err != nil {
		return nil, err
	}
	domain, err = NormalizeDomain(domain)
	if err != nil {
		return nil, err
	}
	token, err := newToken()
	if err != nil {
		return nil, err
	}

	row := db.QueryRow(`
		INSERT INTO monitors (email, domain, token)
		VALUES ($1, $2, $3)
		ON CONFLICT (email, domain) DO UPDATE SET created_at = NOW()
		RETURNING id, email, domain, token, created_at
	`, email, domain, token)

	m := &Monitor{}
	if err := row.Scan(&m.ID, &m.Email, &m.Domain, &m.Token, &m.CreatedAt); err != nil {
		return nil, err
	}
	return m, nil
}

func GetMonitorByToken(db *sql.DB, token string) (*Monitor, error) {
	row := db.QueryRow(`
		SELECT id, email, domain, token, last_score, last_signals_hash,
		       created_at, last_checked_at, last_notified_at
		FROM monitors WHERE token = $1
	`, token)
	m := &Monitor{}
	if err := row.Scan(&m.ID, &m.Email, &m.Domain, &m.Token,
		&m.LastScore, &m.LastSignalsHash, &m.CreatedAt,
		&m.LastCheckedAt, &m.LastNotifiedAt); err != nil {
		return nil, err
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
		       created_at, last_checked_at, last_notified_at
		FROM monitors
		WHERE last_checked_at IS NULL OR last_checked_at < $1
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
		if err := rows.Scan(&m.ID, &m.Email, &m.Domain, &m.Token,
			&m.LastScore, &m.LastSignalsHash, &m.CreatedAt,
			&m.LastCheckedAt, &m.LastNotifiedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
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
