package models

import (
	"database/sql"
	"time"
)

type GeoFixJob struct {
	ID              int64      `json:"id"`
	Host            string     `json:"host"`
	RepoURL         *string    `json:"repo_url,omitempty"`
	Email           string     `json:"email"`
	Notes           *string    `json:"notes,omitempty"`
	StripeSessionID *string    `json:"stripe_session_id,omitempty"`
	PriceCents      int        `json:"price_cents"`
	Currency        string     `json:"currency"`
	Status          string     `json:"status"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CreateGeoFixJob inserts an intake record before redirect to Stripe. Status
// starts as "pending"; webhook flips to "paid" and sets paid_at.
func CreateGeoFixJob(db *sql.DB, j *GeoFixJob) error {
	if j.PriceCents == 0 {
		j.PriceCents = 19900
	}
	if j.Currency == "" {
		j.Currency = "usd"
	}
	if j.Status == "" {
		j.Status = "pending"
	}
	return db.QueryRow(`
		INSERT INTO geo_fix_jobs (host, repo_url, email, notes, price_cents, currency, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`,
		j.Host, j.RepoURL, j.Email, j.Notes, j.PriceCents, j.Currency, j.Status,
	).Scan(&j.ID, &j.CreatedAt, &j.UpdatedAt)
}

// SetGeoFixJobSession attaches the Stripe checkout session id to an intake row.
func SetGeoFixJobSession(db *sql.DB, id int64, sessionID string) error {
	_, err := db.Exec(`
		UPDATE geo_fix_jobs
		SET stripe_session_id = $1, updated_at = NOW()
		WHERE id = $2`, sessionID, id)
	return err
}

// MarkGeoFixJobPaid is called from the Stripe webhook on checkout.session.completed.
// Idempotent — safe to call on already-paid rows.
func MarkGeoFixJobPaid(db *sql.DB, sessionID string) (*GeoFixJob, error) {
	j := &GeoFixJob{}
	var repoURL, notes sql.NullString
	var paidAt, completedAt sql.NullTime
	err := db.QueryRow(`
		UPDATE geo_fix_jobs
		SET status = 'paid',
		    paid_at = COALESCE(paid_at, NOW()),
		    updated_at = NOW()
		WHERE stripe_session_id = $1
		RETURNING id, host, repo_url, email, notes, stripe_session_id,
		          price_cents, currency, status, paid_at, completed_at,
		          created_at, updated_at`, sessionID,
	).Scan(&j.ID, &j.Host, &repoURL, &j.Email, &notes, &j.StripeSessionID,
		&j.PriceCents, &j.Currency, &j.Status, &paidAt, &completedAt,
		&j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if repoURL.Valid {
		j.RepoURL = &repoURL.String
	}
	if notes.Valid {
		j.Notes = &notes.String
	}
	if paidAt.Valid {
		j.PaidAt = &paidAt.Time
	}
	if completedAt.Valid {
		j.CompletedAt = &completedAt.Time
	}
	return j, nil
}

// ListGeoFixJobs returns all intake rows, newest first. Admin-only.
func ListGeoFixJobs(db *sql.DB, limit int) ([]GeoFixJob, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := db.Query(`
		SELECT id, host, COALESCE(repo_url, ''), email, COALESCE(notes, ''),
		       COALESCE(stripe_session_id, ''), price_cents, currency,
		       status, paid_at, completed_at, created_at, updated_at
		FROM geo_fix_jobs
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GeoFixJob
	for rows.Next() {
		var j GeoFixJob
		var repoURL, notes, sessionID string
		var paidAt, completedAt sql.NullTime
		if err := rows.Scan(&j.ID, &j.Host, &repoURL, &j.Email, &notes,
			&sessionID, &j.PriceCents, &j.Currency, &j.Status,
			&paidAt, &completedAt, &j.CreatedAt, &j.UpdatedAt); err != nil {
			continue
		}
		if repoURL != "" {
			j.RepoURL = &repoURL
		}
		if notes != "" {
			j.Notes = &notes
		}
		if sessionID != "" {
			j.StripeSessionID = &sessionID
		}
		if paidAt.Valid {
			j.PaidAt = &paidAt.Time
		}
		if completedAt.Valid {
			j.CompletedAt = &completedAt.Time
		}
		out = append(out, j)
	}
	return out, nil
}
