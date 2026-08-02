package models

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const AnonymousMonthlyQuota = 100

type APIKey struct {
	ID                   int64
	KeyPrefix            string
	Email                string
	Label                string
	Plan                 string
	MonthlyLimit         int
	Status               string
	StripeCustomerID     string
	StripeSubscriptionID string
}

type APIPlan struct {
	Name         string
	MonthlyLimit int
	PriceCents   int64
}

func APIPlans() []APIPlan {
	return []APIPlan{APIPlanFor("unlimited")}
}

// APIPlanFor returns the single legacy-compatible priority-throughput plan:
// $9.99/mo with 50,000 higher-ceiling calls per month. "unlimited" remains the
// stored identifier for existing Stripe subscriptions; baseline discovery is
// free and falls back to public safety limits rather than a payment wall.
func APIPlanFor(name string) APIPlan {
	return APIPlan{Name: "unlimited", MonthlyLimit: 50000, PriceCents: 999}
}

func HashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func HashAnonymousID(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:16])
}

func GenerateAPIKey(prefix string) (string, string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	raw := prefix + "_" + hex.EncodeToString(b[:])
	displayPrefix := raw
	if len(displayPrefix) > 16 {
		displayPrefix = displayPrefix[:16]
	}
	return raw, displayPrefix, nil
}

func ResolveAPIKey(db *sql.DB, raw string) (*APIKey, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, sql.ErrNoRows
	}
	return resolveAPIKeyByHash(db, HashAPIKey(raw))
}

func resolveAPIKeyByHash(db *sql.DB, hash string) (*APIKey, error) {
	var k APIKey
	err := db.QueryRow(`
		SELECT id, key_prefix, email, label, plan, monthly_limit, status,
		       stripe_customer_id, stripe_subscription_id
		FROM api_keys
		WHERE key_hash=$1
	`, hash).Scan(&k.ID, &k.KeyPrefix, &k.Email, &k.Label, &k.Plan, &k.MonthlyLimit, &k.Status, &k.StripeCustomerID, &k.StripeSubscriptionID)
	if err != nil {
		return nil, err
	}
	if k.Status != "active" {
		return nil, errors.New("api key inactive")
	}
	return &k, nil
}

func ActivateAPIKeyForCheckout(db *sql.DB, sessionID, email, planName, customerID, subscriptionID string) (raw string, key *APIKey, err error) {
	plan := APIPlanFor(planName)

	var existing APIKey
	err = db.QueryRow(`
		SELECT id, key_prefix, email, label, plan, monthly_limit, status,
		       stripe_customer_id, stripe_subscription_id
		FROM api_keys
		WHERE stripe_checkout_session_id=$1
	`, sessionID).Scan(&existing.ID, &existing.KeyPrefix, &existing.Email, &existing.Label, &existing.Plan, &existing.MonthlyLimit, &existing.Status, &existing.StripeCustomerID, &existing.StripeSubscriptionID)
	if err == nil {
		return "", &existing, nil
	}
	if err != sql.ErrNoRows {
		return "", nil, err
	}

	raw, prefix, err := GenerateAPIKey("nhs_live")
	if err != nil {
		return "", nil, err
	}
	hash := HashAPIKey(raw)
	var created APIKey
	err = db.QueryRow(`
		INSERT INTO api_keys (
			key_hash, key_prefix, email, label, plan, monthly_limit, status,
			stripe_customer_id, stripe_subscription_id, stripe_checkout_session_id
		)
		VALUES ($1,$2,$3,$4,$5,$6,'active',$7,$8,$9)
		RETURNING id, key_prefix, email, label, plan, monthly_limit, status,
		          stripe_customer_id, stripe_subscription_id
	`, hash, prefix, email, "Stripe "+plan.Name, plan.Name, plan.MonthlyLimit, customerID, subscriptionID, sessionID).
		Scan(&created.ID, &created.KeyPrefix, &created.Email, &created.Label, &created.Plan, &created.MonthlyLimit, &created.Status, &created.StripeCustomerID, &created.StripeSubscriptionID)
	if err != nil {
		return "", nil, err
	}
	return raw, &created, nil
}

func UpsertSubscriptionStatus(db *sql.DB, customerID, subscriptionID, status, planName string) error {
	if subscriptionID == "" {
		return nil
	}
	plan := APIPlanFor(planName)
	keyStatus := "active"
	switch status {
	case "active", "trialing":
		keyStatus = "active"
	default:
		keyStatus = "inactive"
	}
	_, err := db.Exec(`
		UPDATE api_keys
		SET status=$1, plan=$2, monthly_limit=$3, stripe_customer_id=COALESCE(NULLIF($4,''), stripe_customer_id), updated_at=NOW()
		WHERE stripe_subscription_id=$5
	`, keyStatus, plan.Name, plan.MonthlyLimit, customerID, subscriptionID)
	return err
}

