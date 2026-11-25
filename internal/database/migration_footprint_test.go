package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestProviderCommercialProofFootprintProbesCoverInheritedDDL(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "022_provider_commercial_proof.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	want := map[string]bool{}
	alterColumnPattern := regexp.MustCompile(`(?m)^ALTER TABLE ([a-z0-9_]+)\s+ADD COLUMN IF NOT EXISTS ([a-z0-9_]+)`)
	for _, match := range alterColumnPattern.FindAllStringSubmatch(migrationSQL, -1) {
		probe := migrationFootprintProbe{
			kind:     migrationFootprintColumn,
			relation: match[1],
			name:     match[2],
		}
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		want[key] = true
	}
	if len(want) != 6 {
		t.Fatalf("022 ALTER-column footprint count=%d, want 6", len(want))
	}
	for _, probe := range []migrationFootprintProbe{
		{kind: migrationFootprintTrigger, relation: "provider_offers", name: "provider_offer_commercial_immutability_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_offer_commercial_immutability"},
		{kind: migrationFootprintTrigger, relation: "provider_action_handoff_receipts", name: "provider_action_handoff_receipt_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_action_handoff_receipt"},
		{kind: migrationFootprintTrigger, relation: "action_tickets", name: "action_ticket_observed_handoff_status_enforced"},
		{kind: migrationFootprintTrigger, relation: "action_tickets", name: "action_ticket_observed_handoff_insert_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_action_ticket_observed_handoff_status"},
	} {
		if !strings.Contains(migrationSQL, probe.name) {
			t.Fatalf("022 migration does not declare footprint marker %s", probe.name)
		}
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		want[key] = true
	}

	spec := protectedMigrationSpecs["022_provider_commercial_proof.sql"]
	got := make(map[string]bool, len(spec.footprintProbes))
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatalf("invalid 022 footprint probe: %v", err)
		}
		if got[key] {
			t.Fatalf("duplicate 022 footprint probe %s", key)
		}
		got[key] = true
	}
	if len(got) != len(want) {
		t.Fatalf("022 footprint probe count=%d, want %d; got=%v want=%v", len(got), len(want), got, want)
	}
	for key := range want {
		if !got[key] {
			t.Fatalf("022 protected contract is missing footprint probe %s", key)
		}
	}
}

func TestInspectProtectedMigrationDetectsEach022InheritedFootprint(t *testing.T) {
	spec := protectedMigrationSpecs["022_provider_commercial_proof.sql"]
	migration := migrationFile{
		name:   "022_provider_commercial_proof.sql",
		sha256: strings.Repeat("a", 64),
	}
	tests := []struct {
		name     string
		existing map[string]bool
		want     bool
	}{
		{name: "clean", existing: map[string]bool{}, want: false},
	}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		tests = append(tests, struct {
			name     string
			existing map[string]bool
			want     bool
		}{name: key, existing: map[string]bool{key: true}, want: true})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := sql.OpenDB(&migrationFootprintTestConnector{existing: test.existing})
			t.Cleanup(func() { _ = db.Close() })
			state, err := inspectProtectedMigration(context.Background(), db, migration.name, spec)
			if err != nil {
				t.Fatalf("inspect 022 footprint: %v", err)
			}
			if state.anyFootprint != test.want {
				t.Fatalf("anyFootprint=%t, want %t", state.anyFootprint, test.want)
			}
			err = validateProtectedMigrationState(migration, state, false)
			if !test.want && err != nil {
				t.Fatalf("clean 022 state rejected: %v", err)
			}
			if test.want && (err == nil || !strings.Contains(err.Error(), "ambiguous_prior_022")) {
				t.Fatalf("partial 022 footprint error=%v, want ambiguous_prior_022", err)
			}
		})
	}

	matched := protectedMigrationState{
		receiptExists:       true,
		receiptSHA256:       migration.sha256,
		receiptSchemaSHA256: strings.Repeat("b", 64),
		currentSchemaSHA256: strings.Repeat("b", 64),
		anyFootprint:        true,
		complete:            true,
	}
	if err := validateProtectedMigrationState(migration, matched, true); err != nil {
		t.Fatalf("exactly receipted 022 footprint rejected: %v", err)
	}
}

