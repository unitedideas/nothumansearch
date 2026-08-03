package models

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
)

func TestProviderPilotEnrollmentEligibilityConstraintsMapToBoundedError(t *testing.T) {
	for _, constraint := range []string{
		"provider_pilot_enrollment_fresh_company_claim",
		"provider_pilot_enrollment_stage1_topic_eligibility",
	} {
		err := mapProviderPilotEnrollmentError(&pq.Error{Constraint: constraint})
		if !errors.Is(err, ErrProviderPilotEnrollmentNotEligible) {
			t.Fatalf("constraint %q mapped to %v", constraint, err)
		}
	}
}

func TestProviderPilotEnrollmentResponseExposesOnlyOpaqueEligibilityProof(t *testing.T) {
	enrollment := ProviderPilotEnrollment{
		ID:                      "123e4567-e89b-42d3-a456-426614174000",
		ProviderPilotEpochID:    "123e4567-e89b-42d3-a456-426614174001",
		ProviderPilotCompanyID:  "123e4567-e89b-42d3-a456-426614174002",
		ProviderClaimID:         "123e4567-e89b-42d3-a456-426614174003",
		OwnerReference:          "owner:eligibility-test",
		EvidenceReference:       "evidence:eligibility-test",
		Stage1EligibilityStatus: "eligible",
		Stage1EligibilitySHA256: strings.Repeat("a", 64),
		EnrolledAt:              time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC),
	}
	encoded, err := json.Marshal(enrollment)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"stage1_eligibility_status", "stage1_eligibility_sha256",
	} {
		if _, ok := document[required]; !ok {
			t.Fatalf("enrollment response omitted %q: %s", required, encoded)
		}
	}
	for _, forbidden := range []string{
		"stage1_eligibility_site_id_snapshot",
		"stage1_eligibility_domain_sha256",
		"stage1_eligibility_bound_at",
		"search_receipt_id",
		"organic_position",
		"eligible_domain_count",
		"eligible_domains",
	} {
		if _, ok := document[forbidden]; ok {
			t.Fatalf("enrollment response exposed %q: %s", forbidden, encoded)
		}
	}
}

func readyProviderPilotStage1Proof() *Stage1DemandProof {
	startedAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	return &Stage1DemandProof{
		Days:                             30,
		RetentionDays:                    30,
		AsOf:                             startedAt.Add(15 * 24 * time.Hour),
		Stage1StartedAt:                  startedAt,
		Stage1EpochEnforced:              true,
		SyntheticExcluded:                true,
		CountsAreReceiptsNotUniqueAgents: true,
		CommercialProof:                  false,
		MeaningfulSearchReceipts:         120,
		ResultSelections:                 30,
		SearchReceiptsWithSelection:      25,
		ActionInterestReceipts:           14,
		SearchReceiptsWithActionInterest: 12,
		DistinctInterestDomains:          10,
		BucketReceiptThreshold:           20,
		TopicBucketsMayOverlap:           true,
		DemandTopics: []Stage1DemandBucket{
			{Value: "jobs", ReceiptCount: 32},
			{Value: "developer-tools", ReceiptCount: 24},
		},
		PilotCandidateTopics: []Stage1DemandBucket{
			{Value: "jobs", ReceiptCount: 32},
			{Value: "developer-tools", ReceiptCount: 24},
		},
		PilotCandidateTopicAvailable: true,
		ActionTypes: []Stage1DemandBucket{
			{Value: "application", ReceiptCount: 22},
			{Value: "demo", ReceiptCount: 20},
		},
		ObservationWindowDays:  Stage1ObservationWindowDays,
		ObservationSpanSeconds: int64(14 * 24 * time.Hour / time.Second),
		ObservationSpanDays:    14,
		ObservationWindowMet:   true,
		Stage1Ready:            true,
		Targets: map[string]int{
			"meaningful_search_receipts":           100,
			"search_receipts_with_selection":       20,
			"search_receipts_with_action_interest": 10,
			"pilot_candidate_topic_receipts":       20,
			"observation_window_days":              14,
		},
		TargetsMet: map[string]bool{
			"meaningful_search_receipts":           true,
			"search_receipts_with_selection":       true,
			"search_receipts_with_action_interest": true,
			"pilot_candidate_topic_receipts":       true,
			"observation_window_days":              true,
		},
	}
}

