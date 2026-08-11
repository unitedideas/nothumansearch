package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

const (
	protectedMigrationStart = "019_"
	migrationLockClass      = 0x4e4853 // "NHS"
	migrationLockID         = 1
)

var (
	migrationOverallTimeout              = 2 * time.Minute
	migrationTransactionLockTimeout      = 15 * time.Second
	migrationTransactionStatementTimeout = 90 * time.Second
)

const migrationReceiptTableSQL = `
CREATE TABLE IF NOT EXISTS public.nhs_schema_migrations (
    name              TEXT PRIMARY KEY,
    sha256            TEXT NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    schema_sha256     TEXT NOT NULL CHECK (schema_sha256 ~ '^[0-9a-f]{64}$'),
    applied_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_by_commit TEXT NOT NULL CHECK (applied_by_commit ~ '^[0-9a-f]{40}$')
)`

type migrationFile struct {
	name   string
	data   []byte
	sha256 string
}

type migrationRelation struct {
	name    string
	relkind string
	parent  string
}

type migrationRule struct {
	relation string
	name     string
}

type migrationFootprintKind string

const (
	migrationFootprintColumn   migrationFootprintKind = "column"
	migrationFootprintTrigger  migrationFootprintKind = "trigger"
	migrationFootprintFunction migrationFootprintKind = "function"
)

// migrationFootprintProbe names a migration-specific catalog marker that is
// not itself part of the cumulative relation/rule completeness contract. These
// probes catch unreceipted ALTERs and behavior attached to inherited tables.
// They are intentionally not carried into later migration specs: once this
// migration has a receipt, the cumulative schema fingerprint owns them.
type migrationFootprintProbe struct {
	kind     migrationFootprintKind
	relation string
	name     string
}

type protectedMigrationSpec struct {
	relations []migrationRelation
	rules     []migrationRule
	// fingerprintTables adds inherited behavior-bearing tables without
	// pretending those legacy relations were created by this migration. It is
	// used when a protected ALTER/trigger migration needs the full table
	// definition in its latest schema receipt.
	fingerprintTables []string
	// fingerprintFunctions carries standalone behavior-bearing routines whose
	// definitions are not embedded in a table trigger. Entries are canonical
	// regprocedure identities such as function_name(uuid,text,uuid).
	fingerprintFunctions   []string
	allObjectsAreFootprint bool
	footprintRelations     map[string]bool
	footprintRules         map[string]bool
	footprintFunctions     map[string]bool
	footprintProbes        []migrationFootprintProbe
}

// ProtectedMigrationPreflight runs inside the exact protected-migration
// transaction only when that migration has no receipt and is about to apply.
// It may acquire migration-specific locks and must not commit independently.
type ProtectedMigrationPreflight func(context.Context, *sql.Tx, string) error

// Every protected migration must declare the PostgreSQL objects that prove its
// complete application. A new protected migration without a contract fails
// before any older migration is replayed. This prevents an unrecorded partial
// schema from being adopted merely because its DDL uses IF NOT EXISTS.
var protectedMigrationSpecs = buildProtectedMigrationSpecs()

var requiredProtectedMigrationNames = []string{
	"019_provider_exchange.sql",
	"020_action_interest_receipts.sql",
	"021_provider_capacity_reservations.sql",
	"022_provider_commercial_proof.sql",
	"023_provider_controlled_intent_disclosure.sql",
	"024_provider_pilot_boundary.sql",
	"025_stage1_fact_integrity.sql",
	"026_provider_pilot_proof_integrity.sql",
	"027_provider_pilot_review_evidence.sql",
	"028_provider_commercial_proof_manifest.sql",
	"029_provider_settlement_receipts.sql",
	"030_provider_processor_net_receipts.sql",
	"031_action_interest_attempt_funnel.sql",
}

// Historical migration tests deliberately replay prefix releases one at a
// time. Production code has no setter for this test-only in-process escape.
var allowPartialProtectedMigrationsForTests bool