func TestProviderControlledIntentDisclosureFootprintProbesCoverInheritedDDL(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "023_provider_controlled_intent_disclosure.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	want := map[string]bool{}
	alterColumnPattern := regexp.MustCompile(`(?m)^ALTER TABLE ([a-z0-9_]+)\s+ADD COLUMN IF NOT EXISTS ([a-z0-9_]+)`)
	for _, match := range alterColumnPattern.FindAllStringSubmatch(migrationSQL, -1) {
		key, err := migrationFootprintProbeKey(migrationFootprintProbe{
			kind: migrationFootprintColumn, relation: match[1], name: match[2],
		})
		if err != nil {
			t.Fatal(err)
		}
		want[key] = true
	}
	if len(want) != 2 {
		t.Fatalf("023 ALTER-column footprint count=%d, want 2", len(want))
	}
	for _, probe := range []migrationFootprintProbe{
		{kind: migrationFootprintTrigger, relation: "action_tickets", name: "action_ticket_controlled_intent_immutability_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_action_ticket_controlled_intent_immutability"},
	} {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		want[key] = true
	}
	spec := protectedMigrationSpecs["023_provider_controlled_intent_disclosure.sql"]
	got := map[string]bool{}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		got[key] = true
	}
	if len(got) != len(want) {
		t.Fatalf("023 footprint probes=%v, want %v", got, want)
	}
	for key := range want {
		if !got[key] {
			t.Fatalf("023 protected contract is missing footprint probe %s", key)
		}
	}
}

func TestInspectProtectedMigrationDetectsEach023InheritedFootprint(t *testing.T) {
	name := "023_provider_controlled_intent_disclosure.sql"
	spec := protectedMigrationSpecs[name]
	migration := migrationFile{name: name, sha256: strings.Repeat("c", 64)}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(key, func(t *testing.T) {
			db := sql.OpenDB(&migrationFootprintTestConnector{existing: map[string]bool{key: true}})
			t.Cleanup(func() { _ = db.Close() })
			state, err := inspectProtectedMigration(context.Background(), db, name, spec)
			if err != nil {
				t.Fatal(err)
			}
			if !state.anyFootprint {
				t.Fatal("023 inherited footprint was not detected")
			}
			if err := validateProtectedMigrationState(migration, state, false); err == nil ||
				!strings.Contains(err.Error(), "ambiguous_prior_023") {
				t.Fatalf("023 partial footprint error=%v", err)
			}
		})
	}
}

func TestProviderPilotBoundaryFootprintProbesCoverInheritedDDL(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "024_provider_pilot_boundary.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	want := map[string]bool{}
	alterColumnPattern := regexp.MustCompile(`(?m)^ALTER TABLE ([a-z0-9_]+)\s+ADD COLUMN IF NOT EXISTS ([a-z0-9_]+)`)
	for _, match := range alterColumnPattern.FindAllStringSubmatch(migrationSQL, -1) {
		key, err := migrationFootprintProbeKey(migrationFootprintProbe{
			kind: migrationFootprintColumn, relation: match[1], name: match[2],
		})
		if err != nil {
			t.Fatal(err)
		}
		want[key] = true
	}
	if len(want) != 3 {
		t.Fatalf("024 ALTER-column footprint count=%d, want 3", len(want))
	}
	for _, probe := range []migrationFootprintProbe{
		{kind: migrationFootprintTrigger, relation: "provider_pilot_epochs", name: "provider_pilot_epoch_insert_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_pilot_epoch_insert"},
		{kind: migrationFootprintTrigger, relation: "provider_pilot_enrollments", name: "provider_pilot_enrollment_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_pilot_enrollment"},
		{kind: migrationFootprintTrigger, relation: "provider_pilot_epoch_events", name: "provider_pilot_epoch_event_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_pilot_epoch_event"},
		{kind: migrationFootprintTrigger, relation: "provider_pilot_epochs", name: "provider_pilot_epoch_transition_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_pilot_epoch_transition"},
		{kind: migrationFootprintTrigger, relation: "provider_offers", name: "provider_offer_pilot_binding_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_pilot_offer_binding"},
		{kind: migrationFootprintTrigger, relation: "provider_offers_returned", name: "provider_pilot_returned_offer_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_pilot_returned_offer"},
		{kind: migrationFootprintTrigger, relation: "action_tickets", name: "action_ticket_pilot_insert_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_action_ticket_pilot_insert"},
		{kind: migrationFootprintTrigger, relation: "action_tickets", name: "action_ticket_pilot_binding_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_action_ticket_pilot_binding"},
		{kind: migrationFootprintTrigger, relation: "provider_action_handoff_receipts", name: "zz_provider_action_handoff_pilot_boundary_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_pilot_handoff_insert"},
	} {
		if !strings.Contains(migrationSQL, probe.name) {
			t.Fatalf("024 migration does not declare footprint marker %s", probe.name)
		}
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		want[key] = true
	}

	spec := protectedMigrationSpecs["024_provider_pilot_boundary.sql"]
	got := map[string]bool{}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		if got[key] {
			t.Fatalf("duplicate 024 footprint probe %s", key)
		}
		got[key] = true
	}
	if len(got) != len(want) {
		t.Fatalf("024 footprint probes=%v, want %v", got, want)
	}
	for key := range want {
		if !got[key] {
			t.Fatalf("024 protected contract is missing footprint probe %s", key)
		}
	}
}