func TestProviderPilotStage1SnapshotIsDeterministicAndControlled(t *testing.T) {
	proof := readyProviderPilotStage1Proof()
	first, err := ProviderPilotStage1SnapshotSHA256(proof)
	if err != nil {
		t.Fatal(err)
	}
	proof.DemandTopics[0], proof.DemandTopics[1] = proof.DemandTopics[1], proof.DemandTopics[0]
	proof.PilotCandidateTopics[0], proof.PilotCandidateTopics[1] =
		proof.PilotCandidateTopics[1], proof.PilotCandidateTopics[0]
	proof.ActionTypes[0], proof.ActionTypes[1] = proof.ActionTypes[1], proof.ActionTypes[0]
	second, err := ProviderPilotStage1SnapshotSHA256(proof)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 64 || first != second {
		t.Fatalf("Stage 1 snapshot hashes first=%q second=%q", first, second)
	}

	unsafe := readyProviderPilotStage1Proof()
	unsafe.DemandTopics[0].Value = "jobs/private-query"
	if _, err := ProviderPilotStage1SnapshotSHA256(unsafe); !errors.Is(err, ErrInvalidProviderPilotSnapshot) {
		t.Fatalf("unsafe demand bucket error = %v", err)
	}
	unsafe = readyProviderPilotStage1Proof()
	unsafe.PilotCandidateTopics[0].Value = "other"
	if _, err := ProviderPilotStage1SnapshotSHA256(unsafe); !errors.Is(err, ErrInvalidProviderPilotSnapshot) {
		t.Fatalf("other candidate bucket error = %v", err)
	}
}

func TestProviderPilotEventSnapshotUsesOpaqueIDsOnly(t *testing.T) {
	pilotID := "123e4567-e89b-42d3-a456-426614174000"
	claimID := "123e4567-e89b-42d3-a456-426614174001"
	created, err := providerPilotEventSnapshotSHA256("created", pilotID, "")
	if err != nil || len(created) != 64 {
		t.Fatalf("created snapshot hash=%q err=%v", created, err)
	}
	enrolled, err := providerPilotEventSnapshotSHA256("provider_enrolled", pilotID, claimID)
	if err != nil || len(enrolled) != 64 || enrolled == created {
		t.Fatalf("enrollment snapshot hash=%q err=%v", enrolled, err)
	}
	for _, input := range []struct {
		eventType string
		claimID   string
	}{
		{eventType: "provider_enrolled"},
		{eventType: "created", claimID: claimID},
		{eventType: "provider@example.com", claimID: claimID},
	} {
		if _, err := providerPilotEventSnapshotSHA256(input.eventType, pilotID, input.claimID); !errors.Is(err, ErrInvalidProviderPilotSnapshot) {
			t.Fatalf("unsafe event %#v error = %v", input, err)
		}
	}
}

func TestProviderPilotEpochInputRequiresControlledNonOtherTopicAndBoundedCaps(t *testing.T) {
	valid := ProviderPilotEpochInput{
		DemandTopic:       " Jobs ",
		CohortLimit:       3,
		ProviderTicketCap: 3,
		TotalTicketCap:    5,
		OwnerReference:    "owner:pilot/2026-08-02",
		EvidenceReference: "evidence:pilot/2026-08-02",
	}
	normalized, err := normalizeProviderPilotEpochInput(valid)
	if err != nil || normalized.DemandTopic != "jobs" {
		t.Fatalf("normalized input=%#v err=%v", normalized, err)
	}
	for _, mutate := range []func(*ProviderPilotEpochInput){
		func(input *ProviderPilotEpochInput) { input.DemandTopic = "other" },
		func(input *ProviderPilotEpochInput) { input.DemandTopic = "raw customer request" },
		func(input *ProviderPilotEpochInput) { input.CohortLimit = 2 },
		func(input *ProviderPilotEpochInput) { input.CohortLimit = 21 },
		func(input *ProviderPilotEpochInput) { input.ProviderTicketCap = 101 },
		func(input *ProviderPilotEpochInput) { input.TotalTicketCap = 4 },
	} {
		input := valid
		mutate(&input)
		if _, err := normalizeProviderPilotEpochInput(input); !errors.Is(err, ErrInvalidProviderExchange) {
			t.Fatalf("invalid input %#v error = %v", input, err)
		}
	}
}