func buildProtectedMigrationSpecs() map[string]protectedMigrationSpec {
	providerExchange := protectedMigrationSpec{
		allObjectsAreFootprint: true,
		relations: []migrationRelation{
			{name: "provider_claims", relkind: "r"},
			{name: "idx_provider_claims_one_verified_site", relkind: "i", parent: "provider_claims"},
			{name: "idx_provider_claims_one_pending_account_site", relkind: "i", parent: "provider_claims"},
			{name: "idx_provider_claims_account_status", relkind: "i", parent: "provider_claims"},
			{name: "idx_provider_claims_pending_expiry", relkind: "i", parent: "provider_claims"},
			{name: "idx_provider_claims_verification_due", relkind: "i", parent: "provider_claims"},
			{name: "idx_provider_claims_verification_freshness", relkind: "i", parent: "provider_claims"},
			{name: "provider_api_keys", relkind: "r"},
			{name: "idx_provider_api_keys_claim_status", relkind: "i", parent: "provider_api_keys"},
			{name: "idx_provider_api_keys_one_active_claim", relkind: "i", parent: "provider_api_keys"},
			{name: "provider_offers", relkind: "r"},
			{name: "idx_provider_offers_claim_status", relkind: "i", parent: "provider_offers"},
			{name: "idx_provider_offers_public_active", relkind: "i", parent: "provider_offers"},
			{name: "provider_offers_returned", relkind: "r"},
			{name: "idx_provider_offers_returned_offer", relkind: "i", parent: "provider_offers_returned"},
			{name: "action_tickets", relkind: "r"},
			{name: "idx_action_tickets_offer_status", relkind: "i", parent: "action_tickets"},
			{name: "idx_action_tickets_receipt", relkind: "i", parent: "action_tickets"},
			{name: "idx_action_tickets_expiry", relkind: "i", parent: "action_tickets"},
			{name: "provider_budget_ledger", relkind: "r"},
			{name: "idx_provider_budget_offer_created", relkind: "i", parent: "provider_budget_ledger"},
			{name: "idx_provider_budget_claim_created", relkind: "i", parent: "provider_budget_ledger"},
			{name: "idx_provider_budget_one_charge_per_ticket", relkind: "i", parent: "provider_budget_ledger"},
			{name: "idx_provider_budget_one_credit_per_ticket", relkind: "i", parent: "provider_budget_ledger"},
			{name: "idx_provider_budget_unique_funding_reference", relkind: "i", parent: "provider_budget_ledger"},
			{name: "outcome_receipts", relkind: "r"},
			{name: "idx_outcome_receipts_ticket_created", relkind: "i", parent: "outcome_receipts"},
			{name: "idx_outcome_receipts_claim_created", relkind: "i", parent: "outcome_receipts"},
			{name: "provider_admin_audit_events", relkind: "r"},
			{name: "idx_provider_admin_audit_offer_created", relkind: "i", parent: "provider_admin_audit_events"},
		},
		rules: []migrationRule{
			{relation: "provider_offers_returned", name: "provider_offers_returned_no_update"},
			{relation: "provider_budget_ledger", name: "provider_budget_ledger_no_update"},
			{relation: "provider_budget_ledger", name: "provider_budget_ledger_no_delete"},
			{relation: "outcome_receipts", name: "outcome_receipts_no_update"},
			{relation: "outcome_receipts", name: "outcome_receipts_no_delete"},
			{relation: "provider_admin_audit_events", name: "provider_admin_audit_no_update"},
			{relation: "provider_admin_audit_events", name: "provider_admin_audit_no_delete"},
			{relation: "search_receipts", name: "redact_action_ticket_intent_on_receipt_delete"},
		},
	}

	actionInterests := protectedMigrationSpec{
		// Later contracts are cumulative so the latest receipt fingerprints every
		// protected invariant. Only objects first introduced by 020 count as its
		// ambiguity footprint on a database that legitimately already has 019.
		relations: append(append([]migrationRelation(nil), providerExchange.relations...),
			migrationRelation{name: "idx_search_receipts_id_synthetic", relkind: "i", parent: "search_receipts"},
			migrationRelation{name: "action_interest_receipts", relkind: "r"},
			migrationRelation{name: "idx_action_interest_receipts_domain_created", relkind: "i", parent: "action_interest_receipts"},
			migrationRelation{name: "idx_action_interest_receipts_action_created", relkind: "i", parent: "action_interest_receipts"},
			migrationRelation{name: "idx_action_interest_receipts_expires", relkind: "i", parent: "action_interest_receipts"},
		),
		rules: append(append([]migrationRule(nil), providerExchange.rules...),
			migrationRule{relation: "action_interest_receipts", name: "action_interest_receipts_no_update"},
		),
		footprintRelations: map[string]bool{
			"idx_search_receipts_id_synthetic":            true,
			"action_interest_receipts":                    true,
			"idx_action_interest_receipts_domain_created": true,
			"idx_action_interest_receipts_action_created": true,
			"idx_action_interest_receipts_expires":        true,
		},
		footprintRules: map[string]bool{
			"action_interest_receipts_no_update": true,
		},
	}

	capacityReservations := protectedMigrationSpec{
		// The latest protected contract remains cumulative. Only the append-only
		// reservation event table and its explicit indexes/rules are the 021
		// ambiguity footprint; inherited 019/020 objects still participate in the
		// latest schema fingerprint.
		relations: append(append([]migrationRelation(nil), actionInterests.relations...),
			migrationRelation{name: "provider_capacity_events", relkind: "r"},
			migrationRelation{name: "idx_provider_capacity_events_offer_created", relkind: "i", parent: "provider_capacity_events"},
			migrationRelation{name: "idx_provider_capacity_events_ticket_created", relkind: "i", parent: "provider_capacity_events"},
			migrationRelation{name: "idx_provider_capacity_one_terminal_per_ticket", relkind: "i", parent: "provider_capacity_events"},
		),
		rules: append(append([]migrationRule(nil), actionInterests.rules...),
			migrationRule{relation: "provider_capacity_events", name: "provider_capacity_events_no_update"},
			migrationRule{relation: "provider_capacity_events", name: "provider_capacity_events_no_delete"},
		),
		footprintRelations: map[string]bool{
			"provider_capacity_events":                      true,
			"idx_provider_capacity_events_offer_created":    true,
			"idx_provider_capacity_events_ticket_created":   true,
			"idx_provider_capacity_one_terminal_per_ticket": true,
		},
		footprintRules: map[string]bool{
			"provider_capacity_events_no_update": true,
			"provider_capacity_events_no_delete": true,
		},
	}

	commercialProof := protectedMigrationSpec{
		// 022 keeps the cumulative protected contract and adds only immutable
		// provider/company acceptance plus owner-verified commitment evidence.
		// ALTERed columns and behavior attached to inherited tables need explicit
		// ambiguity probes in addition to the cumulative schema fingerprint.
		relations: append(append([]migrationRelation(nil), capacityReservations.relations...),
			migrationRelation{name: "provider_commercial_acceptance_events", relkind: "r"},
			migrationRelation{name: "idx_provider_commercial_acceptance_claim_created", relkind: "i", parent: "provider_commercial_acceptance_events"},
			migrationRelation{name: "idx_provider_commercial_acceptance_offer_created", relkind: "i", parent: "provider_commercial_acceptance_events"},
			migrationRelation{name: "idx_provider_commercial_acceptance_one_renewal", relkind: "i", parent: "provider_commercial_acceptance_events"},
			migrationRelation{name: "provider_pilot_companies", relkind: "r"},
			migrationRelation{name: "idx_provider_pilot_companies_claim", relkind: "i", parent: "provider_pilot_companies"},
			migrationRelation{name: "provider_commercial_commitment_events", relkind: "r"},
			migrationRelation{name: "idx_provider_commercial_commitment_company_created", relkind: "i", parent: "provider_commercial_commitment_events"},
			migrationRelation{name: "idx_provider_commercial_commitment_offer_created", relkind: "i", parent: "provider_commercial_commitment_events"},
			migrationRelation{name: "idx_provider_commercial_commitment_related", relkind: "i", parent: "provider_commercial_commitment_events"},
			migrationRelation{name: "idx_provider_commercial_one_terms_renewal", relkind: "i", parent: "provider_commercial_commitment_events"},
			migrationRelation{name: "provider_action_handoff_receipts", relkind: "r"},
			migrationRelation{name: "idx_provider_action_handoff_offer_observed", relkind: "i", parent: "provider_action_handoff_receipts"},
		),
		rules: append(append([]migrationRule(nil), capacityReservations.rules...),
			migrationRule{relation: "provider_commercial_acceptance_events", name: "provider_commercial_acceptance_no_update"},
			migrationRule{relation: "provider_commercial_acceptance_events", name: "provider_commercial_acceptance_no_delete"},
			migrationRule{relation: "provider_pilot_companies", name: "provider_pilot_companies_no_update"},
			migrationRule{relation: "provider_pilot_companies", name: "provider_pilot_companies_no_delete"},
			migrationRule{relation: "provider_commercial_commitment_events", name: "provider_commercial_commitment_no_update"},
			migrationRule{relation: "provider_commercial_commitment_events", name: "provider_commercial_commitment_no_delete"},
			migrationRule{relation: "provider_action_handoff_receipts", name: "provider_action_handoff_no_update"},
			migrationRule{relation: "provider_action_handoff_receipts", name: "provider_action_handoff_no_delete"},
		),
		footprintRelations: map[string]bool{
			"provider_commercial_acceptance_events":              true,
			"idx_provider_commercial_acceptance_claim_created":   true,
			"idx_provider_commercial_acceptance_offer_created":   true,
			"idx_provider_commercial_acceptance_one_renewal":     true,
			"provider_pilot_companies":                           true,
			"idx_provider_pilot_companies_claim":                 true,
			"provider_commercial_commitment_events":              true,
			"idx_provider_commercial_commitment_company_created": true,
			"idx_provider_commercial_commitment_offer_created":   true,
			"idx_provider_commercial_commitment_related":         true,
			"idx_provider_commercial_one_terms_renewal":          true,
			"provider_action_handoff_receipts":                   true,
			"idx_provider_action_handoff_offer_observed":         true,
		},
		footprintRules: map[string]bool{
			"provider_commercial_acceptance_no_update": true,
			"provider_commercial_acceptance_no_delete": true,
			"provider_pilot_companies_no_update":       true,
			"provider_pilot_companies_no_delete":       true,
			"provider_commercial_commitment_no_update": true,
			"provider_commercial_commitment_no_delete": true,
			"provider_action_handoff_no_update":        true,
			"provider_action_handoff_no_delete":        true,
		},
		footprintProbes: []migrationFootprintProbe{
			{kind: migrationFootprintColumn, relation: "provider_offers", name: "commercial_terms_contract_version"},
			{kind: migrationFootprintColumn, relation: "provider_offers", name: "commercial_terms_sha256"},
			{kind: migrationFootprintColumn, relation: "provider_offers_returned", name: "commercial_terms_contract_version_snapshot"},
			{kind: migrationFootprintColumn, relation: "provider_offers_returned", name: "commercial_terms_sha256_snapshot"},
			{kind: migrationFootprintColumn, relation: "action_tickets", name: "commercial_terms_contract_version_snapshot"},
			{kind: migrationFootprintColumn, relation: "action_tickets", name: "commercial_terms_sha256_snapshot"},
			{kind: migrationFootprintTrigger, relation: "provider_offers", name: "provider_offer_commercial_immutability_enforced"},
			{kind: migrationFootprintFunction, name: "enforce_provider_offer_commercial_immutability"},
			{kind: migrationFootprintTrigger, relation: "provider_action_handoff_receipts", name: "provider_action_handoff_receipt_enforced"},
			{kind: migrationFootprintFunction, name: "enforce_provider_action_handoff_receipt"},
			{kind: migrationFootprintTrigger, relation: "action_tickets", name: "action_ticket_observed_handoff_status_enforced"},
			{kind: migrationFootprintTrigger, relation: "action_tickets", name: "action_ticket_observed_handoff_insert_enforced"},
			{kind: migrationFootprintFunction, name: "enforce_action_ticket_observed_handoff_status"},
		},
	}

	controlledIntentDisclosure := protectedMigrationSpec{
		// 023 adds an opt-in disclosure pair to the existing append-only handoff
		// receipt and makes the separately disclosed ticket bundle immutable except
		// for one-way redaction. The cumulative 022 contract remains intact; altered
		// columns and behavior on inherited tables are explicit ambiguity probes.
		relations:          append([]migrationRelation(nil), commercialProof.relations...),
		rules:              append([]migrationRule(nil), commercialProof.rules...),
		footprintRelations: map[string]bool{},
		footprintRules:     map[string]bool{},
		footprintProbes: []migrationFootprintProbe{
			{kind: migrationFootprintColumn, relation: "provider_action_handoff_receipts", name: "principal_controlled_intent_disclosure_consent"},
			{kind: migrationFootprintColumn, relation: "provider_action_handoff_receipts", name: "controlled_intent_disclosure_consent_version"},
			{kind: migrationFootprintTrigger, relation: "action_tickets", name: "action_ticket_controlled_intent_immutability_enforced"},
			{kind: migrationFootprintFunction, name: "enforce_action_ticket_controlled_intent_immutability"},
		},
	}

	providerPilotBoundary := protectedMigrationSpec{
		// 024 turns the documentary Stage 2 cohort into a database-enforced
		// owner-authorized epoch. The latest protected fingerprint remains
		// cumulative; only the new pilot tables, indexes, rules, and transition
		// behavior count as this migration's ambiguity footprint.
		relations: append(append([]migrationRelation(nil), controlledIntentDisclosure.relations...),
			migrationRelation{name: "provider_pilot_epochs", relkind: "r"},
			migrationRelation{name: "idx_provider_pilot_one_open_epoch", relkind: "i", parent: "provider_pilot_epochs"},
			migrationRelation{name: "provider_pilot_enrollments", relkind: "r"},
			migrationRelation{name: "idx_provider_pilot_enrollments_claim", relkind: "i", parent: "provider_pilot_enrollments"},
			migrationRelation{name: "provider_pilot_epoch_events", relkind: "r"},
			migrationRelation{name: "idx_provider_pilot_epoch_singleton_events", relkind: "i", parent: "provider_pilot_epoch_events"},
			migrationRelation{name: "idx_provider_pilot_epoch_enrollment_events", relkind: "i", parent: "provider_pilot_epoch_events"},
			migrationRelation{name: "idx_action_tickets_pilot_epoch_claim", relkind: "i", parent: "action_tickets"},
		),
		rules: append(append([]migrationRule(nil), controlledIntentDisclosure.rules...),
			migrationRule{relation: "provider_pilot_enrollments", name: "provider_pilot_enrollments_no_update"},
			migrationRule{relation: "provider_pilot_enrollments", name: "provider_pilot_enrollments_no_delete"},
			migrationRule{relation: "provider_pilot_epoch_events", name: "provider_pilot_epoch_events_no_update"},
			migrationRule{relation: "provider_pilot_epoch_events", name: "provider_pilot_epoch_events_no_delete"},
			migrationRule{relation: "provider_pilot_epochs", name: "provider_pilot_epochs_no_delete"},
		),
		fingerprintFunctions: []string{
			"provider_pilot_stage1_eligibility_snapshot_sha256(text,uuid,text,timestamptz,timestamptz,uuid,uuid,uuid,text,timestamptz)",
			"provider_pilot_enrollment_eligibility_is_current(uuid,uuid)",
		},
		footprintRelations: map[string]bool{
			"provider_pilot_epochs":                      true,
			"idx_provider_pilot_one_open_epoch":          true,
			"provider_pilot_enrollments":                 true,
			"idx_provider_pilot_enrollments_claim":       true,
			"provider_pilot_epoch_events":                true,
			"idx_provider_pilot_epoch_singleton_events":  true,
			"idx_provider_pilot_epoch_enrollment_events": true,
			"idx_action_tickets_pilot_epoch_claim":       true,
		},
		footprintRules: map[string]bool{
			"provider_pilot_enrollments_no_update":  true,
			"provider_pilot_enrollments_no_delete":  true,
			"provider_pilot_epoch_events_no_update": true,
			"provider_pilot_epoch_events_no_delete": true,
			"provider_pilot_epochs_no_delete":       true,
		},
		footprintFunctions: map[string]bool{
			"provider_pilot_stage1_eligibility_snapshot_sha256(text,uuid,text,timestamptz,timestamptz,uuid,uuid,uuid,text,timestamptz)": true,
			"provider_pilot_enrollment_eligibility_is_current(uuid,uuid)":                                                               true,
		},
		footprintProbes: []migrationFootprintProbe{
			{kind: migrationFootprintColumn, relation: "provider_offers", name: "provider_pilot_epoch_id"},
			{kind: migrationFootprintColumn, relation: "provider_offers_returned", name: "provider_pilot_epoch_id_snapshot"},
			{kind: migrationFootprintColumn, relation: "action_tickets", name: "provider_pilot_epoch_id"},
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
		},
	}

	stage1FactIntegrity := protectedMigrationSpec{
		// 025 makes the protected migration ledger and every fact counted by the
		// Stage 1 release gate immutable, with database-owned observation clocks.
		// The explicit inherited-table fingerprint set brings their complete
		// definitions (including the new timestamp/update triggers) into the
		// latest receipt without treating legacy tables as 025-created objects.
		relations: append([]migrationRelation(nil), providerPilotBoundary.relations...),
		rules: append(append([]migrationRule(nil), providerPilotBoundary.rules...),
			migrationRule{relation: "nhs_schema_migrations", name: "nhs_schema_migrations_no_update"},
			migrationRule{relation: "nhs_schema_migrations", name: "nhs_schema_migrations_no_delete"},
		),
		fingerprintTables: []string{
			"nhs_schema_migrations",
			"search_receipts",
			"organic_results_returned",
			"result_selections",
			"action_interest_receipts",
		},
		fingerprintFunctions: append([]string(nil), providerPilotBoundary.fingerprintFunctions...),
		footprintRelations:   map[string]bool{},
		footprintRules: map[string]bool{
			"nhs_schema_migrations_no_update": true,
			"nhs_schema_migrations_no_delete": true,
		},
		footprintProbes: []migrationFootprintProbe{
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
		},
	}

	providerPilotProofIntegrity := protectedMigrationSpec{
		// 026 adds no new relation. It strengthens outcome receipts and pilot
		// lifecycle rows inherited through 025, so the complete cumulative
		// relation/rule contract and inherited-table fingerprints must carry
		// forward while only its new functions/triggers form the ambiguity
		// footprint.
		relations:         append([]migrationRelation(nil), stage1FactIntegrity.relations...),
		rules:             append([]migrationRule(nil), stage1FactIntegrity.rules...),
		fingerprintTables: append([]string(nil), stage1FactIntegrity.fingerprintTables...),
		fingerprintFunctions: append(
			[]string(nil), stage1FactIntegrity.fingerprintFunctions...,
		),
		footprintRelations: map[string]bool{},
		footprintRules:     map[string]bool{},
		footprintProbes: []migrationFootprintProbe{
			{kind: migrationFootprintFunction, name: "enforce_provider_pilot_outcome_receipt"},
			{kind: migrationFootprintTrigger, relation: "outcome_receipts", name: "provider_pilot_outcome_receipt_enforced"},
			{kind: migrationFootprintFunction, name: "require_provider_pilot_lifecycle_event"},
			{kind: migrationFootprintTrigger, relation: "provider_pilot_epochs", name: "provider_pilot_epoch_created_event_required"},
			{kind: migrationFootprintTrigger, relation: "provider_pilot_enrollments", name: "provider_pilot_enrollment_event_required"},
			{kind: migrationFootprintTrigger, relation: "provider_pilot_epochs", name: "provider_pilot_epoch_transition_event_required"},
		},
	}

	providerPilotReviewEvidence := protectedMigrationSpec{
		// 027 adds append-only owner review evidence. The generic review subject
		// is resolved and hash-checked by a table trigger, while the standalone
		// canonical snapshot routine is carried explicitly in every later schema
		// fingerprint so changing its body cannot preserve a green receipt.
		relations: append(append([]migrationRelation(nil), providerPilotProofIntegrity.relations...),
			migrationRelation{name: "provider_pilot_review_events", relkind: "r"},
			migrationRelation{name: "idx_provider_pilot_reviews_pilot_type", relkind: "i", parent: "provider_pilot_review_events"},
			migrationRelation{name: "idx_provider_pilot_reviews_claim_type", relkind: "i", parent: "provider_pilot_review_events"},
		),
		rules: append(append([]migrationRule(nil), providerPilotProofIntegrity.rules...),
			migrationRule{relation: "provider_pilot_review_events", name: "provider_pilot_review_events_no_update"},
			migrationRule{relation: "provider_pilot_review_events", name: "provider_pilot_review_events_no_delete"},
		),
		fingerprintTables: append([]string(nil), providerPilotProofIntegrity.fingerprintTables...),
		fingerprintFunctions: append(
			append([]string(nil), providerPilotProofIntegrity.fingerprintFunctions...),
			"provider_pilot_review_snapshot_sha256(uuid,text,uuid)",
		),
		footprintRelations: map[string]bool{
			"provider_pilot_review_events":          true,
			"idx_provider_pilot_reviews_pilot_type": true,
			"idx_provider_pilot_reviews_claim_type": true,
		},
		footprintRules: map[string]bool{
			"provider_pilot_review_events_no_update": true,
			"provider_pilot_review_events_no_delete": true,
		},
		footprintFunctions: map[string]bool{
			"provider_pilot_review_snapshot_sha256(uuid,text,uuid)": true,
		},
		footprintProbes: []migrationFootprintProbe{
			{kind: migrationFootprintTrigger, relation: "provider_pilot_review_events", name: "provider_pilot_review_event_enforced"},
			{kind: migrationFootprintFunction, name: "enforce_provider_pilot_review_event"},
			{kind: migrationFootprintTrigger, relation: "provider_pilot_epochs", name: "provider_pilot_activation_provider_reviews"},
			{kind: migrationFootprintFunction, name: "enforce_provider_pilot_epoch_provider_reviews"},
			{kind: migrationFootprintTrigger, relation: "provider_offers", name: "provider_offer_activation_review"},
			{kind: migrationFootprintFunction, name: "enforce_provider_offer_pre_activation_review"},
			{kind: migrationFootprintTrigger, relation: "provider_action_handoff_receipts", name: "provider_handoff_ticket_review"},
			{kind: migrationFootprintFunction, name: "enforce_provider_handoff_ticket_review"},
		},
	}

	providerCommercialProofManifest := protectedMigrationSpec{
		// 028 adds one append-only, aggregate signed proof per exact closed
		// pilot. The table trigger owns JSON/relational/time/hash bindings;
		// production signing-key retention separately re-verifies the HMAC.
		relations: append(append([]migrationRelation(nil), providerPilotReviewEvidence.relations...),
			migrationRelation{name: "provider_commercial_proof_manifests", relkind: "r"},
			migrationRelation{name: "idx_provider_proof_manifests_issued", relkind: "i", parent: "provider_commercial_proof_manifests"},
			migrationRelation{name: "idx_provider_proof_manifests_key", relkind: "i", parent: "provider_commercial_proof_manifests"},
		),
		rules: append(append([]migrationRule(nil), providerPilotReviewEvidence.rules...),
			migrationRule{relation: "provider_commercial_proof_manifests", name: "provider_commercial_proof_manifests_no_update"},
			migrationRule{relation: "provider_commercial_proof_manifests", name: "provider_commercial_proof_manifests_no_delete"},
		),
		fingerprintTables:    append([]string(nil), providerPilotReviewEvidence.fingerprintTables...),
		fingerprintFunctions: append([]string(nil), providerPilotReviewEvidence.fingerprintFunctions...),
		footprintRelations: map[string]bool{
			"provider_commercial_proof_manifests": true,
			"idx_provider_proof_manifests_issued": true,
			"idx_provider_proof_manifests_key":    true,
		},
		footprintRules: map[string]bool{
			"provider_commercial_proof_manifests_no_update": true,
			"provider_commercial_proof_manifests_no_delete": true,
		},
		footprintProbes: []migrationFootprintProbe{
			{kind: migrationFootprintTrigger, relation: "provider_commercial_proof_manifests", name: "provider_commercial_proof_manifest_enforced"},
			{kind: migrationFootprintFunction, name: "enforce_provider_commercial_proof_manifest"},
		},
	}

	providerSettlementReceipts := protectedMigrationSpec{
		// 029 makes collection a first-class, append-only external fact. The
		// current fingerprint remains cumulative; the new relations, rules, and
		// insert triggers are the only 029 ambiguity footprint.
		relations: append(append([]migrationRelation(nil), providerCommercialProofManifest.relations...),
			migrationRelation{name: "provider_settlement_orders", relkind: "r"},
			migrationRelation{name: "idx_provider_settlement_orders_claim_created", relkind: "i", parent: "provider_settlement_orders"},
			migrationRelation{name: "idx_provider_settlement_orders_offer_created", relkind: "i", parent: "provider_settlement_orders"},
			migrationRelation{name: "provider_settlement_checkout_sessions", relkind: "r"},
			migrationRelation{name: "idx_provider_settlement_checkout_created", relkind: "i", parent: "provider_settlement_checkout_sessions"},
			migrationRelation{name: "provider_settlement_payment_receipts", relkind: "r"},
			migrationRelation{name: "idx_provider_settlement_payment_paid", relkind: "i", parent: "provider_settlement_payment_receipts"},
		),
		rules: append(append([]migrationRule(nil), providerCommercialProofManifest.rules...),
			migrationRule{relation: "provider_settlement_orders", name: "provider_settlement_orders_no_update"},
			migrationRule{relation: "provider_settlement_orders", name: "provider_settlement_orders_no_delete"},
			migrationRule{relation: "provider_settlement_checkout_sessions", name: "provider_settlement_checkout_sessions_no_update"},
			migrationRule{relation: "provider_settlement_checkout_sessions", name: "provider_settlement_checkout_sessions_no_delete"},
			migrationRule{relation: "provider_settlement_payment_receipts", name: "provider_settlement_payment_receipts_no_update"},
			migrationRule{relation: "provider_settlement_payment_receipts", name: "provider_settlement_payment_receipts_no_delete"},
		),
		fingerprintTables:    append([]string(nil), providerCommercialProofManifest.fingerprintTables...),
		fingerprintFunctions: append([]string(nil), providerCommercialProofManifest.fingerprintFunctions...),
		footprintRelations: map[string]bool{
			"provider_settlement_orders":                   true,
			"idx_provider_settlement_orders_claim_created": true,
			"idx_provider_settlement_orders_offer_created": true,
			"provider_settlement_checkout_sessions":        true,
			"idx_provider_settlement_checkout_created":     true,
			"provider_settlement_payment_receipts":         true,
			"idx_provider_settlement_payment_paid":         true,
		},
		footprintRules: map[string]bool{
			"provider_settlement_orders_no_update":            true,
			"provider_settlement_orders_no_delete":            true,
			"provider_settlement_checkout_sessions_no_update": true,
			"provider_settlement_checkout_sessions_no_delete": true,
			"provider_settlement_payment_receipts_no_update":  true,
			"provider_settlement_payment_receipts_no_delete":  true,
		},
		footprintProbes: []migrationFootprintProbe{
			{kind: migrationFootprintTrigger, relation: "provider_settlement_orders", name: "provider_settlement_order_enforced"},
			{kind: migrationFootprintFunction, name: "enforce_provider_settlement_order"},
			{kind: migrationFootprintTrigger, relation: "provider_settlement_checkout_sessions", name: "provider_settlement_checkout_session_enforced"},
			{kind: migrationFootprintFunction, name: "enforce_provider_settlement_checkout_session"},
			{kind: migrationFootprintTrigger, relation: "provider_settlement_payment_receipts", name: "provider_settlement_payment_receipt_enforced"},
			{kind: migrationFootprintFunction, name: "enforce_provider_settlement_payment_receipt"},
		},
	}

	providerProcessorNetReceipts := protectedMigrationSpec{
		// 030 binds actual processor gross/fee/net evidence to every paid
		// settlement and requires a separate availability observation before
		// retained value can enter mechanism proof.
		relations: append(append([]migrationRelation(nil), providerSettlementReceipts.relations...),
			migrationRelation{name: "provider_settlement_processor_balance_receipts", relkind: "r"},
			migrationRelation{name: "idx_provider_settlement_processor_balance_created", relkind: "i", parent: "provider_settlement_processor_balance_receipts"},
			migrationRelation{name: "provider_settlement_processor_availability_receipts", relkind: "r"},
			migrationRelation{name: "idx_provider_settlement_processor_available_created", relkind: "i", parent: "provider_settlement_processor_availability_receipts"},
		),
		rules: append(append([]migrationRule(nil), providerSettlementReceipts.rules...),
			migrationRule{relation: "provider_settlement_processor_balance_receipts", name: "provider_settlement_processor_balance_receipts_no_update"},
			migrationRule{relation: "provider_settlement_processor_balance_receipts", name: "provider_settlement_processor_balance_receipts_no_delete"},
			migrationRule{relation: "provider_settlement_processor_availability_receipts", name: "provider_settlement_processor_availability_receipts_no_update"},
			migrationRule{relation: "provider_settlement_processor_availability_receipts", name: "provider_settlement_processor_availability_receipts_no_delete"},
		),
		fingerprintTables:    append([]string(nil), providerSettlementReceipts.fingerprintTables...),
		fingerprintFunctions: append([]string(nil), providerSettlementReceipts.fingerprintFunctions...),
		footprintRelations: map[string]bool{
			"provider_settlement_processor_balance_receipts":      true,
			"idx_provider_settlement_processor_balance_created":   true,
			"provider_settlement_processor_availability_receipts": true,
			"idx_provider_settlement_processor_available_created": true,
		},
		footprintRules: map[string]bool{
			"provider_settlement_processor_balance_receipts_no_update":      true,
			"provider_settlement_processor_balance_receipts_no_delete":      true,
			"provider_settlement_processor_availability_receipts_no_update": true,
			"provider_settlement_processor_availability_receipts_no_delete": true,
		},
		footprintProbes: []migrationFootprintProbe{
			{kind: migrationFootprintTrigger, relation: "provider_settlement_processor_balance_receipts", name: "provider_settlement_processor_balance_receipt_enforced"},
			{kind: migrationFootprintFunction, name: "enforce_provider_settlement_processor_balance_receipt"},
			{kind: migrationFootprintTrigger, relation: "provider_settlement_processor_availability_receipts", name: "provider_settlement_processor_availability_receipt_enforced"},
			{kind: migrationFootprintFunction, name: "enforce_provider_settlement_processor_availability_receipt"},
		},
	}

	actionInterestAttemptFunnel := protectedMigrationSpec{
		// 031 adds only privacy-safe UTC-day/surface/outcome counters. It is
		// cumulative with the settlement proof schema but remains operational
		// diagnostics rather than demand or commercial evidence.
		relations: append(append([]migrationRelation(nil), providerProcessorNetReceipts.relations...),
			migrationRelation{name: "action_interest_attempt_daily", relkind: "r"},
			migrationRelation{name: "idx_action_interest_attempt_daily_recent", relkind: "i", parent: "action_interest_attempt_daily"},
		),
		rules:                append([]migrationRule(nil), providerProcessorNetReceipts.rules...),
		fingerprintTables:    append(append([]string(nil), providerProcessorNetReceipts.fingerprintTables...), "action_interest_attempt_daily"),
		fingerprintFunctions: append([]string(nil), providerProcessorNetReceipts.fingerprintFunctions...),
		footprintRelations: map[string]bool{
			"action_interest_attempt_daily":            true,
			"idx_action_interest_attempt_daily_recent": true,
		},
		footprintProbes: []migrationFootprintProbe{
			{kind: migrationFootprintTrigger, relation: "action_interest_attempt_daily", name: "action_interest_attempt_aggregate_owned"},
			{kind: migrationFootprintFunction, name: "own_action_interest_attempt_aggregate"},
		},
	}

	return map[string]protectedMigrationSpec{
		"019_provider_exchange.sql":                     providerExchange,
		"020_action_interest_receipts.sql":              actionInterests,
		"021_provider_capacity_reservations.sql":        capacityReservations,
		"022_provider_commercial_proof.sql":             commercialProof,
		"023_provider_controlled_intent_disclosure.sql": controlledIntentDisclosure,
		"024_provider_pilot_boundary.sql":               providerPilotBoundary,
		"025_stage1_fact_integrity.sql":                 stage1FactIntegrity,
		"026_provider_pilot_proof_integrity.sql":        providerPilotProofIntegrity,
		"027_provider_pilot_review_evidence.sql":        providerPilotReviewEvidence,
		"028_provider_commercial_proof_manifest.sql":    providerCommercialProofManifest,
		"029_provider_settlement_receipts.sql":          providerSettlementReceipts,
		"030_provider_processor_net_receipts.sql":       providerProcessorNetReceipts,
		"031_action_interest_attempt_funnel.sql":        actionInterestAttemptFunnel,
	}
}

