package models

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	ErrProviderSettlementNotEligible = errors.New("provider outcome is not eligible for settlement")
	ErrProviderSettlementConflict    = errors.New("provider settlement conflicts with an existing immutable record")
)

// ProviderSettlementOrder freezes the exact provider obligation already
// created by a charged terms outcome. Creating an order never proves payment.
// ProviderBillingEmail is deliberately omitted: it is read from the provider's
// existing account only long enough to create the private Stripe Checkout.
type ProviderSettlementOrder struct {
	ID                   string    `json:"id"`
	ProviderClaimID      string    `json:"provider_claim_id"`
	ProviderOfferID      string    `json:"provider_offer_id"`
	ActionTicketID       string    `json:"action_ticket_id"`
	OutcomeReceiptID     string    `json:"outcome_receipt_id"`
	Outcome              string    `json:"outcome"`
	OfferVersionSnapshot int       `json:"offer_version"`
	TermsContractVersion string    `json:"commercial_terms_contract_version"`
	TermsSHA256          string    `json:"commercial_terms_sha256"`
	AmountCents          int64     `json:"amount_cents"`
	Currency             string    `json:"currency"`
	CreatedAt            time.Time `json:"created_at"`
	ProviderBillingEmail string    `json:"-"`
}

// ProviderSettlementPaymentInput contains only Stripe identifiers and exact
// currency/amount facts received through a signature-verified webhook. It
// intentionally accepts neither a caller-selected price nor free-form notes.
type ProviderSettlementPaymentInput struct {
	OrderID                    string
	StripeCheckoutSessionID    string
	StripePaymentIntentID      string
	StripeEventID              string
	StripeChargeID             string
	StripeBalanceTransactionID string
	AmountCents                int64
	ProcessorFeeCents          int64
	ProcessorNetCents          int64
	Currency                   string
	ProcessorStatus            string
	ProcessorAvailableOn       time.Time
	ProcessorObservedAt        time.Time
	PaidAt                     time.Time
}

type ProviderSettlementPaymentReceipt struct {
	ID                    string    `json:"id"`
	OrderID               string    `json:"order_id"`
	AmountCents           int64     `json:"amount_cents"`
	ProcessorFeeCents     int64     `json:"processor_fee_cents"`
	ProcessorNetCents     int64     `json:"processor_net_cents"`
	Currency              string    `json:"currency"`
	ProcessorAvailableOn  time.Time `json:"processor_available_on"`
	ProcessorNetAvailable bool      `json:"processor_net_available"`
	PaidAt                time.Time `json:"paid_at"`
	CreatedAt             time.Time `json:"created_at"`
}

// ProviderSettlementAvailabilityInput carries one authenticated Stripe
// balance-transaction refresh. The balance transaction identifier is a private
// binding coordinate and is never projected by owner status or pilot proof.
type ProviderSettlementAvailabilityInput struct {
	OrderID                    string
	StripeBalanceTransactionID string
	GrossAmountCents           int64
	FeeCents                   int64
	NetCents                   int64
	Currency                   string
	AvailableOn                time.Time
	ProcessorVerifiedAt        time.Time
}

// ProviderSettlementProcessorReference is an owner-only lookup coordinate used
// to refresh one already-recorded Stripe balance transaction. The identifier
// must never be serialized into an API response or proof artifact.
type ProviderSettlementProcessorReference struct {
	StripeBalanceTransactionID string    `json:"-"`
	GrossAmountCents           int64     `json:"gross_amount_cents"`
	FeeCents                   int64     `json:"fee_cents"`
	NetCents                   int64     `json:"net_cents"`
	Currency                   string    `json:"currency"`
	AvailableOn                time.Time `json:"available_on"`
}

