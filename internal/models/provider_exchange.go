package models

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/lib/pq"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

const (
	ProviderDisclosureLabel            = "Provider-funded action"
	ProviderPrincipalConsentV1         = "nhs-principal-consent-v1"
	ProviderClaimChallengeTTL          = 24 * time.Hour
	ProviderClaimVerificationFreshness = 7 * 24 * time.Hour
	ProviderClaimDNSRecheckInterval    = 24 * time.Hour
	ProviderClaimDNSFailureRetry       = time.Hour
	ProviderClaimDNSLeaseDuration      = 2 * time.Minute
	ProviderClaimDNSFailureLimit       = 3
	ProviderClaimDNSMaximumBatch       = 100
	ActionTicketDefaultTTL             = 30 * 24 * time.Hour
	ActionTicketMaximumTTL             = 90 * 24 * time.Hour
	ActionTicketIntentRetention        = 30 * 24 * time.Hour
	OutcomeReceiptValidity             = 10 * 365 * 24 * time.Hour
	// Exported USD-only pilot caps keep each commercial field bounded according
	// to its actual risk instead of sharing one ambiguous generic ceiling.
	ProviderBountyMaximumCents          = int64(1_000_000)
	ProviderPrincipalPriceMaximumCents  = int64(100_000_000)
	ProviderTermsCreditMaximumCents     = int64(10_000_000)
	ProviderMoneyMaximumCents           = int64(100_000_000)
	ProviderOfferMaximumPerClaim        = 20
	ProviderActiveOfferMaximumPerAction = 3
	providerOfferLockNamespace          = "nhs-provider-offer"
	providerIdempotencyNamespace        = "nhs-provider-outcome"
)

var (
	ErrProviderClaimExists            = errors.New("a live provider claim already exists for this account and site")
	ErrProviderSiteClaimed            = errors.New("site already has a live provider claim")
	ErrProviderClaimNotVerified       = errors.New("provider claim is not verified")
	ErrProviderClaimVerificationStale = errors.New("provider claim DNS verification is stale")
	ErrProviderChallengeExpired       = errors.New("provider claim challenge expired")
	ErrProviderChallengeMismatch      = errors.New("provider claim challenge did not match")
	ErrProviderDNSLeaseLost           = errors.New("provider DNS verification lease is no longer active")
	ErrProviderAPIKeyExists           = errors.New("provider claim already has an active callback key")
	ErrProviderOfferNotPublic         = errors.New("provider offer is not eligible for public action discovery")
	ErrProviderOfferLimit             = errors.New("provider offer limit reached")
	ErrProviderOfferRevoked           = errors.New("provider offer authorization was revoked by an emergency pause")
	ErrInsufficientProviderFunds      = errors.New("insufficient prepaid provider budget")
	ErrProviderBudgetLimit            = errors.New("provider budget limit reached")
	ErrProviderTermsCreditLimit       = errors.New("provider terms credit limit reached")
	ErrProviderIdempotency            = errors.New("idempotency key was already used with a different payload")
	ErrProviderOutcomeExists          = errors.New("outcome already recorded for action ticket")
	ErrProviderOutcomeTransition      = errors.New("invalid action ticket outcome transition")
	ErrActionTicketExists             = errors.New("action ticket already exists for this search receipt and offer")
	ErrActionTicketExpired            = errors.New("action ticket expired")
	ErrInvalidProviderExchange        = errors.New("invalid provider exchange input")
)

var (
	providerUUIDPattern         = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	providerReferencePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{2,199}$`)
	providerOpaquePattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)
	providerHashPattern         = regexp.MustCompile(`^[0-9a-f]{64}$`)
	providerSigningKeyIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{2,63}$`)
	providerRegionPattern       = regexp.MustCompile(`^[A-Z]{2}(-[A-Z0-9]{1,3})?$`)
)

var providerActionTypes = map[string]bool{
	"lead": true, "demo": true, "trial": true, "signup": true,
	"purchase": true, "quote": true, "application": true, "booking": true,
}

var providerChargeEvents = map[string]bool{
	"accepted": true, "activated": true, "converted": true,
}

var providerOutcomeTypes = map[string]bool{
	"accepted": true, "activated": true, "converted": true,
	"rejected": true, "duplicate": true, "invalid": true,
}

var providerDemandTopics = map[string]bool{
	"payments": true, "commerce": true, "jobs": true, "data": true,
	"search": true, "weather": true, "maps": true, "email": true,
	"messaging": true, "image": true, "video": true, "audio": true,
	"documents": true, "security": true, "finance": true, "health": true,
	"education": true, "news": true, "analytics": true, "automation": true,
	"productivity": true, "identity": true, "storage": true, "ai-tools": true,
	"developer-tools": true, "other": true,
}

var providerBudgetBands = map[string]bool{
	"unspecified": true, "under_100": true, "100_499": true,
	"500_1999": true, "2000_plus": true,
}

var providerUrgencies = map[string]bool{
	"unspecified": true, "now": true, "7_days": true,
	"30_days": true, "researching": true,
}

var providerRequirementFlags = map[string]bool{
	"api_access": true, "mcp": true, "sandbox": true, "self_serve": true,
	"enterprise": true, "compliance": true, "multilingual": true,
	"human_support": true,
}

type rowScanner interface {
	Scan(dest ...any) error
}

// ProviderClaim represents account ownership of one indexed site. The raw DNS
// challenge is returned once by creation/rotation and is never stored.
type ProviderClaim struct {
	ID                              string     `json:"id"`
	AccountID                       int64      `json:"-"`
	SiteID                          string     `json:"site_id"`
	Domain                          string     `json:"domain"`
	VerificationMethod              string     `json:"verification_method"`
	VerificationRecordName          string     `json:"verification_record_name"`
	Status                          string     `json:"status"`
	ChallengeExpiresAt              time.Time  `json:"challenge_expires_at"`
	VerifiedAt                      *time.Time `json:"verified_at,omitempty"`
	VerificationLastSucceededAt     *time.Time `json:"verification_last_succeeded_at,omitempty"`
	VerificationLastAttemptedAt     *time.Time `json:"verification_last_attempted_at,omitempty"`
	VerificationConsecutiveFailures int        `json:"verification_consecutive_failures"`
	VerificationNextCheckAt         *time.Time `json:"verification_next_check_at,omitempty"`
	RevokedAt                       *time.Time `json:"revoked_at,omitempty"`
	CreatedAt                       time.Time  `json:"created_at"`
	UpdatedAt                       time.Time  `json:"updated_at"`
}

// ProviderClaimDNSLease contains only the minimum work description needed by
// the internal verifier. The persisted token hash and raw DNS proof are never
// returned through this type or any public handler response.
type ProviderClaimDNSLease struct {
	ClaimID                string    `json:"-"`
	LeaseID                string    `json:"-"`
	VerificationRecordName string    `json:"-"`
	LeaseUntil             time.Time `json:"-"`
}

type ProviderClaimDNSCheckResult struct {
	ClaimID                     string
	Matched                     bool
	Revoked                     bool
	ConsecutiveFailures         int
	VerificationLastSucceededAt *time.Time
}

// ProviderAPIKey is a claim-scoped provider callback key. It intentionally has
// no email, label, request fingerprint, or raw-key field.
type ProviderAPIKey struct {
	ID              int64      `json:"id"`
	ProviderClaimID string     `json:"provider_claim_id"`
	KeyPrefix       string     `json:"key_prefix"`
	Status          string     `json:"status"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ProviderOffer struct {
	ID                     string     `json:"id"`
	ProviderClaimID        string     `json:"provider_claim_id"`
	SiteID                 string     `json:"site_id"`
	Domain                 string     `json:"domain"`
	Status                 string     `json:"status"`
	Version                int        `json:"version"`
	OfferName              string     `json:"name"`
	OfferSummary           string     `json:"summary"`
	ActionType             string     `json:"action_type"`
	ActionURL              string     `json:"action_url"`
	DisclosureLabel        string     `json:"disclosure"`
	ChargeEvent            string     `json:"charge_event"`
	BountyCents            int64      `json:"bounty_cents"`
	Currency               string     `json:"currency"`
	PrincipalPriceMode     string     `json:"principal_price_mode"`
	PrincipalPriceCents    *int64     `json:"principal_price_cents,omitempty"`
	PrincipalCurrency      string     `json:"principal_currency"`
	BillingMode            string     `json:"billing_mode"`
	TermsCreditLimitCents  *int64     `json:"terms_credit_limit_cents,omitempty"`
	TermsPeriodDays        *int       `json:"terms_period_days,omitempty"`
	TermsPeriodAnchorAt    *time.Time `json:"terms_period_anchor_at,omitempty"`
	TermsEvidenceReference string     `json:"terms_evidence_reference,omitempty"`
	OrganicPosition        int        `json:"organic_position,omitempty"`
	BudgetBalanceCents     int64      `json:"budget_balance_cents"`
	ActivatedAt            *time.Time `json:"activated_at,omitempty"`
	PausedAt               *time.Time `json:"paused_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// PublicProviderOffer is the deliberately narrow sponsored-action shape. It
// excludes claim ownership, evidence references, and provider budget balances.
type PublicProviderOffer struct {
	OfferID                   string `json:"offer_id"`
	OfferVersion              int    `json:"offer_version"`
	SiteID                    string `json:"site_id"`
	Domain                    string `json:"domain"`
	OfferName                 string `json:"name"`
	OfferSummary              string `json:"summary"`
	ActionType                string `json:"action_type"`
	ChargeEvent               string `json:"charge_event"`
	DisclosureLabel           string `json:"disclosure"`
	ProviderFundedBountyCents int64  `json:"provider_funded_bounty_cents"`
	ProviderFundedCurrency    string `json:"provider_funded_currency"`
	PrincipalPriceMode        string `json:"principal_price_mode"`
	PrincipalPriceCents       *int64 `json:"principal_price_cents,omitempty"`
	PrincipalCurrency         string `json:"principal_currency"`
	OrganicPosition           int    `json:"organic_position"`
}

type ProviderOfferInput struct {
	OfferName              string
	OfferSummary           string
	ActionType             string
	ActionURL              string
	ChargeEvent            string
	BountyCents            int64
	Currency               string
	PrincipalPriceMode     string
	PrincipalPriceCents    *int64
	PrincipalCurrency      string
	BillingMode            string
	TermsCreditLimitCents  *int64
	TermsPeriodDays        *int
	TermsEvidenceReference string
}

type ProviderBudgetEntry struct {
	ID                int64     `json:"id"`
	ProviderClaimID   string    `json:"provider_claim_id"`
	ProviderOfferID   string    `json:"provider_offer_id"`
	ActionTicketID    string    `json:"action_ticket_id,omitempty"`
	EntryType         string    `json:"entry_type"`
	AmountCents       int64     `json:"amount_cents"`
	Currency          string    `json:"currency"`
	ExternalReference string    `json:"external_reference"`
	BalanceAfterCents int64     `json:"balance_after_cents"`
	Replayed          bool      `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
}

type ActionTicket struct {
	ID                             string     `json:"id"`
	ProviderClaimID                string     `json:"provider_claim_id"`
	ProviderOfferID                string     `json:"provider_offer_id"`
	SearchReceiptID                string     `json:"search_receipt_id,omitempty"`
	SourceIsSynthetic              bool       `json:"-"`
	TokenHash                      string     `json:"-"`
	TokenNonce                     string     `json:"-"`
	CreationRequestHash            string     `json:"-"`
	OfferVersionSnapshot           int        `json:"offer_version"`
	OfferNameSnapshot              string     `json:"offer_name"`
	OfferSummarySnapshot           string     `json:"offer_summary"`
	ActionTypeSnapshot             string     `json:"action_type"`
	ActionURLSnapshot              string     `json:"action_url"`
	DisclosureSnapshot             string     `json:"disclosure"`
	ChargeEventSnapshot            string     `json:"charge_event"`
	BountyCentsSnapshot            int64      `json:"bounty_cents"`
	CurrencySnapshot               string     `json:"currency"`
	BillingModeSnapshot            string     `json:"billing_mode"`
	TermsEvidenceReferenceSnapshot string     `json:"-"`
	TermsCreditLimitCentsSnapshot  *int64     `json:"-"`
	TermsPeriodDaysSnapshot        *int       `json:"-"`
	TermsPeriodAnchorAtSnapshot    *time.Time `json:"-"`
	AttributionKeyIDSnapshot       string     `json:"-"`
	PrincipalPriceModeSnapshot     string     `json:"principal_price_mode"`
	PrincipalPriceCentsSnapshot    *int64     `json:"principal_price_cents,omitempty"`
	PrincipalCurrencySnapshot      string     `json:"principal_currency"`
	DemandTopic                    string     `json:"demand_topic"`
	RegionCode                     string     `json:"region_code,omitempty"`
	BudgetBand                     string     `json:"budget_band"`
	Urgency                        string     `json:"urgency"`
	RequirementFlags               []string   `json:"requirement_flags"`
	PrincipalConsent               bool       `json:"principal_consent"`
	ConsentVersion                 string     `json:"consent_version"`
	Status                         string     `json:"status"`
	ExpiresAt                      time.Time  `json:"expires_at"`
	IntentRedactedAt               *time.Time `json:"intent_redacted_at,omitempty"`
	AuthorizationRevokedAt         *time.Time `json:"authorization_revoked_at,omitempty"`
	Replayed                       bool       `json:"-"`
	CreatedAt                      time.Time  `json:"created_at"`
	UpdatedAt                      time.Time  `json:"updated_at"`
}

type ActionTicketInput struct {
	ProviderOfferID       string
	SearchReceiptPublicID string
	DemandTopic           string
	RegionCode            string
	BudgetBand            string
	Urgency               string
	RequirementFlags      []string
	PrincipalConsent      bool
	ConsentVersion        string
	TTL                   time.Duration
}

type ProviderOutcomeInput struct {
	ActionTicketID string
	IdempotencyKey string
	PayloadHash    string
	Outcome        string
}

type OutcomeReceipt struct {
	ID                 string    `json:"id"`
	NHSEventID         string    `json:"nhs_event_id"`
	ProviderClaimID    string    `json:"provider_claim_id"`
	ProviderOfferID    string    `json:"provider_offer_id"`
	ActionTicketID     string    `json:"action_ticket_id"`
	ProviderAPIKeyID   int64     `json:"provider_api_key_id"`
	IdempotencyKeyHash string    `json:"-"`
	PayloadHash        string    `json:"-"`
	Outcome            string    `json:"outcome"`
	BilledCents        int64     `json:"billed_cents"`
	ChargeStatus       string    `json:"charge_status"`
	Currency           string    `json:"currency"`
	SignedReceipt      string    `json:"signed_receipt"`
	Signature          string    `json:"signature"`
	ProviderReportedAt time.Time `json:"provider_reported_at"`
	CreatedAt          time.Time `json:"created_at"`
}

// PublicOutcomeReceiptState is a provider/account-free current-state view for
// public receipt verification. Signature validity is intentionally evaluated
// separately by providerexchange; this shape reports mutable billing state.
type PublicOutcomeReceiptState struct {
	ReceiptID                   string `json:"receipt_id"`
	ActionTicketID              string `json:"action_ticket_id"`
	ReceiptOutcome              string `json:"receipt_outcome"`
	CurrentTicketStatus         string `json:"current_ticket_status"`
	OriginalChargeCredited      bool   `json:"original_charge_credited"`
	SupersededByLaterState      bool   `json:"superseded_by_later_state"`
	AuthorizationRevoked        bool   `json:"authorization_revoked"`
	NetCommercialEffectCents    int64  `json:"net_commercial_effect_cents"`
	NetCommercialEffectCurrency string `json:"net_commercial_effect_currency"`
}

type ProviderExchangeProof struct {
	OperatorRecordedProviderBudgets     int              `json:"operator_recorded_provider_budgets"`
	ProviderReportedAcceptedHandoffs    int              `json:"provider_reported_accepted_handoffs"`
	ProviderReportedActivations         int              `json:"provider_reported_activations"`
	RenewedProviderBudgets              int              `json:"renewed_provider_budgets"`
	ProviderReportedConversions         int              `json:"provider_reported_conversions"`
	PrepaidNetDebitedByCurrency         map[string]int64 `json:"prepaid_net_debited_by_currency"`
	TermsNetReceivableByCurrency        map[string]int64 `json:"terms_net_receivable_by_currency"`
	OperatorRecordedCollectedByCurrency map[string]int64 `json:"operator_recorded_collected_by_currency"`
	PilotThresholdsMet                  bool             `json:"pilot_thresholds_met"`
}

type ProviderAdminAuditEvent struct {
	ID                int64     `json:"id"`
	ProviderClaimID   string    `json:"provider_claim_id"`
	ProviderOfferID   string    `json:"provider_offer_id"`
	EventType         string    `json:"event_type"`
	OperatorReference string    `json:"operator_reference"`
	EvidenceReference string    `json:"evidence_reference"`
	PreviousStatus    string    `json:"previous_status"`
	NewStatus         string    `json:"new_status"`
	CreatedAt         time.Time `json:"created_at"`
}

func newProviderUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(b[:])
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" +
		encoded[16:20] + "-" + encoded[20:32], nil
}

func newProviderSecret(prefix string) (raw, displayPrefix string, err error) {
	var b [32]byte
	if _, err = rand.Read(b[:]); err != nil {
		return "", "", err
	}
	raw = prefix + "_" + hex.EncodeToString(b[:])
	displayPrefix = raw
	if len(displayPrefix) > 20 {
		displayPrefix = displayPrefix[:20]
	}
	return raw, displayPrefix, nil
}

func HashProviderSecret(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func providerClaimVerificationFresh(lastSucceededAt, now time.Time) bool {
	if lastSucceededAt.IsZero() || now.IsZero() {
		return false
	}
	return lastSucceededAt.After(now.UTC().Add(-ProviderClaimVerificationFreshness))
}

func providerDNSTokenMatches(tokenHash string, observedDNSTokens []string) bool {
	want := []byte(strings.TrimSpace(tokenHash))
	if !providerHashPattern.MatchString(string(want)) {
		return false
	}
	for _, raw := range observedDNSTokens {
		raw = strings.TrimSpace(raw)
		if len(raw) < 32 || len(raw) > 128 {
			continue
		}
		got := []byte(HashProviderSecret(raw))
		if len(want) == len(got) && subtle.ConstantTimeCompare(want, got) == 1 {
			return true
		}
	}
	return false
}

func validProviderUUID(value string) bool {
	return providerUUIDPattern.MatchString(strings.ToLower(strings.TrimSpace(value)))
}

func validProviderReference(value string) bool {
	return providerReferencePattern.MatchString(strings.TrimSpace(value))
}

func validProviderOpaqueValue(value string) bool {
	return providerOpaquePattern.MatchString(strings.TrimSpace(value))
}

func validProviderText(value string, maximum int) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maximum {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func scanProviderClaim(row rowScanner) (*ProviderClaim, error) {
	var claim ProviderClaim
	var verifiedAt, verificationLastSucceededAt, verificationLastAttemptedAt sql.NullTime
	var verificationNextCheckAt, revokedAt sql.NullTime
	err := row.Scan(
		&claim.ID, &claim.AccountID, &claim.SiteID, &claim.Domain,
		&claim.VerificationMethod, &claim.VerificationRecordName,
		&claim.Status, &claim.ChallengeExpiresAt, &verifiedAt,
		&verificationLastSucceededAt, &verificationLastAttemptedAt,
		&claim.VerificationConsecutiveFailures, &verificationNextCheckAt,
		&revokedAt,
		&claim.CreatedAt, &claim.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if verifiedAt.Valid {
		claim.VerifiedAt = &verifiedAt.Time
	}
	if verificationLastSucceededAt.Valid {
		claim.VerificationLastSucceededAt = &verificationLastSucceededAt.Time
	}
	if verificationLastAttemptedAt.Valid {
		claim.VerificationLastAttemptedAt = &verificationLastAttemptedAt.Time
	}
	if verificationNextCheckAt.Valid {
		claim.VerificationNextCheckAt = &verificationNextCheckAt.Time
	}
	if revokedAt.Valid {
		claim.RevokedAt = &revokedAt.Time
	}
	return &claim, nil
}

const providerClaimColumns = `
	id::text, account_id, site_id::text, domain_snapshot,
	verification_method, verification_record_name, status,
	challenge_expires_at, verified_at,
	verification_last_succeeded_at, verification_last_attempted_at,
	verification_consecutive_failures, verification_next_check_at,
	revoked_at, created_at, updated_at`

// CreateProviderClaim starts a bounded DNS TXT challenge. Multiple accounts may
// hold parallel pending challenges; pending state never reserves the site.
func CreateProviderClaim(db *sql.DB, accountID int64, siteID string) (*ProviderClaim, string, error) {
	if db == nil || accountID < 1 || !validProviderUUID(siteID) {
		return nil, "", ErrInvalidProviderExchange
	}
	rawToken, _, err := newProviderSecret("nhs_claim")
	if err != nil {
		return nil, "", err
	}
	claimID, err := newProviderUUID()
	if err != nil {
		return nil, "", err
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()

	var domain string
	if err := tx.QueryRow(`SELECT domain FROM sites WHERE id=$1::uuid FOR UPDATE`, siteID).Scan(&domain); err != nil {
		return nil, "", err
	}
	domain = NormalizeProviderDomain(domain)
	if domain == "" {
		return nil, "", ErrInvalidProviderExchange
	}
	if _, err := tx.Exec(`
		UPDATE provider_claims
		SET status='revoked', revoked_at=NOW(), updated_at=NOW()
		WHERE site_id=$1::uuid AND status='pending' AND challenge_expires_at <= NOW()`, siteID); err != nil {
		return nil, "", err
	}
	var verifiedAccountID int64
	err = tx.QueryRow(`
		SELECT account_id FROM provider_claims
		WHERE site_id=$1::uuid AND status='verified'`, siteID).Scan(&verifiedAccountID)
	if err == nil {
		if verifiedAccountID == accountID {
			return nil, "", ErrProviderClaimExists
		}
		return nil, "", ErrProviderSiteClaimed
	}
	if err != sql.ErrNoRows {
		return nil, "", err
	}
	var ownPendingExists bool
	if err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM provider_claims
			WHERE site_id=$1::uuid AND account_id=$2 AND status='pending'
		)`, siteID, accountID).Scan(&ownPendingExists); err != nil {
		return nil, "", err
	}
	if ownPendingExists {
		return nil, "", ErrProviderClaimExists
	}

	expiresAt := time.Now().UTC().Add(ProviderClaimChallengeTTL)
	claim, err := scanProviderClaim(tx.QueryRow(`
		INSERT INTO provider_claims (
			id, account_id, site_id, domain_snapshot,
			verification_record_name, verification_token_hash,
			challenge_expires_at
		) VALUES ($1::uuid,$2,$3::uuid,$4,$5,$6,$7)
		RETURNING `+providerClaimColumns,
		claimID, accountID, siteID, domain, "_nothumansearch."+domain,
		HashProviderSecret(rawToken), expiresAt))
	if err != nil {
		return nil, "", err
	}
	if err := tx.Commit(); err != nil {
		return nil, "", err
	}
	return claim, rawToken, nil
}