func Connect() error {
	return ConnectWithReleaseRevision("development")
}

// ConnectWithReleaseRevision tags every pooled server connection with the
// exact compiled release. Cutover preflight can therefore distinguish old
// application sessions from the candidate without exposing the database DSN.
func ConnectWithReleaseRevision(releaseRevision string) error {
	return ConnectWithReleaseRevisionContext(context.Background(), releaseRevision)
}

// ConnectWithReleaseRevisionContext applies the caller's deadline to the
// initial database handshake. Cutover preflight must not outlive its advertised
// bound because lib/pq's default background ping may otherwise stall on a
// broken route or unavailable database.
func ConnectWithReleaseRevisionContext(ctx context.Context, releaseRevision string) error {
	if ctx == nil {
		return fmt.Errorf("database connection context is unavailable")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}
	taggedDSN, err := databaseDSNWithReleaseApplicationName(dsn, releaseRevision)
	if err != nil {
		return err
	}

	DB, err = sql.Open("postgres", taggedDSN)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	if err := DB.PingContext(ctx); err != nil {
		_ = DB.Close()
		DB = nil
		return fmt.Errorf("ping: %w", err)
	}

	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	return nil
}

func databaseDSNWithReleaseApplicationName(dsn, releaseRevision string) (string, error) {
	revision := strings.ToLower(strings.TrimSpace(releaseRevision))
	if revision != "development" {
		if _, err := protectedMigrationRevision(revision); err != nil {
			return "", fmt.Errorf("database application identity: %w", err)
		}
	}
	applicationName := "nhs-server:" + revision
	parsed, err := url.Parse(dsn)
	if err == nil && (parsed.Scheme == "postgres" || parsed.Scheme == "postgresql") {
		query := parsed.Query()
		query.Set("application_name", applicationName)
		parsed.RawQuery = query.Encode()
		return parsed.String(), nil
	}
	// applicationName is constructed only from a fixed prefix and a validated
	// hexadecimal revision (or the fixed development label), so quoting this
	// lib/pq keyword parameter cannot introduce DSN syntax.
	return strings.TrimSpace(dsn) + " application_name='" + applicationName + "'", nil
}

