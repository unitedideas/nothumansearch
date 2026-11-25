package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

const (
	ProviderPilotEpochContractV1        = "nhs-provider-pilot-v1"
	providerPilotStage1SnapshotV1       = "nhs-provider-pilot-stage1-snapshot-v1"
	providerPilotStage1SnapshotV2       = "nhs-provider-pilot-stage1-snapshot-v2"
	providerPilotEventSnapshotV1        = "nhs-provider-pilot-event-snapshot-v1"
	ProviderPilotMinimumCohort          = 3
	ProviderPilotMaximumCohort          = 20
	ProviderPilotMaximumProviderTickets = 100
	ProviderPilotMinimumTotalTickets    = 5
	ProviderPilotMaximumTotalTickets    = 2000
)

var (
	ErrProviderPilotStage1NotReady        = errors.New("Stage 1 demand proof is not ready for a provider pilot")
	ErrProviderPilotTopicNotCandidate     = errors.New("demand topic is not a Stage 1 pilot candidate")
	ErrProviderPilotNotDraft              = errors.New("provider pilot is not draft")
	ErrProviderPilotNotActive             = errors.New("provider pilot is not active for this provider")
	ErrProviderPilotCohortFull            = errors.New("provider pilot cohort is full")
	ErrProviderPilotCohortNotReady        = errors.New("provider pilot cohort is not ready")
	ErrProviderPilotEnrollmentConflict    = errors.New("provider pilot enrollment conflicts with the existing enrollment")
	ErrProviderPilotEnrollmentNotEligible = errors.New("provider claim does not have a current Stage 1 topic eligibility binding")
	ErrProviderPilotTicketCap             = errors.New("provider pilot ticket cap reached")
	ErrInvalidProviderPilotSnapshot       = errors.New("invalid provider pilot snapshot")
)

type ProviderPilotEpochInput struct {
	DemandTopic       string
	CohortLimit       int
	ProviderTicketCap int
	TotalTicketCap    int
	OwnerReference    string
	EvidenceReference string
}