// ProviderSettlementStatus is an owner-only, privacy-bounded view. It does
// not expose the Stripe session, payment-intent, event identifier, provider
// email, agent identity, or original search context.
type ProviderSettlementStatus struct {
	OrderID               string     `json:"order_id"`
	OutcomeReceiptID      string     `json:"outcome_receipt_id"`
	AmountCents           int64      `json:"amount_cents"`
	Currency              string     `json:"currency"`
	CheckoutCreated       bool       `json:"checkout_created"`
	Paid                  bool       `json:"paid"`
	PaymentReceiptID      string     `json:"payment_receipt_id,omitempty"`
	ProcessorNetRecorded  bool       `json:"processor_net_recorded"`
	ProcessorNetAvailable bool       `json:"processor_net_available"`
	ProcessorFeeCents     int64      `json:"processor_fee_cents,omitempty"`
	ProcessorNetCents     int64      `json:"processor_net_cents,omitempty"`
	ProcessorAvailableOn  *time.Time `json:"processor_available_on,omitempty"`
	PaidAt                *time.Time `json:"paid_at,omitempty"`
}

const providerSettlementOrderColumns = `
	id::text, provider_claim_id::text, provider_offer_id::text,
	action_ticket_id::text, outcome_receipt_id::text, outcome,
	offer_version_snapshot, terms_contract_version_snapshot,
	terms_sha256_snapshot, amount_cents, currency, created_at`