func RunMigrations(dir, releaseRevision string) error {
	return RunMigrationsWithPreflight(dir, releaseRevision, nil)
}

func RunMigrationsWithPreflight(dir, releaseRevision string, preflight ProtectedMigrationPreflight) error {
	if DB == nil {
		return fmt.Errorf("database is not connected")
	}
	migrations, err := loadMigrations(dir)
	if err != nil {
		return err
	}

	ctx, cancelMigration := context.WithTimeout(context.Background(), migrationOverallTimeout)
	defer cancelMigration()
	conn, err := DB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve migration connection: %w", err)
	}
	defer conn.Close()
	lockCtx, cancelLock := context.WithTimeout(ctx, 30*time.Second)
	defer cancelLock()
	if _, err := conn.ExecContext(lockCtx, `SELECT pg_advisory_lock($1, $2)`, migrationLockClass, migrationLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		unlockCtx, cancelUnlock := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelUnlock()
		if _, err := conn.ExecContext(unlockCtx, `SELECT pg_advisory_unlock($1, $2)`, migrationLockClass, migrationLockID); err != nil {
			log.Printf("release migration lock: %v", err)
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()

	// Check every protected receipt and schema footprint before replaying even
	// the legacy migrations. Ambiguity must stop all schema mutation.
	latestReceipt, err := preflightProtectedMigrations(ctx, conn, migrations, releaseRevision)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if isProtectedMigration(migration.name) {
			spec := protectedMigrationSpecs[migration.name]
			applied, err := applyProtectedMigration(ctx, conn, migration, spec, releaseRevision, migration.name == latestReceipt, preflight)
			if err != nil {
				return err
			}
			if applied {
				log.Printf("protected migration applied with exact receipt: %s", migration.name)
			} else {
				log.Printf("protected migration receipt verified: %s", migration.name)
			}
			continue
		}
		if err := applyLegacyMigration(ctx, conn, migration); err != nil {
			return err
		}
	}
	return nil
}

