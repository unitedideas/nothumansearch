package models

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestValidateProviderPilotQueueState(t *testing.T) {
	t.Parallel()
	for _, state := range []string{
		"", "all", "review_required", "pre_event_review_required",
		"provider_review_required", "offer_review_required",
		"ticket_review_required", "handoff_review_required",
		"callback_review_required", " pending_company ", "pending_terms", "activation_review",
		"expiring_terms", "handoff_awaiting_callback", "recent_callback",
	} {
		if err := ValidateProviderPilotQueueState(state); err != nil {
			t.Fatalf("valid state %q rejected: %v", state, err)
		}
	}
	for _, state := range []string{"first_handoff", "raw_queries", "provider_key", "everything"} {
		if err := ValidateProviderPilotQueueState(state); err == nil {
			t.Fatalf("invalid state %q accepted", state)
		}
	}
}

func TestProviderPilotReviewRequiredQueueCarriesOnlyBoundedReviewCoordinates(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(&ProviderPilotQueueItem{
		State:                 "ticket_review_required",
		ProviderPilotEpochID:  "623e4567-e89b-42d3-a456-426614174000",
		ProviderClaimID:       "123e4567-e89b-42d3-a456-426614174000",
		Domain:                "provider.example",
		TicketID:              "423e4567-e89b-42d3-a456-426614174000",
		ReviewType:            "ticket",
		SubjectID:             "423e4567-e89b-42d3-a456-426614174000",
		SubjectSnapshotSHA256: strings.Repeat("a", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, required := range []string{
		`"provider_pilot_epoch_id"`, `"review_type":"ticket"`,
		`"subject_id"`, `"subject_snapshot_sha256"`,
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("review queue item missing %s: %s", required, text)
		}
	}
	for _, forbidden := range []string{
		"query", "attribution_token", "action_url", "contact", "agent_identity",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("review queue item exposed %s: %s", forbidden, text)
		}
	}
}

func TestProviderPilotQueueNamesBoundedCountsTruthfully(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(&ProviderPilotQueue{
		ReturnedCounts: map[string]int{"pending_terms": 25},
		Items:          []ProviderPilotQueueItem{},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	if !strings.Contains(text, `"returned_counts":{"pending_terms":25}`) {
		t.Fatalf("returned-count contract missing: %s", text)
	}
	if strings.Contains(text, `"counts":`) {
		t.Fatalf("bounded returned items misrepresented as total counts: %s", text)
	}
}

func TestProviderPilotActivationSurfacesFailClosedOnUnverifiedLedgerRows(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("provider_pilot_status.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		"AS activation_ready",
		`ProviderPilotQueueItem{State: "activation_review"}`,
		"FROM provider_budget_ledger unverified",
		"linked.budget_ledger_entry_id=unverified.id",
		"linked.event_type='prepaid_fund'",
		"linked.event_type='fund_reversal'",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("pilot readiness surface missing contamination guard %q", required)
		}
	}
	if strings.Count(text, "FROM provider_budget_ledger unverified") < 2 {
		t.Fatal("both provider status and owner activation queue must reject unverified ledger contamination")
	}
}

func TestOfferReviewSurfacesUseCurrentVerifiedCommercialCommitments(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"queue":     "provider_pilot_status.go",
		"candidate": "provider_pilot_review.go",
		"snapshot":  "../../migrations/027_provider_pilot_review_evidence.sql",
	}
	for surface, path := range files {
		surface, path := surface, path
		t.Run(surface, func(t *testing.T) {
			t.Parallel()
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(source)
			for _, required := range []string{
				"event_type='prepaid_fund'",
				"event_type IN ('terms_acceptance','terms_renewal')",
				"SUM(reversal.amount_cents)",
				"event_type='fund_reversal'",
				"FROM provider_budget_ledger unverified",
				"linked.budget_ledger_entry_id=unverified.id",
				"commercial_terms_sha256 ~ '^[0-9a-f]{64}$'",
			} {
				if !strings.Contains(text, required) {
					t.Fatalf("%s offer review surface missing current commitment predicate %q", surface, required)
				}
			}
		})
	}
}