func TestInspectProtectedMigrationDetectsEach024InheritedFootprint(t *testing.T) {
	name := "024_provider_pilot_boundary.sql"
	spec := protectedMigrationSpecs[name]
	migration := migrationFile{name: name, sha256: strings.Repeat("d", 64)}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(key, func(t *testing.T) {
			db := sql.OpenDB(&migrationFootprintTestConnector{existing: map[string]bool{key: true}})
			t.Cleanup(func() { _ = db.Close() })
			state, err := inspectProtectedMigration(context.Background(), db, name, spec)
			if err != nil {
				t.Fatal(err)
			}
			if !state.anyFootprint {
				t.Fatal("024 inherited footprint was not detected")
			}
			if err := validateProtectedMigrationState(migration, state, false); err == nil ||
				!strings.Contains(err.Error(), "ambiguous_prior_024") {
				t.Fatalf("024 partial footprint error=%v", err)
			}
		})
	}
}

func TestStage1FactIntegrityFootprintProbesCoverInheritedDDL(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "025_stage1_fact_integrity.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	want := map[string]bool{}
	for _, probe := range []migrationFootprintProbe{
		{kind: migrationFootprintColumn, relation: "search_receipts", name: "stage1_integrity_generation"},
		{kind: migrationFootprintColumn, relation: "organic_results_returned", name: "stage1_integrity_generation"},
		{kind: migrationFootprintColumn, relation: "result_selections", name: "stage1_integrity_generation"},
		{kind: migrationFootprintColumn, relation: "action_interest_receipts", name: "stage1_integrity_generation"},
		{kind: migrationFootprintFunction, name: "enforce_search_receipt_stage1_immutability"},
		{kind: migrationFootprintFunction, name: "enforce_organic_result_stage1_immutability"},
		{kind: migrationFootprintFunction, name: "enforce_result_selection_stage1_immutability"},
		{kind: migrationFootprintFunction, name: "own_search_receipt_created_at"},
		{kind: migrationFootprintFunction, name: "own_organic_result_returned_at"},
		{kind: migrationFootprintFunction, name: "own_result_selection_selected_at"},
		{kind: migrationFootprintFunction, name: "own_action_interest_created_at"},
		{kind: migrationFootprintTrigger, relation: "provider_pilot_epochs", name: "aa_provider_pilot_stage1_epoch_anchor_locked"},
		{kind: migrationFootprintFunction, name: "lock_provider_pilot_stage1_epoch_anchor"},
		{kind: migrationFootprintTrigger, relation: "provider_pilot_epochs", name: "ab_provider_pilot_stage1_generation_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_pilot_stage1_generation"},
	} {
		if !strings.Contains(migrationSQL, probe.name) {
			t.Fatalf("025 migration does not declare footprint marker %s", probe.name)
		}
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		want[key] = true
	}

	spec := protectedMigrationSpecs["025_stage1_fact_integrity.sql"]
	got := map[string]bool{}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		if got[key] {
			t.Fatalf("duplicate 025 footprint probe %s", key)
		}
		got[key] = true
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("025 footprint probes=%v, want %v", got, want)
	}
}