type ProtectedMigrationReadiness struct {
	Name                   string `json:"name"`
	ExpectedSHA256         string `json:"expected_sha256"`
	ReceiptExists          bool   `json:"receipt_exists"`
	ReceiptMatchesFile     bool   `json:"receipt_matches_file"`
	SchemaComplete         bool   `json:"schema_complete"`
	LatestSchemaMatches    bool   `json:"latest_schema_matches"`
	UnreceiptedFootprint   bool   `json:"unreceipted_footprint"`
	ReceiptAppliedByCommit string `json:"receipt_applied_by_commit,omitempty"`
}

type MigrationReadinessReport struct {
	CandidateRevision      string                        `json:"candidate_revision"`
	LatestProtectedReceipt string                        `json:"latest_protected_receipt,omitempty"`
	PendingProtectedCount  int                           `json:"pending_protected_count"`
	DatabaseAhead          bool                          `json:"database_ahead"`
	Protected              []ProtectedMigrationReadiness `json:"protected"`
}

// InspectMigrationReadiness executes the exact protected migration receipt,
// checksum, schema-fingerprint, database-ahead, and unreceipted-footprint gates
// without acquiring the migration advisory lock or changing database state.
func InspectMigrationReadiness(ctx context.Context, dir, releaseRevision string) (*MigrationReadinessReport, error) {
	if ctx == nil || DB == nil {
		return nil, fmt.Errorf("database is not connected")
	}
	revision, err := protectedMigrationRevision(releaseRevision)
	if err != nil {
		return nil, err
	}
	migrations, err := loadMigrations(dir)
	if err != nil {
		return nil, err
	}
	conn, err := DB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("reserve migration inspection connection: %w", err)
	}
	defer conn.Close()
	latestReceipt, err := preflightProtectedMigrations(ctx, conn, migrations, revision)
	if err != nil {
		return nil, err
	}
	report := &MigrationReadinessReport{
		CandidateRevision:      revision,
		LatestProtectedReceipt: latestReceipt,
		DatabaseAhead:          false,
		Protected:              []ProtectedMigrationReadiness{},
	}
	for _, migration := range migrations {
		if !isProtectedMigration(migration.name) {
			continue
		}
		state, err := inspectProtectedMigration(ctx, conn, migration.name, protectedMigrationSpecs[migration.name])
		if err != nil {
			return nil, fmt.Errorf("inspect protected migration %s: %w", migration.name, err)
		}
		item := ProtectedMigrationReadiness{
			Name:               migration.name,
			ExpectedSHA256:     migration.sha256,
			ReceiptExists:      state.receiptExists,
			ReceiptMatchesFile: state.receiptExists && state.receiptSHA256 == migration.sha256,
			SchemaComplete:     state.complete,
			LatestSchemaMatches: migration.name != latestReceipt ||
				(state.receiptExists && state.receiptSchemaSHA256 == state.currentSchemaSHA256),
			UnreceiptedFootprint:   state.anyFootprint && !state.receiptExists,
			ReceiptAppliedByCommit: state.receiptRevision,
		}
		if !state.receiptExists {
			report.PendingProtectedCount++
		}
		report.Protected = append(report.Protected, item)
	}
	return report, nil
}

func loadMigrations(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	migrations := make([]migrationFile, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		digest := sha256.Sum256(data)
		migrations = append(migrations, migrationFile{
			name: f, data: data, sha256: hex.EncodeToString(digest[:]),
		})
	}
	return migrations, nil
}

func isProtectedMigration(name string) bool {
	return name >= protectedMigrationStart
}

func validateRequiredProtectedMigrationSet(migrations []migrationFile) error {
	present := make(map[string]bool, len(migrations))
	protectedCount := 0
	for _, migration := range migrations {
		if !isProtectedMigration(migration.name) {
			continue
		}
		protectedCount++
		present[migration.name] = true
	}
	for _, required := range requiredProtectedMigrationNames {
		if !present[required] {
			return fmt.Errorf("required protected migration is missing: %s", required)
		}
	}
	if protectedCount != len(requiredProtectedMigrationNames) {
		return fmt.Errorf("protected migration set is not exact through %s", requiredProtectedMigrationNames[len(requiredProtectedMigrationNames)-1])
	}
	return nil
}