func GetProviderClaim(db *sql.DB, accountID int64, claimID string) (*ProviderClaim, error) {
	if db == nil || accountID < 1 || !validProviderUUID(claimID) {
		return nil, ErrInvalidProviderExchange
	}
	return scanProviderClaim(db.QueryRow(`
		SELECT `+providerClaimColumns+`
		FROM provider_claims WHERE id=$1::uuid AND account_id=$2`, claimID, accountID))
}

func ListProviderClaims(db *sql.DB, accountID int64) ([]ProviderClaim, error) {
	if db == nil || accountID < 1 {
		return nil, ErrInvalidProviderExchange
	}
	rows, err := db.Query(`
		SELECT `+providerClaimColumns+`
		FROM provider_claims WHERE account_id=$1
		ORDER BY created_at DESC, id DESC`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	claims := []ProviderClaim{}
	for rows.Next() {
		claim, err := scanProviderClaim(rows)
		if err != nil {
			return nil, err
		}
		claims = append(claims, *claim)
	}
	return claims, rows.Err()
}

func RotateProviderClaimChallenge(db *sql.DB, accountID int64, claimID string) (*ProviderClaim, string, error) {
	if db == nil || accountID < 1 || !validProviderUUID(claimID) {
		return nil, "", ErrInvalidProviderExchange
	}
	rawToken, _, err := newProviderSecret("nhs_claim")
	if err != nil {
		return nil, "", err
	}
	claim, err := scanProviderClaim(db.QueryRow(`
		UPDATE provider_claims
		SET verification_token_hash=$1, challenge_expires_at=$2, updated_at=NOW()
		WHERE id=$3::uuid AND account_id=$4 AND status='pending'
		RETURNING `+providerClaimColumns,
		HashProviderSecret(rawToken), time.Now().UTC().Add(ProviderClaimChallengeTTL), claimID, accountID))
	if err != nil {
		return nil, "", err
	}
	return claim, rawToken, nil
}

// VerifyProviderClaim persists verification only after the caller has obtained
// observedDNSToken from the claim's DNS TXT record. Supplying the originally
// issued token without a DNS observation is not sufficient at the handler layer.
func VerifyProviderClaim(db *sql.DB, accountID int64, claimID, observedDNSToken string) (*ProviderClaim, error) {
	if db == nil || accountID < 1 || !validProviderUUID(claimID) || strings.TrimSpace(observedDNSToken) == "" {
		return nil, ErrInvalidProviderExchange
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var siteID string
	if err := tx.QueryRow(`
		SELECT site_id::text FROM provider_claims
		WHERE id=$1::uuid AND account_id=$2`, claimID, accountID).Scan(&siteID); err != nil {
		return nil, err
	}
	// The site lock makes DNS verification a first-valid-proof-wins operation.
	if _, err := tx.Exec(`SELECT id FROM sites WHERE id=$1::uuid FOR UPDATE`, siteID); err != nil {
		return nil, err
	}
	var tokenHash, status string
	var expiresAt time.Time
	err = tx.QueryRow(`
		SELECT verification_token_hash, status, challenge_expires_at
		FROM provider_claims
		WHERE id=$1::uuid AND account_id=$2
		FOR UPDATE`, claimID, accountID).Scan(&tokenHash, &status, &expiresAt)
	if err != nil {
		return nil, err
	}
	checkedAt := time.Now().UTC()
	if status == "verified" {
		if !providerDNSTokenMatches(tokenHash, []string{observedDNSToken}) {
			return nil, ErrProviderChallengeMismatch
		}
		claim, err := scanProviderClaim(tx.QueryRow(`
			UPDATE provider_claims
			SET verification_last_succeeded_at=$1,
			    verification_last_attempted_at=$1,
			    verification_consecutive_failures=0,
			    verification_next_check_at=$2,
			    verification_lease_id=NULL,
			    verification_lease_until=NULL,
			    updated_at=$1
			WHERE id=$3::uuid AND status='verified'
			RETURNING `+providerClaimColumns,
			checkedAt, checkedAt.Add(ProviderClaimDNSRecheckInterval), claimID))
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return claim, nil
	}
	if status != "pending" {
		var winningClaimID string
		if err := tx.QueryRow(`
			SELECT id::text FROM provider_claims
			WHERE site_id=$1::uuid AND status='verified'`, siteID).Scan(&winningClaimID); err == nil {
			return nil, ErrProviderSiteClaimed
		} else if err != sql.ErrNoRows {
			return nil, err
		}
		return nil, ErrProviderClaimNotVerified
	}
	if !expiresAt.After(checkedAt) {
		if _, err := tx.Exec(`
			UPDATE provider_claims SET status='revoked', revoked_at=NOW(), updated_at=NOW()
			WHERE id=$1::uuid`, claimID); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return nil, ErrProviderChallengeExpired
	}
	if !providerDNSTokenMatches(tokenHash, []string{observedDNSToken}) {
		return nil, ErrProviderChallengeMismatch
	}
	var winningClaimID string
	err = tx.QueryRow(`
		SELECT id::text FROM provider_claims
		WHERE site_id=$1::uuid AND status='verified'`, siteID).Scan(&winningClaimID)
	if err == nil && winningClaimID != claimID {
		if _, err := tx.Exec(`
			UPDATE provider_claims
			SET status='revoked', revoked_at=NOW(), updated_at=NOW()
			WHERE id=$1::uuid AND status='pending'`, claimID); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return nil, ErrProviderSiteClaimed
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	claim, err := scanProviderClaim(tx.QueryRow(`
		UPDATE provider_claims
		SET status='verified', verified_at=$1,
		    verification_last_succeeded_at=$1,
		    verification_last_attempted_at=$1,
		    verification_consecutive_failures=0,
		    verification_next_check_at=$2,
		    verification_lease_id=NULL,
		    verification_lease_until=NULL,
		    updated_at=$1
		WHERE id=$3::uuid AND status='pending'
		RETURNING `+providerClaimColumns,
		checkedAt, checkedAt.Add(ProviderClaimDNSRecheckInterval), claimID))
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`
		UPDATE provider_claims
		SET status='revoked', revoked_at=NOW(), updated_at=NOW()
		WHERE site_id=$1::uuid AND id <> $2::uuid AND status='pending'`, siteID, claimID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return claim, nil
}

func RevokeProviderClaim(db *sql.DB, accountID int64, claimID string) error {
	if db == nil || accountID < 1 || !validProviderUUID(claimID) {
		return ErrInvalidProviderExchange
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err := tx.QueryRow(`
		SELECT status FROM provider_claims
		WHERE id=$1::uuid AND account_id=$2 FOR UPDATE`, claimID, accountID).Scan(&status); err != nil {
		return err
	}
	if status == "revoked" {
		return sql.ErrNoRows
	}
	if err := revokeProviderClaimTx(tx, claimID, time.Now().UTC()); err != nil {
		return err
	}
	return tx.Commit()
}

// revokeProviderClaimTx is the single atomic revocation boundary used by owner
// revocation and DNS-freshness enforcement. Its explicit timestamp keeps the
// claim and formerly active key bound for the narrow charged-ticket credit path.
func revokeProviderClaimTx(tx *sql.Tx, claimID string, revokedAt time.Time) error {
	if tx == nil || !validProviderUUID(claimID) || revokedAt.IsZero() {
		return ErrInvalidProviderExchange
	}
	revokedAt = revokedAt.UTC()
	result, err := tx.Exec(`
		UPDATE provider_claims
		SET status='revoked', revoked_at=COALESCE(revoked_at,$2),
		    verification_next_check_at=NULL,
		    verification_lease_id=NULL,
		    verification_lease_until=NULL,
		    updated_at=$2
		WHERE id=$1::uuid AND status <> 'revoked'`, claimID, revokedAt)
	if err != nil {
		return err
	}
	if changed, err := result.RowsAffected(); err != nil {
		return err
	} else if changed == 0 {
		return sql.ErrNoRows
	}
	if _, err := tx.Exec(`
		UPDATE provider_api_keys
		SET status='revoked', revoked_at=COALESCE(revoked_at,$2), updated_at=$2
		WHERE provider_claim_id=$1::uuid AND status='active'`, claimID, revokedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE provider_offers
		SET status='paused', paused_at=COALESCE(paused_at,$2), updated_at=$2
		WHERE provider_claim_id=$1::uuid AND status='active'`, claimID, revokedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE action_tickets
		SET authorization_revoked_at=COALESCE(authorization_revoked_at,$2),
		    updated_at=$2
		WHERE provider_claim_id=$1::uuid
		  AND authorization_revoked_at IS NULL
		  AND expires_at > $2
		  AND status IN ('created','redirected','accepted','activated','converted')`, claimID, revokedAt); err != nil {
		return err
	}
	return nil
}

// LeaseDueProviderClaimDNSChecks uses row locks plus SKIP LOCKED and a unique
// per-acquisition lease UUID so multiple server instances can safely run the
// verifier. A crashed worker simply leaves a short lease that another instance
// may acquire after expiry.
func LeaseDueProviderClaimDNSChecks(db *sql.DB, checkedAt time.Time, limit int) ([]ProviderClaimDNSLease, error) {
	if db == nil || checkedAt.IsZero() || limit < 1 || limit > ProviderClaimDNSMaximumBatch {
		return nil, ErrInvalidProviderExchange
	}
	checkedAt = checkedAt.UTC()
	leaseUntil := checkedAt.Add(ProviderClaimDNSLeaseDuration)
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.Query(`
		WITH due AS (
			SELECT id
			FROM provider_claims
			WHERE status='verified'
			  AND verification_next_check_at <= $1
			  AND (verification_lease_until IS NULL OR verification_lease_until <= $1)
			ORDER BY verification_next_check_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE provider_claims claim
		SET verification_lease_id=uuid_generate_v4(),
		    verification_lease_until=$3
		FROM due
		WHERE claim.id=due.id
		RETURNING claim.id::text, claim.verification_lease_id::text,
		          claim.verification_record_name, claim.verification_lease_until`,
		checkedAt, limit, leaseUntil)
	if err != nil {
		return nil, err
	}
	leases := make([]ProviderClaimDNSLease, 0, limit)
	for rows.Next() {
		var lease ProviderClaimDNSLease
		if err := rows.Scan(&lease.ClaimID, &lease.LeaseID, &lease.VerificationRecordName, &lease.LeaseUntil); err != nil {
			rows.Close()
			return nil, err
		}
		leases = append(leases, lease)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return leases, nil
}

// CompleteProviderClaimDNSCheck accepts transient DNS candidates but performs
// the match against the persisted hash inside the locked transaction. Raw TXT
// token values are never written to a table or returned in the result.
func CompleteProviderClaimDNSCheck(
	db *sql.DB,
	claimID, leaseID string,
	observedDNSTokens []string,
	checkedAt time.Time,
) (*ProviderClaimDNSCheckResult, error) {
	if db == nil || !validProviderUUID(claimID) || !validProviderUUID(leaseID) ||
		checkedAt.IsZero() || len(observedDNSTokens) > 8 {
		return nil, ErrInvalidProviderExchange
	}
	checkedAt = checkedAt.UTC()
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var tokenHash, status string
	var lastSucceededAt sql.NullTime
	var consecutiveFailures int
	err = tx.QueryRow(`
		SELECT verification_token_hash, status,
		       verification_last_succeeded_at,
		       verification_consecutive_failures
		FROM provider_claims
		WHERE id=$1::uuid AND verification_lease_id=$2::uuid
		  AND verification_lease_until > $3
		FOR UPDATE`, claimID, leaseID, checkedAt).
		Scan(&tokenHash, &status, &lastSucceededAt, &consecutiveFailures)
	if err == sql.ErrNoRows {
		return nil, ErrProviderDNSLeaseLost
	}
	if err != nil {
		return nil, err
	}
	if status != "verified" {
		return nil, ErrProviderDNSLeaseLost
	}

	matched := providerDNSTokenMatches(tokenHash, observedDNSTokens)
	result := &ProviderClaimDNSCheckResult{ClaimID: claimID, Matched: matched}
	if matched {
		if _, err := tx.Exec(`
			UPDATE provider_claims
			SET verification_last_succeeded_at=$1,
			    verification_last_attempted_at=$1,
			    verification_consecutive_failures=0,
			    verification_next_check_at=$2,
			    verification_lease_id=NULL,
			    verification_lease_until=NULL,
			    updated_at=$1
			WHERE id=$3::uuid AND status='verified'`,
			checkedAt, checkedAt.Add(ProviderClaimDNSRecheckInterval), claimID); err != nil {
			return nil, err
		}
		result.VerificationLastSucceededAt = &checkedAt
	} else {
		lastSuccess := time.Time{}
		if lastSucceededAt.Valid {
			lastSuccess = lastSucceededAt.Time
			value := lastSucceededAt.Time
			result.VerificationLastSucceededAt = &value
		}
		result.ConsecutiveFailures = min(consecutiveFailures+1, ProviderClaimDNSFailureLimit)
		result.Revoked, err = recordProviderClaimDNSFailureTx(
			tx, claimID, checkedAt, lastSuccess, result.ConsecutiveFailures,
		)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

// RecordProviderClaimDNSFailure records a real owner-triggered repeat lookup
// failure. Pending challenges have no prior proof and therefore do not use the
// verified-claim failure counter.
func RecordProviderClaimDNSFailure(db *sql.DB, accountID int64, claimID string, checkedAt time.Time) (*ProviderClaimDNSCheckResult, error) {
	if db == nil || accountID < 1 || !validProviderUUID(claimID) || checkedAt.IsZero() {
		return nil, ErrInvalidProviderExchange
	}
	checkedAt = checkedAt.UTC()
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var status string
	var lastSucceededAt sql.NullTime
	var consecutiveFailures int
	if err := tx.QueryRow(`
		SELECT status, verification_last_succeeded_at,
		       verification_consecutive_failures
		FROM provider_claims
		WHERE id=$1::uuid AND account_id=$2
		FOR UPDATE`, claimID, accountID).
		Scan(&status, &lastSucceededAt, &consecutiveFailures); err != nil {
		return nil, err
	}
	if status != "verified" {
		return nil, ErrProviderClaimNotVerified
	}
	lastSuccess := time.Time{}
	result := &ProviderClaimDNSCheckResult{
		ClaimID:             claimID,
		ConsecutiveFailures: min(consecutiveFailures+1, ProviderClaimDNSFailureLimit),
	}
	if lastSucceededAt.Valid {
		lastSuccess = lastSucceededAt.Time
		value := lastSucceededAt.Time
		result.VerificationLastSucceededAt = &value
	}
	result.Revoked, err = recordProviderClaimDNSFailureTx(
		tx, claimID, checkedAt, lastSuccess, result.ConsecutiveFailures,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func recordProviderClaimDNSFailureTx(
	tx *sql.Tx,
	claimID string,
	checkedAt, lastSucceededAt time.Time,
	consecutiveFailures int,
) (bool, error) {
	if tx == nil || !validProviderUUID(claimID) || checkedAt.IsZero() ||
		consecutiveFailures < 1 || consecutiveFailures > ProviderClaimDNSFailureLimit {
		return false, ErrInvalidProviderExchange
	}
	revoke := consecutiveFailures >= ProviderClaimDNSFailureLimit ||
		!providerClaimVerificationFresh(lastSucceededAt, checkedAt)
	nextCheckAt := checkedAt.Add(ProviderClaimDNSFailureRetry)
	if _, err := tx.Exec(`
		UPDATE provider_claims
		SET verification_last_attempted_at=$1,
		    verification_consecutive_failures=$2,
		    verification_next_check_at=CASE WHEN $3 THEN verification_next_check_at ELSE $4 END,
		    verification_lease_id=NULL,
		    verification_lease_until=NULL,
		    updated_at=$1
		WHERE id=$5::uuid AND status='verified'`,
		checkedAt, consecutiveFailures, revoke, nextCheckAt, claimID); err != nil {
		return false, err
	}
	if !revoke {
		return false, nil
	}
	if err := revokeProviderClaimTx(tx, claimID, checkedAt); err != nil {
		return false, err
	}
	return true, nil
}

func scanProviderAPIKey(row rowScanner) (*ProviderAPIKey, error) {
	var key ProviderAPIKey
	var lastUsed sql.NullTime
	err := row.Scan(&key.ID, &key.ProviderClaimID, &key.KeyPrefix, &key.Status, &lastUsed, &key.CreatedAt)
	if err != nil {
		return nil, err
	}
	if lastUsed.Valid {
		key.LastUsedAt = &lastUsed.Time
	}
	return &key, nil
}

func CreateProviderAPIKey(db *sql.DB, accountID int64, claimID string) (string, *ProviderAPIKey, error) {
	if db == nil || accountID < 1 || !validProviderUUID(claimID) {
		return "", nil, ErrInvalidProviderExchange
	}
	raw, prefix, err := newProviderSecret("nhs_provider")
	if err != nil {
		return "", nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback()
	var status string
	var lastSucceededAt time.Time
	if err := tx.QueryRow(`
		SELECT status, verification_last_succeeded_at FROM provider_claims
		WHERE id=$1::uuid AND account_id=$2 FOR UPDATE`, claimID, accountID).
		Scan(&status, &lastSucceededAt); err != nil {
		return "", nil, err
	}
	if status != "verified" {
		return "", nil, ErrProviderClaimNotVerified
	}
	if !providerClaimVerificationFresh(lastSucceededAt, time.Now().UTC()) {
		return "", nil, ErrProviderClaimVerificationStale
	}
	var activeKeyExists bool
	if err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM provider_api_keys
			WHERE provider_claim_id=$1::uuid AND status='active'
		)`, claimID).Scan(&activeKeyExists); err != nil {
		return "", nil, err
	}
	if activeKeyExists {
		return "", nil, ErrProviderAPIKeyExists
	}
	key, err := scanProviderAPIKey(tx.QueryRow(`
		INSERT INTO provider_api_keys (provider_claim_id, key_hash, key_prefix)
		VALUES ($1::uuid,$2,$3)
		RETURNING id, provider_claim_id::text, key_prefix, status, last_used_at, created_at`,
		claimID, HashProviderSecret(raw), prefix))
	if err != nil {
		return "", nil, err
	}
	if err := tx.Commit(); err != nil {
		return "", nil, err
	}
	return raw, key, nil
}

// RotateProviderAPIKey atomically revokes the current key and returns one new
// raw key. The claim row lock and partial unique index enforce one active key.
func RotateProviderAPIKey(db *sql.DB, accountID int64, claimID string) (string, *ProviderAPIKey, error) {
	if db == nil || accountID < 1 || !validProviderUUID(claimID) {
		return "", nil, ErrInvalidProviderExchange
	}
	raw, prefix, err := newProviderSecret("nhs_provider")
	if err != nil {
		return "", nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback()
	var status string
	var lastSucceededAt time.Time
	if err := tx.QueryRow(`
		SELECT status, verification_last_succeeded_at FROM provider_claims
		WHERE id=$1::uuid AND account_id=$2 FOR UPDATE`, claimID, accountID).
		Scan(&status, &lastSucceededAt); err != nil {
		return "", nil, err
	}
	if status != "verified" {
		return "", nil, ErrProviderClaimNotVerified
	}
	if !providerClaimVerificationFresh(lastSucceededAt, time.Now().UTC()) {
		return "", nil, ErrProviderClaimVerificationStale
	}
	if _, err := tx.Exec(`
		UPDATE provider_api_keys
		SET status='revoked', revoked_at=NOW(), updated_at=NOW()
		WHERE provider_claim_id=$1::uuid AND status='active'`, claimID); err != nil {
		return "", nil, err
	}
	key, err := scanProviderAPIKey(tx.QueryRow(`
		INSERT INTO provider_api_keys (provider_claim_id, key_hash, key_prefix)
		VALUES ($1::uuid,$2,$3)
		RETURNING id, provider_claim_id::text, key_prefix, status, last_used_at, created_at`,
		claimID, HashProviderSecret(raw), prefix))
	if err != nil {
		return "", nil, err
	}
	if err := tx.Commit(); err != nil {
		return "", nil, err
	}
	return raw, key, nil
}

func ResolveProviderAPIKey(db *sql.DB, raw string) (*ProviderAPIKey, error) {
	if db == nil || strings.TrimSpace(raw) == "" {
		return nil, sql.ErrNoRows
	}
	return scanProviderAPIKey(db.QueryRow(`
		SELECT key.id, key.provider_claim_id::text, key.key_prefix,
		       key.status, key.last_used_at, key.created_at
		FROM provider_api_keys key
		JOIN provider_claims claim ON claim.id=key.provider_claim_id
		WHERE key.key_hash=$1 AND key.status='active' AND claim.status='verified'
		  AND claim.verification_last_succeeded_at >
		      NOW() - $2::bigint * INTERVAL '1 second'`,
		HashProviderSecret(raw), int64(ProviderClaimVerificationFreshness/time.Second)))
}

// ResolveProviderAPIKeyForChargeResolution is the only callback-key recovery
// path after an owner revokes a provider claim. It binds the exact raw key that
// was active and revoked atomically with that claim to an exact already-charged
// ticket. Earlier rotated/revoked keys cannot use this recovery path. It grants
// no general provider access; callers may use it only with RecordProviderOutcome
// for invalid/duplicate resolution.
func ResolveProviderAPIKeyForChargeResolution(db *sql.DB, raw, ticketID string) (*ProviderAPIKey, error) {
	if db == nil || strings.TrimSpace(raw) == "" || !validProviderUUID(ticketID) {
		return nil, sql.ErrNoRows
	}
	return scanProviderAPIKey(db.QueryRow(`
		SELECT key.id, key.provider_claim_id::text, key.key_prefix,
		       key.status, key.last_used_at, key.created_at
		FROM provider_api_keys key
		JOIN provider_claims claim ON claim.id=key.provider_claim_id
		JOIN action_tickets ticket
		  ON ticket.id=$2::uuid AND ticket.provider_claim_id=key.provider_claim_id
		JOIN outcome_receipts charged
		  ON charged.action_ticket_id=ticket.id
		 AND charged.charge_status='charged'
		WHERE key.key_hash=$1
		  AND (
			(key.status='active' AND claim.status='verified') OR
			(key.status='revoked' AND claim.status='revoked'
			 AND key.revoked_at=claim.revoked_at)
		  )`, HashProviderSecret(raw), ticketID))
}

func RevokeProviderAPIKey(db *sql.DB, accountID, keyID int64) error {
	if db == nil || accountID < 1 || keyID < 1 {
		return ErrInvalidProviderExchange
	}
	result, err := db.Exec(`
		UPDATE provider_api_keys key
		SET status='revoked', revoked_at=COALESCE(key.revoked_at,NOW()), updated_at=NOW()
		FROM provider_claims claim
		WHERE key.id=$1 AND claim.id=key.provider_claim_id
		  AND claim.account_id=$2 AND key.status='active'`, keyID, accountID)
	if err != nil {
		return err
	}
	if changed, err := result.RowsAffected(); err != nil {
		return err
	} else if changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func normalizeProviderOfferInput(input ProviderOfferInput) (ProviderOfferInput, error) {
	input.OfferName = strings.TrimSpace(input.OfferName)
	input.OfferSummary = strings.TrimSpace(input.OfferSummary)
	input.ActionType = strings.ToLower(strings.TrimSpace(input.ActionType))
	input.ActionURL = strings.TrimSpace(input.ActionURL)
	input.ChargeEvent = strings.ToLower(strings.TrimSpace(input.ChargeEvent))
	input.Currency = strings.ToLower(strings.TrimSpace(input.Currency))
	input.PrincipalPriceMode = strings.ToLower(strings.TrimSpace(input.PrincipalPriceMode))
	input.PrincipalCurrency = strings.ToLower(strings.TrimSpace(input.PrincipalCurrency))
	input.BillingMode = strings.ToLower(strings.TrimSpace(input.BillingMode))
	input.TermsEvidenceReference = strings.TrimSpace(input.TermsEvidenceReference)

	if !validProviderText(input.OfferName, 80) || !validProviderText(input.OfferSummary, 280) ||
		!providerActionTypes[input.ActionType] || !providerChargeEvents[input.ChargeEvent] ||
		input.BountyCents < 1 || input.BountyCents > ProviderBountyMaximumCents ||
		input.Currency != "usd" || input.PrincipalCurrency != "usd" ||
		(input.BillingMode != "prepaid" && input.BillingMode != "terms") {
		return ProviderOfferInput{}, ErrInvalidProviderExchange
	}
	parsed, err := url.Parse(input.ActionURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Hostname() == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return ProviderOfferInput{}, ErrInvalidProviderExchange
	}
	input.ActionURL = parsed.String()
	if len(input.ActionURL) > providerexchange.ActionURLBaseMaximumBytes {
		return ProviderOfferInput{}, ErrInvalidProviderExchange
	}
	switch input.PrincipalPriceMode {
	case "free":
		if input.PrincipalPriceCents == nil || *input.PrincipalPriceCents != 0 {
			return ProviderOfferInput{}, ErrInvalidProviderExchange
		}
	case "fixed":
		if input.PrincipalPriceCents == nil || *input.PrincipalPriceCents < 1 ||
			*input.PrincipalPriceCents > ProviderPrincipalPriceMaximumCents {
			return ProviderOfferInput{}, ErrInvalidProviderExchange
		}
	case "quote", "provider_pricing":
		if input.PrincipalPriceCents != nil {
			return ProviderOfferInput{}, ErrInvalidProviderExchange
		}
	default:
		return ProviderOfferInput{}, ErrInvalidProviderExchange
	}
	if input.TermsEvidenceReference != "" && !validProviderReference(input.TermsEvidenceReference) {
		return ProviderOfferInput{}, ErrInvalidProviderExchange
	}
	switch input.BillingMode {
	case "prepaid":
		if input.TermsCreditLimitCents != nil || input.TermsPeriodDays != nil {
			return ProviderOfferInput{}, ErrInvalidProviderExchange
		}
	case "terms":
		if input.TermsCreditLimitCents == nil || *input.TermsCreditLimitCents < 1 ||
			*input.TermsCreditLimitCents > ProviderTermsCreditMaximumCents ||
			input.TermsPeriodDays == nil || *input.TermsPeriodDays < 1 || *input.TermsPeriodDays > 90 {
			return ProviderOfferInput{}, ErrInvalidProviderExchange
		}
	}
	return input, nil
}

func scanProviderOffer(row rowScanner) (*ProviderOffer, error) {
	var offer ProviderOffer
	var principalPrice, termsCreditLimit, termsPeriodDays sql.NullInt64
	var termsPeriodAnchor, activatedAt, pausedAt sql.NullTime
	err := row.Scan(
		&offer.ID, &offer.ProviderClaimID, &offer.SiteID, &offer.Domain,
		&offer.Status, &offer.Version, &offer.OfferName, &offer.OfferSummary,
		&offer.ActionType, &offer.ActionURL, &offer.DisclosureLabel,
		&offer.ChargeEvent, &offer.BountyCents, &offer.Currency,
		&offer.PrincipalPriceMode, &principalPrice, &offer.PrincipalCurrency,
		&offer.BillingMode, &termsCreditLimit, &termsPeriodDays,
		&termsPeriodAnchor, &offer.TermsEvidenceReference,
		&offer.BudgetBalanceCents, &activatedAt, &pausedAt,
		&offer.CreatedAt, &offer.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if principalPrice.Valid {
		value := principalPrice.Int64
		offer.PrincipalPriceCents = &value
	}
	if termsCreditLimit.Valid {
		value := termsCreditLimit.Int64
		offer.TermsCreditLimitCents = &value
	}
	if termsPeriodDays.Valid {
		value := int(termsPeriodDays.Int64)
		offer.TermsPeriodDays = &value
	}
	if termsPeriodAnchor.Valid {
		offer.TermsPeriodAnchorAt = &termsPeriodAnchor.Time
	}
	if activatedAt.Valid {
		offer.ActivatedAt = &activatedAt.Time
	}
	if pausedAt.Valid {
		offer.PausedAt = &pausedAt.Time
	}
	return &offer, nil
}

const providerOfferSelectColumns = `
	offer.id::text, offer.provider_claim_id::text, claim.site_id::text,
	claim.domain_snapshot, offer.status, offer.version, offer.offer_name, offer.offer_summary,
	offer.action_type, offer.action_url, offer.disclosure_label,
	offer.charge_event, offer.bounty_cents, offer.currency,
	offer.principal_price_mode, offer.principal_price_cents,
	offer.principal_currency, offer.billing_mode,
	offer.terms_credit_limit_cents, offer.terms_period_days,
	offer.terms_period_anchor_at,
	offer.terms_evidence_reference,
	COALESCE((SELECT SUM(ledger.amount_cents)
	          FROM provider_budget_ledger ledger
	          WHERE ledger.provider_offer_id=offer.id),0)::bigint,
	offer.activated_at, offer.paused_at, offer.created_at, offer.updated_at`

func getProviderOfferTx(tx *sql.Tx, offerID string) (*ProviderOffer, error) {
	return scanProviderOffer(tx.QueryRow(`
		SELECT `+providerOfferSelectColumns+`
		FROM provider_offers offer
		JOIN provider_claims claim ON claim.id=offer.provider_claim_id
		WHERE offer.id=$1::uuid`, offerID))
}

func CreateProviderOffer(db *sql.DB, accountID int64, claimID string, input ProviderOfferInput) (*ProviderOffer, error) {
	if db == nil || accountID < 1 || !validProviderUUID(claimID) {
		return nil, ErrInvalidProviderExchange
	}
	input, err := normalizeProviderOfferInput(input)
	if err != nil {
		return nil, err
	}
	offerID, err := newProviderUUID()
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var claimStatus string
	var lastSucceededAt time.Time
	if err := tx.QueryRow(`
		SELECT status, verification_last_succeeded_at FROM provider_claims
		WHERE id=$1::uuid AND account_id=$2 FOR UPDATE`, claimID, accountID).
		Scan(&claimStatus, &lastSucceededAt); err != nil {
		return nil, err
	}
	if claimStatus != "verified" {
		return nil, ErrProviderClaimNotVerified
	}
	if !providerClaimVerificationFresh(lastSucceededAt, time.Now().UTC()) {
		return nil, ErrProviderClaimVerificationStale
	}
	var liveOfferCount int
	if err := tx.QueryRow(`
		SELECT COUNT(*)::int FROM provider_offers
		WHERE provider_claim_id=$1::uuid AND status IN ('draft','active')`, claimID).
		Scan(&liveOfferCount); err != nil {
		return nil, err
	}
	if liveOfferCount >= ProviderOfferMaximumPerClaim {
		return nil, ErrProviderOfferLimit
	}
	if _, err := tx.Exec(`
		INSERT INTO provider_offers (
			id, provider_claim_id, offer_name, offer_summary, action_type,
			action_url, charge_event, bounty_cents, currency,
			principal_price_mode, principal_price_cents, principal_currency,
			billing_mode, terms_credit_limit_cents, terms_period_days,
			terms_evidence_reference
		) VALUES (
			$1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16
		)`,
		offerID, claimID, input.OfferName, input.OfferSummary, input.ActionType,
		input.ActionURL, input.ChargeEvent, input.BountyCents, input.Currency,
		input.PrincipalPriceMode, input.PrincipalPriceCents, input.PrincipalCurrency,
		input.BillingMode, input.TermsCreditLimitCents, input.TermsPeriodDays,
		input.TermsEvidenceReference); err != nil {
		return nil, err
	}
	offer, err := getProviderOfferTx(tx, offerID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return offer, nil
}

// UpdateProviderOffer edits only a never-activated provider-owned draft. Once
// commercial terms have activated, they are immutable so historic ledger cents
// can never be reinterpreted under a different bounty, currency, or charge event.
func UpdateProviderOffer(db *sql.DB, accountID int64, offerID string, input ProviderOfferInput) (*ProviderOffer, error) {
	if db == nil || accountID < 1 || !validProviderUUID(offerID) {
		return nil, ErrInvalidProviderExchange
	}
	input, err := normalizeProviderOfferInput(input)
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := lockVerifiedProviderOfferClaim(tx, offerID); err != nil {
		return nil, err
	}
	var currentStatus string
	var hasLedgerEntries bool
	if err := tx.QueryRow(`
		SELECT offer.status, EXISTS(
			SELECT 1 FROM provider_budget_ledger ledger
			WHERE ledger.provider_offer_id=offer.id
		)
		FROM provider_offers offer
		JOIN provider_claims claim ON claim.id=offer.provider_claim_id
		WHERE offer.id=$1::uuid AND claim.account_id=$2
		FOR UPDATE OF offer`, offerID, accountID).Scan(&currentStatus, &hasLedgerEntries); err != nil {
		return nil, err
	}
	if currentStatus != "draft" || hasLedgerEntries {
		return nil, errors.New("activated provider offer terms are immutable; create a new draft")
	}
	if _, err := tx.Exec(`
		UPDATE provider_offers SET
			offer_name=$1, offer_summary=$2, action_type=$3, action_url=$4,
			charge_event=$5, bounty_cents=$6, currency=$7,
			principal_price_mode=$8, principal_price_cents=$9,
			principal_currency=$10, billing_mode=$11,
			terms_credit_limit_cents=$12, terms_period_days=$13,
			terms_evidence_reference=$14, version=version+1, updated_at=NOW()
		WHERE id=$15::uuid`,
		input.OfferName, input.OfferSummary, input.ActionType, input.ActionURL,
		input.ChargeEvent, input.BountyCents, input.Currency,
		input.PrincipalPriceMode, input.PrincipalPriceCents, input.PrincipalCurrency,
		input.BillingMode, input.TermsCreditLimitCents, input.TermsPeriodDays,
		input.TermsEvidenceReference, offerID); err != nil {
		return nil, err
	}
	offer, err := getProviderOfferTx(tx, offerID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return offer, nil
}

func GetProviderOffer(db *sql.DB, accountID int64, offerID string) (*ProviderOffer, error) {
	if db == nil || accountID < 1 || !validProviderUUID(offerID) {
		return nil, ErrInvalidProviderExchange
	}
	return scanProviderOffer(db.QueryRow(`
		SELECT `+providerOfferSelectColumns+`
		FROM provider_offers offer
		JOIN provider_claims claim ON claim.id=offer.provider_claim_id
		WHERE offer.id=$1::uuid AND claim.account_id=$2`, offerID, accountID))
}

// ListProviderOffers is tenant-scoped by both owning account and claim.
func ListProviderOffers(db *sql.DB, accountID int64, claimID string) ([]ProviderOffer, error) {
	if db == nil || accountID < 1 || !validProviderUUID(claimID) {
		return nil, ErrInvalidProviderExchange
	}
	rows, err := db.Query(`
		SELECT `+providerOfferSelectColumns+`
		FROM provider_offers offer
		JOIN provider_claims claim ON claim.id=offer.provider_claim_id
		WHERE claim.id=$1::uuid AND claim.account_id=$2
		ORDER BY offer.created_at DESC, offer.id DESC`, claimID, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	offers := []ProviderOffer{}
	for rows.Next() {
		offer, err := scanProviderOffer(rows)
		if err != nil {
			return nil, err
		}
		offers = append(offers, *offer)
	}
	return offers, rows.Err()
}

func PauseProviderOffer(db *sql.DB, accountID int64, offerID string) (*ProviderOffer, error) {
	if db == nil || accountID < 1 || !validProviderUUID(offerID) {
		return nil, ErrInvalidProviderExchange
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`
		UPDATE provider_offers offer
		SET status='paused', paused_at=NOW(), updated_at=NOW()
		FROM provider_claims claim
		WHERE offer.id=$1::uuid AND claim.id=offer.provider_claim_id
		  AND claim.account_id=$2 AND offer.status='active'`, offerID, accountID)
	if err != nil {
		return nil, err
	}
	if changed, err := result.RowsAffected(); err != nil {
		return nil, err
	} else if changed == 0 {
		return nil, sql.ErrNoRows
	}
	offer, err := getProviderOfferTx(tx, offerID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return offer, nil
}

// AdminPauseProviderOffer is the cross-tenant emergency stop for a misleading
// or compromised sponsored action. It revokes every still-live ticket
// authorization without editing organic discovery. Tickets already charged
// may still report invalid/duplicate so the atomic credit path remains open.
func AdminPauseProviderOffer(db *sql.DB, offerID, operatorReference, evidenceReference string) (*ProviderOffer, error) {
	operatorReference = strings.TrimSpace(operatorReference)
	evidenceReference = strings.TrimSpace(evidenceReference)
	if db == nil || !validProviderUUID(offerID) ||
		!validProviderReference(operatorReference) || !validProviderReference(evidenceReference) {
		return nil, ErrInvalidProviderExchange
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := lockProviderOffer(tx, offerID); err != nil {
		return nil, err
	}
	var claimID, previousStatus string
	if err := tx.QueryRow(`
		SELECT provider_claim_id::text, status FROM provider_offers
		WHERE id=$1::uuid FOR UPDATE`, offerID).Scan(&claimID, &previousStatus); err != nil {
		return nil, err
	}
	if previousStatus != "active" && previousStatus != "paused" {
		return nil, sql.ErrNoRows
	}
	var emergencyPauseExists bool
	if err := tx.QueryRow(`
		SELECT COALESCE((
			SELECT event_type='emergency_pause'
			FROM provider_admin_audit_events
			WHERE provider_offer_id=$1::uuid
			ORDER BY created_at DESC, id DESC LIMIT 1
		),false)`, offerID).Scan(&emergencyPauseExists); err != nil {
		return nil, err
	}
	if emergencyPauseExists {
		return nil, sql.ErrNoRows
	}
	result, err := tx.Exec(`
		UPDATE provider_offers
		SET status='paused', paused_at=COALESCE(paused_at,NOW()), updated_at=NOW()
		WHERE id=$1::uuid AND status IN ('active','paused')`, offerID)
	if err != nil {
		return nil, err
	}
	if changed, err := result.RowsAffected(); err != nil {
		return nil, err
	} else if changed == 0 {
		return nil, sql.ErrNoRows
	}
	if _, err := tx.Exec(`
		UPDATE action_tickets
		SET authorization_revoked_at=COALESCE(authorization_revoked_at,NOW()),
		    updated_at=NOW()
		WHERE provider_offer_id=$1::uuid
		  AND authorization_revoked_at IS NULL
		  AND expires_at > NOW()
		  AND status IN ('created','redirected','accepted','activated','converted')`, offerID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`
		INSERT INTO provider_admin_audit_events (
			provider_claim_id, provider_offer_id, event_type,
			operator_reference, evidence_reference, previous_status, new_status
		) VALUES ($1::uuid,$2::uuid,'emergency_pause',$3,$4,$5,'paused')`,
		claimID, offerID, operatorReference, evidenceReference, previousStatus); err != nil {
		return nil, err
	}
	offer, err := getProviderOfferTx(tx, offerID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return offer, nil
}

func ListProviderAdminAuditEvents(db *sql.DB, offerID string, limit int) ([]ProviderAdminAuditEvent, error) {
	if db == nil || !validProviderUUID(offerID) {
		return nil, ErrInvalidProviderExchange
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := db.Query(`
		SELECT id, provider_claim_id::text, provider_offer_id::text,
		       event_type, operator_reference, evidence_reference,
		       previous_status, new_status, created_at
		FROM provider_admin_audit_events
		WHERE provider_offer_id=$1::uuid
		ORDER BY created_at DESC, id DESC
		LIMIT $2`, offerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []ProviderAdminAuditEvent{}
	for rows.Next() {
		var event ProviderAdminAuditEvent
		if err := rows.Scan(
			&event.ID, &event.ProviderClaimID, &event.ProviderOfferID,
			&event.EventType, &event.OperatorReference, &event.EvidenceReference,
			&event.PreviousStatus, &event.NewStatus, &event.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func lockProviderOffer(tx *sql.Tx, offerID string) error {
	_, err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, providerOfferLockNamespace, offerID)
	return err
}

// providerDatabaseClock is the only clock used for capacity authorization.
// Callers acquire the offer advisory lock before using it, so ticket expiry,
// budget mutation, terms-period math, and provider outcomes share one database
// time domain and one serialized capacity boundary.
func providerDatabaseClock(tx *sql.Tx) (time.Time, error) {
	var now time.Time
	if err := tx.QueryRow(`SELECT clock_timestamp()`).Scan(&now); err != nil {
		return time.Time{}, err
	}
	return now.UTC(), nil
}

func lockVerifiedProviderOfferClaim(tx *sql.Tx, offerID string) (string, error) {
	var claimID, status string
	var lastSucceededAt, databaseNow time.Time
	if err := tx.QueryRow(`
		SELECT provider_claim_id::text FROM provider_offers
		WHERE id=$1::uuid`, offerID).Scan(&claimID); err != nil {
		return "", err
	}
	if err := tx.QueryRow(`
		SELECT status, verification_last_succeeded_at, clock_timestamp()
		FROM provider_claims
		WHERE id=$1::uuid FOR UPDATE`, claimID).
		Scan(&status, &lastSucceededAt, &databaseNow); err != nil {
		return "", err
	}
	if status != "verified" {
		return "", ErrProviderClaimNotVerified
	}
	if !providerClaimVerificationFresh(lastSucceededAt, databaseNow.UTC()) {
		return "", ErrProviderClaimVerificationStale
	}
	return claimID, nil
}

func providerOfferBalance(tx *sql.Tx, offerID string) (int64, error) {
	var raw string
	if err := tx.QueryRow(`
		SELECT COALESCE(SUM(amount_cents::numeric),0)::text
		FROM provider_budget_ledger WHERE provider_offer_id=$1::uuid`, offerID).Scan(&raw); err != nil {
		return 0, err
	}
	return parseBoundedProviderMoney(raw)
}

// providerActiveReservedCapacity returns bounty capacity promised to live,
// uncharged tickets. Reserve/consume/release events are append-only; expiry and
// emergency revocation release capacity logically even before a cleanup event.
// Every caller already holds the offer advisory lock, so this value is
// serialized with ticket creation, outcomes, and operator budget changes.
func providerActiveReservedCapacity(tx *sql.Tx, offerID string) (int64, error) {
	now, err := providerDatabaseClock(tx)
	if err != nil {
		return 0, err
	}
	var raw string
	if err := tx.QueryRow(`
		SELECT COALESCE(SUM(reserve.amount_cents::numeric),0)::text
		FROM provider_capacity_events reserve
		JOIN action_tickets ticket ON ticket.id=reserve.action_ticket_id
		WHERE reserve.provider_offer_id=$1::uuid
		  AND reserve.event_type='reserve'
		  AND ticket.expires_at > $2
		  AND ticket.authorization_revoked_at IS NULL
		  AND ticket.status IN ('created','redirected','accepted','activated')
		  AND NOT EXISTS (
			SELECT 1 FROM provider_capacity_events terminal
			WHERE terminal.action_ticket_id=reserve.action_ticket_id
			  AND terminal.event_type IN ('consume','release')
		  )`, offerID, now).Scan(&raw); err != nil {
		return 0, err
	}
	reserved, err := parseBoundedProviderMoney(raw)
	if err != nil || reserved < 0 {
		return 0, ErrProviderBudgetLimit
	}
	return reserved, nil
}

func reserveProviderTicketCapacity(
	tx *sql.Tx,
	claimID, offerID, ticketID string,
	amountCents int64,
	currency string,
	createdAt time.Time,
) error {
	if amountCents < 1 || amountCents > ProviderBountyMaximumCents || currency != "usd" {
		return ErrInvalidProviderExchange
	}
	result, err := tx.Exec(`
		INSERT INTO provider_capacity_events (
			provider_claim_id, provider_offer_id, action_ticket_id,
			event_type, event_reason, amount_cents, currency, created_at
		) VALUES ($1::uuid,$2::uuid,$3::uuid,'reserve','ticket_created',$4,$5,$6)`,
		claimID, offerID, ticketID, amountCents, currency, createdAt)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 1 {
		return nil
	}
	return errors.New("provider ticket capacity reservation was not created")
}

func settleProviderTicketCapacity(
	tx *sql.Tx,
	claimID, offerID, ticketID, eventType string,
	amountCents int64,
	currency string,
	createdAt time.Time,
) error {
	reason := ""
	switch eventType {
	case "consume":
		reason = "charge_recorded"
	case "release":
		reason = "terminal_without_charge"
	default:
		return ErrInvalidProviderExchange
	}
	result, err := tx.Exec(`
		INSERT INTO provider_capacity_events (
			provider_claim_id, provider_offer_id, action_ticket_id,
			event_type, event_reason, amount_cents, currency, created_at
		)
		SELECT $1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8
		WHERE EXISTS (
			SELECT 1 FROM provider_capacity_events reserve
			WHERE reserve.action_ticket_id=$3::uuid
			  AND reserve.provider_claim_id=$1::uuid
			  AND reserve.provider_offer_id=$2::uuid
			  AND reserve.event_type='reserve'
			  AND reserve.amount_cents=$6 AND reserve.currency=$7
		)
		  AND NOT EXISTS (
			SELECT 1 FROM provider_capacity_events terminal
			WHERE terminal.action_ticket_id=$3::uuid
			  AND terminal.event_type IN ('consume','release')
		)`, claimID, offerID, ticketID, eventType, reason, amountCents, currency, createdAt)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return errors.New("provider ticket capacity reservation is unavailable")
	}
	return nil
}

// providerOutstandingCreditExposure reserves headroom for every charged ticket
// that can still receive its one allowed invalid/duplicate credit. Offer-row
// locking serializes this value with funding, charging, and crediting writes.
func providerOutstandingCreditExposure(tx *sql.Tx, offerID string) (int64, error) {
	var raw string
	if err := tx.QueryRow(`
		SELECT COALESCE(SUM((-charge.amount_cents)::numeric),0)::text
		FROM provider_budget_ledger charge
		WHERE charge.provider_offer_id=$1::uuid
		  AND charge.entry_type='charge'
		  AND NOT EXISTS (
			SELECT 1 FROM provider_budget_ledger credit
			WHERE credit.action_ticket_id=charge.action_ticket_id
			  AND credit.entry_type='credit'
		  )`, offerID).Scan(&raw); err != nil {
		return 0, err
	}
	exposure, err := parseProviderMoney(raw)
	if err != nil || exposure < 0 || exposure > ProviderMoneyMaximumCents {
		return 0, ErrProviderBudgetLimit
	}
	return exposure, nil
}

func parseBoundedProviderMoney(raw string) (int64, error) {
	value, err := parseProviderMoney(raw)
	if err != nil || value < -ProviderMoneyMaximumCents || value > ProviderMoneyMaximumCents {
		return 0, ErrProviderBudgetLimit
	}
	return value, nil
}

// parseProviderMoney protects platform-wide aggregates from BIGINT conversion
// overflow without applying the per-offer ledger cap to sums across providers.
func parseProviderMoney(raw string) (int64, error) {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, ErrProviderBudgetLimit
	}
	return value, nil
}

func ensureTermsCreditCapacity(tx *sql.Tx, offerID string, creditLimit int64, periodDays int, periodAnchor time.Time, additionalCents int64) error {
	if creditLimit < 1 || creditLimit > ProviderTermsCreditMaximumCents ||
		periodDays < 1 || periodDays > 90 || periodAnchor.IsZero() ||
		additionalCents < 1 || additionalCents > ProviderBountyMaximumCents || additionalCents > creditLimit {
		return ErrInvalidProviderExchange
	}
	now, err := providerDatabaseClock(tx)
	if err != nil {
		return err
	}
	period := time.Duration(periodDays) * 24 * time.Hour
	periodStart := periodAnchor
	if now.After(periodAnchor) {
		periodStart = periodAnchor.Add(time.Duration(int64(now.Sub(periodAnchor)/period)) * period)
	}
	var outstandingRaw, periodReceivableRaw string
	err = tx.QueryRow(`
		SELECT
			GREATEST(-COALESCE(SUM(amount_cents::numeric),0),0)::text,
			GREATEST(-COALESCE(SUM(amount_cents::numeric) FILTER (
				WHERE entry_type IN ('charge','credit') AND created_at >= $2
			),0),0)::text
		FROM provider_budget_ledger
		WHERE provider_offer_id=$1::uuid`, offerID, periodStart).
		Scan(&outstandingRaw, &periodReceivableRaw)
	if err != nil {
		return err
	}
	outstanding, err := parseBoundedProviderMoney(outstandingRaw)
	if err != nil {
		return err
	}
	periodReceivable, err := parseBoundedProviderMoney(periodReceivableRaw)
	if err != nil {
		return err
	}
	reserved, err := providerActiveReservedCapacity(tx, offerID)
	if err != nil {
		return err
	}
	if reserved > creditLimit-additionalCents ||
		outstanding > creditLimit-additionalCents-reserved ||
		periodReceivable > creditLimit-additionalCents-reserved {
		return ErrProviderTermsCreditLimit
	}
	return nil
}

// ActivateProviderOffer is an admin boundary: evidenceReference must identify
// the exact non-secret budget/CPA terms reviewed by the operator. Prepaid offers
// cannot activate without at least one bounty of cleared ledger balance.
func ActivateProviderOffer(db *sql.DB, offerID, operatorReference, evidenceReference string) (*ProviderOffer, error) {
	operatorReference = strings.TrimSpace(operatorReference)
	evidenceReference = strings.TrimSpace(evidenceReference)
	if db == nil || !validProviderUUID(offerID) ||
		!validProviderReference(operatorReference) || !validProviderReference(evidenceReference) {
		return nil, ErrInvalidProviderExchange
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	claimID, err := lockVerifiedProviderOfferClaim(tx, offerID)
	if err != nil {
		return nil, err
	}
	if err := lockProviderOffer(tx, offerID); err != nil {
		return nil, err
	}
	var offerStatus, billingMode, actionType string
	var bounty int64
	err = tx.QueryRow(`
		SELECT offer.status, offer.billing_mode, offer.bounty_cents, offer.action_type
		FROM provider_offers offer
		WHERE offer.id=$1::uuid
		FOR UPDATE OF offer`, offerID).Scan(&offerStatus, &billingMode, &bounty, &actionType)
	if err != nil {
		return nil, err
	}
	if offerStatus != "draft" && offerStatus != "paused" {
		return nil, errors.New("provider offer is not activatable")
	}
	var otherLiveOfferCount, activeActionOfferCount int
	if err := tx.QueryRow(`
		SELECT
			COUNT(*) FILTER (WHERE status IN ('draft','active'))::int,
			COUNT(*) FILTER (WHERE status='active' AND action_type=$2)::int
		FROM provider_offers
		WHERE provider_claim_id=$1::uuid AND id<>$3::uuid`, claimID, actionType, offerID).
		Scan(&otherLiveOfferCount, &activeActionOfferCount); err != nil {
		return nil, err
	}
	if otherLiveOfferCount >= ProviderOfferMaximumPerClaim ||
		activeActionOfferCount >= ProviderActiveOfferMaximumPerAction {
		return nil, ErrProviderOfferLimit
	}
	if billingMode == "prepaid" {
		balance, err := providerOfferBalance(tx, offerID)
		if err != nil {
			return nil, err
		}
		if balance < bounty {
			return nil, ErrInsufficientProviderFunds
		}
	}
	if _, err := tx.Exec(`
		UPDATE provider_offers
		SET status='active', terms_evidence_reference=$1,
		    terms_period_anchor_at=CASE
		        WHEN billing_mode='terms' THEN COALESCE(terms_period_anchor_at,NOW())
		        ELSE NULL
		    END,
		    activated_at=COALESCE(activated_at,NOW()), paused_at=NULL, updated_at=NOW()
		WHERE id=$2::uuid`, evidenceReference, offerID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`
		INSERT INTO provider_admin_audit_events (
			provider_claim_id, provider_offer_id, event_type,
			operator_reference, evidence_reference, previous_status, new_status
		) VALUES ($1::uuid,$2::uuid,'activate',$3,$4,$5,'active')`,
		claimID, offerID, operatorReference, evidenceReference, offerStatus); err != nil {
		return nil, err
	}
	offer, err := getProviderOfferTx(tx, offerID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return offer, nil
}

func recordProviderBudgetEntry(db *sql.DB, offerID, entryType string, amountCents int64, currency, externalReference string) (*ProviderBudgetEntry, error) {
	currency = strings.ToLower(strings.TrimSpace(currency))
	externalReference = strings.TrimSpace(externalReference)
	if db == nil || !validProviderUUID(offerID) ||
		currency != "usd" ||
		!validProviderReference(externalReference) {
		return nil, ErrInvalidProviderExchange
	}
	if amountCents < -ProviderMoneyMaximumCents || amountCents > ProviderMoneyMaximumCents ||
		(entryType == "fund" && amountCents < 1) || (entryType == "adjustment" && amountCents == 0) {
		return nil, ErrInvalidProviderExchange
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	lockedClaimID, err := lockVerifiedProviderOfferClaim(tx, offerID)
	if err != nil {
		return nil, err
	}
	if err := lockProviderOffer(tx, offerID); err != nil {
		return nil, err
	}
	var claimID, offerCurrency, billingMode string
	var termsCreditLimit sql.NullInt64
	if err := tx.QueryRow(`
		SELECT offer.provider_claim_id::text, offer.currency, offer.billing_mode,
		       offer.terms_credit_limit_cents
		FROM provider_offers offer
		WHERE offer.id=$1::uuid FOR UPDATE OF offer`, offerID).
		Scan(&claimID, &offerCurrency, &billingMode, &termsCreditLimit); err != nil {
		return nil, err
	}
	if claimID != lockedClaimID {
		return nil, errors.New("provider offer claim changed during budget operation")
	}
	if offerCurrency != currency {
		return nil, errors.New("provider budget currency does not match offer")
	}
	entry := &ProviderBudgetEntry{
		ProviderClaimID: claimID, ProviderOfferID: offerID, EntryType: entryType,
		AmountCents: amountCents, Currency: currency, ExternalReference: externalReference,
	}
	var existingOfferID string
	if entryType == "fund" {
		if _, err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`,
			"nhs-provider-funding-reference", externalReference); err != nil {
			return nil, err
		}
		err = tx.QueryRow(`
			SELECT id, provider_offer_id::text, amount_cents, created_at
			FROM provider_budget_ledger
			WHERE entry_type='fund' AND external_reference=$1`, externalReference).
			Scan(&entry.ID, &existingOfferID, &entry.AmountCents, &entry.CreatedAt)
	} else {
		err = tx.QueryRow(`
			SELECT id, provider_offer_id::text, amount_cents, created_at
			FROM provider_budget_ledger
			WHERE provider_offer_id=$1::uuid AND entry_type=$2 AND external_reference=$3`,
			offerID, entryType, externalReference).
			Scan(&entry.ID, &existingOfferID, &entry.AmountCents, &entry.CreatedAt)
	}
	if err == nil {
		if existingOfferID != offerID || entry.AmountCents != amountCents {
			return nil, ErrProviderIdempotency
		}
		entry.Replayed = true
		entry.BalanceAfterCents, err = providerOfferBalance(tx, offerID)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return entry, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	balance, err := providerOfferBalance(tx, offerID)
	if err != nil {
		return nil, err
	}
	proposedBalance := balance + amountCents
	if proposedBalance < -ProviderMoneyMaximumCents || proposedBalance > ProviderMoneyMaximumCents {
		return nil, ErrProviderBudgetLimit
	}
	if amountCents > 0 {
		creditExposure, err := providerOutstandingCreditExposure(tx, offerID)
		if err != nil {
			return nil, err
		}
		if creditExposure > ProviderMoneyMaximumCents-proposedBalance {
			return nil, ErrProviderBudgetLimit
		}
	}
	reserved, err := providerActiveReservedCapacity(tx, offerID)
	if err != nil {
		return nil, err
	}
	if billingMode == "prepaid" {
		if proposedBalance < 0 || reserved > proposedBalance {
			return nil, ErrInsufficientProviderFunds
		}
	}
	if billingMode == "terms" {
		if !termsCreditLimit.Valid || reserved > termsCreditLimit.Int64 {
			return nil, ErrProviderTermsCreditLimit
		}
		outstanding := int64(0)
		if proposedBalance < 0 {
			outstanding = -proposedBalance
		}
		if outstanding > termsCreditLimit.Int64-reserved {
			return nil, ErrProviderTermsCreditLimit
		}
	}
	if err := tx.QueryRow(`
		INSERT INTO provider_budget_ledger (
			provider_claim_id, provider_offer_id, entry_type,
			amount_cents, currency, external_reference
		) VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6)
		RETURNING id, created_at`,
		claimID, offerID, entryType, amountCents, currency, externalReference).
		Scan(&entry.ID, &entry.CreatedAt); err != nil {
		return nil, err
	}
	entry.BalanceAfterCents, err = providerOfferBalance(tx, offerID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return entry, nil
}

func FundProviderOffer(db *sql.DB, offerID string, amountCents int64, currency, externalReference string) (*ProviderBudgetEntry, error) {
	return recordProviderBudgetEntry(db, offerID, "fund", amountCents, currency, externalReference)
}

func AdjustProviderOfferBudget(db *sql.DB, offerID string, amountCents int64, currency, externalReference string) (*ProviderBudgetEntry, error) {
	return recordProviderBudgetEntry(db, offerID, "adjustment", amountCents, currency, externalReference)
}

// ListPublicProviderOffersForOrganicResults is a separate paid-offer lookup. It
// accepts only the already-computed organic result list and preserves that list's
// position; it never modifies search scores, membership, or organic ordering.
func ListPublicProviderOffersForOrganicResults(db *sql.DB, organicSites []Site) ([]PublicProviderOffer, error) {
	if db == nil || len(organicSites) == 0 {
		return []PublicProviderOffer{}, nil
	}
	ids := make([]string, 0, len(organicSites))
	domains := make([]string, 0, len(organicSites))
	seen := map[string]bool{}
	for _, site := range organicSites {
		id := strings.ToLower(strings.TrimSpace(site.ID))
		domain := NormalizeProviderDomain(site.Domain)
		if validProviderUUID(id) && domain != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
			domains = append(domains, domain)
		}
	}
	if len(ids) == 0 {
		return []PublicProviderOffer{}, nil
	}
	rows, err := db.Query(`
		WITH organic(site_id, domain, organic_position) AS (
			SELECT item.site_id::uuid, item.domain, item.ordinality::integer
			FROM unnest($1::text[], $2::text[]) WITH ORDINALITY AS item(site_id, domain, ordinality)
		)
		SELECT `+providerOfferSelectColumns+`, organic.organic_position
		FROM organic
			JOIN provider_claims claim
			  ON claim.site_id=organic.site_id
			 AND claim.domain_snapshot=organic.domain
			 AND claim.status='verified'
			 AND claim.verification_last_succeeded_at >
			     NOW() - $3::bigint * INTERVAL '1 second'
		JOIN provider_offers offer ON offer.provider_claim_id=claim.id AND offer.status='active'
		LEFT JOIN LATERAL (
			SELECT COALESCE(SUM(reserve.amount_cents::numeric),0) AS reserved_cents
			FROM provider_capacity_events reserve
			JOIN action_tickets reserved_ticket
			  ON reserved_ticket.id=reserve.action_ticket_id
			WHERE reserve.provider_offer_id=offer.id
			  AND reserve.event_type='reserve'
			  AND reserved_ticket.expires_at > NOW()
			  AND reserved_ticket.authorization_revoked_at IS NULL
			  AND reserved_ticket.status IN ('created','redirected','accepted','activated')
			  AND NOT EXISTS (
				SELECT 1 FROM provider_capacity_events terminal
				WHERE terminal.action_ticket_id=reserve.action_ticket_id
				  AND terminal.event_type IN ('consume','release')
			  )
		) capacity ON TRUE
		WHERE (
			offer.billing_mode='prepaid'
			AND COALESCE((
				SELECT SUM(ledger.amount_cents::numeric)
				FROM provider_budget_ledger ledger
				WHERE ledger.provider_offer_id=offer.id
			),0) >= offer.bounty_cents + capacity.reserved_cents
		) OR (
			offer.billing_mode='terms'
			AND offer.terms_credit_limit_cents >= offer.bounty_cents
			AND GREATEST(-COALESCE((
				SELECT SUM(ledger.amount_cents::numeric)
				FROM provider_budget_ledger ledger
				WHERE ledger.provider_offer_id=offer.id
			),0),0) <= offer.terms_credit_limit_cents-offer.bounty_cents-capacity.reserved_cents
			AND GREATEST(-COALESCE((
				SELECT SUM(ledger.amount_cents::numeric)
				FROM provider_budget_ledger ledger
				WHERE ledger.provider_offer_id=offer.id
				  AND ledger.entry_type IN ('charge','credit')
				  AND ledger.created_at >= offer.terms_period_anchor_at + make_interval(
					secs => (GREATEST(FLOOR(
						EXTRACT(EPOCH FROM (NOW()-offer.terms_period_anchor_at)) /
						(offer.terms_period_days*86400)
					),0)*(offer.terms_period_days*86400))::double precision
				  )
			),0),0) <= offer.terms_credit_limit_cents-offer.bounty_cents-capacity.reserved_cents
		)
			ORDER BY organic.organic_position, offer.action_type, offer.id`,
		pq.Array(ids), pq.Array(domains), int64(ProviderClaimVerificationFreshness/time.Second))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	offers := []PublicProviderOffer{}
	for rows.Next() {
		offer, err := scanProviderOfferWithPosition(rows)
		if err != nil {
			return nil, err
		}
		offers = append(offers, PublicProviderOffer{
			OfferID: offer.ID, OfferVersion: offer.Version,
			SiteID: offer.SiteID, Domain: offer.Domain,
			OfferName: offer.OfferName, OfferSummary: offer.OfferSummary,
			ActionType: offer.ActionType, ChargeEvent: offer.ChargeEvent,
			DisclosureLabel:           offer.DisclosureLabel,
			ProviderFundedBountyCents: offer.BountyCents,
			ProviderFundedCurrency:    offer.Currency,
			PrincipalPriceMode:        offer.PrincipalPriceMode,
			PrincipalPriceCents:       offer.PrincipalPriceCents,
			PrincipalCurrency:         offer.PrincipalCurrency,
			OrganicPosition:           offer.OrganicPosition,
		})
	}
	return offers, rows.Err()
}

// RecordProviderOffersReturned commits the exact paid offer/version evidence
// before a REST or MCP response discloses it. Every row is independently tied
// to a non-synthetic receipt and the already-returned organic site/domain.
func RecordProviderOffersReturned(db *sql.DB, searchPublicID string, offers []PublicProviderOffer) error {
	searchPublicID = strings.TrimSpace(searchPublicID)
	if db == nil || !strings.HasPrefix(searchPublicID, "nhs_sr_") {
		return ErrInvalidProviderExchange
	}
	if len(offers) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, offer := range offers {
		offerID := strings.ToLower(strings.TrimSpace(offer.OfferID))
		if !validProviderUUID(offerID) || offer.OfferVersion < 1 {
			return ErrInvalidProviderExchange
		}
		if _, err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`,
			"nhs-provider-offer-returned:"+searchPublicID, offerID); err != nil {
			return err
		}
		result, err := tx.Exec(`
			INSERT INTO provider_offers_returned (
				search_receipt_id, provider_offer_id, provider_claim_id,
				offer_version_snapshot, offer_name_snapshot, action_type_snapshot,
				disclosure_snapshot, bounty_cents_snapshot, currency_snapshot,
				charge_event_snapshot
			)
			SELECT receipt.id, offer.id, offer.provider_claim_id,
			       offer.version, offer.offer_name, offer.action_type,
			       offer.disclosure_label, offer.bounty_cents, offer.currency,
			       offer.charge_event
			FROM search_receipts receipt
			JOIN provider_offers offer ON offer.id=$2::uuid
			JOIN provider_claims claim
			  ON claim.id=offer.provider_claim_id
			 AND claim.status='verified'
			 AND claim.verification_last_succeeded_at >
			     NOW() - $4::bigint * INTERVAL '1 second'
			JOIN organic_results_returned organic
			  ON organic.search_receipt_id=receipt.id
			 AND organic.site_id=claim.site_id
			 AND organic.site_domain_snapshot=claim.domain_snapshot
			WHERE receipt.public_id=$1 AND NOT receipt.is_synthetic
			  AND offer.version=$3 AND offer.status='active'
			  AND NOT EXISTS (
				SELECT 1 FROM provider_offers_returned existing
				WHERE existing.search_receipt_id=receipt.id
				  AND existing.provider_offer_id=offer.id
			  )`,
			searchPublicID, offerID, offer.OfferVersion,
			int64(ProviderClaimVerificationFreshness/time.Second))
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed == 0 {
			var exact bool
			if err := tx.QueryRow(`
				SELECT EXISTS(
					SELECT 1 FROM provider_offers_returned returned
					JOIN search_receipts receipt ON receipt.id=returned.search_receipt_id
					WHERE receipt.public_id=$1
					  AND returned.provider_offer_id=$2::uuid
					  AND returned.offer_version_snapshot=$3
				)`, searchPublicID, offerID, offer.OfferVersion).Scan(&exact); err != nil {
				return err
			}
			if !exact {
				return ErrProviderOfferNotPublic
			}
		}
	}
	return tx.Commit()
}

func scanProviderOfferWithPosition(row rowScanner) (*ProviderOffer, error) {
	var offer ProviderOffer
	var principalPrice, termsCreditLimit, termsPeriodDays sql.NullInt64
	var termsPeriodAnchor, activatedAt, pausedAt sql.NullTime
	err := row.Scan(
		&offer.ID, &offer.ProviderClaimID, &offer.SiteID, &offer.Domain,
		&offer.Status, &offer.Version, &offer.OfferName, &offer.OfferSummary,
		&offer.ActionType, &offer.ActionURL, &offer.DisclosureLabel,
		&offer.ChargeEvent, &offer.BountyCents, &offer.Currency,
		&offer.PrincipalPriceMode, &principalPrice, &offer.PrincipalCurrency,
		&offer.BillingMode, &termsCreditLimit, &termsPeriodDays,
		&termsPeriodAnchor, &offer.TermsEvidenceReference,
		&offer.BudgetBalanceCents, &activatedAt, &pausedAt,
		&offer.CreatedAt, &offer.UpdatedAt, &offer.OrganicPosition,
	)
	if err != nil {
		return nil, err
	}
	if principalPrice.Valid {
		value := principalPrice.Int64
		offer.PrincipalPriceCents = &value
	}
	if termsCreditLimit.Valid {
		value := termsCreditLimit.Int64
		offer.TermsCreditLimitCents = &value
	}
	if termsPeriodDays.Valid {
		value := int(termsPeriodDays.Int64)
		offer.TermsPeriodDays = &value
	}
	if termsPeriodAnchor.Valid {
		offer.TermsPeriodAnchorAt = &termsPeriodAnchor.Time
	}
	if activatedAt.Valid {
		offer.ActivatedAt = &activatedAt.Time
	}
	if pausedAt.Valid {
		offer.PausedAt = &pausedAt.Time
	}
	return &offer, nil
}

func normalizeActionTicketInput(input ActionTicketInput) (ActionTicketInput, error) {
	input.ProviderOfferID = strings.ToLower(strings.TrimSpace(input.ProviderOfferID))
	input.SearchReceiptPublicID = strings.TrimSpace(input.SearchReceiptPublicID)
	input.DemandTopic = strings.ToLower(strings.TrimSpace(input.DemandTopic))
	input.RegionCode = strings.ToUpper(strings.TrimSpace(input.RegionCode))
	input.BudgetBand = strings.ToLower(strings.TrimSpace(input.BudgetBand))
	input.Urgency = strings.ToLower(strings.TrimSpace(input.Urgency))
	input.ConsentVersion = strings.TrimSpace(input.ConsentVersion)
	if input.BudgetBand == "" {
		input.BudgetBand = "unspecified"
	}
	if input.Urgency == "" {
		input.Urgency = "unspecified"
	}
	if input.TTL == 0 {
		input.TTL = ActionTicketDefaultTTL
	}
	input.TTL = input.TTL.Truncate(time.Second)
	if !validProviderUUID(input.ProviderOfferID) ||
		!strings.HasPrefix(input.SearchReceiptPublicID, "nhs_sr_") ||
		len(input.SearchReceiptPublicID) > 100 ||
		!regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(input.SearchReceiptPublicID) ||
		!providerDemandTopics[input.DemandTopic] ||
		(input.RegionCode != "" && !providerRegionPattern.MatchString(input.RegionCode)) ||
		!providerBudgetBands[input.BudgetBand] || !providerUrgencies[input.Urgency] ||
		!input.PrincipalConsent || input.ConsentVersion != ProviderPrincipalConsentV1 ||
		input.TTL < time.Hour || input.TTL > ActionTicketMaximumTTL {
		return ActionTicketInput{}, ErrInvalidProviderExchange
	}
	seen := map[string]bool{}
	flags := make([]string, 0, len(input.RequirementFlags))
	for _, raw := range input.RequirementFlags {
		flag := strings.ToLower(strings.TrimSpace(raw))
		if !providerRequirementFlags[flag] {
			return ActionTicketInput{}, ErrInvalidProviderExchange
		}
		if !seen[flag] {
			seen[flag] = true
			flags = append(flags, flag)
		}
	}
	if len(flags) > 8 {
		return ActionTicketInput{}, ErrInvalidProviderExchange
	}
	sort.Strings(flags)
	input.RequirementFlags = flags
	return input, nil
}

func scanActionTicket(row rowScanner) (*ActionTicket, error) {
	var ticket ActionTicket
	var flags pq.StringArray
	var searchReceiptID sql.NullString
	var principalPrice, termsCreditLimit, termsPeriodDays sql.NullInt64
	var termsPeriodAnchor, redactedAt, authorizationRevokedAt sql.NullTime
	err := row.Scan(
		&ticket.ID, &ticket.ProviderClaimID, &ticket.ProviderOfferID,
		&searchReceiptID, &ticket.SourceIsSynthetic, &ticket.TokenHash,
		&ticket.TokenNonce, &ticket.CreationRequestHash,
		&ticket.OfferVersionSnapshot, &ticket.OfferNameSnapshot,
		&ticket.OfferSummarySnapshot, &ticket.ActionTypeSnapshot,
		&ticket.ActionURLSnapshot, &ticket.DisclosureSnapshot,
		&ticket.ChargeEventSnapshot, &ticket.BountyCentsSnapshot,
		&ticket.CurrencySnapshot, &ticket.BillingModeSnapshot,
		&ticket.TermsEvidenceReferenceSnapshot,
		&termsCreditLimit, &termsPeriodDays, &termsPeriodAnchor,
		&ticket.AttributionKeyIDSnapshot,
		&ticket.PrincipalPriceModeSnapshot, &principalPrice,
		&ticket.PrincipalCurrencySnapshot,
		&ticket.DemandTopic, &ticket.RegionCode, &ticket.BudgetBand,
		&ticket.Urgency, &flags, &ticket.PrincipalConsent,
		&ticket.ConsentVersion, &ticket.Status, &ticket.ExpiresAt,
		&redactedAt, &authorizationRevokedAt, &ticket.CreatedAt, &ticket.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if searchReceiptID.Valid {
		ticket.SearchReceiptID = searchReceiptID.String
	}
	if principalPrice.Valid {
		value := principalPrice.Int64
		ticket.PrincipalPriceCentsSnapshot = &value
	}
	if termsCreditLimit.Valid {
		value := termsCreditLimit.Int64
		ticket.TermsCreditLimitCentsSnapshot = &value
	}
	if termsPeriodDays.Valid {
		value := int(termsPeriodDays.Int64)
		ticket.TermsPeriodDaysSnapshot = &value
	}
	if termsPeriodAnchor.Valid {
		ticket.TermsPeriodAnchorAtSnapshot = &termsPeriodAnchor.Time
	}
	if redactedAt.Valid {
		ticket.IntentRedactedAt = &redactedAt.Time
	}
	if authorizationRevokedAt.Valid {
		ticket.AuthorizationRevokedAt = &authorizationRevokedAt.Time
	}
	ticket.RequirementFlags = []string(flags)
	return &ticket, nil
}

const actionTicketColumns = `
	id::text, provider_claim_id::text, provider_offer_id::text,
	search_receipt_id::text, source_is_synthetic, token_hash,
	token_nonce, creation_request_hash,
	offer_version_snapshot, offer_name_snapshot, offer_summary_snapshot,
	action_type_snapshot, action_url_snapshot, disclosure_snapshot,
	charge_event_snapshot, bounty_cents_snapshot, currency_snapshot,
	billing_mode_snapshot, terms_evidence_reference_snapshot,
	terms_credit_limit_cents_snapshot, terms_period_days_snapshot,
	terms_period_anchor_at_snapshot,
	attribution_key_id_snapshot,
	principal_price_mode_snapshot,
	principal_price_cents_snapshot, principal_currency_snapshot,
	demand_topic, region_code, budget_band, urgency, requirement_flags,
	principal_consent, consent_version, status, expires_at, intent_redacted_at,
	authorization_revoked_at,
	created_at, updated_at`

func actionTicketRequestHash(input ActionTicketInput) string {
	controlled := []string{
		input.ProviderOfferID, input.SearchReceiptPublicID, input.DemandTopic,
		input.RegionCode, input.BudgetBand, input.Urgency,
		strings.Join(input.RequirementFlags, ","), input.ConsentVersion,
		strconv.FormatInt(int64(input.TTL/time.Second), 10),
	}
	sum := sha256.Sum256([]byte(strings.Join(controlled, "\x00")))
	return hex.EncodeToString(sum[:])
}

// CreateActionTicket proves four things in one transaction: the site appeared
// in a real organic search receipt, this exact paid offer/version was returned
// beside it, the offer remains independently active, and the principal
// consented to the controlled action fields. The attribution token is signed
// after the UUID is generated and only its hash is persisted.
func CreateActionTicket(db *sql.DB, input ActionTicketInput, signer *providerexchange.Signer) (*ActionTicket, *ProviderOffer, string, error) {
	if db == nil || signer == nil {
		return nil, nil, "", ErrInvalidProviderExchange
	}
	input, err := normalizeActionTicketInput(input)
	if err != nil {
		return nil, nil, "", err
	}
	attributionKeyID := signer.ActiveKeyID()
	if !providerSigningKeyIDPattern.MatchString(attributionKeyID) {
		return nil, nil, "", ErrInvalidProviderExchange
	}
	requestHash := actionTicketRequestHash(input)
	tx, err := db.Begin()
	if err != nil {
		return nil, nil, "", err
	}
	defer tx.Rollback()
	lockedClaimID, err := lockVerifiedProviderOfferClaim(tx, input.ProviderOfferID)
	if err != nil {
		return nil, nil, "", err
	}
	if err := lockProviderOffer(tx, input.ProviderOfferID); err != nil {
		return nil, nil, "", err
	}

	var searchReceiptID string
	var sourceIsSynthetic bool
	err = tx.QueryRow(`
		SELECT receipt.id::text, receipt.is_synthetic
		FROM provider_offers offer
			JOIN provider_claims claim
			  ON claim.id=offer.provider_claim_id AND claim.status='verified'
			 AND claim.verification_last_succeeded_at >
			     clock_timestamp() - $4::bigint * INTERVAL '1 second'
		JOIN search_receipts receipt
		  ON receipt.public_id=$2 AND NOT receipt.is_synthetic
		 AND $3=ANY(receipt.demand_topics)
		JOIN provider_offers_returned returned
		  ON returned.search_receipt_id=receipt.id
		 AND returned.provider_offer_id=offer.id
		 AND returned.offer_version_snapshot=offer.version
		JOIN organic_results_returned organic
		  ON organic.search_receipt_id=receipt.id
		 AND organic.site_id=claim.site_id
		 AND organic.site_domain_snapshot=claim.domain_snapshot
		WHERE offer.id=$1::uuid AND offer.status='active'
		FOR UPDATE OF offer`, input.ProviderOfferID, input.SearchReceiptPublicID, input.DemandTopic,
		int64(ProviderClaimVerificationFreshness/time.Second)).
		Scan(&searchReceiptID, &sourceIsSynthetic)
	if err == sql.ErrNoRows {
		return nil, nil, "", ErrProviderOfferNotPublic
	}
	if err != nil {
		return nil, nil, "", err
	}
	offer, err := getProviderOfferTx(tx, input.ProviderOfferID)
	if err != nil {
		return nil, nil, "", err
	}
	if offer.ProviderClaimID != lockedClaimID {
		return nil, nil, "", errors.New("provider offer claim changed during ticket creation")
	}
	// Provider self-service pause takes the offer row lock without the advisory
	// lock. Read the authoritative clock only after the eligibility query has
	// acquired that row lock, then recheck ownership freshness at the same
	// issuance boundary.
	authorizationAt, err := providerDatabaseClock(tx)
	if err != nil {
		return nil, nil, "", err
	}
	var claimStatus string
	var verificationLastSucceededAt time.Time
	if err := tx.QueryRow(`
		SELECT status, verification_last_succeeded_at
		FROM provider_claims WHERE id=$1::uuid`, lockedClaimID).
		Scan(&claimStatus, &verificationLastSucceededAt); err != nil {
		return nil, nil, "", err
	}
	if claimStatus != "verified" {
		return nil, nil, "", ErrProviderClaimNotVerified
	}
	if !providerClaimVerificationFresh(verificationLastSucceededAt, authorizationAt) {
		return nil, nil, "", ErrProviderClaimVerificationStale
	}
	issuedAt := authorizationAt.Truncate(time.Second)
	expiresAt := issuedAt.Add(input.TTL)
	existing, err := scanActionTicket(tx.QueryRow(`
		SELECT `+actionTicketColumns+`
		FROM action_tickets
		WHERE search_receipt_id=$1::uuid AND provider_offer_id=$2::uuid`,
		searchReceiptID, input.ProviderOfferID))
	if err == nil {
		if existing.CreationRequestHash != requestHash {
			return nil, nil, "", ErrActionTicketExists
		}
		if existing.AuthorizationRevokedAt != nil {
			return nil, nil, "", ErrProviderOfferRevoked
		}
		if !existing.ExpiresAt.After(authorizationAt) {
			return nil, nil, "", ErrActionTicketExpired
		}
		var reservationMatches bool
		if err := tx.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM provider_capacity_events reserve
				WHERE reserve.action_ticket_id=$1::uuid
				  AND reserve.provider_claim_id=$2::uuid
				  AND reserve.provider_offer_id=$3::uuid
				  AND reserve.event_type='reserve'
				  AND reserve.amount_cents=$4 AND reserve.currency=$5
			)`, existing.ID, existing.ProviderClaimID, existing.ProviderOfferID,
			existing.BountyCentsSnapshot, existing.CurrencySnapshot).
			Scan(&reservationMatches); err != nil {
			return nil, nil, "", err
		}
		if !reservationMatches {
			return nil, nil, "", errors.New("provider ticket capacity reservation is missing")
		}
		claims := providerexchange.AttributionClaims{
			Version:  providerexchange.AttributionTokenVersion,
			KeyID:    existing.AttributionKeyIDSnapshot,
			TicketID: existing.ID, OfferID: existing.ProviderOfferID,
			IssuedAt: existing.CreatedAt.Unix(), ExpiresAt: existing.ExpiresAt.Unix(),
			Nonce: existing.TokenNonce,
		}
		rawToken, err := signer.SignAttribution(claims)
		if err != nil {
			return nil, nil, "", err
		}
		if _, err := providerexchange.ActionURLWithAttribution(existing.ActionURLSnapshot, rawToken); err != nil {
			return nil, nil, "", ErrInvalidProviderExchange
		}
		want, got := []byte(existing.TokenHash), []byte(HashProviderSecret(rawToken))
		if len(want) != len(got) || subtle.ConstantTimeCompare(want, got) != 1 {
			return nil, nil, "", errors.New("stored action ticket token binding mismatch")
		}
		existing.Replayed = true
		if err := tx.Commit(); err != nil {
			return nil, nil, "", err
		}
		return existing, offer, rawToken, nil
	}
	if err != sql.ErrNoRows {
		return nil, nil, "", err
	}
	if offer.BillingMode == "prepaid" {
		balance, err := providerOfferBalance(tx, input.ProviderOfferID)
		if err != nil {
			return nil, nil, "", err
		}
		reserved, err := providerActiveReservedCapacity(tx, input.ProviderOfferID)
		if err != nil {
			return nil, nil, "", err
		}
		if balance < offer.BountyCents || reserved > balance-offer.BountyCents {
			return nil, nil, "", ErrInsufficientProviderFunds
		}
	} else if offer.TermsCreditLimitCents == nil || offer.TermsPeriodDays == nil || offer.TermsPeriodAnchorAt == nil {
		return nil, nil, "", ErrInvalidProviderExchange
	} else if err := ensureTermsCreditCapacity(
		tx, offer.ID, *offer.TermsCreditLimitCents, *offer.TermsPeriodDays,
		*offer.TermsPeriodAnchorAt, offer.BountyCents,
	); err != nil {
		return nil, nil, "", err
	}
	ticketID, err := newProviderUUID()
	if err != nil {
		return nil, nil, "", err
	}
	nonce, err := providerexchange.NewNonce()
	if err != nil {
		return nil, nil, "", err
	}
	claims := providerexchange.AttributionClaims{
		Version:  providerexchange.AttributionTokenVersion,
		KeyID:    attributionKeyID,
		TicketID: ticketID, OfferID: input.ProviderOfferID,
		IssuedAt: issuedAt.Unix(), ExpiresAt: expiresAt.Unix(), Nonce: nonce,
	}
	rawToken, err := signer.SignAttribution(claims)
	if err != nil {
		return nil, nil, "", err
	}
	if _, err := providerexchange.ActionURLWithAttribution(offer.ActionURL, rawToken); err != nil {
		return nil, nil, "", ErrInvalidProviderExchange
	}
	ticket, err := scanActionTicket(tx.QueryRow(`
		INSERT INTO action_tickets (
			id, provider_claim_id, provider_offer_id, search_receipt_id,
			source_is_synthetic, token_hash, token_nonce, creation_request_hash,
			offer_version_snapshot, offer_name_snapshot, offer_summary_snapshot,
			action_type_snapshot, action_url_snapshot, disclosure_snapshot,
			charge_event_snapshot, bounty_cents_snapshot, currency_snapshot,
			billing_mode_snapshot, terms_evidence_reference_snapshot,
			terms_credit_limit_cents_snapshot, terms_period_days_snapshot,
			terms_period_anchor_at_snapshot,
			attribution_key_id_snapshot,
			principal_price_mode_snapshot,
			principal_price_cents_snapshot, principal_currency_snapshot,
			demand_topic, region_code, budget_band, urgency,
			requirement_flags, principal_consent, consent_version,
			expires_at, created_at, updated_at
		) VALUES (
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,
			$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,
			$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$35
		)
		RETURNING `+actionTicketColumns,
		ticketID, offer.ProviderClaimID, input.ProviderOfferID, searchReceiptID,
		sourceIsSynthetic, HashProviderSecret(rawToken), nonce, requestHash,
		offer.Version, offer.OfferName, offer.OfferSummary, offer.ActionType,
		offer.ActionURL, offer.DisclosureLabel, offer.ChargeEvent,
		offer.BountyCents, offer.Currency, offer.BillingMode,
		offer.TermsEvidenceReference, offer.TermsCreditLimitCents,
		offer.TermsPeriodDays, offer.TermsPeriodAnchorAt, attributionKeyID,
		offer.PrincipalPriceMode,
		offer.PrincipalPriceCents, offer.PrincipalCurrency,
		input.DemandTopic, input.RegionCode, input.BudgetBand, input.Urgency,
		pq.Array(input.RequirementFlags), input.PrincipalConsent,
		input.ConsentVersion, expiresAt, issuedAt))
	if err != nil {
		return nil, nil, "", err
	}
	if err := reserveProviderTicketCapacity(
		tx, offer.ProviderClaimID, input.ProviderOfferID, ticket.ID,
		offer.BountyCents, offer.Currency, issuedAt,
	); err != nil {
		return nil, nil, "", err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, "", err
	}
	return ticket, offer, rawToken, nil
}

func ResolveActionTicket(db *sql.DB, ticketID, rawToken string) (*ActionTicket, error) {
	if db == nil || !validProviderUUID(ticketID) || strings.TrimSpace(rawToken) == "" {
		return nil, ErrInvalidProviderExchange
	}
	return scanActionTicket(db.QueryRow(`
		SELECT `+actionTicketColumns+`
		FROM action_tickets
		WHERE id=$1::uuid AND token_hash=$2 AND expires_at > NOW()
		  AND authorization_revoked_at IS NULL
		  AND EXISTS (
			SELECT 1 FROM provider_claims claim
			WHERE claim.id=action_tickets.provider_claim_id
			  AND claim.status='verified'
			  AND claim.verification_last_succeeded_at >
			      NOW() - $3::bigint * INTERVAL '1 second'
		  )`, ticketID, HashProviderSecret(rawToken),
		int64(ProviderClaimVerificationFreshness/time.Second)))
}

// ResolveActionTicketForChargeResolution proves only that the caller holds the
// exact token bound to an already charged ticket. It intentionally ignores
// attribution expiry and emergency revocation so a provider can authenticate a
// late invalid/duplicate credit. This function does not authorize redirects,
// positive outcomes, or a credit; RecordProviderOutcome re-locks the ticket and
// enforces the existing-charge and invalid/duplicate-only resolution contract.
func ResolveActionTicketForChargeResolution(db *sql.DB, ticketID, rawToken string) (*ActionTicket, error) {
	if db == nil || !validProviderUUID(ticketID) || strings.TrimSpace(rawToken) == "" {
		return nil, ErrInvalidProviderExchange
	}
	return scanActionTicket(db.QueryRow(`
		SELECT `+actionTicketColumns+`
		FROM action_tickets ticket
		WHERE ticket.id=$1::uuid AND ticket.token_hash=$2
		  AND EXISTS (
			SELECT 1 FROM provider_budget_ledger charge
			WHERE charge.action_ticket_id=ticket.id AND charge.entry_type='charge'
		  )`, ticketID, HashProviderSecret(rawToken)))
}

func MarkActionTicketRedirected(db *sql.DB, ticketID, rawToken string) (*ActionTicket, error) {
	if db == nil || !validProviderUUID(ticketID) || strings.TrimSpace(rawToken) == "" {
		return nil, ErrInvalidProviderExchange
	}
	return scanActionTicket(db.QueryRow(`
		UPDATE action_tickets
		SET status='redirected', updated_at=NOW()
		WHERE id=$1::uuid AND token_hash=$2 AND expires_at > NOW()
		  AND authorization_revoked_at IS NULL
		  AND status IN ('created','redirected')
		  AND EXISTS (
			SELECT 1 FROM provider_claims claim
			WHERE claim.id=action_tickets.provider_claim_id
			  AND claim.status='verified'
			  AND claim.verification_last_succeeded_at >
			      NOW() - $3::bigint * INTERVAL '1 second'
		  )
		RETURNING `+actionTicketColumns, ticketID, HashProviderSecret(rawToken),
		int64(ProviderClaimVerificationFreshness/time.Second)))
}

// RedactExpiredActionTicketIntent enforces the 30-day controlled-intent
// retention boundary even when search-receipt pruning is delayed. Commercial
// IDs, consent attestation, immutable offer terms, and outcome proof remain.
func RedactExpiredActionTicketIntent(db *sql.DB) (int64, error) {
	if db == nil {
		return 0, ErrInvalidProviderExchange
	}
	result, err := db.Exec(`
		UPDATE action_tickets
		SET search_receipt_id=NULL,
		    demand_topic='redacted', region_code='',
		    budget_band='unspecified', urgency='unspecified',
		    requirement_flags='{}',
		    intent_redacted_at=COALESCE(intent_redacted_at,NOW()),
		    updated_at=NOW()
		WHERE intent_redacted_at IS NULL
		  AND created_at < NOW() - $1::bigint * INTERVAL '1 second'`,
		int64(ActionTicketIntentRetention/time.Second))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanOutcomeReceipt(row rowScanner) (*OutcomeReceipt, error) {
	var receipt OutcomeReceipt
	err := row.Scan(
		&receipt.ID, &receipt.NHSEventID, &receipt.ProviderClaimID,
		&receipt.ProviderOfferID, &receipt.ActionTicketID,
		&receipt.ProviderAPIKeyID, &receipt.IdempotencyKeyHash,
		&receipt.PayloadHash, &receipt.Outcome, &receipt.BilledCents,
		&receipt.ChargeStatus, &receipt.Currency, &receipt.SignedReceipt,
		&receipt.Signature, &receipt.ProviderReportedAt, &receipt.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

const outcomeReceiptColumns = `
	id::text, nhs_event_id::text, provider_claim_id::text,
	provider_offer_id::text, action_ticket_id::text, provider_api_key_id,
	idempotency_key_hash, payload_hash, outcome, billed_cents,
	charge_status, currency, signed_receipt, signature,
	provider_reported_at, created_at`

func validProviderOutcomeTransition(current, outcome string) bool {
	switch current {
	case "created", "redirected":
		return outcome == "accepted" || outcome == "rejected" ||
			outcome == "duplicate" || outcome == "invalid"
	case "accepted":
		return outcome == "activated" || outcome == "duplicate" || outcome == "invalid"
	case "activated":
		return outcome == "converted" || outcome == "duplicate" || outcome == "invalid"
	case "converted":
		return outcome == "duplicate" || outcome == "invalid"
	default:
		return false
	}
}

func nextActionTicketStatus(current, outcome string) string {
	if !validProviderOutcomeTransition(current, outcome) {
		return current
	}
	return outcome
}

// providerChargeResolutionAllowed is the only outcome path that may cross an
// attribution expiry or emergency authorization revocation. It is deliberately
// limited to reversing an existing charge; it can never authorize a redirect,
// positive outcome, or new debit.
func providerChargeResolutionOutcome(outcome string) bool {
	return outcome == "invalid" || outcome == "duplicate"
}

func providerChargeResolutionAllowed(outcome string, chargedCents int64) bool {
	return chargedCents > 0 && providerChargeResolutionOutcome(outcome)
}

// RecordProviderOutcome serializes one provider event, ticket, and offer before
// it checks idempotency or moves money. It atomically inserts the outcome,
// prepaid debit (or invalid/duplicate credit), signed receipt, and ticket state.
func RecordProviderOutcome(db *sql.DB, key *ProviderAPIKey, input ProviderOutcomeInput, signer *providerexchange.Signer) (*OutcomeReceipt, bool, error) {
	input.ActionTicketID = strings.ToLower(strings.TrimSpace(input.ActionTicketID))
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	input.PayloadHash = strings.ToLower(strings.TrimSpace(input.PayloadHash))
	input.Outcome = strings.ToLower(strings.TrimSpace(input.Outcome))
	if db == nil || key == nil || signer == nil || key.ID < 1 ||
		!validProviderUUID(key.ProviderClaimID) || !validProviderUUID(input.ActionTicketID) ||
		!validProviderOpaqueValue(input.IdempotencyKey) ||
		!providerHashPattern.MatchString(input.PayloadHash) || !providerOutcomeTypes[input.Outcome] {
		return nil, false, ErrInvalidProviderExchange
	}
	receiptID, err := newProviderUUID()
	if err != nil {
		return nil, false, err
	}
	nhsEventID, err := newProviderUUID()
	if err != nil {
		return nil, false, err
	}
	idempotencyHash := HashProviderSecret(input.IdempotencyKey)
	tx, err := db.Begin()
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()

	var claimStatus string
	var verificationLastSucceededAt sql.NullTime
	if err := tx.QueryRow(`
		SELECT status, verification_last_succeeded_at FROM provider_claims
		WHERE id=$1::uuid FOR UPDATE`, key.ProviderClaimID).
		Scan(&claimStatus, &verificationLastSucceededAt); err != nil {
		return nil, false, err
	}
	if claimStatus != "verified" &&
		!(claimStatus == "revoked" && providerChargeResolutionOutcome(input.Outcome)) {
		return nil, false, ErrProviderClaimNotVerified
	}

	var offerID string
	if err := tx.QueryRow(`
		SELECT provider_offer_id::text FROM action_tickets
		WHERE id=$1::uuid`, input.ActionTicketID).Scan(&offerID); err != nil {
		return nil, false, err
	}
	if err := lockProviderOffer(tx, offerID); err != nil {
		return nil, false, err
	}
	var claimID, ticketStatus, chargeEvent, billingMode, currency string
	var expiresAt time.Time
	var bountyCents int64
	var termsCreditLimit, termsPeriodDays sql.NullInt64
	var termsPeriodAnchor, authorizationRevokedAt sql.NullTime
	err = tx.QueryRow(`
		SELECT ticket.provider_claim_id::text, ticket.status,
		       ticket.expires_at, ticket.charge_event_snapshot,
		       ticket.billing_mode_snapshot, ticket.bounty_cents_snapshot,
		       ticket.currency_snapshot, ticket.terms_credit_limit_cents_snapshot,
		       ticket.terms_period_days_snapshot, ticket.terms_period_anchor_at_snapshot,
		       ticket.authorization_revoked_at
		FROM action_tickets ticket
		JOIN provider_offers offer ON offer.id=ticket.provider_offer_id
		WHERE ticket.id=$1::uuid AND offer.id=$2::uuid
		FOR UPDATE OF ticket, offer`, input.ActionTicketID, offerID).
		Scan(
			&claimID, &ticketStatus, &expiresAt, &chargeEvent, &billingMode,
			&bountyCents, &currency, &termsCreditLimit, &termsPeriodDays,
			&termsPeriodAnchor, &authorizationRevokedAt,
		)
	if err != nil {
		return nil, false, err
	}
	if claimID != key.ProviderClaimID {
		return nil, false, sql.ErrNoRows
	}
	// Decide authorization against the database wall clock only after both the
	// offer advisory lock and the ticket/offer row locks are held. A callback
	// that began before expiry but waited for either lock must not consume
	// capacity using a stale process timestamp.
	authorizationAt, err := providerDatabaseClock(tx)
	if err != nil {
		return nil, false, err
	}
	// recordedAt remains exact for authorization guards; only values written
	// into receipts and state transitions are truncated to whole seconds.
	recordedAt := authorizationAt
	receiptRecordedAt := authorizationAt.Truncate(time.Second)
	providerReportedAt := receiptRecordedAt
	freshClaim := verificationLastSucceededAt.Valid &&
		providerClaimVerificationFresh(verificationLastSucceededAt.Time, authorizationAt)
	activeClaim := claimStatus == "verified" && freshClaim
	staleClaimResolution := claimStatus == "verified" && !freshClaim &&
		providerChargeResolutionOutcome(input.Outcome)
	revokedClaimResolution := claimStatus == "revoked" && providerChargeResolutionOutcome(input.Outcome)
	if !activeClaim && !staleClaimResolution && !revokedClaimResolution {
		if claimStatus == "verified" {
			return nil, false, ErrProviderClaimVerificationStale
		}
		return nil, false, ErrProviderClaimNotVerified
	}
	var authenticatedClaimID string
	err = tx.QueryRow(`
		SELECT key.provider_claim_id::text
		FROM provider_api_keys key
			WHERE key.id=$1 AND key.provider_claim_id=$2::uuid
			  AND (
				($3::boolean AND key.status='active') OR
				($4::boolean AND key.status='active' AND EXISTS (
					SELECT 1 FROM outcome_receipts charged
					WHERE charged.action_ticket_id=$6::uuid
					  AND charged.provider_claim_id=key.provider_claim_id
					  AND charged.charge_status='charged'
				)) OR
				($5::boolean AND key.status='revoked' AND EXISTS (
					SELECT 1 FROM outcome_receipts charged
					WHERE charged.action_ticket_id=$6::uuid
					  AND charged.provider_claim_id=key.provider_claim_id
					  AND charged.charge_status='charged'
			) AND key.revoked_at=(
				SELECT claim.revoked_at FROM provider_claims claim
				WHERE claim.id=key.provider_claim_id
			))
			  )
			FOR UPDATE`, key.ID, key.ProviderClaimID, activeClaim,
		staleClaimResolution, revokedClaimResolution, input.ActionTicketID).Scan(&authenticatedClaimID)
	if err != nil {
		return nil, false, err
	}
	if authenticatedClaimID != claimID {
		return nil, false, sql.ErrNoRows
	}
	if _, err := tx.Exec(`
		SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`,
		providerIdempotencyNamespace+":"+authenticatedClaimID, idempotencyHash); err != nil {
		return nil, false, err
	}

	existing, err := scanOutcomeReceipt(tx.QueryRow(`
		SELECT `+outcomeReceiptColumns+`
		FROM outcome_receipts
		WHERE provider_claim_id=$1::uuid AND idempotency_key_hash=$2`,
		authenticatedClaimID, idempotencyHash))
	if err == nil {
		if existing.PayloadHash != input.PayloadHash {
			return nil, false, ErrProviderIdempotency
		}
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}
	if bountyCents < 1 || bountyCents > ProviderBountyMaximumCents || currency != "usd" {
		return nil, false, ErrInvalidProviderExchange
	}
	if !validProviderOutcomeTransition(ticketStatus, input.Outcome) {
		return nil, false, ErrProviderOutcomeTransition
	}
	var sameOutcomeExists bool
	if err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM outcome_receipts
			WHERE action_ticket_id=$1::uuid AND outcome=$2
		)`, input.ActionTicketID, input.Outcome).Scan(&sameOutcomeExists); err != nil {
		return nil, false, err
	}
	if sameOutcomeExists {
		return nil, false, ErrProviderOutcomeExists
	}

	var chargedCents int64
	err = tx.QueryRow(`
		SELECT -amount_cents
		FROM provider_budget_ledger
		WHERE action_ticket_id=$1::uuid AND entry_type='charge'`, input.ActionTicketID).
		Scan(&chargedCents)
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}
	if err == sql.ErrNoRows {
		chargedCents = 0
	}
	if chargedCents < 0 || chargedCents > ProviderBountyMaximumCents {
		return nil, false, ErrInvalidProviderExchange
	}
	chargeResolutionAllowed := providerChargeResolutionAllowed(input.Outcome, chargedCents)
	if !expiresAt.After(recordedAt) && !chargeResolutionAllowed {
		return nil, false, ErrActionTicketExpired
	}
	if authorizationRevokedAt.Valid && !chargeResolutionAllowed {
		return nil, false, ErrProviderOfferRevoked
	}
	billedCents := int64(0)
	chargeStatus := providerexchange.ChargeStatusNone
	if input.Outcome == chargeEvent && chargedCents == 0 {
		balance, err := providerOfferBalance(tx, offerID)
		if err != nil {
			return nil, false, err
		}
		if balance < -ProviderMoneyMaximumCents+bountyCents {
			return nil, false, ErrProviderBudgetLimit
		}
		if billingMode == "prepaid" {
			if balance < bountyCents {
				return nil, false, ErrInsufficientProviderFunds
			}
		} else {
			if !termsCreditLimit.Valid || !termsPeriodDays.Valid || !termsPeriodAnchor.Valid {
				return nil, false, ErrInvalidProviderExchange
			}
			// Terms capacity was reserved atomically when the ticket was
			// created. Consuming that reservation and inserting the charge in
			// this transaction keeps total exposure unchanged.
		}
		if _, err := tx.Exec(`
			INSERT INTO provider_budget_ledger (
				provider_claim_id, provider_offer_id, action_ticket_id,
				entry_type, amount_cents, currency, external_reference
			) VALUES ($1::uuid,$2::uuid,$3::uuid,'charge',$4,$5,$6)`,
			claimID, offerID, input.ActionTicketID, -bountyCents, currency,
			"outcome:"+nhsEventID); err != nil {
			return nil, false, err
		}
		if err := settleProviderTicketCapacity(
			tx, claimID, offerID, input.ActionTicketID, "consume",
			bountyCents, currency, receiptRecordedAt,
		); err != nil {
			return nil, false, err
		}
		chargedCents = bountyCents
		billedCents = bountyCents
		chargeStatus = providerexchange.ChargeStatusCharged
	} else if (input.Outcome == "invalid" || input.Outcome == "duplicate") && chargedCents > 0 {
		var creditExists bool
		if err := tx.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM provider_budget_ledger
				WHERE action_ticket_id=$1::uuid AND entry_type='credit'
			)`, input.ActionTicketID).Scan(&creditExists); err != nil {
			return nil, false, err
		}
		if !creditExists {
			balance, err := providerOfferBalance(tx, offerID)
			if err != nil {
				return nil, false, err
			}
			if balance > ProviderMoneyMaximumCents-chargedCents {
				return nil, false, ErrProviderBudgetLimit
			}
			if _, err := tx.Exec(`
				INSERT INTO provider_budget_ledger (
					provider_claim_id, provider_offer_id, action_ticket_id,
					entry_type, amount_cents, currency, external_reference
				) VALUES ($1::uuid,$2::uuid,$3::uuid,'credit',$4,$5,$6)`,
				claimID, offerID, input.ActionTicketID, chargedCents, currency,
				"outcome:"+nhsEventID); err != nil {
				return nil, false, err
			}
			billedCents = chargedCents
			chargeStatus = providerexchange.ChargeStatusCredited
		}
	}
	if chargedCents == 0 &&
		(input.Outcome == "rejected" || input.Outcome == "duplicate" || input.Outcome == "invalid") {
		if err := settleProviderTicketCapacity(
			tx, claimID, offerID, input.ActionTicketID, "release",
			bountyCents, currency, receiptRecordedAt,
		); err != nil {
			return nil, false, err
		}
	}

	signedReceipt, signature, err := signer.SignOutcomeReceipt(providerexchange.OutcomeReceipt{
		Version:            providerexchange.OutcomeReceiptVersion,
		ReceiptID:          receiptID,
		TicketID:           input.ActionTicketID,
		OfferID:            offerID,
		NHSEventID:         nhsEventID,
		Outcome:            providerexchange.Outcome(input.Outcome),
		ProviderReportedAt: providerReportedAt.Unix(),
		RecordedAt:         receiptRecordedAt.Unix(),
		ExpiresAt:          receiptRecordedAt.Add(OutcomeReceiptValidity).Unix(),
		ChargedMinor:       billedCents,
		Currency:           currency,
		ChargeStatus:       chargeStatus,
	})
	if err != nil {
		return nil, false, err
	}
	receipt, err := scanOutcomeReceipt(tx.QueryRow(`
		INSERT INTO outcome_receipts (
			id, nhs_event_id, provider_claim_id, provider_offer_id,
			action_ticket_id, provider_api_key_id, idempotency_key_hash,
			payload_hash, outcome, billed_cents, charge_status, currency,
			signed_receipt, signature, provider_reported_at, created_at
		) VALUES (
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16
		)
		RETURNING `+outcomeReceiptColumns,
		receiptID, nhsEventID, claimID, offerID, input.ActionTicketID, key.ID,
		idempotencyHash, input.PayloadHash, input.Outcome, billedCents,
		string(chargeStatus), currency, signedReceipt, signature,
		providerReportedAt, receiptRecordedAt))
	if err != nil {
		return nil, false, err
	}
	if _, err := tx.Exec(`
		UPDATE action_tickets SET status=$1, updated_at=$2
		WHERE id=$3::uuid`, nextActionTicketStatus(ticketStatus, input.Outcome), receiptRecordedAt, input.ActionTicketID); err != nil {
		return nil, false, err
	}
	if _, err := tx.Exec(`UPDATE provider_api_keys SET last_used_at=$1 WHERE id=$2`, receiptRecordedAt, key.ID); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return receipt, true, nil
}

func GetOutcomeReceipt(db *sql.DB, key *ProviderAPIKey, receiptID string) (*OutcomeReceipt, error) {
	if db == nil || key == nil || key.ID < 1 || !validProviderUUID(key.ProviderClaimID) || !validProviderUUID(receiptID) {
		return nil, ErrInvalidProviderExchange
	}
	return scanOutcomeReceipt(db.QueryRow(`
		SELECT `+outcomeReceiptColumns+`
		FROM outcome_receipts receipt
		JOIN provider_api_keys key
		  ON key.id=$1 AND key.provider_claim_id=receipt.provider_claim_id
		WHERE receipt.id=$2::uuid AND key.status='active'`, key.ID, receiptID))
}

// GetPublicOutcomeReceiptState returns only the current ticket and commercial
// effect for an exact NHS receipt/ticket pair. It exposes no claim, provider,
// account, action URL, intent, contact, or principal fields. Callers must verify
// the signed receipt independently; a valid historic signature does not imply
// that its charge or ticket authorization remains current.
func GetPublicOutcomeReceiptState(db *sql.DB, receiptID, ticketID string) (*PublicOutcomeReceiptState, error) {
	receiptID = strings.ToLower(strings.TrimSpace(receiptID))
	ticketID = strings.ToLower(strings.TrimSpace(ticketID))
	if db == nil || !validProviderUUID(receiptID) || !validProviderUUID(ticketID) {
		return nil, ErrInvalidProviderExchange
	}
	state := &PublicOutcomeReceiptState{}
	var rawNet string
	err := db.QueryRow(`
		SELECT receipt.id::text, ticket.id::text, receipt.outcome,
		       CASE
		           WHEN ticket.authorization_revoked_at IS NOT NULL THEN 'revoked'
		           WHEN ticket.expires_at <= NOW()
		                AND ticket.status IN ('created','redirected','accepted','activated') THEN 'expired'
		           ELSE ticket.status
		       END AS current_ticket_status,
		       receipt.charge_status='charged' AND EXISTS (
		           SELECT 1 FROM provider_budget_ledger credit
		           WHERE credit.action_ticket_id=ticket.id AND credit.entry_type='credit'
		       ) AS original_charge_credited,
		       ticket.authorization_revoked_at IS NOT NULL OR
		       ticket.status <> receipt.outcome OR
		       (ticket.expires_at <= NOW() AND
		        ticket.status IN ('created','redirected','accepted','activated'))
		       AS superseded_by_later_state,
		       ticket.authorization_revoked_at IS NOT NULL,
		       GREATEST(-COALESCE((
		           SELECT SUM(ledger.amount_cents::numeric)
		           FROM provider_budget_ledger ledger
		           WHERE ledger.action_ticket_id=ticket.id
		             AND ledger.entry_type IN ('charge','credit')
		       ),0),0)::text AS net_commercial_effect_cents,
		       receipt.currency
		FROM outcome_receipts receipt
		JOIN action_tickets ticket ON ticket.id=receipt.action_ticket_id
		WHERE receipt.id=$1::uuid AND ticket.id=$2::uuid`, receiptID, ticketID).Scan(
		&state.ReceiptID, &state.ActionTicketID, &state.ReceiptOutcome,
		&state.CurrentTicketStatus, &state.OriginalChargeCredited,
		&state.SupersededByLaterState, &state.AuthorizationRevoked,
		&rawNet, &state.NetCommercialEffectCurrency,
	)
	if err != nil {
		return nil, err
	}
	state.NetCommercialEffectCents, err = parseBoundedProviderMoney(rawNet)
	if err != nil {
		return nil, err
	}
	return state, nil
}

// GetProviderExchangeProof returns authenticated provider assertions and
// operator-recorded evidence aggregates. PilotThresholdsMet is a pilot progress
// signal, never an independently audited business or outcome proof.
func GetProviderExchangeProof(db *sql.DB) (*ProviderExchangeProof, error) {
	if db == nil {
		return nil, ErrInvalidProviderExchange
	}
	proof := &ProviderExchangeProof{
		PrepaidNetDebitedByCurrency:         map[string]int64{},
		TermsNetReceivableByCurrency:        map[string]int64{},
		OperatorRecordedCollectedByCurrency: map[string]int64{},
	}
	err := db.QueryRow(`
		WITH real_tickets AS (
			SELECT ticket.id
			FROM action_tickets ticket
			WHERE NOT ticket.source_is_synthetic
			  AND ticket.authorization_revoked_at IS NULL
			  AND ticket.status IN ('accepted','activated','converted')
			  AND NOT EXISTS (
				SELECT 1 FROM provider_budget_ledger credited
				WHERE credited.action_ticket_id=ticket.id
				  AND credited.entry_type='credit'
			  )
		), operator_recorded_budgets AS (
			SELECT DISTINCT claim.account_id
			FROM provider_claims claim
			JOIN provider_offers offer
			  ON offer.provider_claim_id=claim.id AND offer.status='active'
			WHERE claim.status='verified'
			  AND offer.terms_evidence_reference <> ''
			  AND (
				offer.billing_mode='terms' OR
				EXISTS (
					SELECT 1 FROM provider_budget_ledger ledger
					WHERE ledger.provider_offer_id=offer.id
					  AND ledger.entry_type='fund' AND ledger.amount_cents > 0
				)
			  )
		), renewed_budgets AS (
			SELECT DISTINCT claim.account_id
			FROM provider_offers offer
			JOIN provider_claims claim ON claim.id=offer.provider_claim_id
			JOIN provider_budget_ledger funding ON funding.provider_offer_id=offer.id
			WHERE funding.entry_type='fund' AND funding.amount_cents > 0
			  AND EXISTS (
				SELECT 1 FROM provider_budget_ledger charged
				JOIN action_tickets charged_ticket
				  ON charged_ticket.id=charged.action_ticket_id
				WHERE charged.provider_offer_id=offer.id
				  AND charged.entry_type='charge'
				  AND funding.amount_cents >= -charged.amount_cents
				  AND NOT charged_ticket.source_is_synthetic
				  AND charged_ticket.status IN ('accepted','activated','converted')
				  AND NOT EXISTS (
					SELECT 1 FROM provider_budget_ledger credited
					WHERE credited.action_ticket_id=charged_ticket.id
					  AND credited.entry_type='credit'
				  )
				  AND (charged.created_at, charged.id) < (funding.created_at, funding.id)
			  )
		)
		SELECT
			(SELECT COUNT(*)::int FROM operator_recorded_budgets),
			COUNT(DISTINCT outcome.action_ticket_id)
			  FILTER (WHERE outcome.outcome='accepted' AND NOT EXISTS (
				SELECT 1 FROM outcome_receipts terminal
				WHERE terminal.action_ticket_id=outcome.action_ticket_id
				  AND terminal.outcome IN ('duplicate','invalid')
			  ))::int,
			COUNT(DISTINCT outcome.action_ticket_id)
			  FILTER (WHERE outcome.outcome IN ('activated','converted') AND NOT EXISTS (
				SELECT 1 FROM outcome_receipts terminal
				WHERE terminal.action_ticket_id=outcome.action_ticket_id
				  AND terminal.outcome IN ('duplicate','invalid')
			  ))::int,
			(SELECT COUNT(*)::int FROM renewed_budgets),
			COUNT(DISTINCT outcome.action_ticket_id)
			  FILTER (WHERE outcome.outcome='converted' AND NOT EXISTS (
				SELECT 1 FROM outcome_receipts terminal
				WHERE terminal.action_ticket_id=outcome.action_ticket_id
				  AND terminal.outcome IN ('duplicate','invalid')
			  ))::int
		FROM outcome_receipts outcome
		JOIN real_tickets ticket ON ticket.id=outcome.action_ticket_id`,
	).Scan(
		&proof.OperatorRecordedProviderBudgets, &proof.ProviderReportedAcceptedHandoffs,
		&proof.ProviderReportedActivations, &proof.RenewedProviderBudgets,
		&proof.ProviderReportedConversions,
	)
	if err != nil {
		return nil, err
	}
	prepaidRows, err := db.Query(`
		SELECT ledger.currency,
		       COALESCE(-SUM(ledger.amount_cents::numeric),0)::text AS net_debited_cents
		FROM provider_budget_ledger ledger
		JOIN action_tickets ticket ON ticket.id=ledger.action_ticket_id
		WHERE ledger.entry_type IN ('charge','credit')
		  AND ticket.billing_mode_snapshot='prepaid'
		  AND NOT ticket.source_is_synthetic
		GROUP BY ledger.currency
		ORDER BY ledger.currency`)
	if err != nil {
		return nil, err
	}
	for prepaidRows.Next() {
		var currency, rawNet string
		if err := prepaidRows.Scan(&currency, &rawNet); err != nil {
			prepaidRows.Close()
			return nil, err
		}
		net, err := parseProviderMoney(rawNet)
		if err != nil {
			prepaidRows.Close()
			return nil, err
		}
		proof.PrepaidNetDebitedByCurrency[currency] = net
	}
	if err := prepaidRows.Err(); err != nil {
		prepaidRows.Close()
		return nil, err
	}
	prepaidRows.Close()

	termsRows, err := db.Query(`
		WITH outstanding_by_offer AS (
			SELECT offer.id, offer.currency,
			       GREATEST(-COALESCE(SUM(ledger.amount_cents::numeric),0),0) AS receivable_cents
			FROM provider_offers offer
			LEFT JOIN provider_budget_ledger ledger ON ledger.provider_offer_id=offer.id
			WHERE offer.billing_mode='terms'
			GROUP BY offer.id, offer.currency
		)
		SELECT currency, COALESCE(SUM(receivable_cents),0)::text
		FROM outstanding_by_offer
		GROUP BY currency
		ORDER BY currency`)
	if err != nil {
		return nil, err
	}
	for termsRows.Next() {
		var currency, rawReceivable string
		if err := termsRows.Scan(&currency, &rawReceivable); err != nil {
			termsRows.Close()
			return nil, err
		}
		receivable, err := parseProviderMoney(rawReceivable)
		if err != nil {
			termsRows.Close()
			return nil, err
		}
		proof.TermsNetReceivableByCurrency[currency] = receivable
	}
	if err := termsRows.Err(); err != nil {
		termsRows.Close()
		return nil, err
	}
	termsRows.Close()

	collectedRows, err := db.Query(`
		SELECT currency, COALESCE(SUM(amount_cents::numeric),0)::text
		FROM provider_budget_ledger
		WHERE entry_type='fund'
		GROUP BY currency
		ORDER BY currency`)
	if err != nil {
		return nil, err
	}
	for collectedRows.Next() {
		var currency, rawCollected string
		if err := collectedRows.Scan(&currency, &rawCollected); err != nil {
			collectedRows.Close()
			return nil, err
		}
		collected, err := parseProviderMoney(rawCollected)
		if err != nil {
			collectedRows.Close()
			return nil, err
		}
		proof.OperatorRecordedCollectedByCurrency[currency] = collected
	}
	if err := collectedRows.Err(); err != nil {
		collectedRows.Close()
		return nil, err
	}
	collectedRows.Close()
	proof.PilotThresholdsMet = proof.OperatorRecordedProviderBudgets >= 3 &&
		proof.ProviderReportedAcceptedHandoffs >= 5 &&
		proof.ProviderReportedActivations >= 2 &&
		proof.RenewedProviderBudgets >= 1
	return proof, nil
}