func TestProviderPilotEpochSourceEnforcesStage2Boundary(t *testing.T) {
	source, err := os.ReadFile("provider_pilot_epoch.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		"GetStage1DemandProof(db, 30)",
		"!proof.Stage1Ready",
		"proof.PilotCandidateTopics",
		"candidate.Value == input.DemandTopic",
		"'draft'",
		"JOIN provider_pilot_companies company",
		"providerClaimVerificationFresh",
		"FOR UPDATE OF claim",
		"FOR SHARE OF claim",
		"len(claims) < ProviderPilotMinimumCohort",
		"providerDatabaseClock(tx)",
		"status='active'",
		"status='closed'",
		"FOR UPDATE OF epoch, enrollment",
		"epoch.demand_topic=$2",
		"COUNT(*) FILTER (WHERE ticket.provider_claim_id=$2::uuid)",
		"AND NOT ticket.source_is_synthetic",
		"ticket.provider_pilot_epoch_id=$1::uuid",
		"appendProviderPilotEpochEvent",
		"provider_pilot_enrollment_eligibility_is_current",
		"ErrProviderPilotEnrollmentNotEligible",
		"stage1_eligibility_snapshot_sha256",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("provider pilot epoch source missing %q", required)
		}
	}
	start := strings.Index(text, "func providerPilotEventSnapshotSHA256")
	end := strings.Index(text[start:], "func appendProviderPilotEpochEvent")
	if start < 0 || end < 0 {
		t.Fatal("could not isolate provider pilot event snapshot hash")
	}
	hashSource := text[start : start+end]
	for _, forbidden := range []string{
		"ownerReference", "evidenceReference", "companyKey", "domain", "query", "principal", "agent",
	} {
		if strings.Contains(hashSource, forbidden) {
			t.Fatalf("event snapshot hash accepts forbidden value %q", forbidden)
		}
	}
	activationStart := strings.Index(text, "func ActivateProviderPilotEpoch")
	if activationStart < 0 {
		t.Fatal("could not find provider pilot activation")
	}
	activationEnd := strings.Index(text[activationStart:], "func CloseProviderPilotEpoch")
	if activationEnd < 0 {
		t.Fatal("could not isolate provider pilot activation")
	}
	activation := text[activationStart : activationStart+activationEnd]
	eligibilityCheck := strings.Index(activation, "!claim.eligibilityCurrent")
	reviewCheck := strings.Index(activation, "requireCurrentProviderPilotReview")
	if eligibilityCheck < 0 || reviewCheck < 0 || eligibilityCheck > reviewCheck ||
		strings.Count(activation[:reviewCheck], "for _, claim := range claims") < 2 {
		t.Fatal("provider pilot activation does not validate the whole eligibility cohort before provider reviews")
	}
}