func preflightProtectedMigrations(ctx context.Context, conn *sql.Conn, migrations []migrationFile, releaseRevision string) (string, error) {
	if !allowPartialProtectedMigrationsForTests {
		if err := validateRequiredProtectedMigrationSet(migrations); err != nil {
			return "", err
		}
	}
	known := make(map[string]bool, len(migrations))
	for _, migration := range migrations {
		if isProtectedMigration(migration.name) {
			known[migration.name] = true
		}
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		return "", err
	}

	var ledgerExists bool
	if err := conn.QueryRowContext(ctx, `SELECT to_regclass('public.nhs_schema_migrations') IS NOT NULL`).Scan(&ledgerExists); err != nil {
		return "", fmt.Errorf("inspect migration receipt table: %w", err)
	}
	latestReceipt := ""
	if ledgerExists {
		rows, err := conn.QueryContext(ctx, `
			SELECT name
			FROM public.nhs_schema_migrations
			WHERE name >= $1
			ORDER BY name`, protectedMigrationStart)
		if err != nil {
			return "", fmt.Errorf("read protected migration receipts: %w", err)
		}
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				_ = rows.Close()
				return "", fmt.Errorf("scan protected migration receipt: %w", err)
			}
			if !known[name] {
				_ = rows.Close()
				return "", fmt.Errorf("database_ahead_of_binary: protected migration receipt %s is absent from this release", name)
			}
			latestReceipt = name
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return "", fmt.Errorf("read protected migration receipts: %w", err)
		}
		if err := rows.Close(); err != nil {
			return "", fmt.Errorf("close protected migration receipts: %w", err)
		}
	}

	for _, migration := range migrations {
		if !isProtectedMigration(migration.name) {
			continue
		}
		spec := protectedMigrationSpecs[migration.name]
		state, err := inspectProtectedMigration(ctx, conn, migration.name, spec)
		if err != nil {
			return "", fmt.Errorf("preflight protected migration %s: %w", migration.name, err)
		}
		if err := validateProtectedMigrationState(migration, state, migration.name == latestReceipt); err != nil {
			return "", err
		}
		if !state.receiptExists {
			if _, err := protectedMigrationRevision(releaseRevision); err != nil {
				return "", fmt.Errorf("protected migration %s: %w", migration.name, err)
			}
		}
	}
	return latestReceipt, nil
}

// Each later protected specification is cumulative so the latest receipt can
// fingerprint the complete protected schema after reviewed ALTER migrations.
// Earlier receipts retain immutable file checksums but no longer own the final
// catalog shape after a successor migration has been applied.
func validateProtectedMigrationSpecChain(migrations []migrationFile) error {
	priorRelations := map[string]migrationRelation{}
	priorRules := map[string]migrationRule{}
	priorFingerprintTables := map[string]bool{}
	priorFingerprintFunctions := map[string]bool{}
	first := true
	for _, migration := range migrations {
		if !isProtectedMigration(migration.name) {
			continue
		}
		spec, ok := protectedMigrationSpecs[migration.name]
		if !ok {
			return fmt.Errorf("protected migration %s has no schema-footprint contract", migration.name)
		}
		if first && !spec.allObjectsAreFootprint {
			return fmt.Errorf("first protected migration %s must treat all declared objects as its footprint", migration.name)
		}
		if !first && spec.allObjectsAreFootprint {
			return fmt.Errorf("protected migration %s cannot treat cumulative prior objects as a new footprint", migration.name)
		}
		currentRelations := map[string]migrationRelation{}
		for _, relation := range spec.relations {
			if _, duplicate := currentRelations[relation.name]; duplicate {
				return fmt.Errorf("protected migration %s repeats relation %s", migration.name, relation.name)
			}
			currentRelations[relation.name] = relation
		}
		currentRules := map[string]migrationRule{}
		for _, rule := range spec.rules {
			if _, duplicate := currentRules[rule.name]; duplicate {
				return fmt.Errorf("protected migration %s repeats rule %s", migration.name, rule.name)
			}
			currentRules[rule.name] = rule
		}
		currentFingerprintTables := map[string]bool{}
		for _, table := range spec.fingerprintTables {
			if table == "" {
				return fmt.Errorf("protected migration %s has an empty fingerprint table", migration.name)
			}
			if currentFingerprintTables[table] {
				return fmt.Errorf("protected migration %s repeats fingerprint table %s", migration.name, table)
			}
			currentFingerprintTables[table] = true
		}
		currentFingerprintFunctions := map[string]bool{}
		for _, function := range spec.fingerprintFunctions {
			if strings.TrimSpace(function) == "" {
				return fmt.Errorf("protected migration %s has an empty fingerprint function", migration.name)
			}
			if currentFingerprintFunctions[function] {
				return fmt.Errorf("protected migration %s repeats fingerprint function %s", migration.name, function)
			}
			currentFingerprintFunctions[function] = true
		}
		for name, prior := range priorRelations {
			current, ok := currentRelations[name]
			if !ok || current.relkind != prior.relkind || current.parent != prior.parent {
				return fmt.Errorf("protected migration %s does not carry forward relation contract %s", migration.name, name)
			}
		}
		for name, prior := range priorRules {
			current, ok := currentRules[name]
			if !ok || current.relation != prior.relation {
				return fmt.Errorf("protected migration %s does not carry forward rule contract %s", migration.name, name)
			}
		}
		for table := range priorFingerprintTables {
			if !currentFingerprintTables[table] {
				return fmt.Errorf("protected migration %s does not carry forward fingerprint table %s", migration.name, table)
			}
		}
		for function := range priorFingerprintFunctions {
			if !currentFingerprintFunctions[function] {
				return fmt.Errorf("protected migration %s does not carry forward fingerprint function %s", migration.name, function)
			}
		}
		for name := range spec.footprintRelations {
			if _, ok := currentRelations[name]; !ok {
				return fmt.Errorf("protected migration %s names unknown footprint relation %s", migration.name, name)
			}
		}
		for name := range spec.footprintRules {
			if _, ok := currentRules[name]; !ok {
				return fmt.Errorf("protected migration %s names unknown footprint rule %s", migration.name, name)
			}
		}
		for name := range spec.footprintFunctions {
			if !currentFingerprintFunctions[name] {
				return fmt.Errorf("protected migration %s names unknown footprint function %s", migration.name, name)
			}
		}
		seenProbes := map[string]bool{}
		for _, probe := range spec.footprintProbes {
			key, err := migrationFootprintProbeKey(probe)
			if err != nil {
				return fmt.Errorf("protected migration %s has invalid footprint probe: %w", migration.name, err)
			}
			if seenProbes[key] {
				return fmt.Errorf("protected migration %s repeats footprint probe %s", migration.name, key)
			}
			seenProbes[key] = true
			if probe.kind == migrationFootprintColumn || probe.kind == migrationFootprintTrigger {
				relation, ok := currentRelations[probe.relation]
				declaredTable := ok && relation.relkind == "r"
				if !declaredTable && !currentFingerprintTables[probe.relation] {
					return fmt.Errorf("protected migration %s footprint probe %s names unknown table %s", migration.name, key, probe.relation)
				}
			}
		}
		if !first {
			for name := range currentRelations {
				_, inherited := priorRelations[name]
				marked := spec.footprintRelations[name]
				switch {
				case inherited && marked:
					return fmt.Errorf("protected migration %s marks inherited relation %s as a new footprint", migration.name, name)
				case !inherited && !marked:
					return fmt.Errorf("protected migration %s omits new relation %s from its footprint", migration.name, name)
				}
			}
			for name := range currentRules {
				_, inherited := priorRules[name]
				marked := spec.footprintRules[name]
				switch {
				case inherited && marked:
					return fmt.Errorf("protected migration %s marks inherited rule %s as a new footprint", migration.name, name)
				case !inherited && !marked:
					return fmt.Errorf("protected migration %s omits new rule %s from its footprint", migration.name, name)
				}
			}
			for name := range currentFingerprintFunctions {
				inherited := priorFingerprintFunctions[name]
				marked := spec.footprintFunctions[name]
				switch {
				case inherited && marked:
					return fmt.Errorf("protected migration %s marks inherited fingerprint function %s as a new footprint", migration.name, name)
				case !inherited && !marked:
					return fmt.Errorf("protected migration %s omits new fingerprint function %s from its footprint", migration.name, name)
				}
			}
		}
		priorRelations = currentRelations
		priorRules = currentRules
		priorFingerprintTables = currentFingerprintTables
		priorFingerprintFunctions = currentFingerprintFunctions
		first = false
	}
	return nil
}

func migrationFootprintProbeKey(probe migrationFootprintProbe) (string, error) {
	if probe.name == "" {
		return "", fmt.Errorf("%s footprint has an empty name", probe.kind)
	}
	switch probe.kind {
	case migrationFootprintColumn, migrationFootprintTrigger:
		if probe.relation == "" {
			return "", fmt.Errorf("%s footprint %s has an empty relation", probe.kind, probe.name)
		}
		return string(probe.kind) + ":" + probe.relation + ":" + probe.name, nil
	case migrationFootprintFunction:
		if probe.relation != "" {
			return "", fmt.Errorf("function footprint %s cannot name relation %s", probe.name, probe.relation)
		}
		return string(probe.kind) + ":" + probe.name, nil
	default:
		return "", fmt.Errorf("unknown footprint kind %q for %s", probe.kind, probe.name)
	}
}

type migrationQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type protectedMigrationState struct {
	ledgerExists        bool
	receiptExists       bool
	receiptSHA256       string
	receiptSchemaSHA256 string
	receiptRevision     string
	currentSchemaSHA256 string
	anyFootprint        bool
	complete            bool
}

