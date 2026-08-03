package models

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"
)

type providerSigningKeyProofQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

var providerSignaturePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

const (
	ProviderSigningProofAttribution = "attribution"
	ProviderSigningProofOutcome     = "outcome"
	ProviderSigningProofManifest    = "manifest"
)

// ProviderSigningKeyProof is one non-secret persisted proof sample for a key
// and signing domain. Startup reconstructs or verifies it to detect both a
// removed key ID and accidental replacement of material under the same ID.
type ProviderSigningKeyProof struct {
	KeyID          string
	Kind           string
	TicketID       string
	OfferID        string
	IssuedAt       time.Time
	ExpiresAt      time.Time
	TokenNonce     string
	TokenHash      string
	SignedReceipt  string
	SignedManifest string
	Signature      string
}

// ProviderSigningKeyProofsInUse returns at most one attribution, outcome, and
// commercial-proof manifest sample per referenced key. The small bounded result
// makes exact material verification practical on every production startup
// without exposing secrets.
func ProviderSigningKeyProofsInUse(db *sql.DB) ([]ProviderSigningKeyProof, error) {
	if db == nil {
		return nil, ErrInvalidProviderExchange
	}
	return providerSigningKeyProofsInUse(context.Background(), db)
}

// ProviderSigningKeyProofsInUseContext is the bounded read-only variant used
// by cutover preflight so its database work shares the caller's deadline.
func ProviderSigningKeyProofsInUseContext(ctx context.Context, db *sql.DB) ([]ProviderSigningKeyProof, error) {
	if ctx == nil || db == nil {
		return nil, ErrInvalidProviderExchange
	}
	return providerSigningKeyProofsInUse(ctx, db)
}

// ProviderSigningKeyProofsInUseTx reads the same bounded proof set through the
// protected migration transaction, after its writer lock has been acquired.
// This prevents a ticket/outcome writer from racing the one-way migration 022
// signing-key compatibility decision.
func ProviderSigningKeyProofsInUseTx(ctx context.Context, tx *sql.Tx) ([]ProviderSigningKeyProof, error) {
	if ctx == nil || tx == nil {
		return nil, ErrInvalidProviderExchange
	}
	return providerSigningKeyProofsInUse(ctx, tx)
}

func providerSigningKeyProofsInUse(ctx context.Context, queryer providerSigningKeyProofQueryer) ([]ProviderSigningKeyProof, error) {
	manifestTableRows, err := queryer.QueryContext(ctx, `
		SELECT to_regclass('public.provider_commercial_proof_manifests') IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	manifestTableExists := false
	if manifestTableRows.Next() {
		if err := manifestTableRows.Scan(&manifestTableExists); err != nil {
			manifestTableRows.Close()
			return nil, err
		}
	}
	if err := manifestTableRows.Err(); err != nil {
		manifestTableRows.Close()
		return nil, err
	}
	manifestTableRows.Close()
	manifestProofUnion := ""
	if manifestTableExists {
		manifestProofUnion = `
			UNION ALL
			SELECT key_id,
			       'manifest'::text AS proof_kind,
			       NULL::text AS ticket_id, NULL::text AS offer_id,
			       NULL::timestamptz AS issued_at, NULL::timestamptz AS expires_at,
			       NULL::text AS token_nonce, NULL::text AS token_hash,
			       signed_manifest AS signed_receipt, signature,
			       issued_at AS proof_created_at, id::text AS proof_order
			FROM provider_commercial_proof_manifests`
	}
	rows, err := queryer.QueryContext(ctx, `
		SELECT DISTINCT ON (key_id, proof_kind)
		       key_id, proof_kind, ticket_id, offer_id, issued_at, expires_at,
		       token_nonce, token_hash, signed_receipt, signature
		FROM (
			SELECT attribution_key_id_snapshot AS key_id,
			       'attribution'::text AS proof_kind,
			       id::text AS ticket_id, provider_offer_id::text AS offer_id,
			       created_at AS issued_at, expires_at, token_nonce, token_hash,
			       NULL::text AS signed_receipt, NULL::text AS signature,
			       created_at AS proof_created_at, id::text AS proof_order
			FROM action_tickets
			UNION ALL
			SELECT signed_receipt::jsonb->>'kid' AS key_id,
			       'outcome'::text AS proof_kind,
			       NULL::text AS ticket_id, NULL::text AS offer_id,
			       NULL::timestamptz AS issued_at, NULL::timestamptz AS expires_at,
			       NULL::text AS token_nonce, NULL::text AS token_hash,
			       signed_receipt, signature,
			       created_at AS proof_created_at, id::text AS proof_order
			FROM outcome_receipts
		`+manifestProofUnion+`
		) proofs
		WHERE key_id IS NOT NULL AND key_id <> ''
		ORDER BY key_id, proof_kind, proof_created_at, proof_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	proofs := []ProviderSigningKeyProof{}
	for rows.Next() {
		var proof ProviderSigningKeyProof
		var ticketID, offerID, tokenNonce, tokenHash, signedReceipt, signature sql.NullString
		var issuedAt, expiresAt sql.NullTime
		if err := rows.Scan(
			&proof.KeyID, &proof.Kind, &ticketID, &offerID, &issuedAt, &expiresAt,
			&tokenNonce, &tokenHash, &signedReceipt, &signature,
		); err != nil {
			return nil, err
		}
		if !providerSigningKeyIDPattern.MatchString(proof.KeyID) {
			return nil, fmt.Errorf("%w: invalid persisted signing key id", ErrInvalidProviderExchange)
		}
		switch proof.Kind {
		case ProviderSigningProofAttribution:
			if !ticketID.Valid || !offerID.Valid || !issuedAt.Valid || !expiresAt.Valid ||
				!tokenNonce.Valid || !tokenHash.Valid || !providerHashPattern.MatchString(tokenHash.String) {
				return nil, fmt.Errorf("%w: incomplete persisted attribution proof", ErrInvalidProviderExchange)
			}
			proof.TicketID, proof.OfferID = ticketID.String, offerID.String
			proof.IssuedAt, proof.ExpiresAt = issuedAt.Time, expiresAt.Time
			proof.TokenNonce, proof.TokenHash = tokenNonce.String, tokenHash.String
		case ProviderSigningProofOutcome:
			if !signedReceipt.Valid || signedReceipt.String == "" ||
				!signature.Valid || !providerSignaturePattern.MatchString(signature.String) {
				return nil, fmt.Errorf("%w: incomplete persisted outcome proof", ErrInvalidProviderExchange)
			}
			proof.SignedReceipt, proof.Signature = signedReceipt.String, signature.String
		case ProviderSigningProofManifest:
			if !signedReceipt.Valid || signedReceipt.String == "" ||
				!signature.Valid || !providerSignaturePattern.MatchString(signature.String) {
				return nil, fmt.Errorf("%w: incomplete persisted manifest proof", ErrInvalidProviderExchange)
			}
			proof.SignedManifest, proof.Signature = signedReceipt.String, signature.String
		default:
			return nil, fmt.Errorf("%w: invalid persisted signing proof kind", ErrInvalidProviderExchange)
		}
		proofs = append(proofs, proof)
	}
	return proofs, rows.Err()
}
