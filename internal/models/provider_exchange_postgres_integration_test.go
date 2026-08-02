package models_test

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/models"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

// TestProviderExchangePostgresReleaseRegressions is opt-in because it requires
// an isolated PostgreSQL database that the test may migrate and populate. It
// exercises transaction and PostgreSQL locking behavior that cannot be proved
// by the source-contract unit tests.
func TestProviderExchangePostgresReleaseRegressions(t *testing.T) {
	dsn := os.Getenv("NHS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set NHS_TEST_POSTGRES_DSN to an isolated disposable PostgreSQL database")
	}
	t.Setenv("DATABASE_URL", dsn)
	if err := database.Connect(); err != nil {
		t.Fatalf("connect isolated PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = database.DB.Close() })
	migrationDir := filepath.Join("..", "..", "migrations")
	const releaseRevision = "1111111111111111111111111111111111111111"
	if err := database.RunMigrations(migrationDir, releaseRevision); err != nil {
		t.Fatalf("apply all migrations through repository runner: %v", err)
	}
	if err := database.RunMigrations(migrationDir, releaseRevision); err != nil {
		t.Fatalf("immediately replay all migrations through repository runner: %v", err)
	}
	db := database.DB

	proofs, err := models.ProviderSigningKeyProofsInUse(db)
	if err != nil {
		t.Fatalf("signing-key proofs on empty provider store: %v", err)
	}
	if len(proofs) != 0 {
		t.Fatalf("signing-key proofs on empty provider store = %d, want 0", len(proofs))
	}

	accountID, site := createPostgresProviderFixture(t, db)
	exerciseActionInterestPostgres(t, db, site)
	claim, rawChallenge, err := models.CreateProviderClaim(db, accountID, site.ID)
	if err != nil {
		t.Fatalf("create provider claim: %v", err)
	}
	claim, err = models.VerifyProviderClaim(db, accountID, claim.ID, rawChallenge)
	if err != nil {
		t.Fatalf("verify provider claim: %v", err)
	}
	_, providerKey, err := models.CreateProviderAPIKey(db, accountID, claim.ID)
	if err != nil {
		t.Fatalf("create provider API key: %v", err)
	}
	signer, err := providerexchange.NewSignerKeyring(
		"pg-release-v1",
		strings.Repeat("s", 32),
		nil,
	)
	if err != nil {
		t.Fatalf("create provider signer: %v", err)
	}

	const bountyCents int64 = 10_000
	zero := int64(0)
	offer, err := models.CreateProviderOffer(db, accountID, claim.ID, models.ProviderOfferInput{
		OfferName:           "PostgreSQL release offer",
		OfferSummary:        "Exercises capped prepaid accounting and late credits.",
		ActionType:          "lead",
		ActionURL:           "https://provider.example/start",
		ChargeEvent:         "accepted",
		BountyCents:         bountyCents,
		Currency:            "usd",
		PrincipalPriceMode:  "free",
		PrincipalPriceCents: &zero,
		PrincipalCurrency:   "usd",
		BillingMode:         "prepaid",
	})
	if err != nil {
		t.Fatalf("create provider offer: %v", err)
	}
	if _, err := models.FundProviderOffer(
		db, offer.ID, models.ProviderMoneyMaximumCents, "usd", "pg-release-initial-cap",
	); err != nil {
		t.Fatalf("fund provider offer to cap: %v", err)
	}
	offer, err = models.ActivateProviderOffer(
		db, offer.ID, "operator:pg-release", "evidence:pg-release",
	)
	if err != nil {
		t.Fatalf("activate provider offer: %v", err)
	}

	// Organic site membership alone is insufficient. The exact paid
	// offer/version must have been committed as returned evidence before a
	// ticket can be minted.
	missingEvidenceSearchID := recordPostgresDemandReceipt(t, db, site, "missing-evidence")
	_, _, _, err = models.CreateActionTicket(db, models.ActionTicketInput{
		ProviderOfferID:       offer.ID,
		SearchReceiptPublicID: missingEvidenceSearchID,
		DemandTopic:           "developer-tools",
		PrincipalConsent:      true,
		ConsentVersion:        models.ProviderPrincipalConsentV1,
		TTL:                   time.Hour,
	}, signer)
	if !errors.Is(err, models.ErrProviderOfferNotPublic) {
		t.Fatalf("ticket without exact returned-offer evidence error = %v, want ErrProviderOfferNotPublic", err)
	}
	assertPostgresOfferTicketCount(t, db, offer.ID, 0)

	staleOffer, err := models.CreateProviderOffer(db, accountID, claim.ID, models.ProviderOfferInput{
		OfferName:           "Version-bound disclosure offer",
		OfferSummary:        "Exercises exact paid-offer version evidence.",
		ActionType:          "trial",
		ActionURL:           "https://provider.example/versioned",
		ChargeEvent:         "accepted",
		BountyCents:         bountyCents,
		Currency:            "usd",
		PrincipalPriceMode:  "free",
		PrincipalPriceCents: &zero,
		PrincipalCurrency:   "usd",
		BillingMode:         "prepaid",
	})
	if err != nil {
		t.Fatalf("create version-bound offer: %v", err)
	}
	if _, err := models.FundProviderOffer(db, staleOffer.ID, bountyCents, "usd", "pg-release-version-fund"); err != nil {
		t.Fatalf("fund version-bound offer: %v", err)
	}
	staleOffer, err = models.ActivateProviderOffer(
		db, staleOffer.ID, "operator:pg-release", "evidence:pg-release-version",
	)
	if err != nil {
		t.Fatalf("activate version-bound offer: %v", err)
	}
	staleSearchID := recordPostgresDemandReceipt(t, db, site, "stale-version")
	recordPostgresReturnedOffer(t, db, site, staleSearchID, staleOffer.ID)
	// Simulate a migrated/corrected active offer changing after a disclosure.
	// Application edits are intentionally draft-only, but the ticket boundary
	// must still reject stale evidence if persisted state ever drifts.
	if _, err := db.Exec(`UPDATE provider_offers SET version=version+1 WHERE id=$1::uuid`, staleOffer.ID); err != nil {
		t.Fatalf("advance persisted offer version: %v", err)
	}
	_, _, _, err = models.CreateActionTicket(db, models.ActionTicketInput{
		ProviderOfferID:       staleOffer.ID,
		SearchReceiptPublicID: staleSearchID,
		DemandTopic:           "developer-tools",
		PrincipalConsent:      true,
		ConsentVersion:        models.ProviderPrincipalConsentV1,
		TTL:                   time.Hour,
	}, signer)
	if !errors.Is(err, models.ErrProviderOfferNotPublic) {
		t.Fatalf("ticket from stale offer-version disclosure error = %v, want ErrProviderOfferNotPublic", err)
	}
	assertPostgresOfferTicketCount(t, db, staleOffer.ID, 0)

	ticketOne := createPostgresActionTicket(t, db, signer, site, offer.ID, "sequence-one")
	chargedOne := recordPostgresOutcome(t, db, signer, providerKey, ticketOne.ID, "accepted", "sequence-one-charge")
	if chargedOne.ChargeStatus != string(providerexchange.ChargeStatusCharged) || chargedOne.BilledCents != bountyCents {
		t.Fatalf("first charge = status %q cents %d", chargedOne.ChargeStatus, chargedOne.BilledCents)
	}
	assertPostgresProviderBalance(t, db, offer.ID, models.ProviderMoneyMaximumCents-bountyCents)

	if _, err := models.FundProviderOffer(db, offer.ID, bountyCents, "usd", "pg-release-blocked-topup"); !errors.Is(err, models.ErrProviderBudgetLimit) {
		t.Fatalf("top-up before late credit error = %v, want ErrProviderBudgetLimit", err)
	}
	creditedOne := recordPostgresOutcome(t, db, signer, providerKey, ticketOne.ID, "invalid", "sequence-one-credit")
	if creditedOne.ChargeStatus != string(providerexchange.ChargeStatusCredited) || creditedOne.BilledCents != bountyCents {
		t.Fatalf("first late credit = status %q cents %d", creditedOne.ChargeStatus, creditedOne.BilledCents)
	}
	assertPostgresProviderBalance(t, db, offer.ID, models.ProviderMoneyMaximumCents)

	// Queue a top-up and late credit at the same time. Both code paths take the
	// same offer lock, so either database ordering must leave the credit usable,
	// reject the top-up, and preserve the hard balance cap.
	ticketTwo := createPostgresActionTicket(t, db, signer, site, offer.ID, "concurrent-two")
	recordPostgresOutcome(t, db, signer, providerKey, ticketTwo.ID, "accepted", "concurrent-two-charge")
	assertPostgresProviderBalance(t, db, offer.ID, models.ProviderMoneyMaximumCents-bountyCents)

	type concurrentResult struct {
		operation string
		receipt   *models.OutcomeReceipt
		err       error
	}
	start := make(chan struct{})
	results := make(chan concurrentResult, 2)
	go func() {
		<-start
		_, err := models.FundProviderOffer(db, offer.ID, bountyCents, "usd", "pg-release-concurrent-topup")
		results <- concurrentResult{operation: "topup", err: err}
	}()
	go func() {
		<-start
		receipt, _, err := models.RecordProviderOutcome(db, providerKey, models.ProviderOutcomeInput{
			ActionTicketID: ticketTwo.ID,
			IdempotencyKey: "pg-concurrent-credit-0001",
			PayloadHash:    postgresPayloadHash("concurrent-two-credit"),
			Outcome:        "invalid",
		}, signer)
		results <- concurrentResult{operation: "credit", receipt: receipt, err: err}
	}()
	close(start)
	var topupResult, creditResult concurrentResult
	for range 2 {
		result := <-results
		switch result.operation {
		case "topup":
			topupResult = result
		case "credit":
			creditResult = result
		default:
			t.Fatalf("unknown concurrent result %q", result.operation)
		}
	}
	if !errors.Is(topupResult.err, models.ErrProviderBudgetLimit) {
		t.Fatalf("concurrent top-up error = %v, want ErrProviderBudgetLimit", topupResult.err)
	}
	if creditResult.err != nil {
		t.Fatalf("concurrent late credit: %v", creditResult.err)
	}
	if creditResult.receipt == nil ||
		creditResult.receipt.ChargeStatus != string(providerexchange.ChargeStatusCredited) ||
		creditResult.receipt.BilledCents != bountyCents {
		t.Fatalf("concurrent late credit receipt = %#v", creditResult.receipt)
	}
	assertPostgresProviderBalance(t, db, offer.ID, models.ProviderMoneyMaximumCents)

	// A subsequent ticket can still charge, proving the cap/credit sequence did
	// not brick the active offer.
	ticketThree := createPostgresActionTicket(t, db, signer, site, offer.ID, "post-race-three")
	recordPostgresOutcome(t, db, signer, providerKey, ticketThree.ID, "accepted", "post-race-three-charge")
	assertPostgresProviderBalance(t, db, offer.ID, models.ProviderMoneyMaximumCents-bountyCents)

	// Pruning the query-free search receipt must cascade-delete its exact
	// returned-offer evidence while preserving the commercial ticket via SET
	// NULL and redacting every controlled intent field in the same operation.
	retentionSearchID := recordPostgresDemandReceipt(t, db, site, "retention-four")
	recordPostgresReturnedOffer(t, db, site, retentionSearchID, offer.ID)
	retentionTicket, _, _, err := models.CreateActionTicket(db, models.ActionTicketInput{
		ProviderOfferID:       offer.ID,
		SearchReceiptPublicID: retentionSearchID,
		DemandTopic:           "developer-tools",
		RegionCode:            "US-WA",
		BudgetBand:            "500_1999",
		Urgency:               "7_days",
		RequirementFlags:      []string{"api_access", "human_support"},
		PrincipalConsent:      true,
		ConsentVersion:        models.ProviderPrincipalConsentV1,
		TTL:                   time.Hour,
	}, signer)
	if err != nil {
		t.Fatalf("create retention-bound action ticket: %v", err)
	}
	var returnedBeforeDelete int
	if err := db.QueryRow(`
		SELECT COUNT(*)::int
		FROM provider_offers_returned
		WHERE search_receipt_id=$1::uuid`, retentionTicket.SearchReceiptID,
	).Scan(&returnedBeforeDelete); err != nil {
		t.Fatalf("count exact returned-offer evidence before receipt delete: %v", err)
	}
	if returnedBeforeDelete != 1 {
		t.Fatalf("exact returned-offer evidence before receipt delete = %d, want 1", returnedBeforeDelete)
	}
	if _, err := db.Exec(`DELETE FROM search_receipts WHERE public_id=$1`, retentionSearchID); err != nil {
		t.Fatalf("delete retention-bound search receipt: %v", err)
	}
	var (
		searchReceiptCleared bool
		demandTopic          string
		regionCode           string
		budgetBand           string
		urgency              string
		requirementCount     int
		intentRedacted       bool
		principalConsent     bool
		returnedAfterDelete  int
	)
	if err := db.QueryRow(`
		SELECT search_receipt_id IS NULL, demand_topic, region_code,
		       budget_band, urgency, cardinality(requirement_flags),
		       intent_redacted_at IS NOT NULL, principal_consent
		FROM action_tickets WHERE id=$1::uuid`, retentionTicket.ID,
	).Scan(
		&searchReceiptCleared, &demandTopic, &regionCode, &budgetBand,
		&urgency, &requirementCount, &intentRedacted, &principalConsent,
	); err != nil {
		t.Fatalf("read ticket after search-receipt pruning: %v", err)
	}
	if !searchReceiptCleared || demandTopic != "redacted" || regionCode != "" ||
		budgetBand != "unspecified" || urgency != "unspecified" ||
		requirementCount != 0 || !intentRedacted || !principalConsent {
		t.Fatalf(
			"ticket after receipt delete = cleared:%t topic:%q region:%q budget:%q urgency:%q flags:%d redacted:%t consent:%t",
			searchReceiptCleared, demandTopic, regionCode, budgetBand, urgency,
			requirementCount, intentRedacted, principalConsent,
		)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*)::int
		FROM provider_offers_returned
		WHERE search_receipt_id=$1::uuid`, retentionTicket.SearchReceiptID,
	).Scan(&returnedAfterDelete); err != nil {
		t.Fatalf("count exact returned-offer evidence after receipt delete: %v", err)
	}
	if returnedAfterDelete != 0 {
		t.Fatalf("exact returned-offer evidence after receipt delete = %d, want 0", returnedAfterDelete)
	}

	proofs, err = models.ProviderSigningKeyProofsInUse(db)
	if err != nil {
		t.Fatalf("signing-key proofs with persisted tickets and outcomes: %v", err)
	}
	if len(proofs) != 2 {
		t.Fatalf("signing-key proofs = %d, want one attribution and one outcome proof", len(proofs))
	}
	proofKinds := map[string]bool{}
	for _, proof := range proofs {
		if proof.KeyID != signer.ActiveKeyID() {
			t.Fatalf("proof key ID = %q, want %q", proof.KeyID, signer.ActiveKeyID())
		}
		proofKinds[proof.Kind] = true
		switch proof.Kind {
		case models.ProviderSigningProofAttribution:
			token, err := signer.SignAttribution(providerexchange.AttributionClaims{
				Version: providerexchange.AttributionTokenVersion,
				KeyID:   proof.KeyID, TicketID: proof.TicketID, OfferID: proof.OfferID,
				IssuedAt: proof.IssuedAt.Unix(), ExpiresAt: proof.ExpiresAt.Unix(),
				Nonce: proof.TokenNonce,
			})
			if err != nil {
				t.Fatalf("reconstruct attribution proof: %v", err)
			}
			if models.HashProviderSecret(token) != proof.TokenHash {
				t.Fatal("reconstructed attribution proof does not match persisted token hash")
			}
		case models.ProviderSigningProofOutcome:
			if _, err := signer.VerifyOutcomeReceiptSignature(proof.SignedReceipt, proof.Signature); err != nil {
				t.Fatalf("verify persisted outcome proof: %v", err)
			}
		default:
			t.Fatalf("unexpected signing proof kind %q", proof.Kind)
		}
	}
	if !proofKinds[models.ProviderSigningProofAttribution] || !proofKinds[models.ProviderSigningProofOutcome] {
		t.Fatalf("signing proof kinds = %#v", proofKinds)
	}

	// A legacy/directly persisted Unicode URL can be within the database's raw
	// byte cap yet expand beyond the final 2048-byte URL after percent encoding.
	// Ticket creation must preflight the exact final representation before its
	// transaction commits.
	unicodeURL := "https://provider.example/" + repeatPostgresRune('é', 400)
	unicodeOffer, err := models.CreateProviderOffer(db, accountID, claim.ID, models.ProviderOfferInput{
		OfferName:           "Unicode expansion offer",
		OfferSummary:        "Exercises final attributed URL length preflight.",
		ActionType:          "demo",
		ActionURL:           "https://provider.example/unicode",
		ChargeEvent:         "accepted",
		BountyCents:         bountyCents,
		Currency:            "usd",
		PrincipalPriceMode:  "free",
		PrincipalPriceCents: &zero,
		PrincipalCurrency:   "usd",
		BillingMode:         "prepaid",
	})
	if err != nil {
		t.Fatalf("persist raw-byte-bounded Unicode offer: %v", err)
	}
	if _, err := models.FundProviderOffer(db, unicodeOffer.ID, bountyCents, "usd", "pg-release-unicode-fund"); err != nil {
		t.Fatalf("fund Unicode offer: %v", err)
	}
	if _, err := models.ActivateProviderOffer(
		db, unicodeOffer.ID, "operator:pg-release", "evidence:pg-release-unicode",
	); err != nil {
		t.Fatalf("activate Unicode offer: %v", err)
	}
	// Simulate a legacy/imported row that predates normalized-length validation.
	// The database raw-byte constraint permits it; final URL encoding does not.
	if _, err := db.Exec(`UPDATE provider_offers SET action_url=$1 WHERE id=$2::uuid`, unicodeURL, unicodeOffer.ID); err != nil {
		t.Fatalf("persist legacy Unicode action URL: %v", err)
	}
	searchID := recordPostgresDemandReceipt(t, db, site, "unicode-four")
	recordPostgresReturnedOffer(t, db, site, searchID, unicodeOffer.ID)
	_, _, _, err = models.CreateActionTicket(db, models.ActionTicketInput{
		ProviderOfferID:       unicodeOffer.ID,
		SearchReceiptPublicID: searchID,
		DemandTopic:           "developer-tools",
		PrincipalConsent:      true,
		ConsentVersion:        models.ProviderPrincipalConsentV1,
		TTL:                   time.Hour,
	}, signer)
	if !errors.Is(err, models.ErrInvalidProviderExchange) {
		t.Fatalf("Unicode-expanded action URL ticket error = %v, want ErrInvalidProviderExchange", err)
	}
	var unicodeTicketCount int
	if err := db.QueryRow(
		`SELECT COUNT(*)::int FROM action_tickets WHERE provider_offer_id=$1::uuid`,
		unicodeOffer.ID,
	).Scan(&unicodeTicketCount); err != nil {
		t.Fatalf("count Unicode offer tickets: %v", err)
	}
	if unicodeTicketCount != 0 {
		t.Fatalf("Unicode-expanded action URL left %d committed ticket(s), want 0", unicodeTicketCount)
	}
}

func exerciseActionInterestPostgres(t *testing.T, db *sql.DB, site models.Site) {
	t.Helper()
	commercialBefore, err := models.GetProviderExchangeProof(db)
	if err != nil {
		t.Fatalf("commercial proof before action-interest tests: %v", err)
	}

	columnRows, err := db.Query(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema='public' AND table_name='action_interest_receipts'
		ORDER BY ordinal_position`)
	if err != nil {
		t.Fatalf("read action-interest catalog columns: %v", err)
	}
	var columns []string
	for columnRows.Next() {
		var column string
		if err := columnRows.Scan(&column); err != nil {
			_ = columnRows.Close()
			t.Fatalf("scan action-interest catalog column: %v", err)
		}
		columns = append(columns, column)
	}
	if err := columnRows.Err(); err != nil {
		_ = columnRows.Close()
		t.Fatalf("iterate action-interest catalog columns: %v", err)
	}
	if err := columnRows.Close(); err != nil {
		t.Fatalf("close action-interest catalog columns: %v", err)
	}
	wantColumns := []string{
		"id", "public_id", "search_receipt_id", "source_is_synthetic",
		"site_domain_snapshot", "action_type", "surface",
		"caller_attests_principal_interest", "confirmation_version",
		"created_at", "expires_at",
	}
	if !reflect.DeepEqual(columns, wantColumns) {
		t.Fatalf("action-interest catalog columns = %v, want %v", columns, wantColumns)
	}

	recordSearch := func(synthetic bool, returned []models.Site) string {
		t.Helper()
		searchID, err := models.GenerateDemandSearchID()
		if err != nil {
			t.Fatalf("generate action-interest search ID: %v", err)
		}
		if err := models.RecordDemandSearch(db, models.DemandSearchReceipt{
			PublicID: searchID, Surface: "rest", Category: "developer",
			ResultCount: len(returned), Page: 1, PageSize: 10, Synthetic: synthetic,
		}, returned); err != nil {
			t.Fatalf("record action-interest search: %v", err)
		}
		return searchID
	}
	inputFor := func(searchID, action string) models.ActionInterestInput {
		return models.ActionInterestInput{
			SearchID: searchID, Domain: site.Domain, ActionType: action,
			Surface: "rest", CallerAttestsPrincipalInterest: true,
			ConfirmationVersion: models.ActionInterestConfirmationV1,
		}
	}

	searchID := recordSearch(false, []models.Site{site})
	created, err := models.RecordActionInterest(db, inputFor(searchID, "quote"))
	if err != nil {
		t.Fatalf("create action interest: %v", err)
	}
	if created.Replayed || !strings.HasPrefix(created.PublicID, "nhs_air_") || created.Domain != site.Domain {
		t.Fatalf("new action-interest receipt = %#v", created)
	}
	if ttl := created.ExpiresAt.Sub(created.CreatedAt); ttl <= 29*24*time.Hour || ttl > 30*24*time.Hour {
		t.Fatalf("action-interest TTL = %v, want source-bounded <=30 days", ttl)
	}
	replayInput := inputFor(searchID, "quote")
	replayInput.Surface = "mcp"
	replayed, err := models.RecordActionInterest(db, replayInput)
	if err != nil {
		t.Fatalf("replay action interest: %v", err)
	}
	if !replayed.Replayed || replayed.PublicID != created.PublicID || replayed.Surface != "rest" {
		t.Fatalf("action-interest replay = %#v, want stable original %#v", replayed, created)
	}
	if _, err := models.RecordActionInterest(db, inputFor(searchID, "demo")); !errors.Is(err, models.ErrActionInterestConflict) {
		t.Fatalf("changed action-interest error = %v, want conflict", err)
	}

	for name, input := range map[string]models.ActionInterestInput{
		"false confirmation": func() models.ActionInterestInput {
			v := inputFor(searchID, "quote")
			v.CallerAttestsPrincipalInterest = false
			return v
		}(),
		"wrong version": func() models.ActionInterestInput {
			v := inputFor(searchID, "quote")
			v.ConfirmationVersion = "nhs-action-interest-v2"
			return v
		}(),
		"unknown action": inputFor(searchID, "lead"),
		"url domain": func() models.ActionInterestInput {
			v := inputFor(searchID, "quote")
			v.Domain = "https://" + site.Domain + "/private?token=secret"
			return v
		}(),
	} {
		if _, err := models.RecordActionInterest(db, input); !errors.Is(err, models.ErrInvalidActionInterest) {
			t.Fatalf("%s error = %v, want invalid action interest", name, err)
		}
	}

	if _, err := models.RecordActionInterest(db, inputFor(recordSearch(false, nil), "quote")); !errors.Is(err, models.ErrActionInterestUnavailable) {
		t.Fatalf("non-returned domain error = %v, want unavailable", err)
	}
	syntheticID := recordSearch(true, []models.Site{site})
	if _, err := models.RecordActionInterest(db, inputFor(syntheticID, "quote")); !errors.Is(err, models.ErrActionInterestUnavailable) {
		t.Fatalf("synthetic source error = %v, want unavailable", err)
	}
	if _, err := db.Exec(`
		INSERT INTO action_interest_receipts (
			public_id, search_receipt_id, source_is_synthetic,
			site_domain_snapshot, action_type, surface,
			caller_attests_principal_interest, confirmation_version, expires_at
		)
		SELECT 'nhs_air_AAAAAAAAAAAAAAAA', receipt.id, false,
		       returned.site_domain_snapshot, 'quote', 'rest', true,
		       'nhs-action-interest-v1', NOW() + INTERVAL '1 day'
		FROM search_receipts receipt
		JOIN organic_results_returned returned ON returned.search_receipt_id=receipt.id
		WHERE receipt.public_id=$1`, syntheticID); err == nil {
		t.Fatal("database accepted an action interest bound to a synthetic search receipt")
	} else {
		var pqErr *pq.Error
		if !errors.As(err, &pqErr) || string(pqErr.Code) != "23503" || pqErr.Constraint != "action_interest_receipts_non_synthetic_fk" {
			t.Fatalf("synthetic direct insert error = %#v, want 23503 action_interest_receipts_non_synthetic_fk", err)
		}
	}

	ttlConstraintSearchID := recordSearch(false, []models.Site{site})
	assertTTLConstraint := func(publicID, ttl, wantConstraint string) {
		t.Helper()
		_, err := db.Exec(`
			WITH fixture_clock AS (
				SELECT clock_timestamp() AS at
			)
			INSERT INTO action_interest_receipts (
				public_id, search_receipt_id, source_is_synthetic,
				site_domain_snapshot, action_type, surface,
				caller_attests_principal_interest, confirmation_version,
				created_at, expires_at
			)
			SELECT $2, receipt.id, false,
			       returned.site_domain_snapshot, 'quote', 'rest', true,
			       'nhs-action-interest-v1', fixture_clock.at,
			       fixture_clock.at + $3::interval
			FROM search_receipts receipt
			JOIN organic_results_returned returned
			  ON returned.search_receipt_id=receipt.id
			CROSS JOIN fixture_clock
			WHERE receipt.public_id=$1`, ttlConstraintSearchID, publicID, ttl)
		if err == nil {
			t.Fatalf("database accepted action-interest TTL %q; want %s violation", ttl, wantConstraint)
		}
		var pqErr *pq.Error
		if !errors.As(err, &pqErr) || string(pqErr.Code) != "23514" || pqErr.Constraint != wantConstraint {
			t.Fatalf("action-interest TTL %q error = %#v, want 23514 %s", ttl, err, wantConstraint)
		}
	}
	assertTTLConstraint(
		"nhs_air_BBBBBBBBBBBBBBBB",
		"0 seconds",
		"action_interest_receipts_positive_ttl",
	)
	assertTTLConstraint(
		"nhs_air_CCCCCCCCCCCCCCCC",
		"30 days 1 second",
		"action_interest_receipts_max_ttl",
	)

	staleID := recordSearch(false, []models.Site{site})
	if _, err := db.Exec(`UPDATE search_receipts SET created_at=NOW()-INTERVAL '31 days' WHERE public_id=$1`, staleID); err != nil {
		t.Fatalf("age stale search fixture: %v", err)
	}
	if _, err := models.RecordActionInterest(db, inputFor(staleID, "trial")); !errors.Is(err, models.ErrActionInterestUnavailable) {
		t.Fatalf("stale source error = %v, want unavailable", err)
	}

	disposableSite := models.Site{
		Domain: "action-interest-delete.example",
		URL:    "https://action-interest-delete.example",
	}
	if err := db.QueryRow(`
		INSERT INTO sites (domain, url, name, description, category, crawl_status)
		VALUES ($1,$2,'Disposable interest site','FK referential-action fixture','developer','success')
		RETURNING id::text`, disposableSite.Domain, disposableSite.URL).Scan(&disposableSite.ID); err != nil {
		t.Fatalf("insert disposable action-interest site: %v", err)
	}
	deleteSearchID := recordSearch(false, []models.Site{disposableSite})
	deleteReceipt, err := models.RecordActionInterest(db, models.ActionInterestInput{
		SearchID: deleteSearchID, Domain: disposableSite.Domain, ActionType: "quote",
		Surface: "rest", CallerAttestsPrincipalInterest: true,
		ConfirmationVersion: models.ActionInterestConfirmationV1,
	})
	if err != nil {
		t.Fatalf("create disposable-site action interest: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM sites WHERE id=$1::uuid`, disposableSite.ID); err != nil {
		t.Fatalf("delete site referenced by action-interest source: %v", err)
	}
	var remainingInterest int
	var returnedSiteID sql.NullString
	if err := db.QueryRow(`SELECT COUNT(*)::int FROM action_interest_receipts WHERE public_id=$1`, deleteReceipt.PublicID).Scan(&remainingInterest); err != nil {
		t.Fatalf("count action interest after site delete: %v", err)
	}
	if err := db.QueryRow(`
		SELECT returned.site_id::text
		FROM organic_results_returned returned
		JOIN search_receipts receipt ON receipt.id=returned.search_receipt_id
		WHERE receipt.public_id=$1 AND returned.site_domain_snapshot=$2`, deleteSearchID, disposableSite.Domain).Scan(&returnedSiteID); err != nil {
		t.Fatalf("read returned result after site delete: %v", err)
	}
	if remainingInterest != 1 || returnedSiteID.Valid {
		t.Fatalf("site delete result interest_count=%d returned_site_id=%v, want 1 and NULL", remainingInterest, returnedSiteID)
	}

	expirySearchID := recordSearch(false, []models.Site{site})
	if _, err := db.Exec(`
		UPDATE search_receipts
		SET created_at=clock_timestamp()-INTERVAL '30 days'+INTERVAL '2 seconds'
		WHERE public_id=$1`, expirySearchID); err != nil {
		t.Fatalf("prepare lock-across-expiry fixture: %v", err)
	}
	blockingTx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin expiry row lock: %v", err)
	}
	var returnedID int64
	if err := blockingTx.QueryRow(`
		SELECT returned.id
		FROM organic_results_returned returned
		JOIN search_receipts receipt ON receipt.id=returned.search_receipt_id
		WHERE receipt.public_id=$1 AND returned.site_domain_snapshot=$2
		FOR UPDATE OF returned`, expirySearchID, site.Domain).Scan(&returnedID); err != nil {
		_ = blockingTx.Rollback()
		t.Fatalf("lock returned result across expiry: %v", err)
	}
	expiryStarted := make(chan struct{})
	expiryResult := make(chan error, 1)
	go func() {
		close(expiryStarted)
		_, recordErr := models.RecordActionInterest(db, inputFor(expirySearchID, "trial"))
		expiryResult <- recordErr
	}()
	<-expiryStarted
	time.Sleep(2200 * time.Millisecond)
	if err := blockingTx.Commit(); err != nil {
		t.Fatalf("release expiry row lock: %v", err)
	}
	if err := <-expiryResult; !errors.Is(err, models.ErrActionInterestUnavailable) {
		t.Fatalf("lock-across-expiry error = %v, want unavailable", err)
	}
	var expiredInsertCount int
	if err := db.QueryRow(`
		SELECT COUNT(*)::int
		FROM action_interest_receipts interest
		JOIN search_receipts receipt ON receipt.id=interest.search_receipt_id
		WHERE receipt.public_id=$1`, expirySearchID).Scan(&expiredInsertCount); err != nil {
		t.Fatalf("count lock-across-expiry inserts: %v", err)
	}
	if expiredInsertCount != 0 {
		t.Fatalf("lock-across-expiry created %d receipt(s), want 0", expiredInsertCount)
	}

	concurrentID := recordSearch(false, []models.Site{site})
	start := make(chan struct{})
	type result struct {
		receipt *models.ActionInterestReceipt
		err     error
	}
	results := make(chan result, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			receipt, err := models.RecordActionInterest(db, inputFor(concurrentID, "application"))
			results <- result{receipt: receipt, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	var publicID string
	createdCount, replayCount := 0, 0
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent action interest: %v", result.err)
		}
		if publicID == "" {
			publicID = result.receipt.PublicID
		} else if result.receipt.PublicID != publicID {
			t.Fatalf("concurrent action interests returned %q and %q", publicID, result.receipt.PublicID)
		}
		if result.receipt.Replayed {
			replayCount++
		} else {
			createdCount++
		}
	}
	if createdCount != 1 || replayCount != 1 {
		t.Fatalf("concurrent action-interest outcomes created=%d replayed=%d", createdCount, replayCount)
	}

	cascadeID := recordSearch(false, []models.Site{site})
	cascadeReceipt, err := models.RecordActionInterest(db, inputFor(cascadeID, "booking"))
	if err != nil {
		t.Fatalf("create cascade action interest: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM search_receipts WHERE public_id=$1`, cascadeID); err != nil {
		t.Fatalf("delete cascade search receipt: %v", err)
	}
	var cascadeCount int
	if err := db.QueryRow(`SELECT COUNT(*)::int FROM action_interest_receipts WHERE public_id=$1`, cascadeReceipt.PublicID).Scan(&cascadeCount); err != nil {
		t.Fatalf("count cascaded action interest: %v", err)
	}
	if cascadeCount != 0 {
		t.Fatalf("search receipt deletion left %d action-interest rows", cascadeCount)
	}

	stage1, err := models.GetStage1DemandProof(db, 30)
	if err != nil {
		t.Fatalf("stage 1 demand proof: %v", err)
	}
	if stage1.CommercialProof || !stage1.SyntheticExcluded || !stage1.CountsAreReceiptsNotUniqueAgents ||
		stage1.SearchReceiptsWithActionInterest < 2 || stage1.ObservationWindowMet || stage1.Stage1Ready {
		t.Fatalf("stage 1 proof contract = %#v", stage1)
	}
	if len(stage1.ActionTypes) != 0 || len(stage1.DemandTopics) != 0 {
		t.Fatalf("below-threshold segmented buckets leaked: actions=%v topics=%v", stage1.ActionTypes, stage1.DemandTopics)
	}

	// Put the provider's quote cohort exactly at the privacy threshold so one
	// expired-but-not-yet-deleted row would be observable in both owner and
	// provider analytics if either query forgot the logical expiry predicate.
	var activeQuoteInterests int
	if err := db.QueryRow(`
		SELECT COUNT(*)::int
		FROM action_interest_receipts
		WHERE site_domain_snapshot=$1
		  AND action_type='quote'
		  AND expires_at > clock_timestamp()`, site.Domain).Scan(&activeQuoteInterests); err != nil {
		t.Fatalf("count active quote interests: %v", err)
	}
	for activeQuoteInterests < models.ProviderDemandPrivacyThreshold {
		thresholdSearchID := recordSearch(false, []models.Site{site})
		if _, err := models.RecordActionInterest(db, inputFor(thresholdSearchID, "quote")); err != nil {
			t.Fatalf("create threshold quote interest %d: %v", activeQuoteInterests+1, err)
		}
		activeQuoteInterests++
	}

	expiredFixtureSearchID := recordSearch(false, []models.Site{site})
	stage1BeforeExpiredFixture, err := models.GetStage1DemandProof(db, 30)
	if err != nil {
		t.Fatalf("stage 1 proof before expired fixture: %v", err)
	}
	providerBeforeExpiredFixture, err := models.GetProviderDemandAnalytics(db, site.Domain, 30)
	if err != nil {
		t.Fatalf("provider analytics before expired fixture: %v", err)
	}
	providerSummary, ok := providerBeforeExpiredFixture["summary"].(map[string]any)
	if !ok {
		t.Fatalf("provider analytics summary type = %T", providerBeforeExpiredFixture["summary"])
	}
	providerInterestCount, countOK := providerSummary["action_interest_receipts"].(int)
	if !countOK || providerInterestCount < models.ProviderDemandPrivacyThreshold || providerSummary["action_interest_suppressed"] != false {
		t.Fatalf("provider action-interest threshold fixture not observable: %#v", providerSummary)
	}
	providerActionTypes, ok := providerBeforeExpiredFixture["action_types"].([]map[string]any)
	if !ok {
		t.Fatalf("provider action_types type = %T", providerBeforeExpiredFixture["action_types"])
	}
	quoteBucketVisible := false
	for _, bucket := range providerActionTypes {
		if bucket["action_type"] == "quote" {
			count, ok := bucket["receipt_count"].(int)
			quoteBucketVisible = ok && count >= models.ProviderDemandPrivacyThreshold
			break
		}
	}
	if !quoteBucketVisible {
		t.Fatalf("provider quote threshold bucket missing: %#v", providerActionTypes)
	}

	if _, err := db.Exec(`
		WITH fixture_clock AS (
			SELECT clock_timestamp() AS at
		)
		INSERT INTO action_interest_receipts (
			public_id, search_receipt_id, source_is_synthetic,
			site_domain_snapshot, action_type, surface,
			caller_attests_principal_interest, confirmation_version,
			created_at, expires_at
		)
		SELECT 'nhs_air_DDDDDDDDDDDDDDDD', receipt.id, false,
		       returned.site_domain_snapshot, 'quote', 'rest', true,
		       'nhs-action-interest-v1', fixture_clock.at-INTERVAL '2 days',
		       fixture_clock.at-INTERVAL '1 day'
		FROM search_receipts receipt
		JOIN organic_results_returned returned
		  ON returned.search_receipt_id=receipt.id
		CROSS JOIN fixture_clock
		WHERE receipt.public_id=$1`, expiredFixtureSearchID); err != nil {
		t.Fatalf("insert logically expired action-interest fixture: %v", err)
	}
	var expiredFixtureRows int
	if err := db.QueryRow(`
		SELECT COUNT(*)::int
		FROM action_interest_receipts
		WHERE public_id='nhs_air_DDDDDDDDDDDDDDDD'
		  AND expires_at <= clock_timestamp()`).Scan(&expiredFixtureRows); err != nil {
		t.Fatalf("confirm physically present expired fixture: %v", err)
	}
	if expiredFixtureRows != 1 {
		t.Fatalf("physically present expired fixture rows = %d, want 1", expiredFixtureRows)
	}

	stage1AfterExpiredFixture, err := models.GetStage1DemandProof(db, 30)
	if err != nil {
		t.Fatalf("stage 1 proof after expired fixture: %v", err)
	}
	if !reflect.DeepEqual(stage1BeforeExpiredFixture, stage1AfterExpiredFixture) {
		t.Fatalf("expired row changed stage 1 analytics: before=%#v after=%#v", stage1BeforeExpiredFixture, stage1AfterExpiredFixture)
	}
	providerAfterExpiredFixture, err := models.GetProviderDemandAnalytics(db, site.Domain, 30)
	if err != nil {
		t.Fatalf("provider analytics after expired fixture: %v", err)
	}
	if !reflect.DeepEqual(providerBeforeExpiredFixture, providerAfterExpiredFixture) {
		t.Fatalf("expired row changed provider analytics: before=%#v after=%#v", providerBeforeExpiredFixture, providerAfterExpiredFixture)
	}

	commercialAfter, err := models.GetProviderExchangeProof(db)
	if err != nil {
		t.Fatalf("commercial proof after action-interest tests: %v", err)
	}
	if !reflect.DeepEqual(commercialBefore, commercialAfter) {
		t.Fatalf("action interests changed commercial proof: before=%#v after=%#v", commercialBefore, commercialAfter)
	}
}

func createPostgresProviderFixture(t *testing.T, db *sql.DB) (int64, models.Site) {
	t.Helper()
	var accountID int64
	if err := db.QueryRow(`
		INSERT INTO accounts (email, plan, status, monthly_limit)
		VALUES ('pg-release@example.invalid','provider-pilot','active',0)
		RETURNING id`).Scan(&accountID); err != nil {
		t.Fatalf("insert test account: %v", err)
	}
	site := models.Site{Domain: "provider.example", URL: "https://provider.example", AgenticScore: 90}
	if err := db.QueryRow(`
		INSERT INTO sites (
			domain, url, name, description, has_structured_api,
			agentic_score, category, crawl_status
		) VALUES ($1,$2,'Provider Example','PostgreSQL release fixture',true,$3,'developer','success')
		RETURNING id::text`, site.Domain, site.URL, site.AgenticScore).Scan(&site.ID); err != nil {
		t.Fatalf("insert test site: %v", err)
	}
	return accountID, site
}

func createPostgresActionTicket(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	site models.Site,
	offerID, suffix string,
) *models.ActionTicket {
	t.Helper()
	searchID := recordPostgresDemandReceipt(t, db, site, suffix)
	recordPostgresReturnedOffer(t, db, site, searchID, offerID)
	ticket, _, _, err := models.CreateActionTicket(db, models.ActionTicketInput{
		ProviderOfferID:       offerID,
		SearchReceiptPublicID: searchID,
		DemandTopic:           "developer-tools",
		PrincipalConsent:      true,
		ConsentVersion:        models.ProviderPrincipalConsentV1,
		TTL:                   time.Hour,
	}, signer)
	if err != nil {
		t.Fatalf("create action ticket %q: %v", suffix, err)
	}
	return ticket
}

func recordPostgresReturnedOffer(t *testing.T, db *sql.DB, site models.Site, searchID, offerID string) {
	t.Helper()
	offers, err := models.ListPublicProviderOffersForOrganicResults(db, []models.Site{site})
	if err != nil {
		t.Fatalf("list public provider offers for %q: %v", searchID, err)
	}
	for _, offer := range offers {
		if offer.OfferID != offerID {
			continue
		}
		if err := models.RecordProviderOffersReturned(db, searchID, []models.PublicProviderOffer{offer}); err != nil {
			t.Fatalf("record returned offer %q for %q: %v", offerID, searchID, err)
		}
		return
	}
	t.Fatalf("active offer %q was not eligible beside organic result %q", offerID, searchID)
}

func recordPostgresDemandReceipt(t *testing.T, db *sql.DB, site models.Site, suffix string) string {
	t.Helper()
	searchID := "nhs_sr_pg_release_" + suffix
	err := models.RecordDemandSearch(db, models.DemandSearchReceipt{
		PublicID: searchID, Surface: "rest", Category: "developer",
		ResultCount: 1, Page: 1, PageSize: 10,
	}, []models.Site{site})
	if err != nil {
		t.Fatalf("record demand receipt %q: %v", suffix, err)
	}
	return searchID
}

func recordPostgresOutcome(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	key *models.ProviderAPIKey,
	ticketID, outcome, suffix string,
) *models.OutcomeReceipt {
	t.Helper()
	receipt, created, err := models.RecordProviderOutcome(db, key, models.ProviderOutcomeInput{
		ActionTicketID: ticketID,
		IdempotencyKey: fmt.Sprintf("pg-%s-0001", suffix),
		PayloadHash:    postgresPayloadHash(suffix),
		Outcome:        outcome,
	}, signer)
	if err != nil {
		t.Fatalf("record %s outcome %q: %v", outcome, suffix, err)
	}
	if !created {
		t.Fatalf("record %s outcome %q unexpectedly replayed", outcome, suffix)
	}
	return receipt
}

func assertPostgresProviderBalance(t *testing.T, db *sql.DB, offerID string, want int64) {
	t.Helper()
	var got int64
	if err := db.QueryRow(`
		SELECT COALESCE(SUM(amount_cents),0)::bigint
		FROM provider_budget_ledger WHERE provider_offer_id=$1::uuid`, offerID).Scan(&got); err != nil {
		t.Fatalf("read provider balance: %v", err)
	}
	if got != want {
		t.Fatalf("provider balance = %d, want %d", got, want)
	}
	if got > models.ProviderMoneyMaximumCents {
		t.Fatalf("provider balance %d exceeds hard cap %d", got, models.ProviderMoneyMaximumCents)
	}
}

func assertPostgresOfferTicketCount(t *testing.T, db *sql.DB, offerID string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(
		`SELECT COUNT(*)::int FROM action_tickets WHERE provider_offer_id=$1::uuid`,
		offerID,
	).Scan(&got); err != nil {
		t.Fatalf("count offer tickets: %v", err)
	}
	if got != want {
		t.Fatalf("offer %s ticket count = %d, want %d", offerID, got, want)
	}
}

func postgresPayloadHash(label string) string {
	sum := sha256.Sum256([]byte(label))
	return hex.EncodeToString(sum[:])
}

func repeatPostgresRune(value rune, count int) string {
	runes := make([]rune, count)
	for i := range runes {
		runes[i] = value
	}
	return string(runes)
}