func inspectProtectedMigration(ctx context.Context, db migrationQueryer, name string, spec protectedMigrationSpec) (protectedMigrationState, error) {
	state := protectedMigrationState{complete: true}
	var ledgerExists bool
	if err := db.QueryRowContext(ctx, `SELECT to_regclass('public.nhs_schema_migrations') IS NOT NULL`).Scan(&ledgerExists); err != nil {
		return state, fmt.Errorf("inspect migration receipt table: %w", err)
	}
	state.ledgerExists = ledgerExists
	if ledgerExists {
		err := db.QueryRowContext(ctx, `
			SELECT sha256, schema_sha256, applied_by_commit
			FROM public.nhs_schema_migrations
			WHERE name = $1`, name).Scan(&state.receiptSHA256, &state.receiptSchemaSHA256, &state.receiptRevision)
		switch {
		case err == nil:
			state.receiptExists = true
		case err == sql.ErrNoRows:
		case err != nil:
			return state, fmt.Errorf("read migration receipt: %w", err)
		}
	}

	for _, relation := range spec.relations {
		var actualKind, actualParent string
		if err := db.QueryRowContext(ctx, `
			SELECT COALESCE((
				SELECT c.relkind::text || ':' || COALESCE(parent.relname, '')
				FROM pg_catalog.pg_class c
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				LEFT JOIN pg_catalog.pg_index ix ON ix.indexrelid = c.oid
				LEFT JOIN pg_catalog.pg_class parent ON parent.oid = ix.indrelid
				WHERE n.nspname = 'public' AND c.relname = $1
				LIMIT 1
			), '')`, relation.name).Scan(&actualKind); err != nil {
			return state, fmt.Errorf("inspect relation %s: %w", relation.name, err)
		}
		if actualKind != "" {
			if spec.allObjectsAreFootprint || spec.footprintRelations[relation.name] {
				state.anyFootprint = true
			}
			parts := strings.SplitN(actualKind, ":", 2)
			actualKind = parts[0]
			if len(parts) == 2 {
				actualParent = parts[1]
			}
		}
		if actualKind != relation.relkind || actualParent != relation.parent {
			state.complete = false
		}
	}
	for _, rule := range spec.rules {
		var actualRelation string
		if err := db.QueryRowContext(ctx, `
			SELECT COALESCE((
				SELECT c.relname
				FROM pg_catalog.pg_rewrite r
				JOIN pg_catalog.pg_class c ON c.oid = r.ev_class
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = 'public' AND r.rulename = $1
				LIMIT 1
			), '')`, rule.name).Scan(&actualRelation); err != nil {
			return state, fmt.Errorf("inspect rule %s: %w", rule.name, err)
		}
		if actualRelation != "" {
			if spec.allObjectsAreFootprint || spec.footprintRules[rule.name] {
				state.anyFootprint = true
			}
		}
		if actualRelation != rule.relation {
			state.complete = false
		}
	}
	for _, function := range spec.fingerprintFunctions {
		var exists bool
		if err := db.QueryRowContext(ctx, `
			SELECT to_regprocedure('public.' || $1) IS NOT NULL`, function).Scan(&exists); err != nil {
			return state, fmt.Errorf("inspect fingerprint function %s: %w", function, err)
		}
		if exists && spec.footprintFunctions[function] {
			state.anyFootprint = true
		}
		if !exists {
			state.complete = false
		}
	}
	if !state.anyFootprint {
		for _, probe := range spec.footprintProbes {
			exists, err := protectedMigrationFootprintProbeExists(ctx, db, probe)
			if err != nil {
				return state, err
			}
			if exists {
				state.anyFootprint = true
				break
			}
		}
	}
	if state.complete {
		fingerprint, err := protectedMigrationSchemaFingerprint(ctx, db, spec)
		if err != nil {
			return state, err
		}
		state.currentSchemaSHA256 = fingerprint
	}
	return state, nil
}

func protectedMigrationFootprintProbeExists(ctx context.Context, db migrationQueryer, probe migrationFootprintProbe) (bool, error) {
	var exists bool
	switch probe.kind {
	case migrationFootprintColumn:
		if err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_catalog.pg_attribute a
				JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = 'public'
				  AND c.relname = $1
				  AND c.relkind IN ('r', 'p')
				  AND a.attname = $2
				  AND a.attnum > 0
				  AND NOT a.attisdropped
			)`, probe.relation, probe.name).Scan(&exists); err != nil {
			return false, fmt.Errorf("inspect footprint column %s.%s: %w", probe.relation, probe.name, err)
		}
	case migrationFootprintTrigger:
		if err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_catalog.pg_trigger tg
				JOIN pg_catalog.pg_class c ON c.oid = tg.tgrelid
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = 'public'
				  AND c.relname = $1
				  AND tg.tgname = $2
				  AND NOT tg.tgisinternal
			)`, probe.relation, probe.name).Scan(&exists); err != nil {
			return false, fmt.Errorf("inspect footprint trigger %s.%s: %w", probe.relation, probe.name, err)
		}
	case migrationFootprintFunction:
		if err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_catalog.pg_proc p
				JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
				WHERE n.nspname = 'public'
				  AND p.proname = $1
				  AND p.prorettype = 'pg_catalog.trigger'::pg_catalog.regtype
				  AND pg_catalog.pg_get_function_identity_arguments(p.oid) = ''
			)`, probe.name).Scan(&exists); err != nil {
			return false, fmt.Errorf("inspect footprint function %s: %w", probe.name, err)
		}
	default:
		return false, fmt.Errorf("inspect unknown footprint kind %q for %s", probe.kind, probe.name)
	}
	return exists, nil
}

// protectedMigrationSchemaFingerprint captures the behavior-bearing catalog
// definitions for every table touched by the protected migration. Object names
// alone are insufficient: a same-named non-unique index, changed partial-index
// predicate, or altered append-only rule can violate accounting invariants.
func protectedMigrationSchemaFingerprint(ctx context.Context, db migrationQueryer, spec protectedMigrationSpec) (string, error) {
	tableSet := map[string]bool{}
	for _, relation := range spec.relations {
		if relation.relkind == "r" {
			tableSet[relation.name] = true
		}
	}
	for _, rule := range spec.rules {
		tableSet[rule.relation] = true
	}
	for _, table := range spec.fingerprintTables {
		tableSet[table] = true
	}
	tables := make([]string, 0, len(tableSet))
	for table := range tableSet {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	digest := sha256.New()
	for _, table := range tables {
		var definition string
		if err := db.QueryRowContext(ctx, `
			SELECT jsonb_build_object(
				'relation', c.relname,
				'kind', c.relkind::text,
				'persistence', c.relpersistence::text,
				'replica_identity', c.relreplident::text,
				'row_security', c.relrowsecurity,
				'force_row_security', c.relforcerowsecurity,
				'columns', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						a.attnum,
						a.attname,
						pg_catalog.format_type(a.atttypid, a.atttypmod),
						a.attnotnull,
						COALESCE(pg_catalog.pg_get_expr(d.adbin, d.adrelid, true), ''),
						a.attidentity::text,
						a.attgenerated::text,
						COALESCE(coll.collname, '')
					) ORDER BY a.attnum)
					FROM pg_catalog.pg_attribute a
					LEFT JOIN pg_catalog.pg_attrdef d
						ON d.adrelid = a.attrelid AND d.adnum = a.attnum
					LEFT JOIN pg_catalog.pg_collation coll ON coll.oid = a.attcollation
					WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
				), '[]'::jsonb),
				'constraints', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						con.conname,
						con.contype::text,
						con.condeferrable,
						con.condeferred,
						con.convalidated,
						pg_catalog.pg_get_constraintdef(con.oid, true)
					) ORDER BY con.conname)
					FROM pg_catalog.pg_constraint con
					WHERE con.conrelid = c.oid
				), '[]'::jsonb),
				'indexes', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						ic.relname,
						i.indisunique,
						i.indisprimary,
						i.indisexclusion,
						i.indimmediate,
						i.indisvalid,
						i.indisready,
						pg_catalog.pg_get_indexdef(i.indexrelid),
						COALESCE(pg_catalog.pg_get_expr(i.indpred, i.indrelid, true), ''),
						COALESCE(pg_catalog.pg_get_expr(i.indexprs, i.indrelid, true), '')
					) ORDER BY ic.relname)
					FROM pg_catalog.pg_index i
					JOIN pg_catalog.pg_class ic ON ic.oid = i.indexrelid
					WHERE i.indrelid = c.oid
				), '[]'::jsonb),
				'rules', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						r.rulename,
						r.ev_type::text,
						r.is_instead,
						r.ev_enabled::text,
						r.ev_qual::text,
						r.ev_action::text
					) ORDER BY r.rulename)
					FROM pg_catalog.pg_rewrite r
					WHERE r.ev_class = c.oid AND r.rulename <> '_RETURN'
				), '[]'::jsonb),
				'triggers', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						tg.tgname,
						tg.tgenabled::text,
						pg_catalog.pg_get_triggerdef(tg.oid, true),
						pg_catalog.pg_get_functiondef(tg.tgfoid)
					) ORDER BY tg.tgname)
					FROM pg_catalog.pg_trigger tg
					WHERE tg.tgrelid = c.oid AND NOT tg.tgisinternal
				), '[]'::jsonb),
				'policies', COALESCE((
					SELECT jsonb_agg(jsonb_build_array(
						pol.polname,
						pol.polcmd::text,
						pol.polpermissive,
						pol.polroles::text,
						COALESCE(pg_catalog.pg_get_expr(pol.polqual, pol.polrelid, true), ''),
						COALESCE(pg_catalog.pg_get_expr(pol.polwithcheck, pol.polrelid, true), '')
					) ORDER BY pol.polname)
					FROM pg_catalog.pg_policy pol
					WHERE pol.polrelid = c.oid
				), '[]'::jsonb)
			)::text
			FROM pg_catalog.pg_class c
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'public' AND c.relname = $1`, table).Scan(&definition); err != nil {
			return "", fmt.Errorf("fingerprint relation %s: %w", table, err)
		}
		_, _ = digest.Write([]byte(table))
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(definition))
		_, _ = digest.Write([]byte{0})
	}
	functions := append([]string(nil), spec.fingerprintFunctions...)
	sort.Strings(functions)
	for _, function := range functions {
		var definition string
		if err := db.QueryRowContext(ctx, `
			SELECT COALESCE(pg_catalog.pg_get_functiondef(
				to_regprocedure('public.' || $1)
			), '')`, function).Scan(&definition); err != nil {
			return "", fmt.Errorf("fingerprint function %s: %w", function, err)
		}
		if definition == "" {
			return "", fmt.Errorf("fingerprint function %s is missing", function)
		}
		_, _ = digest.Write([]byte(function))
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(definition))
		_, _ = digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func validateProtectedMigrationState(migration migrationFile, state protectedMigrationState, validateLatestFingerprint bool) error {
	if state.receiptExists {
		if state.receiptSHA256 != migration.sha256 {
			return fmt.Errorf("protected migration %s: checksum drift; recorded receipt does not match repository file", migration.name)
		}
		if !state.complete {
			return fmt.Errorf("protected migration %s: schema drift; recorded receipt is missing required objects", migration.name)
		}
		if validateLatestFingerprint && state.receiptSchemaSHA256 != state.currentSchemaSHA256 {
			return fmt.Errorf("protected migration %s: schema fingerprint drift; invariant-bearing definitions changed", migration.name)
		}
		return nil
	}
	if state.anyFootprint {
		prefix := strings.SplitN(migration.name, "_", 2)[0]
		return fmt.Errorf("protected migration %s: ambiguous_prior_%s; schema footprint exists without an exact receipt", migration.name, prefix)
	}
	// The ledger and 019 receipt are born in the same transaction. A ledger
	// without the first protected receipt is therefore anomalous. For 020+
	// migrations, an existing ledger containing earlier receipts is expected.
	if state.ledgerExists && migration.name == "019_provider_exchange.sql" {
		return fmt.Errorf("protected migration %s: migration receipt table exists without its required receipt", migration.name)
	}
	return nil
}

func applyLegacyMigration(ctx context.Context, conn *sql.Conn, migration migrationFile) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", migration.name, err)
	}
	if err := configureMigrationTransaction(ctx, tx); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("configure migration %s timeouts: %w", migration.name, err)
	}
	statements := migrationStatements(string(migration.data))
	for i, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s statement %d: %w", migration.name, i+1, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", migration.name, err)
	}
	log.Printf("legacy migration replayed atomically: %s (%d statements)", migration.name, len(statements))
	return nil
}

func applyProtectedMigration(ctx context.Context, conn *sql.Conn, migration migrationFile, spec protectedMigrationSpec, releaseRevision string, validateLatestFingerprint bool, preflight ProtectedMigrationPreflight) (bool, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin protected migration %s: %w", migration.name, err)
	}
	rollback := func(err error) (bool, error) {
		_ = tx.Rollback()
		return false, err
	}
	if err := configureMigrationTransaction(ctx, tx); err != nil {
		return rollback(fmt.Errorf("configure protected migration %s timeouts: %w", migration.name, err))
	}
	state, err := inspectProtectedMigration(ctx, tx, migration.name, spec)
	if err != nil {
		return rollback(fmt.Errorf("inspect protected migration %s: %w", migration.name, err))
	}
	if err := validateProtectedMigrationState(migration, state, validateLatestFingerprint); err != nil {
		return rollback(err)
	}
	if state.receiptExists {
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit protected migration receipt check %s: %w", migration.name, err)
		}
		return false, nil
	}
	if preflight != nil {
		if err := preflight(ctx, tx, migration.name); err != nil {
			return rollback(fmt.Errorf("protected migration %s preflight: %w", migration.name, err))
		}
	}

	revision, err := protectedMigrationRevision(releaseRevision)
	if err != nil {
		return rollback(fmt.Errorf("protected migration %s: %w", migration.name, err))
	}
	if _, err := tx.ExecContext(ctx, migrationReceiptTableSQL); err != nil {
		return rollback(fmt.Errorf("create migration receipt table for %s: %w", migration.name, err))
	}
	statements := migrationStatements(string(migration.data))
	for i, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return rollback(fmt.Errorf("protected migration %s statement %d: %w", migration.name, i+1, err))
		}
	}
	postState, err := inspectProtectedMigration(ctx, tx, migration.name, spec)
	if err != nil {
		return rollback(fmt.Errorf("verify protected migration %s: %w", migration.name, err))
	}
	if !postState.complete {
		return rollback(fmt.Errorf("protected migration %s did not create its complete schema footprint", migration.name))
	}
	if postState.currentSchemaSHA256 == "" {
		return rollback(fmt.Errorf("protected migration %s did not produce a schema fingerprint", migration.name))
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO public.nhs_schema_migrations (name, sha256, schema_sha256, applied_by_commit)
		VALUES ($1, $2, $3, $4)`, migration.name, migration.sha256, postState.currentSchemaSHA256, revision); err != nil {
		return rollback(fmt.Errorf("record protected migration %s: %w", migration.name, err))
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit protected migration %s: %w", migration.name, err)
	}
	return true, nil
}