func scanProviderSettlementOrder(row rowScanner) (*ProviderSettlementOrder, error) {
	var order ProviderSettlementOrder
	if err := row.Scan(
		&order.ID, &order.ProviderClaimID, &order.ProviderOfferID,
		&order.ActionTicketID, &order.OutcomeReceiptID, &order.Outcome,
		&order.OfferVersionSnapshot, &order.TermsContractVersion,
		&order.TermsSHA256, &order.AmountCents, &order.Currency, &order.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &order, nil
}

func validProviderStripeID(value, prefix string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, prefix) || len(value) < len(prefix)+8 || len(value) > 258 {
		return false
	}
	for _, r := range value[len(prefix):] {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// PrepareProviderSettlement atomically creates or returns the one settlement
// order permitted for a current, uncredited charged terms outcome. It never
// takes payment and never accepts an externally supplied amount.
func PrepareProviderSettlement(db *sql.DB, outcomeReceiptID string) (*ProviderSettlementOrder, bool, error) {
	outcomeReceiptID = strings.ToLower(strings.TrimSpace(outcomeReceiptID))
	if db == nil || !validProviderUUID(outcomeReceiptID) {
		return nil, false, ErrInvalidProviderExchange
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()

	var billingEmail string
	order := &ProviderSettlementOrder{}
	err = tx.QueryRow(`
		SELECT receipt.provider_claim_id::text, receipt.provider_offer_id::text,
		       receipt.action_ticket_id::text, receipt.outcome, receipt.billed_cents,
		       receipt.currency, ticket.offer_version_snapshot,
		       ticket.commercial_terms_contract_version_snapshot,
		       ticket.commercial_terms_sha256_snapshot, account.email
		FROM outcome_receipts receipt
		JOIN action_tickets ticket ON ticket.id=receipt.action_ticket_id
		JOIN provider_offers offer ON offer.id=receipt.provider_offer_id
		JOIN provider_claims claim ON claim.id=receipt.provider_claim_id
		JOIN accounts account ON account.id=claim.account_id
		WHERE receipt.id=$1::uuid
		  AND receipt.charge_status='charged' AND receipt.billed_cents > 0
		  AND receipt.outcome IN ('accepted','activated','converted')
		  AND (
		      (receipt.outcome='accepted' AND ticket.status IN ('accepted','activated','converted')) OR
		      (receipt.outcome='activated' AND ticket.status IN ('activated','converted')) OR
		      (receipt.outcome='converted' AND ticket.status='converted')
		  )
		  AND ticket.authorization_revoked_at IS NULL
		  AND ticket.billing_mode_snapshot='terms'
		  AND offer.billing_mode='terms'
		  AND NOT EXISTS (
		      SELECT 1 FROM provider_budget_ledger credit
		      WHERE credit.action_ticket_id=ticket.id AND credit.entry_type='credit'
		  )
		FOR UPDATE OF receipt, ticket, offer, claim, account`, outcomeReceiptID).Scan(
		&order.ProviderClaimID, &order.ProviderOfferID, &order.ActionTicketID,
		&order.Outcome, &order.AmountCents, &order.Currency,
		&order.OfferVersionSnapshot, &order.TermsContractVersion,
		&order.TermsSHA256, &billingEmail,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, ErrProviderSettlementNotEligible
	}
	if err != nil {
		return nil, false, err
	}

	existing, err := scanProviderSettlementOrder(tx.QueryRow(`
		SELECT `+providerSettlementOrderColumns+`
		FROM provider_settlement_orders
		WHERE outcome_receipt_id=$1::uuid
		FOR KEY SHARE`, outcomeReceiptID))
	if err == nil {
		existing.ProviderBillingEmail = billingEmail
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	orderID, err := newProviderUUID()
	if err != nil {
		return nil, false, err
	}
	order.ID = orderID
	order.OutcomeReceiptID = outcomeReceiptID
	order.ProviderBillingEmail = billingEmail
	created, err := scanProviderSettlementOrder(tx.QueryRow(`
		INSERT INTO provider_settlement_orders (
			id, provider_claim_id, provider_offer_id, action_ticket_id,
			outcome_receipt_id, outcome, offer_version_snapshot,
			terms_contract_version_snapshot, terms_sha256_snapshot,
			amount_cents, currency
		) VALUES (
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8,$9,$10,$11
		)
		RETURNING `+providerSettlementOrderColumns,
		order.ID, order.ProviderClaimID, order.ProviderOfferID, order.ActionTicketID,
		order.OutcomeReceiptID, order.Outcome, order.OfferVersionSnapshot,
		order.TermsContractVersion, order.TermsSHA256, order.AmountCents, order.Currency,
	))
	if err != nil {
		return nil, false, err
	}
	created.ProviderBillingEmail = billingEmail
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return created, true, nil
}

// RecordProviderSettlementCheckoutSession attaches Stripe's idempotent Checkout
// Session to a prepared order. It is append-only; a different session can never
// replace the first recorded one.
func RecordProviderSettlementCheckoutSession(db *sql.DB, orderID, checkoutSessionID string) (bool, error) {
	orderID = strings.ToLower(strings.TrimSpace(orderID))
	checkoutSessionID = strings.TrimSpace(checkoutSessionID)
	if db == nil || !validProviderUUID(orderID) || !validProviderStripeID(checkoutSessionID, "cs_") {
		return false, ErrInvalidProviderExchange
	}
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var existingSession string
	err = tx.QueryRow(`
		SELECT stripe_checkout_session_id
		FROM provider_settlement_checkout_sessions
		WHERE provider_settlement_order_id=$1::uuid
		FOR KEY SHARE`, orderID).Scan(&existingSession)
	if err == nil {
		if existingSession != checkoutSessionID {
			return false, ErrProviderSettlementConflict
		}
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if _, err := tx.Exec(`
		INSERT INTO provider_settlement_checkout_sessions (
			provider_settlement_order_id, stripe_checkout_session_id
		) VALUES ($1::uuid,$2)`, orderID, checkoutSessionID); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// RecordProviderSettlementPayment appends the only fact that counts as a
// completed provider payment. Callers must have verified the Stripe signature
// and checkout.session.completed payment status before calling it.
func RecordProviderSettlementPayment(db *sql.DB, input ProviderSettlementPaymentInput) (*ProviderSettlementPaymentReceipt, bool, error) {
	input.OrderID = strings.ToLower(strings.TrimSpace(input.OrderID))
	input.StripeCheckoutSessionID = strings.TrimSpace(input.StripeCheckoutSessionID)
	input.StripePaymentIntentID = strings.TrimSpace(input.StripePaymentIntentID)
	input.StripeEventID = strings.TrimSpace(input.StripeEventID)
	input.StripeChargeID = strings.TrimSpace(input.StripeChargeID)
	input.StripeBalanceTransactionID = strings.TrimSpace(input.StripeBalanceTransactionID)
	input.Currency = strings.ToLower(strings.TrimSpace(input.Currency))
	input.ProcessorStatus = strings.ToLower(strings.TrimSpace(input.ProcessorStatus))
	input.PaidAt = input.PaidAt.UTC()
	input.ProcessorAvailableOn = input.ProcessorAvailableOn.UTC()
	input.ProcessorObservedAt = input.ProcessorObservedAt.UTC()
	if db == nil || !validProviderUUID(input.OrderID) ||
		!validProviderStripeID(input.StripeCheckoutSessionID, "cs_") ||
		!validProviderStripeID(input.StripePaymentIntentID, "pi_") ||
		!validProviderStripeID(input.StripeEventID, "evt_") ||
		!validProviderStripeID(input.StripeChargeID, "ch_") ||
		!validProviderStripeID(input.StripeBalanceTransactionID, "txn_") ||
		input.AmountCents < 1 || input.AmountCents > ProviderBountyMaximumCents ||
		input.ProcessorFeeCents < 0 || input.ProcessorFeeCents >= input.AmountCents ||
		input.ProcessorNetCents != input.AmountCents-input.ProcessorFeeCents ||
		input.Currency != "usd" ||
		(input.ProcessorStatus != "pending" && input.ProcessorStatus != "available") ||
		input.PaidAt.IsZero() || input.ProcessorAvailableOn.IsZero() || input.ProcessorObservedAt.IsZero() ||
		input.ProcessorObservedAt.Before(input.PaidAt) {
		return nil, false, ErrInvalidProviderExchange
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	var expectedSession, expectedCurrency string
	var expectedAmount int64
	var ticketID string
	err = tx.QueryRow(`
		SELECT checkout.stripe_checkout_session_id, settlement.amount_cents,
		       settlement.currency, settlement.action_ticket_id::text
		FROM provider_settlement_orders settlement
		JOIN provider_settlement_checkout_sessions checkout
		  ON checkout.provider_settlement_order_id=settlement.id
		WHERE settlement.id=$1::uuid
		FOR UPDATE OF settlement, checkout`, input.OrderID).Scan(
		&expectedSession, &expectedAmount, &expectedCurrency, &ticketID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, ErrProviderSettlementNotEligible
	}
	if err != nil {
		return nil, false, err
	}
	if expectedSession != input.StripeCheckoutSessionID || expectedAmount != input.AmountCents || expectedCurrency != input.Currency {
		return nil, false, ErrProviderSettlementConflict
	}
	var credited bool
	if err := tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM provider_budget_ledger
			WHERE action_ticket_id=$1::uuid AND entry_type='credit'
		)`, ticketID).Scan(&credited); err != nil {
		return nil, false, err
	}
	if credited {
		return nil, false, ErrProviderSettlementNotEligible
	}

	existing := &ProviderSettlementPaymentReceipt{}
	var existingSession, existingIntent, existingEvent string
	var existingCharge, existingBalanceTransaction string
	err = tx.QueryRow(`
		SELECT payment.id::text, payment.provider_settlement_order_id::text,
		       payment.stripe_checkout_session_id, payment.stripe_payment_intent_id,
		       payment.stripe_event_id, balance.stripe_charge_id,
		       balance.stripe_balance_transaction_id,
		       payment.amount_cents, balance.fee_cents, balance.net_cents,
		       payment.currency, balance.available_on,
		       availability.id IS NOT NULL, payment.paid_at, payment.created_at
		FROM provider_settlement_payment_receipts payment
		JOIN provider_settlement_processor_balance_receipts balance
		  ON balance.provider_settlement_payment_receipt_id=payment.id
		LEFT JOIN provider_settlement_processor_availability_receipts availability
		  ON availability.processor_balance_receipt_id=balance.id
		WHERE payment.provider_settlement_order_id=$1::uuid
		FOR KEY SHARE OF payment, balance`, input.OrderID).Scan(
		&existing.ID, &existing.OrderID, &existingSession, &existingIntent,
		&existingEvent, &existingCharge, &existingBalanceTransaction,
		&existing.AmountCents, &existing.ProcessorFeeCents, &existing.ProcessorNetCents,
		&existing.Currency, &existing.ProcessorAvailableOn, &existing.ProcessorNetAvailable,
		&existing.PaidAt, &existing.CreatedAt,
	)
	if err == nil {
		if existingSession != input.StripeCheckoutSessionID || existingIntent != input.StripePaymentIntentID ||
			existingEvent != input.StripeEventID || existingCharge != input.StripeChargeID ||
			existingBalanceTransaction != input.StripeBalanceTransactionID ||
			existing.AmountCents != input.AmountCents || existing.ProcessorFeeCents != input.ProcessorFeeCents ||
			existing.ProcessorNetCents != input.ProcessorNetCents || existing.Currency != input.Currency ||
			!existing.ProcessorAvailableOn.Equal(input.ProcessorAvailableOn) {
			return nil, false, ErrProviderSettlementConflict
		}
		if input.ProcessorStatus == "available" && !existing.ProcessorNetAvailable {
			if _, _, err := recordProviderSettlementAvailabilityTx(tx, ProviderSettlementAvailabilityInput{
				OrderID: input.OrderID, StripeBalanceTransactionID: input.StripeBalanceTransactionID,
				GrossAmountCents: input.AmountCents, FeeCents: input.ProcessorFeeCents,
				NetCents: input.ProcessorNetCents, Currency: input.Currency,
				AvailableOn: input.ProcessorAvailableOn, ProcessorVerifiedAt: input.ProcessorObservedAt,
			}); err != nil {
				return nil, false, err
			}
			existing.ProcessorNetAvailable = true
		}
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	receiptID, err := newProviderUUID()
	if err != nil {
		return nil, false, err
	}
	receipt := &ProviderSettlementPaymentReceipt{}
	err = tx.QueryRow(`
		INSERT INTO provider_settlement_payment_receipts (
			id, provider_settlement_order_id, stripe_checkout_session_id,
			stripe_payment_intent_id, stripe_event_id, amount_cents, currency, paid_at
		) VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8)
		RETURNING id::text, provider_settlement_order_id::text,
		          amount_cents, currency, paid_at, created_at`,
		receiptID, input.OrderID, input.StripeCheckoutSessionID,
		input.StripePaymentIntentID, input.StripeEventID,
		input.AmountCents, input.Currency, input.PaidAt,
	).Scan(&receipt.ID, &receipt.OrderID, &receipt.AmountCents, &receipt.Currency, &receipt.PaidAt, &receipt.CreatedAt)
	if err != nil {
		return nil, false, err
	}
	balanceReceiptID, err := newProviderUUID()
	if err != nil {
		return nil, false, err
	}
	if _, err := tx.Exec(`
		INSERT INTO provider_settlement_processor_balance_receipts (
			id, provider_settlement_payment_receipt_id, stripe_charge_id,
			stripe_balance_transaction_id, gross_amount_cents, fee_cents,
			net_cents, currency, initial_status, available_on, processor_observed_at
		) VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		balanceReceiptID, receipt.ID, input.StripeChargeID,
		input.StripeBalanceTransactionID, input.AmountCents,
		input.ProcessorFeeCents, input.ProcessorNetCents, input.Currency,
		input.ProcessorStatus, input.ProcessorAvailableOn, input.ProcessorObservedAt,
	); err != nil {
		return nil, false, err
	}
	receipt.ProcessorFeeCents = input.ProcessorFeeCents
	receipt.ProcessorNetCents = input.ProcessorNetCents
	receipt.ProcessorAvailableOn = input.ProcessorAvailableOn
	if input.ProcessorStatus == "available" {
		if _, _, err := recordProviderSettlementAvailabilityTx(tx, ProviderSettlementAvailabilityInput{
			OrderID: input.OrderID, StripeBalanceTransactionID: input.StripeBalanceTransactionID,
			GrossAmountCents: input.AmountCents, FeeCents: input.ProcessorFeeCents,
			NetCents: input.ProcessorNetCents, Currency: input.Currency,
			AvailableOn: input.ProcessorAvailableOn, ProcessorVerifiedAt: input.ProcessorObservedAt,
		}); err != nil {
			return nil, false, err
		}
		receipt.ProcessorNetAvailable = true
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return receipt, true, nil
}

func normalizeProviderSettlementAvailabilityInput(input ProviderSettlementAvailabilityInput) (ProviderSettlementAvailabilityInput, error) {
	input.OrderID = strings.ToLower(strings.TrimSpace(input.OrderID))
	input.StripeBalanceTransactionID = strings.TrimSpace(input.StripeBalanceTransactionID)
	input.Currency = strings.ToLower(strings.TrimSpace(input.Currency))
	input.AvailableOn = input.AvailableOn.UTC()
	input.ProcessorVerifiedAt = input.ProcessorVerifiedAt.UTC()
	if !validProviderUUID(input.OrderID) ||
		!validProviderStripeID(input.StripeBalanceTransactionID, "txn_") ||
		input.GrossAmountCents < 1 || input.GrossAmountCents > ProviderBountyMaximumCents ||
		input.FeeCents < 0 || input.FeeCents >= input.GrossAmountCents ||
		input.NetCents != input.GrossAmountCents-input.FeeCents || input.Currency != "usd" ||
		input.AvailableOn.IsZero() || input.ProcessorVerifiedAt.IsZero() ||
		input.ProcessorVerifiedAt.Before(input.AvailableOn) {
		return input, ErrInvalidProviderExchange
	}
	return input, nil
}

func recordProviderSettlementAvailabilityTx(tx *sql.Tx, input ProviderSettlementAvailabilityInput) (string, bool, error) {
	input, err := normalizeProviderSettlementAvailabilityInput(input)
	if err != nil {
		return "", false, err
	}
	var balanceReceiptID string
	var gross, fee, net int64
	var currency string
	var availableOn time.Time
	err = tx.QueryRow(`
		SELECT balance.id::text, balance.gross_amount_cents, balance.fee_cents,
		       balance.net_cents, balance.currency, balance.available_on
		FROM provider_settlement_processor_balance_receipts balance
		JOIN provider_settlement_payment_receipts payment
		  ON payment.id=balance.provider_settlement_payment_receipt_id
		WHERE payment.provider_settlement_order_id=$1::uuid
		  AND balance.stripe_balance_transaction_id=$2
		FOR KEY SHARE OF balance, payment`, input.OrderID, input.StripeBalanceTransactionID).Scan(
		&balanceReceiptID, &gross, &fee, &net, &currency, &availableOn,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, ErrProviderSettlementNotEligible
	}
	if err != nil {
		return "", false, err
	}
	if gross != input.GrossAmountCents || fee != input.FeeCents || net != input.NetCents ||
		currency != input.Currency || !availableOn.Equal(input.AvailableOn) {
		return "", false, ErrProviderSettlementConflict
	}
	var existingID string
	err = tx.QueryRow(`
		SELECT id::text
		FROM provider_settlement_processor_availability_receipts
		WHERE processor_balance_receipt_id=$1::uuid
		FOR KEY SHARE`, balanceReceiptID).Scan(&existingID)
	if err == nil {
		return existingID, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	receiptID, err := newProviderUUID()
	if err != nil {
		return "", false, err
	}
	if err := tx.QueryRow(`
		INSERT INTO provider_settlement_processor_availability_receipts (
			id, processor_balance_receipt_id, gross_amount_cents, fee_cents,
			net_cents, currency, available_on, processor_verified_at
		) VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8)
		RETURNING id::text`, receiptID, balanceReceiptID, gross, fee, net,
		currency, availableOn, input.ProcessorVerifiedAt).Scan(&receiptID); err != nil {
		return "", false, err
	}
	return receiptID, true, nil
}

// RecordProviderSettlementAvailability appends the first authenticated Stripe
// observation that the exact processor net is available. It is idempotent and
// cannot alter the original gross, fee, net, currency, or availability date.
func RecordProviderSettlementAvailability(db *sql.DB, input ProviderSettlementAvailabilityInput) (string, bool, error) {
	if db == nil {
		return "", false, ErrInvalidProviderExchange
	}
	tx, err := db.Begin()
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback()
	id, created, err := recordProviderSettlementAvailabilityTx(tx, input)
	if err != nil {
		return "", false, err
	}
	if err := tx.Commit(); err != nil {
		return "", false, err
	}
	return id, created, nil
}

func GetProviderSettlementProcessorReference(db *sql.DB, orderID string) (*ProviderSettlementProcessorReference, error) {
	orderID = strings.ToLower(strings.TrimSpace(orderID))
	if db == nil || !validProviderUUID(orderID) {
		return nil, ErrInvalidProviderExchange
	}
	reference := &ProviderSettlementProcessorReference{}
	err := db.QueryRow(`
		SELECT balance.stripe_balance_transaction_id, balance.gross_amount_cents,
		       balance.fee_cents, balance.net_cents, balance.currency, balance.available_on
		FROM provider_settlement_processor_balance_receipts balance
		JOIN provider_settlement_payment_receipts payment
		  ON payment.id=balance.provider_settlement_payment_receipt_id
		WHERE payment.provider_settlement_order_id=$1::uuid`, orderID).Scan(
		&reference.StripeBalanceTransactionID, &reference.GrossAmountCents,
		&reference.FeeCents, &reference.NetCents, &reference.Currency, &reference.AvailableOn,
	)
	if err != nil {
		return nil, err
	}
	return reference, nil
}

func GetProviderSettlementStatus(db *sql.DB, orderID string) (*ProviderSettlementStatus, error) {
	orderID = strings.ToLower(strings.TrimSpace(orderID))
	if db == nil || !validProviderUUID(orderID) {
		return nil, ErrInvalidProviderExchange
	}
	status := &ProviderSettlementStatus{}
	var paidAt, processorAvailableOn sql.NullTime
	var processorFeeCents, processorNetCents sql.NullInt64
	err := db.QueryRow(`
		SELECT settlement.id::text, settlement.outcome_receipt_id::text,
		       settlement.amount_cents, settlement.currency,
		       checkout.id IS NOT NULL, COALESCE(payment.id::text,''), payment.paid_at,
		       balance.id IS NOT NULL, availability.id IS NOT NULL,
		       balance.fee_cents, balance.net_cents, balance.available_on
		FROM provider_settlement_orders settlement
		LEFT JOIN provider_settlement_checkout_sessions checkout
		  ON checkout.provider_settlement_order_id=settlement.id
		LEFT JOIN provider_settlement_payment_receipts payment
		  ON payment.provider_settlement_order_id=settlement.id
		LEFT JOIN provider_settlement_processor_balance_receipts balance
		  ON balance.provider_settlement_payment_receipt_id=payment.id
		LEFT JOIN provider_settlement_processor_availability_receipts availability
		  ON availability.processor_balance_receipt_id=balance.id
		WHERE settlement.id=$1::uuid`, orderID).Scan(
		&status.OrderID, &status.OutcomeReceiptID, &status.AmountCents,
		&status.Currency, &status.CheckoutCreated, &status.PaymentReceiptID, &paidAt,
		&status.ProcessorNetRecorded, &status.ProcessorNetAvailable,
		&processorFeeCents, &processorNetCents, &processorAvailableOn,
	)
	if err != nil {
		return nil, err
	}
	status.Paid = status.PaymentReceiptID != ""
	if paidAt.Valid {
		value := paidAt.Time
		status.PaidAt = &value
	}
	if processorFeeCents.Valid {
		status.ProcessorFeeCents = processorFeeCents.Int64
	}
	if processorNetCents.Valid {
		status.ProcessorNetCents = processorNetCents.Int64
	}
	if processorAvailableOn.Valid {
		value := processorAvailableOn.Time
		status.ProcessorAvailableOn = &value
	}
	return status, nil
}