func TestInspectProtectedMigrationDetectsEach025InheritedFootprint(t *testing.T) {
	name := "025_stage1_fact_integrity.sql"
	spec := protectedMigrationSpecs[name]
	migration := migrationFile{name: name, sha256: strings.Repeat("e", 64)}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(key, func(t *testing.T) {
			db := sql.OpenDB(&migrationFootprintTestConnector{existing: map[string]bool{key: true}})
			t.Cleanup(func() { _ = db.Close() })
			state, err := inspectProtectedMigration(context.Background(), db, name, spec)
			if err != nil {
				t.Fatal(err)
			}
			if !state.anyFootprint {
				t.Fatal("025 inherited footprint was not detected")
			}
			if err := validateProtectedMigrationState(migration, state, false); err == nil ||
				!strings.Contains(err.Error(), "ambiguous_prior_025") {
				t.Fatalf("025 partial footprint error=%v", err)
			}
		})
	}
}

func TestProviderPilotProofIntegrityFootprintProbesCoverInheritedDDL(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "026_provider_pilot_proof_integrity.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	want := map[string]bool{}
	for _, probe := range []migrationFootprintProbe{
		{kind: migrationFootprintFunction, name: "enforce_provider_pilot_outcome_receipt"},
		{kind: migrationFootprintTrigger, relation: "outcome_receipts", name: "provider_pilot_outcome_receipt_enforced"},
		{kind: migrationFootprintFunction, name: "require_provider_pilot_lifecycle_event"},
		{kind: migrationFootprintTrigger, relation: "provider_pilot_epochs", name: "provider_pilot_epoch_created_event_required"},
		{kind: migrationFootprintTrigger, relation: "provider_pilot_enrollments", name: "provider_pilot_enrollment_event_required"},
		{kind: migrationFootprintTrigger, relation: "provider_pilot_epochs", name: "provider_pilot_epoch_transition_event_required"},
	} {
		if !strings.Contains(migrationSQL, probe.name) {
			t.Fatalf("026 migration does not declare footprint marker %s", probe.name)
		}
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		want[key] = true
	}

	spec := protectedMigrationSpecs["026_provider_pilot_proof_integrity.sql"]
	got := map[string]bool{}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		if got[key] {
			t.Fatalf("duplicate 026 footprint probe %s", key)
		}
		got[key] = true
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("026 footprint probes=%v, want %v", got, want)
	}
}

func TestInspectProtectedMigrationDetectsEach026InheritedFootprint(t *testing.T) {
	name := "026_provider_pilot_proof_integrity.sql"
	spec := protectedMigrationSpecs[name]
	migration := migrationFile{name: name, sha256: strings.Repeat("f", 64)}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(key, func(t *testing.T) {
			db := sql.OpenDB(&migrationFootprintTestConnector{existing: map[string]bool{key: true}})
			t.Cleanup(func() { _ = db.Close() })
			state, err := inspectProtectedMigration(context.Background(), db, name, spec)
			if err != nil {
				t.Fatal(err)
			}
			if !state.anyFootprint {
				t.Fatal("026 inherited footprint was not detected")
			}
			if err := validateProtectedMigrationState(migration, state, false); err == nil ||
				!strings.Contains(err.Error(), "ambiguous_prior_026") {
				t.Fatalf("026 partial footprint error=%v", err)
			}
		})
	}
}