type ProviderPilotEpoch struct {
	ID                   string     `json:"id"`
	ContractVersion      string     `json:"contract_version"`
	DemandTopic          string     `json:"demand_topic"`
	Stage1StartedAt      time.Time  `json:"stage1_started_at"`
	Stage1EvidenceAsOf   time.Time  `json:"stage1_evidence_as_of"`
	Stage1EvidenceSHA256 string     `json:"stage1_evidence_sha256"`
	CohortLimit          int        `json:"cohort_limit"`
	ProviderTicketCap    int        `json:"provider_ticket_cap"`
	TotalTicketCap       int        `json:"total_ticket_cap"`
	Status               string     `json:"status"`
	OwnerReference       string     `json:"owner_reference"`
	EvidenceReference    string     `json:"evidence_reference"`
	ActivatedAt          *time.Time `json:"activated_at,omitempty"`
	ClosedAt             *time.Time `json:"closed_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type ProviderPilotEnrollmentInput struct {
	ProviderPilotEpochID string
	ProviderClaimID      string
	OwnerReference       string
	EvidenceReference    string
}

type ProviderPilotEnrollment struct {
	ID                      string    `json:"id"`
	ProviderPilotEpochID    string    `json:"provider_pilot_epoch_id"`
	ProviderPilotCompanyID  string    `json:"provider_pilot_company_id"`
	ProviderClaimID         string    `json:"provider_claim_id"`
	OwnerReference          string    `json:"owner_reference"`
	EvidenceReference       string    `json:"evidence_reference"`
	Stage1EligibilityStatus string    `json:"stage1_eligibility_status"`
	Stage1EligibilitySHA256 string    `json:"stage1_eligibility_sha256"`
	EnrolledAt              time.Time `json:"enrolled_at"`
	Replayed                bool      `json:"-"`
}

type ProviderPilotEpochActionInput struct {
	ProviderPilotEpochID string
	OwnerReference       string
	EvidenceReference    string
}

type ProviderPilotEpochStatus struct {
	ProviderPilotEpoch
	AsOf                       time.Time `json:"as_of"`
	EnrolledProviderCount      int       `json:"enrolled_provider_count"`
	FreshEnrolledProviderCount int       `json:"fresh_enrolled_provider_count"`
	NonSyntheticTicketCount    int       `json:"non_synthetic_ticket_count"`
	RemainingTicketCapacity    int       `json:"remaining_ticket_capacity"`
	EventCount                 int       `json:"event_count"`
	CohortReady                bool      `json:"cohort_ready"`
}

const providerPilotEpochColumns = `
	id::text, contract_version, demand_topic, stage1_started_at,
	stage1_evidence_as_of, stage1_evidence_sha256, cohort_limit,
	provider_ticket_cap, total_ticket_cap, status, owner_reference,
	evidence_reference, activated_at, closed_at, created_at, updated_at`

const providerPilotEnrollmentColumns = `
	id::text, provider_pilot_epoch_id::text, provider_pilot_company_id::text,
	provider_claim_id::text, owner_reference, evidence_reference,
	stage1_eligibility_snapshot_sha256, enrolled_at`

var providerPilotStage1TargetNames = []string{
	"meaningful_search_receipts",
	"search_receipts_with_selection",
	"search_receipts_with_action_interest",
	"pilot_candidate_topic_receipts",
	"observation_window_days",
}

func scanProviderPilotEpoch(row rowScanner) (*ProviderPilotEpoch, error) {
	var epoch ProviderPilotEpoch
	var activatedAt, closedAt sql.NullTime
	if err := row.Scan(
		&epoch.ID, &epoch.ContractVersion, &epoch.DemandTopic,
		&epoch.Stage1StartedAt, &epoch.Stage1EvidenceAsOf,
		&epoch.Stage1EvidenceSHA256, &epoch.CohortLimit,
		&epoch.ProviderTicketCap, &epoch.TotalTicketCap, &epoch.Status,
		&epoch.OwnerReference, &epoch.EvidenceReference, &activatedAt,
		&closedAt, &epoch.CreatedAt, &epoch.UpdatedAt,
	); err != nil {
		return nil, err
	}
	epoch.ID = strings.ToLower(epoch.ID)
	if activatedAt.Valid {
		value := activatedAt.Time
		epoch.ActivatedAt = &value
	}
	if closedAt.Valid {
		value := closedAt.Time
		epoch.ClosedAt = &value
	}
	return &epoch, nil
}

func scanProviderPilotEnrollment(row rowScanner) (*ProviderPilotEnrollment, error) {
	var enrollment ProviderPilotEnrollment
	if err := row.Scan(
		&enrollment.ID, &enrollment.ProviderPilotEpochID,
		&enrollment.ProviderPilotCompanyID, &enrollment.ProviderClaimID,
		&enrollment.OwnerReference, &enrollment.EvidenceReference,
		&enrollment.Stage1EligibilitySHA256,
		&enrollment.EnrolledAt,
	); err != nil {
		return nil, err
	}
	enrollment.ID = strings.ToLower(enrollment.ID)
	enrollment.ProviderPilotEpochID = strings.ToLower(enrollment.ProviderPilotEpochID)
	enrollment.ProviderPilotCompanyID = strings.ToLower(enrollment.ProviderPilotCompanyID)
	enrollment.ProviderClaimID = strings.ToLower(enrollment.ProviderClaimID)
	enrollment.Stage1EligibilityStatus = "eligible"
	return &enrollment, nil
}

func mapProviderPilotEnrollmentError(err error) error {
	var postgresError *pq.Error
	if errors.As(err, &postgresError) {
		switch postgresError.Constraint {
		case "provider_pilot_enrollment_fresh_company_claim",
			"provider_pilot_enrollment_stage1_topic_eligibility":
			return ErrProviderPilotEnrollmentNotEligible
		}
	}
	return err
}

func normalizeProviderPilotEpochInput(input ProviderPilotEpochInput) (ProviderPilotEpochInput, error) {
	input.DemandTopic = strings.ToLower(strings.TrimSpace(input.DemandTopic))
	input.OwnerReference = strings.TrimSpace(input.OwnerReference)
	input.EvidenceReference = strings.TrimSpace(input.EvidenceReference)
	if input.DemandTopic == "other" || !providerDemandTopics[input.DemandTopic] ||
		input.CohortLimit < ProviderPilotMinimumCohort ||
		input.CohortLimit > ProviderPilotMaximumCohort ||
		input.ProviderTicketCap < 1 ||
		input.ProviderTicketCap > ProviderPilotMaximumProviderTickets ||
		input.TotalTicketCap < ProviderPilotMinimumTotalTickets ||
		input.TotalTicketCap > ProviderPilotMaximumTotalTickets ||
		input.TotalTicketCap < input.CohortLimit ||
		!validProviderReference(input.OwnerReference) ||
		!validProviderReference(input.EvidenceReference) {
		return ProviderPilotEpochInput{}, ErrInvalidProviderExchange
	}
	return input, nil
}

func normalizeProviderPilotEnrollmentInput(input ProviderPilotEnrollmentInput) (ProviderPilotEnrollmentInput, error) {
	input.ProviderPilotEpochID = strings.ToLower(strings.TrimSpace(input.ProviderPilotEpochID))
	input.ProviderClaimID = strings.ToLower(strings.TrimSpace(input.ProviderClaimID))
	input.OwnerReference = strings.TrimSpace(input.OwnerReference)
	input.EvidenceReference = strings.TrimSpace(input.EvidenceReference)
	if !validProviderUUID(input.ProviderPilotEpochID) ||
		!validProviderUUID(input.ProviderClaimID) ||
		!validProviderReference(input.OwnerReference) ||
		!validProviderReference(input.EvidenceReference) {
		return ProviderPilotEnrollmentInput{}, ErrInvalidProviderExchange
	}
	return input, nil
}

func normalizeProviderPilotEpochActionInput(input ProviderPilotEpochActionInput) (ProviderPilotEpochActionInput, error) {
	input.ProviderPilotEpochID = strings.ToLower(strings.TrimSpace(input.ProviderPilotEpochID))
	input.OwnerReference = strings.TrimSpace(input.OwnerReference)
	input.EvidenceReference = strings.TrimSpace(input.EvidenceReference)
	if !validProviderUUID(input.ProviderPilotEpochID) ||
		!validProviderReference(input.OwnerReference) ||
		!validProviderReference(input.EvidenceReference) {
		return ProviderPilotEpochActionInput{}, ErrInvalidProviderExchange
	}
	return input, nil
}

func appendProviderPilotCanonicalBucket(
	fields []string,
	label string,
	buckets []Stage1DemandBucket,
	candidateTopics bool,
	actionTypes bool,
) ([]string, error) {
	canonical := append([]Stage1DemandBucket(nil), buckets...)
	sort.Slice(canonical, func(i, j int) bool {
		if canonical[i].Value == canonical[j].Value {
			return canonical[i].ReceiptCount < canonical[j].ReceiptCount
		}
		return canonical[i].Value < canonical[j].Value
	})
	seen := map[string]bool{}
	for _, bucket := range canonical {
		value := strings.ToLower(strings.TrimSpace(bucket.Value))
		minimum := ProviderDemandPrivacyThreshold
		valid := providerDemandTopics[value]
		if candidateTopics {
			minimum = Stage1CandidateTopicReceipts
			valid = valid && value != "other"
		}
		if actionTypes {
			valid = ValidActionInterestType(value)
		}
		if bucket.Value != value || !valid || seen[value] || bucket.ReceiptCount < minimum {
			return nil, ErrInvalidProviderPilotSnapshot
		}
		seen[value] = true
		fields = append(fields, label, value, strconv.Itoa(bucket.ReceiptCount))
	}
	return fields, nil
}

// ProviderPilotStage1SnapshotSHA256 hashes only fixed aggregate fields,
// allowlisted bucket names, bounded counts, booleans, and database timestamps.
// No query, search receipt, domain, person, principal, agent, or network value
// is accepted into the canonical snapshot.
func ProviderPilotStage1SnapshotSHA256(proof *Stage1DemandProof) (string, error) {
	legacyPrefixSnapshot := proof != nil && len(proof.EligibleSurfaces) == 0
	if proof == nil || proof.AsOf.IsZero() || proof.Stage1StartedAt.IsZero() ||
		proof.AsOf.Before(proof.Stage1StartedAt) ||
		proof.BucketReceiptThreshold < ProviderDemandPrivacyThreshold ||
		!proof.Stage1EpochEnforced || !proof.SyntheticExcluded ||
		(!legacyPrefixSnapshot && (len(proof.EligibleSurfaces) != 2 ||
			proof.EligibleSurfaces[0] != "mcp" || proof.EligibleSurfaces[1] != "rest")) ||
		!proof.CountsAreReceiptsNotUniqueAgents || proof.CommercialProof {
		return "", ErrInvalidProviderPilotSnapshot
	}
	snapshotVersion := providerPilotStage1SnapshotV2
	if legacyPrefixSnapshot {
		// Migration-ledger prefix tests must still be able to construct the exact
		// v1 receipt enforced by migration 024 before migration 032 replaces the
		// database gate. Current production proofs always name MCP and REST and
		// therefore always use v2.
		snapshotVersion = providerPilotStage1SnapshotV1
	}
	fields := []string{
		snapshotVersion,
		"days", strconv.Itoa(proof.Days),
		"retention_days", strconv.Itoa(proof.RetentionDays),
		"as_of", strconv.FormatInt(proof.AsOf.UTC().UnixMicro(), 10),
		"stage1_started_at", strconv.FormatInt(proof.Stage1StartedAt.UTC().UnixMicro(), 10),
		"stage1_epoch_enforced", strconv.FormatBool(proof.Stage1EpochEnforced),
		"synthetic_excluded", strconv.FormatBool(proof.SyntheticExcluded),
	}
	if !legacyPrefixSnapshot {
		fields = append(fields,
			"eligible_surface", proof.EligibleSurfaces[0],
			"eligible_surface", proof.EligibleSurfaces[1])
	}
	fields = append(fields,
		"counts_are_receipts_not_unique_agents", strconv.FormatBool(proof.CountsAreReceiptsNotUniqueAgents),
		"commercial_proof", strconv.FormatBool(proof.CommercialProof),
		"meaningful_search_receipts", strconv.Itoa(proof.MeaningfulSearchReceipts),
		"result_selections", strconv.Itoa(proof.ResultSelections),
		"search_receipts_with_selection", strconv.Itoa(proof.SearchReceiptsWithSelection),
		"action_interest_receipts", strconv.Itoa(proof.ActionInterestReceipts),
		"search_receipts_with_action_interest", strconv.Itoa(proof.SearchReceiptsWithActionInterest),
		"distinct_interest_domains", strconv.Itoa(proof.DistinctInterestDomains),
		"bucket_receipt_threshold", strconv.Itoa(proof.BucketReceiptThreshold),
		"topic_buckets_may_overlap", strconv.FormatBool(proof.TopicBucketsMayOverlap),
		"pilot_candidate_topic_available", strconv.FormatBool(proof.PilotCandidateTopicAvailable),
		"observation_window_days", strconv.Itoa(proof.ObservationWindowDays),
		"observation_span_seconds", strconv.FormatInt(proof.ObservationSpanSeconds, 10),
		"observation_span_days", strconv.Itoa(proof.ObservationSpanDays),
		"observation_window_met", strconv.FormatBool(proof.ObservationWindowMet),
		"stage1_ready", strconv.FormatBool(proof.Stage1Ready),
	)
	var err error
	if fields, err = appendProviderPilotCanonicalBucket(fields, "demand_topic", proof.DemandTopics, false, false); err != nil {
		return "", err
	}
	if fields, err = appendProviderPilotCanonicalBucket(fields, "pilot_candidate_topic", proof.PilotCandidateTopics, true, false); err != nil {
		return "", err
	}
	if fields, err = appendProviderPilotCanonicalBucket(fields, "action_type", proof.ActionTypes, false, true); err != nil {
		return "", err
	}
	for _, name := range providerPilotStage1TargetNames {
		target, targetOK := proof.Targets[name]
		met, metOK := proof.TargetsMet[name]
		if !targetOK || !metOK || target < 0 {
			return "", ErrInvalidProviderPilotSnapshot
		}
		fields = append(fields, "target", name, strconv.Itoa(target), strconv.FormatBool(met))
	}
	// Every field is a fixed label, allowlisted enum, database count/boolean,
	// or Unix-microsecond timestamp, so newline is an unambiguous separator and
	// can be reproduced by the PostgreSQL insertion gate.
	sum := sha256.Sum256([]byte(strings.Join(fields, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

func providerPilotEventSnapshotSHA256(eventType, pilotID, claimID string) (string, error) {
	eventType = strings.TrimSpace(eventType)
	pilotID = strings.ToLower(strings.TrimSpace(pilotID))
	claimID = strings.ToLower(strings.TrimSpace(claimID))
	validEvent := eventType == "created" || eventType == "provider_enrolled" ||
		eventType == "activated" || eventType == "closed"
	if !validEvent || !validProviderUUID(pilotID) ||
		(eventType == "provider_enrolled") != validProviderUUID(claimID) ||
		(eventType != "provider_enrolled" && claimID != "") {
		return "", ErrInvalidProviderPilotSnapshot
	}
	// Deliberately hash only controlled contract/event labels and opaque UUIDs.
	// Evidence references and every provider identity attribute remain outside.
	sum := sha256.Sum256([]byte(strings.Join([]string{
		providerPilotEventSnapshotV1, eventType, pilotID, claimID,
	}, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

func appendProviderPilotEpochEvent(
	tx *sql.Tx,
	pilotID, eventType, claimID, ownerReference, evidenceReference string,
) error {
	if tx == nil || !validProviderReference(ownerReference) ||
		!validProviderReference(evidenceReference) {
		return ErrInvalidProviderExchange
	}
	snapshotSHA, err := providerPilotEventSnapshotSHA256(eventType, pilotID, claimID)
	if err != nil {
		return err
	}
	var providerClaimID any
	if claimID != "" {
		providerClaimID = claimID
	}
	_, err = tx.Exec(`
		INSERT INTO provider_pilot_epoch_events (
			provider_pilot_epoch_id, event_type, provider_claim_id,
			event_snapshot_sha256, owner_reference, evidence_reference
		) VALUES ($1::uuid,$2,$3::uuid,$4,$5,$6)`,
		pilotID, eventType, providerClaimID, snapshotSHA,
		ownerReference, evidenceReference)
	return err
}

func lockProviderPilotEpoch(tx *sql.Tx, pilotID string) (*ProviderPilotEpoch, error) {
	if tx == nil || !validProviderUUID(pilotID) {
		return nil, ErrInvalidProviderExchange
	}
	return scanProviderPilotEpoch(tx.QueryRow(`
		SELECT `+providerPilotEpochColumns+`
		FROM provider_pilot_epochs
		WHERE id=$1::uuid
		FOR UPDATE`, pilotID))
}

// CreateProviderPilotEpoch takes a fresh 30-day Stage 1 aggregate proof and
// will create a draft only for a controlled, non-other candidate topic.
func CreateProviderPilotEpoch(db *sql.DB, input ProviderPilotEpochInput) (*ProviderPilotEpoch, error) {
	if db == nil {
		return nil, ErrInvalidProviderExchange
	}
	input, err := normalizeProviderPilotEpochInput(input)
	if err != nil {
		return nil, err
	}
	proof, err := GetStage1DemandProof(db, 30)
	if err != nil {
		return nil, err
	}
	if proof == nil || !proof.Stage1Ready ||
		proof.AsOf.Before(proof.Stage1StartedAt.Add(Stage1ObservationWindowDays*24*time.Hour)) {
		return nil, ErrProviderPilotStage1NotReady
	}
	topicCandidate := false
	for _, candidate := range proof.PilotCandidateTopics {
		if candidate.Value == input.DemandTopic && candidate.ReceiptCount >= Stage1CandidateTopicReceipts {
			topicCandidate = true
			break
		}
	}
	if !topicCandidate {
		return nil, ErrProviderPilotTopicNotCandidate
	}
	stage1SnapshotSHA, err := ProviderPilotStage1SnapshotSHA256(proof)
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	epoch, err := scanProviderPilotEpoch(tx.QueryRow(`
		INSERT INTO provider_pilot_epochs (
			contract_version, demand_topic, stage1_started_at,
			stage1_evidence_as_of, stage1_evidence_sha256, cohort_limit,
			provider_ticket_cap, total_ticket_cap, status,
			owner_reference, evidence_reference
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'draft',$9,$10)
		RETURNING `+providerPilotEpochColumns,
		ProviderPilotEpochContractV1, input.DemandTopic,
		proof.Stage1StartedAt, proof.AsOf, stage1SnapshotSHA,
		input.CohortLimit, input.ProviderTicketCap, input.TotalTicketCap,
		input.OwnerReference, input.EvidenceReference))
	if err != nil {
		return nil, err
	}
	if err := appendProviderPilotEpochEvent(
		tx, epoch.ID, "created", "", input.OwnerReference, input.EvidenceReference,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return epoch, nil
}

// EnrollProviderPilotCompany locks the draft epoch as the cohort mutex, then
// binds the exact immutable pilot-company/claim pair only while DNS ownership
// is verified and fresh at a database-clock boundary.
func EnrollProviderPilotCompany(db *sql.DB, input ProviderPilotEnrollmentInput) (*ProviderPilotEnrollment, error) {
	if db == nil {
		return nil, ErrInvalidProviderExchange
	}
	input, err := normalizeProviderPilotEnrollmentInput(input)
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	epoch, err := lockProviderPilotEpoch(tx, input.ProviderPilotEpochID)
	if err != nil {
		return nil, err
	}
	if epoch.Status != "draft" {
		return nil, ErrProviderPilotNotDraft
	}
	var claimStatus, companyID string
	var lastSucceededAt sql.NullTime
	if err := tx.QueryRow(`
		SELECT claim.status, claim.verification_last_succeeded_at,
		       company.id::text
		FROM provider_claims claim
		JOIN provider_pilot_companies company
		  ON company.provider_claim_id=claim.id
		WHERE claim.id=$1::uuid
		FOR UPDATE OF claim`, input.ProviderClaimID).
		Scan(&claimStatus, &lastSucceededAt, &companyID); err != nil {
		return nil, err
	}
	verifiedAt, err := providerDatabaseClock(tx)
	if err != nil {
		return nil, err
	}
	if claimStatus != "verified" {
		return nil, ErrProviderClaimNotVerified
	}
	if !lastSucceededAt.Valid || !providerClaimVerificationFresh(lastSucceededAt.Time, verifiedAt) {
		return nil, ErrProviderClaimVerificationStale
	}
	companyID = strings.ToLower(companyID)
	if !validProviderUUID(companyID) {
		return nil, ErrInvalidProviderExchange
	}
	existing, err := scanProviderPilotEnrollment(tx.QueryRow(`
		SELECT `+providerPilotEnrollmentColumns+`
		FROM provider_pilot_enrollments
		WHERE provider_pilot_epoch_id=$1::uuid
		  AND provider_claim_id=$2::uuid`,
		input.ProviderPilotEpochID, input.ProviderClaimID))
	if err == nil {
		if existing.ProviderPilotCompanyID != companyID ||
			existing.OwnerReference != input.OwnerReference ||
			existing.EvidenceReference != input.EvidenceReference {
			return nil, ErrProviderPilotEnrollmentConflict
		}
		var eligibilityCurrent bool
		if err := tx.QueryRow(`
			SELECT provider_pilot_enrollment_eligibility_is_current($1::uuid,$2::uuid)`,
			input.ProviderPilotEpochID, input.ProviderClaimID).Scan(&eligibilityCurrent); err != nil {
			return nil, err
		}
		if !eligibilityCurrent {
			return nil, ErrProviderPilotEnrollmentNotEligible
		}
		existing.Replayed = true
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	var enrolledCount int
	if err := tx.QueryRow(`
		SELECT COUNT(*)::int
		FROM provider_pilot_enrollments
		WHERE provider_pilot_epoch_id=$1::uuid`, input.ProviderPilotEpochID).
		Scan(&enrolledCount); err != nil {
		return nil, err
	}
	if enrolledCount >= epoch.CohortLimit {
		return nil, ErrProviderPilotCohortFull
	}
	enrollment, err := scanProviderPilotEnrollment(tx.QueryRow(`
		INSERT INTO provider_pilot_enrollments (
			provider_pilot_epoch_id, provider_pilot_company_id,
			provider_claim_id, owner_reference, evidence_reference
		) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5)
		RETURNING `+providerPilotEnrollmentColumns,
		input.ProviderPilotEpochID, companyID, input.ProviderClaimID,
		input.OwnerReference, input.EvidenceReference))
	if err != nil {
		return nil, mapProviderPilotEnrollmentError(err)
	}
	if err := appendProviderPilotEpochEvent(
		tx, epoch.ID, "provider_enrolled", input.ProviderClaimID,
		input.OwnerReference, input.EvidenceReference,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return enrollment, nil
}

type providerPilotActivationClaim struct {
	id                 string
	status             string
	lastSucceededAt    sql.NullTime
	eligibilityCurrent bool
}

// ActivateProviderPilotEpoch re-locks every enrolled claim before consulting
// the database clock. The immutable cohort must contain 3 through its limit,
// and every exact company/claim pair must still be verified and fresh.
func ActivateProviderPilotEpoch(db *sql.DB, input ProviderPilotEpochActionInput) (*ProviderPilotEpoch, error) {
	if db == nil {
		return nil, ErrInvalidProviderExchange
	}
	input, err := normalizeProviderPilotEpochActionInput(input)
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	epoch, err := lockProviderPilotEpoch(tx, input.ProviderPilotEpochID)
	if err != nil {
		return nil, err
	}
	if epoch.Status != "draft" {
		return nil, ErrProviderPilotNotDraft
	}
	rows, err := tx.Query(`
		SELECT claim.id::text, claim.status, claim.verification_last_succeeded_at,
		       provider_pilot_enrollment_eligibility_is_current(
		           enrollment.provider_pilot_epoch_id, enrollment.provider_claim_id
		       )
		FROM provider_pilot_enrollments enrollment
		JOIN provider_pilot_companies company
		  ON company.id=enrollment.provider_pilot_company_id
		 AND company.provider_claim_id=enrollment.provider_claim_id
		JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
		WHERE enrollment.provider_pilot_epoch_id=$1::uuid
		ORDER BY claim.id
		FOR SHARE OF claim`, input.ProviderPilotEpochID)
	if err != nil {
		return nil, err
	}
	claims := []providerPilotActivationClaim{}
	for rows.Next() {
		var claim providerPilotActivationClaim
		if err := rows.Scan(
			&claim.id, &claim.status, &claim.lastSucceededAt, &claim.eligibilityCurrent,
		); err != nil {
			_ = rows.Close()
			return nil, err
		}
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(claims) < ProviderPilotMinimumCohort || len(claims) > epoch.CohortLimit {
		return nil, ErrProviderPilotCohortNotReady
	}
	activatedAt, err := providerDatabaseClock(tx)
	if err != nil {
		return nil, err
	}
	for _, claim := range claims {
		if claim.status != "verified" || !claim.lastSucceededAt.Valid ||
			!claim.eligibilityCurrent ||
			!providerClaimVerificationFresh(claim.lastSucceededAt.Time, activatedAt) {
			return nil, ErrProviderPilotCohortNotReady
		}
	}
	// Validate the whole immutable cohort before consulting review receipts so
	// claim/site drift cannot be hidden by the sort order of an unrelated
	// provider that still needs review.
	for _, claim := range claims {
		if err := requireCurrentProviderPilotReview(
			tx, epoch.ID, "provider", claim.id,
		); err != nil {
			return nil, err
		}
	}
	epoch, err = scanProviderPilotEpoch(tx.QueryRow(`
		UPDATE provider_pilot_epochs
		SET status='active', activated_at=$2::timestamptz
		WHERE id=$1::uuid AND status='draft'
		RETURNING `+providerPilotEpochColumns,
		input.ProviderPilotEpochID, activatedAt))
	if err != nil {
		return nil, err
	}
	if err := appendProviderPilotEpochEvent(
		tx, epoch.ID, "activated", "", input.OwnerReference, input.EvidenceReference,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return epoch, nil
}

func CloseProviderPilotEpoch(db *sql.DB, input ProviderPilotEpochActionInput) (*ProviderPilotEpoch, error) {
	if db == nil {
		return nil, ErrInvalidProviderExchange
	}
	input, err := normalizeProviderPilotEpochActionInput(input)
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	epoch, err := lockProviderPilotEpoch(tx, input.ProviderPilotEpochID)
	if err != nil {
		return nil, err
	}
	if epoch.Status != "active" {
		return nil, ErrProviderPilotNotActive
	}
	closedAt, err := providerDatabaseClock(tx)
	if err != nil {
		return nil, err
	}
	epoch, err = scanProviderPilotEpoch(tx.QueryRow(`
		UPDATE provider_pilot_epochs
		SET status='closed', closed_at=$2::timestamptz
		WHERE id=$1::uuid AND status='active'
		RETURNING `+providerPilotEpochColumns,
		input.ProviderPilotEpochID, closedAt))
	if err != nil {
		return nil, err
	}
	if err := appendProviderPilotEpochEvent(
		tx, epoch.ID, "closed", "", input.OwnerReference, input.EvidenceReference,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return epoch, nil
}

// GetProviderPilotEpochStatus returns only epoch configuration and aggregate
// counts. It does not expose company hashes, domains, searches, principals,
// agents, contacts, network data, or event evidence references.
func GetProviderPilotEpochStatus(db *sql.DB, pilotID string) (*ProviderPilotEpochStatus, error) {
	pilotID = strings.ToLower(strings.TrimSpace(pilotID))
	if db == nil || !validProviderUUID(pilotID) {
		return nil, ErrInvalidProviderExchange
	}
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var asOf time.Time
	if err := tx.QueryRow(`SELECT clock_timestamp()`).Scan(&asOf); err != nil {
		return nil, err
	}
	epoch, err := scanProviderPilotEpoch(tx.QueryRow(`
		SELECT `+providerPilotEpochColumns+`
		FROM provider_pilot_epochs
		WHERE id=$1::uuid`, pilotID))
	if err != nil {
		return nil, err
	}
	status := &ProviderPilotEpochStatus{
		ProviderPilotEpoch: *epoch,
		AsOf:               asOf,
	}
	if err := tx.QueryRow(`
		SELECT
		  (SELECT COUNT(*)::int
		   FROM provider_pilot_enrollments enrollment
		   WHERE enrollment.provider_pilot_epoch_id=$1::uuid),
		  (SELECT COUNT(*)::int
		   FROM provider_pilot_enrollments enrollment
		   JOIN provider_pilot_companies company
		     ON company.id=enrollment.provider_pilot_company_id
		    AND company.provider_claim_id=enrollment.provider_claim_id
		   JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
		   WHERE enrollment.provider_pilot_epoch_id=$1::uuid
		     AND claim.status='verified'
		     AND provider_pilot_enrollment_eligibility_is_current(
		         enrollment.provider_pilot_epoch_id, enrollment.provider_claim_id
		     )
		     AND claim.verification_last_succeeded_at >
		         $2::timestamptz - $3::bigint * INTERVAL '1 second'),
		  (SELECT COUNT(*)::int
		   FROM action_tickets ticket
		   WHERE ticket.provider_pilot_epoch_id=$1::uuid
		     AND NOT ticket.source_is_synthetic),
		  (SELECT COUNT(*)::int
		   FROM provider_pilot_epoch_events event
		   WHERE event.provider_pilot_epoch_id=$1::uuid)`,
		pilotID, asOf, int64(ProviderClaimVerificationFreshness/time.Second)).Scan(
		&status.EnrolledProviderCount, &status.FreshEnrolledProviderCount,
		&status.NonSyntheticTicketCount, &status.EventCount,
	); err != nil {
		return nil, err
	}
	status.RemainingTicketCapacity = epoch.TotalTicketCap - status.NonSyntheticTicketCount
	if status.RemainingTicketCapacity < 0 {
		status.RemainingTicketCapacity = 0
	}
	status.CohortReady = status.EnrolledProviderCount >= ProviderPilotMinimumCohort &&
		status.EnrolledProviderCount <= epoch.CohortLimit &&
		status.FreshEnrolledProviderCount == status.EnrolledProviderCount
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return status, nil
}

// requireActiveProviderPilotEnrollment is deliberately transaction-scoped so
// offer activation and pilot close serialize on the same epoch row.
func requireActiveProviderPilotEnrollment(tx *sql.Tx, claimID string) (pilotID string, demandTopic string, err error) {
	claimID = strings.ToLower(strings.TrimSpace(claimID))
	if tx == nil || !validProviderUUID(claimID) {
		return "", "", ErrInvalidProviderExchange
	}
	err = tx.QueryRow(`
		SELECT epoch.id::text, epoch.demand_topic
		FROM provider_pilot_epochs epoch
		JOIN provider_pilot_enrollments enrollment
		  ON enrollment.provider_pilot_epoch_id=epoch.id
		JOIN provider_pilot_companies company
		  ON company.id=enrollment.provider_pilot_company_id
		 AND company.provider_claim_id=enrollment.provider_claim_id
		WHERE epoch.status='active'
		  AND enrollment.provider_claim_id=$1::uuid
		  AND provider_pilot_enrollment_eligibility_is_current(
		      epoch.id, enrollment.provider_claim_id
		  )
		FOR UPDATE OF epoch, enrollment`, claimID).Scan(&pilotID, &demandTopic)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrProviderPilotNotActive
	}
	if err != nil {
		return "", "", err
	}
	pilotID = strings.ToLower(pilotID)
	if !validProviderUUID(pilotID) || demandTopic == "other" || !providerDemandTopics[demandTopic] {
		return "", "", ErrProviderPilotNotActive
	}
	return pilotID, demandTopic, nil
}

// enforceProviderPilotTicketBoundary locks the active epoch and exact
// enrollment before counting. Holding the epoch row through the caller's
// ticket insert makes the lifetime total cap safe across all enrolled claims;
// the enrollment lock protects the exact per-provider boundary. The caller
// must pass a database-issued clock read inside this transaction.
func enforceProviderPilotTicketBoundary(tx *sql.Tx, claimID, demandTopic string, now time.Time) (pilotID string, err error) {
	claimID = strings.ToLower(strings.TrimSpace(claimID))
	demandTopic = strings.ToLower(strings.TrimSpace(demandTopic))
	if tx == nil || !validProviderUUID(claimID) || now.IsZero() ||
		demandTopic == "other" || !providerDemandTopics[demandTopic] {
		return "", ErrInvalidProviderExchange
	}
	var providerTicketCap, totalTicketCap int
	err = tx.QueryRow(`
		SELECT epoch.id::text, epoch.provider_ticket_cap, epoch.total_ticket_cap
		FROM provider_pilot_epochs epoch
		JOIN provider_pilot_enrollments enrollment
		  ON enrollment.provider_pilot_epoch_id=epoch.id
		JOIN provider_pilot_companies company
		  ON company.id=enrollment.provider_pilot_company_id
		 AND company.provider_claim_id=enrollment.provider_claim_id
		JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
		WHERE epoch.status='active'
		  AND epoch.demand_topic=$2
		  AND epoch.activated_at IS NOT NULL
		  AND epoch.activated_at <= $3::timestamptz
		  AND enrollment.provider_claim_id=$1::uuid
		  AND provider_pilot_enrollment_eligibility_is_current(
		      epoch.id, enrollment.provider_claim_id
		  )
		  AND claim.status='verified'
		  AND claim.verification_last_succeeded_at >
		      $3::timestamptz - $4::bigint * INTERVAL '1 second'
		FOR UPDATE OF epoch, enrollment`,
		claimID, demandTopic, now.UTC(),
		int64(ProviderClaimVerificationFreshness/time.Second)).Scan(
		&pilotID, &providerTicketCap, &totalTicketCap,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrProviderPilotNotActive
	}
	if err != nil {
		return "", err
	}
	pilotID = strings.ToLower(pilotID)
	if !validProviderUUID(pilotID) {
		return "", ErrProviderPilotNotActive
	}
	var providerTicketCount, totalTicketCount int
	if err := tx.QueryRow(`
		SELECT
		  COUNT(*) FILTER (WHERE ticket.provider_claim_id=$2::uuid)::int,
		  COUNT(*)::int
		FROM action_tickets ticket
		WHERE ticket.provider_pilot_epoch_id=$1::uuid
		  AND NOT ticket.source_is_synthetic`, pilotID, claimID).Scan(
		&providerTicketCount, &totalTicketCount,
	); err != nil {
		return "", err
	}
	if providerTicketCount >= providerTicketCap || totalTicketCount >= totalTicketCap {
		return "", ErrProviderPilotTicketCap
	}
	return pilotID, nil
}