func CurrentMonthUsage(db *sql.DB, key *APIKey, anonHash string) (int, error) {
	var used int
	if key != nil {
		err := db.QueryRow(`
			SELECT COALESCE(SUM(units),0)::int
			FROM usage_events
			WHERE api_key_id=$1 AND created_at >= date_trunc('month', NOW()) AND status < 400
		`, key.ID).Scan(&used)
		return used, err
	}
	err := db.QueryRow(`
		SELECT COALESCE(SUM(units),0)::int
		FROM usage_events
		WHERE anonymous_hash=$1 AND api_key_id IS NULL AND created_at >= date_trunc('month', NOW()) AND status < 400
	`, anonHash).Scan(&used)
	return used, err
}

func RecordUsageEvent(db *sql.DB, key *APIKey, anonHash, surface, method, path, tool string, units, status int, ua string) error {
	if db == nil {
		return nil
	}
	if units < 0 {
		units = 0
	}
	var keyID any
	if key != nil {
		keyID = key.ID
	} else {
		keyID = nil
	}
	_, err := db.Exec(`
		INSERT INTO usage_events (api_key_id, anonymous_hash, surface, method, path, tool_name, units, status, user_agent)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, keyID, anonHash, surface, method, path, tool, units, status, ua)
	if err == nil && key != nil && status < 400 {
		_, _ = db.Exec(`UPDATE api_keys SET last_used_at=NOW() WHERE id=$1`, key.ID)
	}
	return err
}

// ReservePriorityUsage atomically consumes one unit from an active API key's
// monthly priority-throughput allocation. A transaction-scoped advisory lock
// serializes the final check and insert for each key, so concurrent requests
// cannot overshoot the advertised allocation. Callers fall back to free safety
// limits when reserved is false or err is non-nil; discovery is never blocked.
func ReservePriorityUsage(db *sql.DB, key *APIKey, surface, method, path, tool string) (reservationID int64, remaining int, reserved bool, err error) {
	if db == nil || key == nil || key.MonthlyLimit < 1 {
		return 0, 0, false, nil
	}
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, false, err
	}
	defer tx.Rollback()

	// Domain-separate these locks from unrelated application advisory locks.
	const priorityUsageLockNamespace int64 = 0x4e48530000000000
	if _, err := tx.Exec(`SELECT pg_advisory_xact_lock($1)`, priorityUsageLockNamespace+key.ID); err != nil {
		return 0, 0, false, err
	}

	var used int
	if err := tx.QueryRow(`
		SELECT COALESCE(SUM(units),0)::int
		FROM usage_events
		WHERE api_key_id=$1
		  AND created_at >= date_trunc('month', NOW())
		  AND status < 400`, key.ID).Scan(&used); err != nil {
		return 0, 0, false, err
	}
	if used >= key.MonthlyLimit {
		return 0, 0, false, nil
	}

	if err := tx.QueryRow(`
		INSERT INTO usage_events (
			api_key_id, anonymous_hash, surface, method, path,
			tool_name, units, status, user_agent
		) VALUES ($1,'',$2,$3,$4,$5,1,200,'')
		RETURNING id`, key.ID, surface, method, path, tool).Scan(&reservationID); err != nil {
		return 0, 0, false, err
	}
	if _, err := tx.Exec(`UPDATE api_keys SET last_used_at=NOW() WHERE id=$1`, key.ID); err != nil {
		return 0, 0, false, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, false, err
	}
	return reservationID, key.MonthlyLimit - used - 1, true, nil
}

// ReleasePriorityUsage refunds one just-reserved unit when NHS rejects a tool
// or cannot perform the admitted request. The exact id + key scope prevents a
// caller from deleting any other usage record.
func ReleasePriorityUsage(db *sql.DB, key *APIKey, reservationID int64) error {
	if db == nil || key == nil || reservationID < 1 {
		return nil
	}
	_, err := db.Exec(`DELETE FROM usage_events WHERE id=$1 AND api_key_id=$2`, reservationID, key.ID)
	return err
}

func QuotaResetUnix() int64 {
	now := time.Now()
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Unix()
}

func QuotaErrorMessage(limit int) string {
	if limit == AnonymousMonthlyQuota {
		return fmt.Sprintf("anonymous free quota exceeded (%d billable calls/month); create an API key", limit)
	}
	return fmt.Sprintf("API quota exceeded (%d billable calls/month)", limit)
}