func TestProviderPilotReviewEvidenceFootprintContractMatchesMigration(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "027_provider_pilot_review_evidence.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	spec := protectedMigrationSpecs["027_provider_pilot_review_evidence.sql"]

	wantRelations := map[string]migrationRelation{
		"provider_pilot_review_events":          {name: "provider_pilot_review_events", relkind: "r"},
		"idx_provider_pilot_reviews_pilot_type": {name: "idx_provider_pilot_reviews_pilot_type", relkind: "i", parent: "provider_pilot_review_events"},
		"idx_provider_pilot_reviews_claim_type": {name: "idx_provider_pilot_reviews_claim_type", relkind: "i", parent: "provider_pilot_review_events"},
	}
	if !reflect.DeepEqual(spec.footprintRelations, map[string]bool{
		"provider_pilot_review_events":          true,
		"idx_provider_pilot_reviews_pilot_type": true,
		"idx_provider_pilot_reviews_claim_type": true,
	}) {
		t.Fatalf("027 relation footprint markers=%v", spec.footprintRelations)
	}
	declaredRelations := make(map[string]migrationRelation, len(wantRelations))
	for _, relation := range spec.relations {
		if spec.footprintRelations[relation.name] {
			declaredRelations[relation.name] = relation
		}
	}
	if !reflect.DeepEqual(declaredRelations, wantRelations) {
		t.Fatalf("027 relation footprint=%v, want %v", declaredRelations, wantRelations)
	}

	wantRules := map[string]migrationRule{
		"provider_pilot_review_events_no_update": {relation: "provider_pilot_review_events", name: "provider_pilot_review_events_no_update"},
		"provider_pilot_review_events_no_delete": {relation: "provider_pilot_review_events", name: "provider_pilot_review_events_no_delete"},
	}
	declaredRules := make(map[string]migrationRule, len(wantRules))
	for _, rule := range spec.rules {
		if spec.footprintRules[rule.name] {
			declaredRules[rule.name] = rule
		}
	}
	if !reflect.DeepEqual(declaredRules, wantRules) {
		t.Fatalf("027 rule footprint=%v, want %v", declaredRules, wantRules)
	}

	const snapshotFunction = "provider_pilot_review_snapshot_sha256(uuid,text,uuid)"
	if !reflect.DeepEqual(spec.footprintFunctions, map[string]bool{snapshotFunction: true}) {
		t.Fatalf("027 standalone-function footprint=%v", spec.footprintFunctions)
	}
	wantProbes := map[string]bool{}
	for _, probe := range []migrationFootprintProbe{
		{kind: migrationFootprintTrigger, relation: "provider_pilot_review_events", name: "provider_pilot_review_event_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_pilot_review_event"},
		{kind: migrationFootprintTrigger, relation: "provider_pilot_epochs", name: "provider_pilot_activation_provider_reviews"},
		{kind: migrationFootprintFunction, name: "enforce_provider_pilot_epoch_provider_reviews"},
		{kind: migrationFootprintTrigger, relation: "provider_offers", name: "provider_offer_activation_review"},
		{kind: migrationFootprintFunction, name: "enforce_provider_offer_pre_activation_review"},
		{kind: migrationFootprintTrigger, relation: "provider_action_handoff_receipts", name: "provider_handoff_ticket_review"},
		{kind: migrationFootprintFunction, name: "enforce_provider_handoff_ticket_review"},
	} {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		wantProbes[key] = true
	}
	gotProbes := map[string]bool{}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		if gotProbes[key] {
			t.Fatalf("duplicate 027 footprint probe %s", key)
		}
		gotProbes[key] = true
	}
	if !reflect.DeepEqual(gotProbes, wantProbes) {
		t.Fatalf("027 trigger-function probes=%v, want %v", gotProbes, wantProbes)
	}

	for name := range wantRelations {
		if !strings.Contains(migrationSQL, name) {
			t.Fatalf("027 migration does not declare relation footprint %s", name)
		}
	}
	for name := range wantRules {
		if !strings.Contains(migrationSQL, name) {
			t.Fatalf("027 migration does not declare rule footprint %s", name)
		}
	}
	for _, name := range []string{
		"provider_pilot_review_snapshot_sha256",
		"provider_pilot_review_event_enforced",
		"enforce_provider_pilot_review_event",
		"provider_pilot_activation_provider_reviews",
		"enforce_provider_pilot_epoch_provider_reviews",
		"provider_offer_activation_review",
		"enforce_provider_offer_pre_activation_review",
		"provider_handoff_ticket_review",
		"enforce_provider_handoff_ticket_review",
	} {
		if !strings.Contains(migrationSQL, name) {
			t.Fatalf("027 migration does not declare function/trigger footprint %s", name)
		}
	}
}