func TestProviderPilotEpochModelMatchesMigration024(t *testing.T) {
	source, err := os.ReadFile("../../migrations/024_provider_pilot_boundary.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		"provider_pilot_epochs",
		"provider_pilot_enrollments",
		"provider_pilot_epoch_events",
		"provider_ticket_cap",
		"total_ticket_cap",
		"provider_pilot_epoch_id UUID",
		"status IN ('draft','active','closed')",
		"event_type IN (",
		"'created','provider_enrolled','activated','closed'",
		"provider_pilot_epoch_configuration_immutable",
		"provider_pilot_epoch_cohort_not_ready",
		"stage1_eligibility_site_id_snapshot UUID NOT NULL",
		"stage1_eligibility_domain_sha256 TEXT NOT NULL",
		"stage1_eligibility_snapshot_sha256 TEXT NOT NULL",
		"provider_pilot_stage1_eligibility_snapshot_sha256",
		"provider_pilot_enrollment_eligibility_is_current",
		"provider_pilot_enrollment_stage1_topic_eligibility",
		"site.category<>'spam'",
		"returned.returned_at >= GREATEST(",
		"returned.returned_at <= NEW.stage1_evidence_as_of",
		"returned.returned_at <= epoch_stage1_evidence_as_of",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("migration 024 missing %q", required)
		}
	}
	if got := strings.Count(text, "returned.returned_at >= GREATEST("); got < 9 {
		t.Fatalf("migration 024 participating-result lower bounds = %d, want at least 9", got)
	}
	if got := strings.Count(text, "returned.returned_at <= NEW.stage1_evidence_as_of"); got < 8 {
		t.Fatalf("migration 024 frozen Stage 1 result cutoffs = %d, want at least 8", got)
	}

	helperStart := strings.Index(text,
		"CREATE OR REPLACE FUNCTION public.provider_pilot_enrollment_eligibility_is_current(")
	if helperStart < 0 {
		t.Fatal("could not find the current enrollment-eligibility helper")
	}
	helperEnd := strings.Index(text[helperStart:],
		"CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_enrollment()")
	if helperEnd < 0 {
		t.Fatal("could not isolate the current enrollment-eligibility helper")
	}
	helper := text[helperStart : helperStart+helperEnd]
	for _, forbidden := range []string{
		"search_receipts", "organic_results_returned", "result_selections",
		"action_interest_receipts",
	} {
		if strings.Contains(helper, forbidden) {
			t.Fatalf("current enrollment-eligibility helper rereads retained Stage 1 relation %q", forbidden)
		}
	}
	for _, required := range []string{
		"stage1_eligibility_site_id_snapshot",
		"stage1_eligibility_domain_sha256",
		"stage1_eligibility_snapshot_sha256",
		"site.domain=claim.domain_snapshot",
		"site.category<>'spam'",
	} {
		if !strings.Contains(helper, required) {
			t.Fatalf("current enrollment-eligibility helper missing %q", required)
		}
	}
}

func TestProviderPilotPositiveOutcomeAndProofRequireCurrentEligibility(t *testing.T) {
	proofIntegrity, err := os.ReadFile("../../migrations/026_provider_pilot_proof_integrity.sql")
	if err != nil {
		t.Fatal(err)
	}
	outcomeSQL := string(proofIntegrity)
	for _, required := range []string{
		"IF NEW.outcome IN ('accepted','activated','converted') THEN",
		"provider_pilot_enrollment_eligibility_is_current",
		"provider_pilot_outcome_enrollment_eligibility",
		"FOR KEY SHARE OF claim, site",
	} {
		if !strings.Contains(outcomeSQL, required) {
			t.Fatalf("migration 026 positive-outcome eligibility gate missing %q", required)
		}
	}

	manifestMigration, err := os.ReadFile("../../migrations/028_provider_commercial_proof_manifest.sql")
	if err != nil {
		t.Fatal(err)
	}
	manifestSQL := string(manifestMigration)
	for _, required := range []string{
		"provider_pilot_enrollment_eligibility_is_current",
		"provider_proof_manifest_enrollment_eligibility",
	} {
		if !strings.Contains(manifestSQL, required) {
			t.Fatalf("migration 028 proof eligibility gate missing %q", required)
		}
	}
}

func TestProviderPilotStage1GenerationGateOwnsFrozenEpochWindow(t *testing.T) {
	integrityMigration, err := os.ReadFile("../../migrations/025_stage1_fact_integrity.sql")
	if err != nil {
		t.Fatal(err)
	}
	integritySQL := string(integrityMigration)
	for _, required := range []string{
		"CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_stage1_generation()",
		"NEW.stage1_started_at IS DISTINCT FROM authoritative_stage1_started_at",
		"receipt.stage1_integrity_generation IS DISTINCT FROM 1",
		"returned.stage1_integrity_generation IS DISTINCT FROM 1",
		"returned.returned_at >= GREATEST(",
		"returned.returned_at <= NEW.stage1_evidence_as_of",
		"provider_pilot_stage1_integrity_generation",
		"BEFORE INSERT ON public.provider_pilot_epochs",
	} {
		if !strings.Contains(integritySQL, required) {
			t.Fatalf("migration 025 frozen-window generation gate missing %q", required)
		}
	}
	if got := strings.Count(integritySQL, "returned.returned_at >= GREATEST("); got < 3 {
		t.Fatalf("migration 025 participating-result lower bounds = %d, want at least 3", got)
	}
	if got := strings.Count(integritySQL, "returned.returned_at <= NEW.stage1_evidence_as_of"); got < 3 {
		t.Fatalf("migration 025 participating-result cutoffs = %d, want at least 3", got)
	}
}
