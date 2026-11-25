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

// Account is a human user identified by email. Activation is driven by Stripe;
// an active account may use the website search UI via a session cookie.
type Account struct {
	ID                   int64
	Email                string
	Plan                 string
	Status               string
	MonthlyLimit         int
	StripeCustomerID     string
	StripeSubscriptionID string
}

// Active reports whether the account holds a live subscription. Nil-safe so
// callers can write acct.Active() on the result of a lookup that may be nil.
func (a *Account) Active() bool { return a != nil && a.Status == "active" }

const (
	magicLinkTTL = 20 * time.Minute
	sessionTTL   = 30 * 24 * time.Hour
)

func newAuthToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func hashAuthToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// EnsureAccount returns the account for email, creating an inactive one if absent.
func EnsureAccount(db *sql.DB, email string) (*Account, error) {
	email = normalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, errors.New("valid email required")
	}
	var a Account
	err := db.QueryRow(`
		INSERT INTO accounts (email) VALUES ($1)
		ON CONFLICT (email) DO UPDATE SET updated_at=NOW()
		RETURNING id, email, plan, status, monthly_limit, stripe_customer_id, stripe_subscription_id
	`, email).Scan(&a.ID, &a.Email, &a.Plan, &a.Status, &a.MonthlyLimit, &a.StripeCustomerID, &a.StripeSubscriptionID)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// SetAccountSubscription upserts an account's subscription state from Stripe.
func SetAccountSubscription(db *sql.DB, email, customerID, subscriptionID, planName, stripeStatus string) (*Account, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, errors.New("email required")
	}
	plan := APIPlanFor(planName)
	status := "inactive"
	switch strings.ToLower(strings.TrimSpace(stripeStatus)) {
	case "active", "trialing", "paid", "complete":
		status = "active"
	}
	var a Account
	err := db.QueryRow(`
		INSERT INTO accounts (email, plan, status, monthly_limit, stripe_customer_id, stripe_subscription_id)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (email) DO UPDATE SET
			plan=EXCLUDED.plan,
			status=EXCLUDED.status,
			monthly_limit=EXCLUDED.monthly_limit,
			stripe_customer_id=COALESCE(NULLIF(EXCLUDED.stripe_customer_id,''), accounts.stripe_customer_id),
			stripe_subscription_id=COALESCE(NULLIF(EXCLUDED.stripe_subscription_id,''), accounts.stripe_subscription_id),
			updated_at=NOW()
		RETURNING id, email, plan, status, monthly_limit, stripe_customer_id, stripe_subscription_id
	`, email, plan.Name, status, plan.MonthlyLimit, customerID, subscriptionID).
		Scan(&a.ID, &a.Email, &a.Plan, &a.Status, &a.MonthlyLimit, &a.StripeCustomerID, &a.StripeSubscriptionID)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// DeactivateAccountBySubscription flips an account inactive when Stripe reports
// its subscription ended (customer.subscription.deleted / unpaid).
func DeactivateAccountBySubscription(db *sql.DB, subscriptionID string) error {
	if db == nil || strings.TrimSpace(subscriptionID) == "" {
		return nil
	}
	_, err := db.Exec(`UPDATE accounts SET status='inactive', updated_at=NOW() WHERE stripe_subscription_id=$1`, subscriptionID)
	return err
}

// CreateMagicLink stores a single-use, 20-minute login token and returns the raw
// value to embed in the emailed link. Only the hash is persisted.
func CreateMagicLink(db *sql.DB, email string) (string, error) {
	email = normalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") {
		return "", errors.New("valid email required")
	}
	raw, err := newAuthToken()
	if err != nil {
		return "", err
	}
	if _, err := db.Exec(`
		INSERT INTO magic_links (email, token_hash, expires_at) VALUES ($1,$2,$3)
	`, email, hashAuthToken(raw), time.Now().Add(magicLinkTTL)); err != nil {
		return "", err
	}
	return raw, nil
}

// ConsumeMagicLink validates and burns a login token, returning the email it was
// issued to. Fails if missing, expired, or already used.
func ConsumeMagicLink(db *sql.DB, rawToken string) (string, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return "", errors.New("token required")
	}
	var email string
	err := db.QueryRow(`
		UPDATE magic_links SET used_at=NOW()
		WHERE token_hash=$1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING email
	`, hashAuthToken(rawToken)).Scan(&email)
	if err == sql.ErrNoRows {
		return "", errors.New("invalid or expired link")
	}
	if err != nil {
		return "", err
	}
	return email, nil
}

// CreateSession mints a 30-day session token for an account; only the hash is stored.
func CreateSession(db *sql.DB, accountID int64) (string, error) {
	raw, err := newAuthToken()
	if err != nil {
		return "", err
	}
	if _, err := db.Exec(`
		INSERT INTO sessions (token_hash, account_id, expires_at) VALUES ($1,$2,$3)
	`, hashAuthToken(raw), accountID, time.Now().Add(sessionTTL)); err != nil {
		return "", err
	}
	return raw, nil
}

// ResolveSession returns the account behind a live session token, or an error.
func ResolveSession(db *sql.DB, rawToken string) (*Account, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, sql.ErrNoRows
	}
	var a Account
	err := db.QueryRow(`
		SELECT a.id, a.email, a.plan, a.status, a.monthly_limit, a.stripe_customer_id, a.stripe_subscription_id
		FROM sessions s JOIN accounts a ON a.id = s.account_id
		WHERE s.token_hash=$1 AND s.expires_at > NOW()
	`, hashAuthToken(rawToken)).Scan(&a.ID, &a.Email, &a.Plan, &a.Status, &a.MonthlyLimit, &a.StripeCustomerID, &a.StripeSubscriptionID)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// DeleteSession revokes a session token (logout).
func DeleteSession(db *sql.DB, rawToken string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM sessions WHERE token_hash=$1`, hashAuthToken(strings.TrimSpace(rawToken)))
	return err
}