func TestInspectProtectedMigrationDetectsEach027Footprint(t *testing.T) {
	const name = "027_provider_pilot_review_evidence.sql"
	spec := protectedMigrationSpecs[name]
	migration := migrationFile{name: name, sha256: strings.Repeat("a", 64)}

	assertAmbiguous := func(t *testing.T, connector *migrationFootprintTestConnector) {
		t.Helper()
		db := sql.OpenDB(connector)
		t.Cleanup(func() { _ = db.Close() })
		state, err := inspectProtectedMigration(context.Background(), db, name, spec)
		if err != nil {
			t.Fatal(err)
		}
		if !state.anyFootprint {
			t.Fatal("027 footprint was not detected")
		}
		if err := validateProtectedMigrationState(migration, state, false); err == nil ||
			!strings.Contains(err.Error(), "ambiguous_prior_027") {
			t.Fatalf("027 partial footprint error=%v", err)
		}
	}

	for _, relation := range spec.relations {
		if !spec.footprintRelations[relation.name] {
			continue
		}
		t.Run("relation:"+relation.name, func(t *testing.T) {
			actual := relation.relkind + ":" + relation.parent
			assertAmbiguous(t, &migrationFootprintTestConnector{
				relations: map[string]string{relation.name: actual},
			})
		})
	}
	for _, rule := range spec.rules {
		if !spec.footprintRules[rule.name] {
			continue
		}
		t.Run("rule:"+rule.name, func(t *testing.T) {
			assertAmbiguous(t, &migrationFootprintTestConnector{
				rules: map[string]string{rule.name: rule.relation},
			})
		})
	}
	for function := range spec.footprintFunctions {
		t.Run("function:"+function, func(t *testing.T) {
			assertAmbiguous(t, &migrationFootprintTestConnector{
				functionDefinitions: map[string]string{function: "CREATE FUNCTION snapshot() RETURNS text LANGUAGE sql AS 'SELECT 1'"},
			})
		})
	}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(key, func(t *testing.T) {
			assertAmbiguous(t, &migrationFootprintTestConnector{existing: map[string]bool{key: true}})
		})
	}
}

func TestProtectedMigrationSchemaFingerprintDetects027StandaloneFunctionDrift(t *testing.T) {
	const function = "provider_pilot_review_snapshot_sha256(uuid,text,uuid)"
	spec := protectedMigrationSpec{fingerprintFunctions: []string{function}}
	fingerprint := func(t *testing.T, definition string) string {
		t.Helper()
		db := sql.OpenDB(&migrationFootprintTestConnector{
			functionDefinitions: map[string]string{function: definition},
		})
		t.Cleanup(func() { _ = db.Close() })
		got, err := protectedMigrationSchemaFingerprint(context.Background(), db, spec)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	original := fingerprint(t, "CREATE FUNCTION public.provider_pilot_review_snapshot_sha256(uuid, text, uuid) RETURNS text LANGUAGE sql AS 'SELECT ''a''' ")
	replay := fingerprint(t, "CREATE FUNCTION public.provider_pilot_review_snapshot_sha256(uuid, text, uuid) RETURNS text LANGUAGE sql AS 'SELECT ''a''' ")
	drifted := fingerprint(t, "CREATE FUNCTION public.provider_pilot_review_snapshot_sha256(uuid, text, uuid) RETURNS text LANGUAGE sql AS 'SELECT ''b''' ")
	if original != replay {
		t.Fatalf("unchanged standalone function fingerprint changed: %s != %s", original, replay)
	}
	if original == drifted {
		t.Fatalf("standalone function body drift preserved fingerprint %s", original)
	}
	state := protectedMigrationState{
		receiptExists:       true,
		receiptSHA256:       strings.Repeat("a", 64),
		receiptSchemaSHA256: original,
		currentSchemaSHA256: drifted,
		complete:            true,
	}
	if err := validateProtectedMigrationState(
		migrationFile{name: "027_provider_pilot_review_evidence.sql", sha256: state.receiptSHA256},
		state,
		true,
	); err == nil || !strings.Contains(err.Error(), "schema fingerprint drift") {
		t.Fatalf("standalone function body drift error=%v", err)
	}

	db := sql.OpenDB(&migrationFootprintTestConnector{})
	t.Cleanup(func() { _ = db.Close() })
	if _, err := protectedMigrationSchemaFingerprint(context.Background(), db, spec); err == nil ||
		!strings.Contains(err.Error(), "fingerprint function "+function+" is missing") {
		t.Fatalf("missing standalone function fingerprint error=%v", err)
	}
}

func TestProviderCommercialProofManifestFootprintContractMatchesMigration(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "028_provider_commercial_proof_manifest.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	spec := protectedMigrationSpecs["028_provider_commercial_proof_manifest.sql"]

	wantRelations := map[string]migrationRelation{
		"provider_commercial_proof_manifests": {name: "provider_commercial_proof_manifests", relkind: "r"},
		"idx_provider_proof_manifests_issued": {name: "idx_provider_proof_manifests_issued", relkind: "i", parent: "provider_commercial_proof_manifests"},
		"idx_provider_proof_manifests_key":    {name: "idx_provider_proof_manifests_key", relkind: "i", parent: "provider_commercial_proof_manifests"},
	}
	if !reflect.DeepEqual(spec.footprintRelations, map[string]bool{
		"provider_commercial_proof_manifests": true,
		"idx_provider_proof_manifests_issued": true,
		"idx_provider_proof_manifests_key":    true,
	}) {
		t.Fatalf("028 relation footprint markers=%v", spec.footprintRelations)
	}
	declaredRelations := make(map[string]migrationRelation, len(wantRelations))
	for _, relation := range spec.relations {
		if spec.footprintRelations[relation.name] {
			declaredRelations[relation.name] = relation
		}
	}
	if !reflect.DeepEqual(declaredRelations, wantRelations) {
		t.Fatalf("028 relation footprint=%v, want %v", declaredRelations, wantRelations)
	}

	wantRules := map[string]migrationRule{
		"provider_commercial_proof_manifests_no_update": {relation: "provider_commercial_proof_manifests", name: "provider_commercial_proof_manifests_no_update"},
		"provider_commercial_proof_manifests_no_delete": {relation: "provider_commercial_proof_manifests", name: "provider_commercial_proof_manifests_no_delete"},
	}
	declaredRules := make(map[string]migrationRule, len(wantRules))
	for _, rule := range spec.rules {
		if spec.footprintRules[rule.name] {
			declaredRules[rule.name] = rule
		}
	}
	if !reflect.DeepEqual(declaredRules, wantRules) {
		t.Fatalf("028 rule footprint=%v, want %v", declaredRules, wantRules)
	}
	if len(spec.footprintFunctions) != 0 {
		t.Fatalf("028 unexpected standalone-function footprint=%v", spec.footprintFunctions)
	}

	wantProbes := map[string]bool{}
	for _, probe := range []migrationFootprintProbe{
		{kind: migrationFootprintTrigger, relation: "provider_commercial_proof_manifests", name: "provider_commercial_proof_manifest_enforced"},
		{kind: migrationFootprintFunction, name: "enforce_provider_commercial_proof_manifest"},
	} {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		wantProbes[key] = true
	}
	gotProbes := map[string]bool{}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		if gotProbes[key] {
			t.Fatalf("duplicate 028 footprint probe %s", key)
		}
		gotProbes[key] = true
	}
	if !reflect.DeepEqual(gotProbes, wantProbes) {
		t.Fatalf("028 trigger-function probes=%v, want %v", gotProbes, wantProbes)
	}

	for name := range wantRelations {
		if !strings.Contains(migrationSQL, name) {
			t.Fatalf("028 migration does not declare relation footprint %s", name)
		}
	}
	for name := range wantRules {
		if !strings.Contains(migrationSQL, name) {
			t.Fatalf("028 migration does not declare rule footprint %s", name)
		}
	}
	for _, name := range []string{
		"provider_commercial_proof_manifest_enforced",
		"enforce_provider_commercial_proof_manifest",
	} {
		if !strings.Contains(migrationSQL, name) {
			t.Fatalf("028 migration does not declare function/trigger footprint %s", name)
		}
	}
}