func configureMigrationTransaction(ctx context.Context, tx *sql.Tx) error {
	if migrationTransactionLockTimeout <= 0 || migrationTransactionStatementTimeout <= 0 {
		return errors.New("migration transaction timeouts must be positive")
	}
	lockTimeout := strconv.FormatInt(migrationTransactionLockTimeout.Milliseconds(), 10) + "ms"
	statementTimeout := strconv.FormatInt(migrationTransactionStatementTimeout.Milliseconds(), 10) + "ms"
	_, err := tx.ExecContext(ctx, `
		SELECT set_config('lock_timeout',$1,true),
		       set_config('statement_timeout',$2,true)`, lockTimeout, statementTimeout)
	return err
}

func protectedMigrationRevision(candidate string) (string, error) {
	revision := strings.ToLower(strings.TrimSpace(candidate))
	if len(revision) != 40 {
		return "", fmt.Errorf("release revision must be the full 40-character Git commit before applying a protected migration")
	}
	if _, err := hex.DecodeString(revision); err != nil {
		return "", fmt.Errorf("release revision must be a hexadecimal Git commit")
	}
	return revision, nil
}

// migrationStatements splits PostgreSQL migration files without breaking
// quoted strings, quoted identifiers, comments, or dollar-quoted function
// bodies. Each file is still executed inside one transaction by RunMigrations,
// so any statement failure rolls the entire file back.
func migrationStatements(data string) []string {
	var statements []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	inLineComment := false
	blockCommentDepth := 0
	dollarTag := ""

	appendStatement := func() {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
		current.Reset()
	}

	for i := 0; i < len(data); {
		if inLineComment {
			if data[i] == '\n' {
				inLineComment = false
				current.WriteByte('\n')
			}
			i++
			continue
		}
		if blockCommentDepth > 0 {
			if i+1 < len(data) && data[i:i+2] == "/*" {
				blockCommentDepth++
				i += 2
				continue
			}
			if i+1 < len(data) && data[i:i+2] == "*/" {
				blockCommentDepth--
				i += 2
				if blockCommentDepth == 0 {
					current.WriteByte(' ')
				}
				continue
			}
			i++
			continue
		}
		if dollarTag != "" {
			if strings.HasPrefix(data[i:], dollarTag) {
				current.WriteString(dollarTag)
				i += len(dollarTag)
				dollarTag = ""
				continue
			}
			current.WriteByte(data[i])
			i++
			continue
		}
		if inSingle {
			current.WriteByte(data[i])
			if data[i] == '\\' && i+1 < len(data) {
				current.WriteByte(data[i+1])
				i += 2
				continue
			}
			if data[i] == '\'' {
				if i+1 < len(data) && data[i+1] == '\'' {
					current.WriteByte(data[i+1])
					i += 2
					continue
				}
				inSingle = false
			}
			i++
			continue
		}
		if inDouble {
			current.WriteByte(data[i])
			if data[i] == '"' {
				if i+1 < len(data) && data[i+1] == '"' {
					current.WriteByte(data[i+1])
					i += 2
					continue
				}
				inDouble = false
			}
			i++
			continue
		}

		if i+1 < len(data) && data[i:i+2] == "--" {
			inLineComment = true
			i += 2
			continue
		}
		if i+1 < len(data) && data[i:i+2] == "/*" {
			blockCommentDepth = 1
			i += 2
			continue
		}
		switch data[i] {
		case '\'':
			inSingle = true
			current.WriteByte(data[i])
			i++
		case '"':
			inDouble = true
			current.WriteByte(data[i])
			i++
		case '$':
			tag, width := postgresDollarQuoteTag(data[i:])
			if width == 0 {
				current.WriteByte(data[i])
				i++
				continue
			}
			dollarTag = tag
			current.WriteString(tag)
			i += width
		case ';':
			appendStatement()
			i++
		default:
			current.WriteByte(data[i])
			i++
		}
	}
	appendStatement()
	return statements
}

func postgresDollarQuoteTag(data string) (string, int) {
	if len(data) < 2 || data[0] != '$' {
		return "", 0
	}
	for i := 1; i < len(data); i++ {
		switch c := data[i]; {
		case c == '$':
			return data[:i+1], i + 1
		case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9':
			continue
		default:
			return "", 0
		}
	}
	return "", 0
}