func TestInspectProtectedMigrationDetectsEach028Footprint(t *testing.T) {
	const name = "028_provider_commercial_proof_manifest.sql"
	spec := protectedMigrationSpecs[name]
	migration := migrationFile{name: name, sha256: strings.Repeat("b", 64)}

	assertAmbiguous := func(t *testing.T, connector *migrationFootprintTestConnector) {
		t.Helper()
		db := sql.OpenDB(connector)
		t.Cleanup(func() { _ = db.Close() })
		state, err := inspectProtectedMigration(context.Background(), db, name, spec)
		if err != nil {
			t.Fatal(err)
		}
		if !state.anyFootprint {
			t.Fatal("028 footprint was not detected")
		}
		if err := validateProtectedMigrationState(migration, state, false); err == nil ||
			!strings.Contains(err.Error(), "ambiguous_prior_028") {
			t.Fatalf("028 partial footprint error=%v", err)
		}
	}

	for _, relation := range spec.relations {
		if !spec.footprintRelations[relation.name] {
			continue
		}
		t.Run("relation:"+relation.name, func(t *testing.T) {
			actual := relation.relkind + ":" + relation.parent
			assertAmbiguous(t, &migrationFootprintTestConnector{
				relations: map[string]string{relation.name: actual},
			})
		})
	}
	for _, rule := range spec.rules {
		if !spec.footprintRules[rule.name] {
			continue
		}
		t.Run("rule:"+rule.name, func(t *testing.T) {
			assertAmbiguous(t, &migrationFootprintTestConnector{
				rules: map[string]string{rule.name: rule.relation},
			})
		})
	}
	for _, probe := range spec.footprintProbes {
		key, err := migrationFootprintProbeKey(probe)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(key, func(t *testing.T) {
			assertAmbiguous(t, &migrationFootprintTestConnector{existing: map[string]bool{key: true}})
		})
	}
}

type migrationFootprintTestConnector struct {
	existing            map[string]bool
	relations           map[string]string
	rules               map[string]string
	functionDefinitions map[string]string
}

func (c *migrationFootprintTestConnector) Connect(context.Context) (driver.Conn, error) {
	return &migrationFootprintTestConn{
		existing:            c.existing,
		relations:           c.relations,
		rules:               c.rules,
		functionDefinitions: c.functionDefinitions,
	}, nil
}

func (*migrationFootprintTestConnector) Driver() driver.Driver {
	return migrationFootprintTestDriver{}
}

type migrationFootprintTestDriver struct{}

func (migrationFootprintTestDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("migration footprint test driver requires OpenDB")
}

type migrationFootprintTestConn struct {
	existing            map[string]bool
	relations           map[string]string
	rules               map[string]string
	functionDefinitions map[string]string
}

func (*migrationFootprintTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("migration footprint test driver does not prepare statements")
}

func (*migrationFootprintTestConn) Close() error { return nil }

func (*migrationFootprintTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("migration footprint test driver does not begin transactions")
}

func (c *migrationFootprintTestConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	row := func(value driver.Value) driver.Rows {
		return &migrationFootprintTestRows{value: value}
	}
	switch {
	case strings.Contains(query, "to_regclass('public.nhs_schema_migrations')"):
		return row(false), nil
	case strings.Contains(query, "FROM pg_catalog.pg_attribute a"):
		key, err := migrationFootprintProbeKey(migrationFootprintProbe{
			kind:     migrationFootprintColumn,
			relation: migrationFootprintTestArgument(args, 0),
			name:     migrationFootprintTestArgument(args, 1),
		})
		if err != nil {
			return nil, err
		}
		return row(c.existing[key]), nil
	case strings.Contains(query, "FROM pg_catalog.pg_trigger tg"):
		key, err := migrationFootprintProbeKey(migrationFootprintProbe{
			kind:     migrationFootprintTrigger,
			relation: migrationFootprintTestArgument(args, 0),
			name:     migrationFootprintTestArgument(args, 1),
		})
		if err != nil {
			return nil, err
		}
		return row(c.existing[key]), nil
	case strings.Contains(query, "to_regprocedure('public.' || $1) IS NOT NULL"):
		_, exists := c.functionDefinitions[migrationFootprintTestArgument(args, 0)]
		return row(exists), nil
	case strings.Contains(query, "FROM pg_catalog.pg_proc p"):
		key, err := migrationFootprintProbeKey(migrationFootprintProbe{
			kind: migrationFootprintFunction,
			name: migrationFootprintTestArgument(args, 0),
		})
		if err != nil {
			return nil, err
		}
		return row(c.existing[key]), nil
	case strings.Contains(query, "c.relkind::text ||"):
		return row(c.relations[migrationFootprintTestArgument(args, 0)]), nil
	case strings.Contains(query, "FROM pg_catalog.pg_rewrite r"):
		return row(c.rules[migrationFootprintTestArgument(args, 0)]), nil
	case strings.Contains(query, "pg_catalog.pg_get_functiondef"):
		return row(c.functionDefinitions[migrationFootprintTestArgument(args, 0)]), nil
	default:
		return nil, errors.New("unexpected migration footprint test query")
	}
}

func migrationFootprintTestArgument(args []driver.NamedValue, index int) string {
	if index >= len(args) {
		return ""
	}
	value, _ := args[index].Value.(string)
	return value
}

type migrationFootprintTestRows struct {
	value driver.Value
	done  bool
}

func (*migrationFootprintTestRows) Columns() []string { return []string{"value"} }
func (*migrationFootprintTestRows) Close() error      { return nil }

func (r *migrationFootprintTestRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = r.value
	return nil
}

var _ driver.QueryerContext = (*migrationFootprintTestConn)(nil)
