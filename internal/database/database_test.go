package database

import (
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestDatabaseDSNIncludesExactReleaseApplicationName(t *testing.T) {
	const revision = "0123456789abcdef0123456789abcdef01234567"
	got, err := databaseDSNWithReleaseApplicationName(
		"postgres://db.example.test/nhs?sslmode=require&application_name=stale",
		revision,
	)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if value := parsed.Query().Get("application_name"); value != "nhs-server:"+revision {
		t.Fatalf("application_name=%q", value)
	}
	if value := parsed.Query().Get("sslmode"); value != "require" {
		t.Fatalf("sslmode=%q", value)
	}

	keyword, err := databaseDSNWithReleaseApplicationName("host=db.example.test dbname=nhs", revision)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(keyword, " application_name='nhs-server:"+revision+"'") {
		t.Fatalf("keyword DSN did not receive bounded application identity")
	}
	if _, err := databaseDSNWithReleaseApplicationName("postgres://db.example.test/nhs", "short"); err == nil {
		t.Fatal("invalid release identity was accepted for database sessions")
	}
}

func TestMigrationStatementsIgnoresSemicolonsInLineComments(t *testing.T) {
	data := `-- Created at /fix/{host} intake; paid_at set by Stripe webhook;
CREATE TABLE IF NOT EXISTS geo_fix_jobs (
    id BIGSERIAL PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'pending',
    paid_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_geo_fix_jobs_status ON geo_fix_jobs(status);
`

	statements := migrationStatements(data)
	if len(statements) != 2 {
		t.Fatalf("expected 2 executable statements, got %d: %#v", len(statements), statements)
	}
	for _, stmt := range statements {
		if strings.HasPrefix(strings.TrimSpace(stmt), "paid_at set by Stripe webhook") {
			t.Fatalf("comment fragment became executable SQL: %q", stmt)
		}
	}
}

func TestGeoFixMigrationHasNoCommentFragments(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "012_geo_fix_jobs.sql"))
	if err != nil {
		t.Fatal(err)
	}
	statements := migrationStatements(string(data))
	if len(statements) == 0 {
		t.Fatal("expected geo-fix migration statements")
	}
	for _, stmt := range statements {
		if strings.HasPrefix(strings.TrimSpace(stmt), "paid_at") {
			t.Fatalf("unexpected paid_at-leading statement: %q", stmt)
		}
	}
}

func TestMigrationStatementsKeepDollarQuotedFunctionAtomic(t *testing.T) {
	data := `
		CREATE OR REPLACE FUNCTION redact_ticket()
		RETURNS TRIGGER LANGUAGE plpgsql AS $$
		BEGIN
			UPDATE action_tickets SET demand_topic='redacted' WHERE id=OLD.id;
			RETURN OLD;
		END;
		$$;
		CREATE TRIGGER redact BEFORE DELETE ON search_receipts
		FOR EACH ROW EXECUTE FUNCTION redact_ticket();
	`
	statements := migrationStatements(data)
	if len(statements) != 2 {
		t.Fatalf("expected function and trigger statements, got %d: %#v", len(statements), statements)
	}
	if !strings.Contains(statements[0], "UPDATE action_tickets") || !strings.Contains(statements[0], "RETURN OLD;") {
		t.Fatalf("dollar-quoted function body was split: %q", statements[0])
	}
}

func TestProviderExchangeMigrationKeepsRedactionRuleAtomic(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "019_provider_exchange.sql"))
	if err != nil {
		t.Fatal(err)
	}
	statements := migrationStatements(string(data))
	redactionRules := 0
	for _, stmt := range statements {
		if strings.Contains(stmt, "CREATE OR REPLACE RULE redact_action_ticket_intent_on_receipt_delete") {
			redactionRules++
			if !strings.Contains(stmt, "UPDATE action_tickets") || !strings.Contains(stmt, "WHERE search_receipt_id=OLD.id") {
				t.Fatalf("provider exchange redaction rule was not kept intact: %q", stmt)
			}
		}
	}
	if redactionRules != 1 {
		t.Fatalf("expected exactly one intact provider exchange redaction rule, got %d", redactionRules)
	}
}

func TestProviderExchangeProtectedContractCoversDeclaredObjects(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "019_provider_exchange.sql"))
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := protectedMigrationSpecs["019_provider_exchange.sql"]
	if !ok {
		t.Fatal("provider exchange migration has no protected schema contract")
	}

	declaredRelations := map[string]string{}
	relationPattern := regexp.MustCompile(`(?m)^CREATE (?:UNIQUE )?(TABLE|INDEX) IF NOT EXISTS ([a-z0-9_]+)`)
	for _, match := range relationPattern.FindAllStringSubmatch(string(data), -1) {
		kind := "i"
		if match[1] == "TABLE" {
			kind = "r"
		}
		declaredRelations[match[2]] = kind
	}
	declaredIndexParents := map[string]string{}
	indexPattern := regexp.MustCompile(`(?m)^CREATE (?:UNIQUE )?INDEX IF NOT EXISTS ([a-z0-9_]+)\s+ON ([a-z0-9_]+)`)
	for _, match := range indexPattern.FindAllStringSubmatch(string(data), -1) {
		declaredIndexParents[match[1]] = match[2]
	}
	if len(declaredRelations) != len(spec.relations) {
		t.Fatalf("protected relation contract has %d entries, migration declares %d", len(spec.relations), len(declaredRelations))
	}
	for _, relation := range spec.relations {
		if want, ok := declaredRelations[relation.name]; !ok || relation.relkind != want {
			t.Fatalf("protected relation contract mismatch for %s: kind=%q declared=%q present=%t", relation.name, relation.relkind, want, ok)
		}
		if relation.relkind == "i" && relation.parent != declaredIndexParents[relation.name] {
			t.Fatalf("protected index %s parent=%q, migration declares %q", relation.name, relation.parent, declaredIndexParents[relation.name])
		}
	}

	declaredRules := map[string]string{}
	rulePattern := regexp.MustCompile(`(?m)^CREATE OR REPLACE RULE ([a-z0-9_]+) AS\s+ON (?:UPDATE|DELETE) TO ([a-z0-9_]+)`)
	for _, match := range rulePattern.FindAllStringSubmatch(string(data), -1) {
		declaredRules[match[1]] = match[2]
	}
	if len(declaredRules) != len(spec.rules) {
		t.Fatalf("protected rule contract has %d entries, migration declares %d", len(spec.rules), len(declaredRules))
	}
	for _, rule := range spec.rules {
		if want, ok := declaredRules[rule.name]; !ok || rule.relation != want {
			t.Fatalf("protected rule contract mismatch for %s: relation=%q declared=%q present=%t", rule.name, rule.relation, want, ok)
		}
	}
}

func TestActionInterestProtectedContractCoversDeclaredDelta(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "020_action_interest_receipts.sql"))
	if err != nil {
		t.Fatal(err)
	}
	prior := protectedMigrationSpecs["019_provider_exchange.sql"]
	current, ok := protectedMigrationSpecs["020_action_interest_receipts.sql"]
	if !ok {
		t.Fatal("action-interest migration has no protected schema contract")
	}

	priorRelations := map[string]bool{}
	for _, relation := range prior.relations {
		priorRelations[relation.name] = true
	}
	deltaRelations := map[string]migrationRelation{}
	for _, relation := range current.relations {
		if !priorRelations[relation.name] {
			deltaRelations[relation.name] = relation
		}
	}
	declaredRelations := map[string]string{}
	relationPattern := regexp.MustCompile(`(?m)^CREATE (?:UNIQUE )?(TABLE|INDEX) IF NOT EXISTS ([a-z0-9_]+)`)
	for _, match := range relationPattern.FindAllStringSubmatch(string(data), -1) {
		kind := "i"
		if match[1] == "TABLE" {
			kind = "r"
		}
		declaredRelations[match[2]] = kind
	}
	declaredIndexParents := map[string]string{}
	indexPattern := regexp.MustCompile(`(?m)^CREATE (?:UNIQUE )?INDEX IF NOT EXISTS ([a-z0-9_]+)\s+ON ([a-z0-9_]+)`)
	for _, match := range indexPattern.FindAllStringSubmatch(string(data), -1) {
		declaredIndexParents[match[1]] = match[2]
	}
	if len(deltaRelations) != len(declaredRelations) {
		t.Fatalf("020 relation delta has %d entries, migration declares %d", len(deltaRelations), len(declaredRelations))
	}
	for name, relation := range deltaRelations {
		if want, ok := declaredRelations[name]; !ok || relation.relkind != want {
			t.Fatalf("020 relation delta mismatch for %s: kind=%q declared=%q present=%t", name, relation.relkind, want, ok)
		}
		if relation.relkind == "i" && relation.parent != declaredIndexParents[name] {
			t.Fatalf("020 index %s parent=%q, migration declares %q", name, relation.parent, declaredIndexParents[name])
		}
		if !current.footprintRelations[name] {
			t.Fatalf("020 relation delta %s is not marked as its new footprint", name)
		}
	}
	for name := range current.footprintRelations {
		if _, ok := deltaRelations[name]; !ok {
			t.Fatalf("020 marks inherited/unknown relation %s as a new footprint", name)
		}
	}

	priorRules := map[string]bool{}
	for _, rule := range prior.rules {
		priorRules[rule.name] = true
	}
	deltaRules := map[string]migrationRule{}
	for _, rule := range current.rules {
		if !priorRules[rule.name] {
			deltaRules[rule.name] = rule
		}
	}
	declaredRules := map[string]string{}
	rulePattern := regexp.MustCompile(`(?m)^CREATE OR REPLACE RULE ([a-z0-9_]+) AS\s+ON (?:UPDATE|DELETE) TO ([a-z0-9_]+)`)
	for _, match := range rulePattern.FindAllStringSubmatch(string(data), -1) {
		declaredRules[match[1]] = match[2]
	}
	if len(deltaRules) != len(declaredRules) {
		t.Fatalf("020 rule delta has %d entries, migration declares %d", len(deltaRules), len(declaredRules))
	}
	for name, rule := range deltaRules {
		if want, ok := declaredRules[name]; !ok || rule.relation != want {
			t.Fatalf("020 rule delta mismatch for %s: relation=%q declared=%q present=%t", name, rule.relation, want, ok)
		}
		if !current.footprintRules[name] {
			t.Fatalf("020 rule delta %s is not marked as its new footprint", name)
		}
	}
	for name := range current.footprintRules {
		if _, ok := deltaRules[name]; !ok {
			t.Fatalf("020 marks inherited/unknown rule %s as a new footprint", name)
		}
	}

	migrations, err := loadMigrations(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("repository protected migration chain: %v", err)
	}
}

func TestProviderCapacityProtectedContractCoversDeclaredDelta(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "021_provider_capacity_reservations.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	statements := migrationStatements(migrationSQL)
	if len(statements) == 0 || !strings.Contains(statements[0], "LOCK TABLE public.action_tickets IN SHARE ROW EXCLUSIVE MODE") {
		t.Fatal("021 must lock action_tickets before taking its backfill snapshot")
	}
	lockAt := strings.Index(migrationSQL, "LOCK TABLE public.action_tickets IN SHARE ROW EXCLUSIVE MODE")
	createCapacityAt := strings.Index(migrationSQL, "CREATE TABLE IF NOT EXISTS provider_capacity_events")
	lastBackfillAt := strings.LastIndex(migrationSQL, "INSERT INTO provider_capacity_events")
	functionAt := strings.Index(migrationSQL, "CREATE OR REPLACE FUNCTION public.enforce_action_ticket_capacity_reservation()")
	triggerAt := strings.Index(migrationSQL, "CREATE CONSTRAINT TRIGGER action_ticket_capacity_reservation_required")
	if lockAt < 0 || createCapacityAt < 0 || lastBackfillAt < 0 || functionAt < 0 || triggerAt < 0 ||
		!(lockAt < createCapacityAt && createCapacityAt < lastBackfillAt && lastBackfillAt < functionAt && functionAt < triggerAt) {
		t.Fatalf("021 compatibility boundary order lock=%d table=%d backfill=%d function=%d trigger=%d", lockAt, createCapacityAt, lastBackfillAt, functionAt, triggerAt)
	}
	for _, required := range []string{
		"AFTER INSERT OR UPDATE ON public.action_tickets",
		"DEFERRABLE INITIALLY DEFERRED",
		"capacity.action_ticket_id = NEW.id",
		"capacity.provider_claim_id = NEW.provider_claim_id",
		"capacity.provider_offer_id = NEW.provider_offer_id",
		"capacity.event_type = 'reserve'",
		"capacity.event_reason = 'ticket_created'",
		"capacity.amount_cents = NEW.bounty_cents_snapshot",
		"capacity.currency = NEW.currency_snapshot",
		"ERRCODE = '23514'",
	} {
		if !strings.Contains(migrationSQL, required) {
			t.Fatalf("021 capacity constraint is missing %q", required)
		}
	}
	prior := protectedMigrationSpecs["020_action_interest_receipts.sql"]
	current, ok := protectedMigrationSpecs["021_provider_capacity_reservations.sql"]
	if !ok {
		t.Fatal("provider-capacity migration has no protected schema contract")
	}

	priorRelations := map[string]bool{}
	for _, relation := range prior.relations {
		priorRelations[relation.name] = true
	}
	deltaRelations := map[string]migrationRelation{}
	for _, relation := range current.relations {
		if !priorRelations[relation.name] {
			deltaRelations[relation.name] = relation
		}
	}
	declaredRelations := map[string]string{}
	relationPattern := regexp.MustCompile(`(?m)^CREATE (?:UNIQUE )?(TABLE|INDEX) IF NOT EXISTS ([a-z0-9_]+)`)
	for _, match := range relationPattern.FindAllStringSubmatch(string(data), -1) {
		kind := "i"
		if match[1] == "TABLE" {
			kind = "r"
		}
		declaredRelations[match[2]] = kind
	}
	declaredIndexParents := map[string]string{}
	indexPattern := regexp.MustCompile(`(?m)^CREATE (?:UNIQUE )?INDEX IF NOT EXISTS ([a-z0-9_]+)\s+ON ([a-z0-9_]+)`)
	for _, match := range indexPattern.FindAllStringSubmatch(string(data), -1) {
		declaredIndexParents[match[1]] = match[2]
	}
	if len(deltaRelations) != len(declaredRelations) {
		t.Fatalf("021 relation delta has %d entries, migration declares %d", len(deltaRelations), len(declaredRelations))
	}
	for name, relation := range deltaRelations {
		if want, present := declaredRelations[name]; !present || relation.relkind != want {
			t.Fatalf("021 relation delta mismatch for %s: kind=%q declared=%q present=%t", name, relation.relkind, want, present)
		}
		if relation.relkind == "i" && relation.parent != declaredIndexParents[name] {
			t.Fatalf("021 index %s parent=%q, migration declares %q", name, relation.parent, declaredIndexParents[name])
		}
		if !current.footprintRelations[name] {
			t.Fatalf("021 relation delta %s is not marked as its new footprint", name)
		}
	}
	for name := range current.footprintRelations {
		if _, present := deltaRelations[name]; !present {
			t.Fatalf("021 marks inherited/unknown relation %s as a new footprint", name)
		}
	}

	priorRules := map[string]bool{}
	for _, rule := range prior.rules {
		priorRules[rule.name] = true
	}
	deltaRules := map[string]migrationRule{}
	for _, rule := range current.rules {
		if !priorRules[rule.name] {
			deltaRules[rule.name] = rule
		}
	}
	declaredRules := map[string]string{}
	rulePattern := regexp.MustCompile(`(?m)^CREATE OR REPLACE RULE ([a-z0-9_]+) AS\s+ON (?:UPDATE|DELETE) TO ([a-z0-9_]+)`)
	for _, match := range rulePattern.FindAllStringSubmatch(string(data), -1) {
		declaredRules[match[1]] = match[2]
	}
	if len(deltaRules) != len(declaredRules) {
		t.Fatalf("021 rule delta has %d entries, migration declares %d", len(deltaRules), len(declaredRules))
	}
	for name, rule := range deltaRules {
		if want, present := declaredRules[name]; !present || rule.relation != want {
			t.Fatalf("021 rule delta mismatch for %s: relation=%q declared=%q present=%t", name, rule.relation, want, present)
		}
		if !current.footprintRules[name] {
			t.Fatalf("021 rule delta %s is not marked as its new footprint", name)
		}
	}

	migrations, err := loadMigrations(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("repository protected migration chain: %v", err)
	}
}

func TestProviderCommercialProofProtectedContractCoversDeclaredDelta(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "022_provider_commercial_proof.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	for _, required := range []string{
		"Nothing in this migration backfills a qualifying company or commitment",
		"provider_action_handoff_cutover_inflight",
		"zero live pre-handoff action tickets",
		"LOCK TABLE public.action_tickets IN ACCESS EXCLUSIVE MODE",
		"ALTER COLUMN commercial_terms_contract_version_snapshot DROP DEFAULT",
		"ALTER COLUMN commercial_terms_sha256_snapshot DROP DEFAULT",
		"CREATE TABLE IF NOT EXISTS provider_commercial_acceptance_events",
		"CREATE TABLE IF NOT EXISTS provider_pilot_companies",
		"CREATE TABLE IF NOT EXISTS provider_commercial_commitment_events",
		"CREATE TABLE IF NOT EXISTS provider_action_handoff_receipts",
		"provider_commercial_acceptance_fresh_claim",
		"provider_pilot_company_exact_acceptance",
		"provider_commercial_commitment_replenishment",
		"provider_commercial_commitment_reversal_amount",
		"provider_commercial_commitment_terms_renewal",
		"provider_action_handoff_receipt_enforced",
		"enforce_provider_action_handoff_receipt",
		"action_ticket_observed_handoff_status_enforced",
		"action_ticket_observed_handoff_insert_enforced",
		"enforce_action_ticket_observed_handoff_status",
		"action_ticket_observed_handoff_required",
		"principal_handoff_consent",
		"handoff_consent_version",
		"source_effective_at <= owner_verified_at",
	} {
		if !strings.Contains(migrationSQL, required) {
			t.Fatalf("022 commercial proof contract is missing %q", required)
		}
	}
	prior := protectedMigrationSpecs["021_provider_capacity_reservations.sql"]
	current, ok := protectedMigrationSpecs["022_provider_commercial_proof.sql"]
	if !ok {
		t.Fatal("provider-commercial-proof migration has no protected schema contract")
	}

	priorRelations := map[string]bool{}
	for _, relation := range prior.relations {
		priorRelations[relation.name] = true
	}
	deltaRelations := map[string]migrationRelation{}
	for _, relation := range current.relations {
		if !priorRelations[relation.name] {
			deltaRelations[relation.name] = relation
		}
	}
	declaredRelations := map[string]string{}
	relationPattern := regexp.MustCompile(`(?m)^CREATE (?:UNIQUE )?(TABLE|INDEX) IF NOT EXISTS ([a-z0-9_]+)`)
	for _, match := range relationPattern.FindAllStringSubmatch(migrationSQL, -1) {
		kind := "i"
		if match[1] == "TABLE" {
			kind = "r"
		}
		declaredRelations[match[2]] = kind
	}
	declaredIndexParents := map[string]string{}
	indexPattern := regexp.MustCompile(`(?m)^CREATE (?:UNIQUE )?INDEX IF NOT EXISTS ([a-z0-9_]+)\s+ON ([a-z0-9_]+)`)
	for _, match := range indexPattern.FindAllStringSubmatch(migrationSQL, -1) {
		declaredIndexParents[match[1]] = match[2]
	}
	if len(deltaRelations) != len(declaredRelations) {
		t.Fatalf("022 relation delta has %d entries, migration declares %d", len(deltaRelations), len(declaredRelations))
	}
	for name, relation := range deltaRelations {
		if want, present := declaredRelations[name]; !present || relation.relkind != want {
			t.Fatalf("022 relation delta mismatch for %s: kind=%q declared=%q present=%t", name, relation.relkind, want, present)
		}
		if relation.relkind == "i" && relation.parent != declaredIndexParents[name] {
			t.Fatalf("022 index %s parent=%q, migration declares %q", name, relation.parent, declaredIndexParents[name])
		}
		if !current.footprintRelations[name] {
			t.Fatalf("022 relation delta %s is not marked as its new footprint", name)
		}
	}
	for name := range current.footprintRelations {
		if _, present := deltaRelations[name]; !present {
			t.Fatalf("022 marks inherited/unknown relation %s as a new footprint", name)
		}
	}

	priorRules := map[string]bool{}
	for _, rule := range prior.rules {
		priorRules[rule.name] = true
	}
	deltaRules := map[string]migrationRule{}
	for _, rule := range current.rules {
		if !priorRules[rule.name] {
			deltaRules[rule.name] = rule
		}
	}
	declaredRules := map[string]string{}
	rulePattern := regexp.MustCompile(`(?m)^CREATE OR REPLACE RULE ([a-z0-9_]+) AS\s+ON (?:UPDATE|DELETE) TO ([a-z0-9_]+)`)
	for _, match := range rulePattern.FindAllStringSubmatch(migrationSQL, -1) {
		declaredRules[match[1]] = match[2]
	}
	if len(deltaRules) != len(declaredRules) {
		t.Fatalf("022 rule delta has %d entries, migration declares %d", len(deltaRules), len(declaredRules))
	}
	for name, rule := range deltaRules {
		if want, present := declaredRules[name]; !present || rule.relation != want {
			t.Fatalf("022 rule delta mismatch for %s: relation=%q declared=%q present=%t", name, rule.relation, want, present)
		}
		if !current.footprintRules[name] {
			t.Fatalf("022 rule delta %s is not marked as its new footprint", name)
		}
	}

	migrations, err := loadMigrations(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("repository protected migration chain: %v", err)
	}
}

func TestControlledIntentDisclosureProtectedContractCarries022(t *testing.T) {
	prior := protectedMigrationSpecs["022_provider_commercial_proof.sql"]
	current, ok := protectedMigrationSpecs["023_provider_controlled_intent_disclosure.sql"]
	if !ok {
		t.Fatal("controlled-intent disclosure migration has no protected schema contract")
	}
	if len(current.relations) != len(prior.relations) || len(current.rules) != len(prior.rules) ||
		len(current.footprintRelations) != 0 || len(current.footprintRules) != 0 ||
		len(current.footprintProbes) != 4 {
		t.Fatalf("023 cumulative contract shape relations=%d/%d rules=%d/%d relation_delta=%d rule_delta=%d probes=%d",
			len(current.relations), len(prior.relations), len(current.rules), len(prior.rules),
			len(current.footprintRelations), len(current.footprintRules), len(current.footprintProbes))
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "023_provider_controlled_intent_disclosure.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"Existing handoffs are deliberately backfilled as declined",
		"principal_controlled_intent_disclosure_consent",
		"controlled_intent_disclosure_consent_version",
		"nhs-provider-controlled-intent-disclosure-consent-v1",
		"provider_handoff_intent_disclosure_consent_pair",
		"action_ticket_controlled_intent_immutability_enforced",
		"enforce_action_ticket_controlled_intent_immutability",
		"only allowed change is the existing one-way privacy redaction transition",
	} {
		if !strings.Contains(string(data), required) {
			t.Fatalf("023 disclosure contract is missing %q", required)
		}
	}
	migrations, err := loadMigrations(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("repository protected migration chain: %v", err)
	}
}

func TestProviderPilotBoundaryProtectedContractCarries023(t *testing.T) {
	prior := protectedMigrationSpecs["023_provider_controlled_intent_disclosure.sql"]
	current, ok := protectedMigrationSpecs["024_provider_pilot_boundary.sql"]
	if !ok {
		t.Fatal("provider-pilot boundary migration has no protected schema contract")
	}
	if len(current.relations) != len(prior.relations)+8 ||
		len(current.rules) != len(prior.rules)+5 ||
		len(current.fingerprintFunctions) != len(prior.fingerprintFunctions)+2 ||
		len(current.footprintRelations) != 8 ||
		len(current.footprintRules) != 5 ||
		len(current.footprintFunctions) != 2 ||
		len(current.footprintProbes) != 21 {
		t.Fatalf(
			"024 cumulative contract shape relations=%d/%d rules=%d/%d fingerprint_functions=%d/%d+2 relation_delta=%d rule_delta=%d function_delta=%d probes=%d",
			len(current.relations), len(prior.relations)+8,
			len(current.rules), len(prior.rules)+5,
			len(current.fingerprintFunctions), len(prior.fingerprintFunctions),
			len(current.footprintRelations), len(current.footprintRules),
			len(current.footprintFunctions),
			len(current.footprintProbes),
		)
	}
	for _, function := range []string{
		"provider_pilot_stage1_eligibility_snapshot_sha256(text,uuid,text,timestamptz,timestamptz,uuid,uuid,uuid,text,timestamptz)",
		"provider_pilot_enrollment_eligibility_is_current(uuid,uuid)",
	} {
		if !current.footprintFunctions[function] {
			t.Fatalf("024 standalone function %q is not protected as migration footprint", function)
		}
	}

	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "024_provider_pilot_boundary.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	for _, required := range []string{
		"provider_pilot_epoch_insert_enforced",
		"provider_pilot_stage1_evidence_window",
		"provider_pilot_stage1_thresholds",
		"provider_pilot_stage1_snapshot_hash",
		"provider_pilot_stage1_eligibility_snapshot_sha256",
		"provider_pilot_enrollment_eligibility_is_current",
		"provider_pilot_enrollment_enforced",
		"FOR UPDATE",
		"provider_pilot_enrollment_cohort_cap",
		"provider_pilot_epoch_event_enforced",
		"provider_pilot_event_snapshot_hash",
		"provider_offer_pilot_enrollment_claim",
		"provider_pilot_returned_offer_enforced",
		"provider_returned_offer_exact_snapshot",
		"provider_returned_offer_organic_result",
		"provider_returned_offer_receipt_window",
		"action_ticket_pilot_insert_enforced",
		"action_ticket_pilot_exact_snapshot",
		"action_ticket_pilot_returned_offer",
		"action_ticket_pilot_provider_cap",
		"action_ticket_pilot_total_cap",
		"action_ticket_pilot_snapshot_immutable",
		"zz_provider_action_handoff_pilot_boundary_enforced",
		"provider_pilot_handoff_active_epoch",
		"provider_pilot_epoch_enrollment_freshness",
		"NEW.activated_at := statement_timestamp()",
		"NEW.closed_at := statement_timestamp()",
		"NEW.enrolled_at := statement_timestamp()",
		"NEW.returned_at := statement_timestamp()",
	} {
		if !strings.Contains(migrationSQL, required) {
			t.Fatalf("024 pilot boundary contract is missing %q", required)
		}
	}

	priorRelations := map[string]bool{}
	for _, relation := range prior.relations {
		priorRelations[relation.name] = true
	}
	declaredRelations := map[string]bool{}
	relationPattern := regexp.MustCompile(`(?m)^CREATE (?:UNIQUE )?(?:TABLE|INDEX) IF NOT EXISTS ([a-z0-9_]+)`)
	for _, match := range relationPattern.FindAllStringSubmatch(migrationSQL, -1) {
		declaredRelations[match[1]] = true
	}
	for _, relation := range current.relations {
		if priorRelations[relation.name] {
			continue
		}
		if !declaredRelations[relation.name] {
			t.Fatalf("024 protected relation delta %s is not declared by the migration", relation.name)
		}
		if !current.footprintRelations[relation.name] {
			t.Fatalf("024 protected relation delta %s is not marked as footprint", relation.name)
		}
	}

	migrations, err := loadMigrations(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("repository protected migration chain: %v", err)
	}
}

func TestStage1FactIntegrityProtectedContractCarries024(t *testing.T) {
	prior := protectedMigrationSpecs["024_provider_pilot_boundary.sql"]
	current, ok := protectedMigrationSpecs["025_stage1_fact_integrity.sql"]
	if !ok {
		t.Fatal("Stage 1 fact-integrity migration has no protected schema contract")
	}
	if len(current.relations) != len(prior.relations) ||
		len(current.rules) != len(prior.rules)+2 ||
		len(current.fingerprintTables) != 5 ||
		len(current.fingerprintFunctions) != len(prior.fingerprintFunctions) ||
		len(current.footprintRelations) != 0 ||
		len(current.footprintRules) != 2 ||
		len(current.footprintProbes) != 15 {
		t.Fatalf(
			"025 cumulative contract shape relations=%d/%d rules=%d/%d fingerprint_tables=%d fingerprint_functions=%d/%d relation_delta=%d rule_delta=%d probes=%d",
			len(current.relations), len(prior.relations),
			len(current.rules), len(prior.rules)+2,
			len(current.fingerprintTables),
			len(current.fingerprintFunctions), len(prior.fingerprintFunctions),
			len(current.footprintRelations),
			len(current.footprintRules), len(current.footprintProbes),
		)
	}
	if !reflect.DeepEqual(current.fingerprintFunctions, prior.fingerprintFunctions) {
		t.Fatalf("025 did not carry forward inherited fingerprint functions: prior=%v current=%v", prior.fingerprintFunctions, current.fingerprintFunctions)
	}

	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "025_stage1_fact_integrity.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	for _, required := range []string{
		"nhs_schema_migrations_no_update",
		"nhs_schema_migrations_no_delete",
		"result_selections_returned_result_fk",
		"stage1_integrity_generation",
		"search_receipt_stage1_immutability_enforced",
		"organic_result_stage1_immutability_enforced",
		"result_selection_stage1_immutability_enforced",
		"stage1_search_receipt_insert_timestamp_owned",
		"stage1_organic_result_insert_timestamp_owned",
		"stage1_result_selection_insert_timestamp_owned",
		"stage1_action_interest_insert_timestamp_owned",
		"NEW.created_at := clock_timestamp()",
		"NEW.returned_at := clock_timestamp()",
		"NEW.selected_at := clock_timestamp()",
		"INTO NEW.expires_at",
		"aa_provider_pilot_stage1_epoch_anchor_locked",
		"lock_provider_pilot_stage1_epoch_anchor",
		"ab_provider_pilot_stage1_generation_enforced",
		"enforce_provider_pilot_stage1_generation",
		"provider_pilot_stage1_integrity_generation",
		"FOR UPDATE",
	} {
		if !strings.Contains(migrationSQL, required) {
			t.Fatalf("025 Stage 1 fact-integrity contract is missing %q", required)
		}
	}
	wantFingerprintTables := map[string]bool{
		"nhs_schema_migrations":    true,
		"search_receipts":          true,
		"organic_results_returned": true,
		"result_selections":        true,
		"action_interest_receipts": true,
	}
	for _, table := range current.fingerprintTables {
		if !wantFingerprintTables[table] {
			t.Fatalf("025 fingerprints unexpected inherited table %q", table)
		}
		delete(wantFingerprintTables, table)
	}
	if len(wantFingerprintTables) != 0 {
		t.Fatalf("025 missing inherited fingerprint tables: %v", wantFingerprintTables)
	}

	migrations, err := loadMigrations(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	// A concurrent successor may already exist in the shared worktree. The
	// chain check still proves 025 registration as long as every successor has
	// carried this contract forward.
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("repository protected migration chain: %v", err)
	}
}

func TestProviderPilotProofIntegrityProtectedContractCarries025(t *testing.T) {
	prior := protectedMigrationSpecs["025_stage1_fact_integrity.sql"]
	current, ok := protectedMigrationSpecs["026_provider_pilot_proof_integrity.sql"]
	if !ok {
		t.Fatal("provider-pilot proof-integrity migration has no protected schema contract")
	}
	if len(current.relations) != len(prior.relations) ||
		len(current.rules) != len(prior.rules) ||
		len(current.fingerprintTables) != len(prior.fingerprintTables) ||
		len(current.fingerprintFunctions) != len(prior.fingerprintFunctions) ||
		len(current.footprintRelations) != 0 ||
		len(current.footprintRules) != 0 ||
		len(current.footprintProbes) != 6 {
		t.Fatalf(
			"026 cumulative contract shape relations=%d/%d rules=%d/%d fingerprint_tables=%d/%d fingerprint_functions=%d/%d relation_delta=%d rule_delta=%d probes=%d",
			len(current.relations), len(prior.relations),
			len(current.rules), len(prior.rules),
			len(current.fingerprintTables), len(prior.fingerprintTables),
			len(current.fingerprintFunctions), len(prior.fingerprintFunctions),
			len(current.footprintRelations), len(current.footprintRules),
			len(current.footprintProbes),
		)
	}
	if !reflect.DeepEqual(current.fingerprintTables, prior.fingerprintTables) {
		t.Fatalf("026 did not carry forward inherited fingerprint tables: prior=%v current=%v", prior.fingerprintTables, current.fingerprintTables)
	}
	if !reflect.DeepEqual(current.fingerprintFunctions, prior.fingerprintFunctions) {
		t.Fatalf("026 did not carry forward inherited fingerprint functions: prior=%v current=%v", prior.fingerprintFunctions, current.fingerprintFunctions)
	}

	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "026_provider_pilot_proof_integrity.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	for _, required := range []string{
		"provider_pilot_outcome_receipt_enforced",
		"enforce_provider_pilot_outcome_receipt",
		"provider_pilot_outcome_database_clock",
		"provider_pilot_outcome_canonical_row",
		"provider_pilot_outcome_exact_ticket",
		"provider_pilot_outcome_exact_handoff",
		"provider_pilot_epoch_created_event_required",
		"provider_pilot_enrollment_event_required",
		"provider_pilot_epoch_transition_event_required",
		"require_provider_pilot_lifecycle_event",
		"provider_pilot_lifecycle_legacy_incomplete",
		"IN SHARE ROW EXCLUSIVE MODE",
		"DEFERRABLE INITIALLY DEFERRED",
	} {
		if !strings.Contains(migrationSQL, required) {
			t.Fatalf("026 provider-pilot proof-integrity contract is missing %q", required)
		}
	}

	migrations, err := loadMigrations(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("repository protected migration chain: %v", err)
	}
}

func TestProviderPilotReviewEvidenceProtectedContractCarries026(t *testing.T) {
	prior := protectedMigrationSpecs["026_provider_pilot_proof_integrity.sql"]
	current, ok := protectedMigrationSpecs["027_provider_pilot_review_evidence.sql"]
	if !ok {
		t.Fatal("provider-pilot review-evidence migration has no protected schema contract")
	}
	if len(current.relations) != len(prior.relations)+3 ||
		len(current.rules) != len(prior.rules)+2 ||
		len(current.fingerprintTables) != len(prior.fingerprintTables) ||
		len(current.fingerprintFunctions) != len(prior.fingerprintFunctions)+1 ||
		len(current.footprintRelations) != 3 ||
		len(current.footprintRules) != 2 ||
		len(current.footprintFunctions) != 1 ||
		len(current.footprintProbes) != 8 {
		t.Fatalf(
			"027 cumulative contract shape relations=%d/%d+3 rules=%d/%d+2 fingerprint_tables=%d/%d fingerprint_functions=%d/%d+1 relation_delta=%d rule_delta=%d function_delta=%d probes=%d",
			len(current.relations), len(prior.relations),
			len(current.rules), len(prior.rules),
			len(current.fingerprintTables), len(prior.fingerprintTables),
			len(current.fingerprintFunctions), len(prior.fingerprintFunctions),
			len(current.footprintRelations), len(current.footprintRules),
			len(current.footprintFunctions), len(current.footprintProbes),
		)
	}
	if !reflect.DeepEqual(current.relations[:len(prior.relations)], prior.relations) {
		t.Fatal("027 did not carry forward the exact 026 relation contract")
	}
	if !reflect.DeepEqual(current.rules[:len(prior.rules)], prior.rules) {
		t.Fatal("027 did not carry forward the exact 026 rule contract")
	}
	if !reflect.DeepEqual(current.fingerprintTables, prior.fingerprintTables) {
		t.Fatalf("027 did not carry forward inherited fingerprint tables: prior=%v current=%v", prior.fingerprintTables, current.fingerprintTables)
	}
	if len(prior.fingerprintFunctions) > 0 &&
		!reflect.DeepEqual(current.fingerprintFunctions[:len(prior.fingerprintFunctions)], prior.fingerprintFunctions) {
		t.Fatal("027 did not carry forward the exact 026 standalone-function fingerprint contract")
	}
	const snapshotFunction = "provider_pilot_review_snapshot_sha256(uuid,text,uuid)"
	if current.fingerprintFunctions[len(current.fingerprintFunctions)-1] != snapshotFunction ||
		!current.footprintFunctions[snapshotFunction] {
		t.Fatalf("027 standalone snapshot function contract=%v footprint=%v", current.fingerprintFunctions, current.footprintFunctions)
	}

	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "027_provider_pilot_review_evidence.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS provider_pilot_review_events",
		"idx_provider_pilot_reviews_pilot_type",
		"idx_provider_pilot_reviews_claim_type",
		"provider_pilot_review_snapshot_sha256",
		"enforce_provider_pilot_review_event",
		"provider_pilot_review_event_enforced",
		"enforce_provider_pilot_epoch_provider_reviews",
		"provider_pilot_activation_provider_reviews",
		"enforce_provider_offer_pre_activation_review",
		"provider_offer_activation_review",
		"enforce_provider_handoff_ticket_review",
		"provider_handoff_ticket_review",
		"provider_pilot_review_events_no_update",
		"provider_pilot_review_events_no_delete",
		"provider_pilot_review_snapshot_hash",
		"provider_pilot_review_subject",
		"nhs-provider-pilot-review-v1",
		"nhs-provider-pilot-review-snapshot-v1",
		"BEFORE INSERT ON public.provider_pilot_review_events",
		"DO INSTEAD NOTHING",
	} {
		if !strings.Contains(migrationSQL, required) {
			t.Fatalf("027 provider-pilot review-evidence contract is missing %q", required)
		}
	}

	migrations, err := loadMigrations(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("repository protected migration chain: %v", err)
	}
}

func TestProviderCommercialProofManifestProtectedContractCarries027(t *testing.T) {
	prior := protectedMigrationSpecs["027_provider_pilot_review_evidence.sql"]
	current, ok := protectedMigrationSpecs["028_provider_commercial_proof_manifest.sql"]
	if !ok {
		t.Fatal("provider commercial-proof manifest migration has no protected schema contract")
	}
	if len(current.relations) != len(prior.relations)+3 ||
		len(current.rules) != len(prior.rules)+2 ||
		len(current.fingerprintTables) != len(prior.fingerprintTables) ||
		len(current.fingerprintFunctions) != len(prior.fingerprintFunctions) ||
		len(current.footprintRelations) != 3 ||
		len(current.footprintRules) != 2 ||
		len(current.footprintFunctions) != 0 ||
		len(current.footprintProbes) != 2 {
		t.Fatalf(
			"028 cumulative contract shape relations=%d/%d+3 rules=%d/%d+2 fingerprint_tables=%d/%d fingerprint_functions=%d/%d relation_delta=%d rule_delta=%d function_delta=%d probes=%d",
			len(current.relations), len(prior.relations),
			len(current.rules), len(prior.rules),
			len(current.fingerprintTables), len(prior.fingerprintTables),
			len(current.fingerprintFunctions), len(prior.fingerprintFunctions),
			len(current.footprintRelations), len(current.footprintRules),
			len(current.footprintFunctions), len(current.footprintProbes),
		)
	}
	if !reflect.DeepEqual(current.relations[:len(prior.relations)], prior.relations) {
		t.Fatal("028 did not carry forward the exact 027 relation contract")
	}
	if !reflect.DeepEqual(current.rules[:len(prior.rules)], prior.rules) {
		t.Fatal("028 did not carry forward the exact 027 rule contract")
	}
	if !reflect.DeepEqual(current.fingerprintTables, prior.fingerprintTables) {
		t.Fatalf("028 did not carry forward inherited fingerprint tables: prior=%v current=%v", prior.fingerprintTables, current.fingerprintTables)
	}
	if !reflect.DeepEqual(current.fingerprintFunctions, prior.fingerprintFunctions) {
		t.Fatalf("028 did not carry forward standalone-function fingerprints: prior=%v current=%v", prior.fingerprintFunctions, current.fingerprintFunctions)
	}

	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "028_provider_commercial_proof_manifest.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS provider_commercial_proof_manifests",
		"idx_provider_proof_manifests_issued",
		"idx_provider_proof_manifests_key",
		"enforce_provider_commercial_proof_manifest",
		"provider_commercial_proof_manifest_enforced",
		"provider_commercial_proof_manifests_no_update",
		"provider_commercial_proof_manifests_no_delete",
		"provider_proof_manifest_contract",
		"provider_proof_manifest_closed_pilot",
		"provider_proof_manifest_json_binding",
		"provider_proof_manifest_json_shape",
		"provider_proof_manifest_json_types",
		"provider_proof_manifest_privacy_shape",
		"provider_proof_manifest_review_shape",
		"provider_proof_manifest_aggregate_relationships",
		"provider_proof_manifest_issued_at",
		"review_evidence_sha256",
		"nhs-provider-proof-review-root-v1",
		"nhs-free-organic-provider-funded-v1",
		"nhs-private-keyring",
		"monetary_amounts_withheld_for_privacy",
		"REVOKE INSERT, UPDATE, DELETE, TRUNCATE",
		"nhs-provider-proof-manifest-v1",
		"date_trunc('second', transaction_timestamp())",
		"BEFORE INSERT ON public.provider_commercial_proof_manifests",
		"DO INSTEAD NOTHING",
	} {
		if !strings.Contains(migrationSQL, required) {
			t.Fatalf("028 provider commercial-proof manifest contract is missing %q", required)
		}
	}

	migrations, err := loadMigrations(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("repository protected migration chain: %v", err)
	}
}

func TestProviderSettlementProtectedContractCarries028(t *testing.T) {
	prior := protectedMigrationSpecs["028_provider_commercial_proof_manifest.sql"]
	current, ok := protectedMigrationSpecs["029_provider_settlement_receipts.sql"]
	if !ok {
		t.Fatal("provider settlement migration has no protected schema contract")
	}
	if len(current.relations) != len(prior.relations)+7 ||
		len(current.rules) != len(prior.rules)+6 ||
		len(current.fingerprintTables) != len(prior.fingerprintTables) ||
		len(current.fingerprintFunctions) != len(prior.fingerprintFunctions) ||
		len(current.footprintRelations) != 7 ||
		len(current.footprintRules) != 6 ||
		len(current.footprintFunctions) != 0 ||
		len(current.footprintProbes) != 6 {
		t.Fatalf(
			"029 cumulative contract shape relations=%d/%d+7 rules=%d/%d+6 fingerprint_tables=%d/%d fingerprint_functions=%d/%d relation_delta=%d rule_delta=%d function_delta=%d probes=%d",
			len(current.relations), len(prior.relations),
			len(current.rules), len(prior.rules),
			len(current.fingerprintTables), len(prior.fingerprintTables),
			len(current.fingerprintFunctions), len(prior.fingerprintFunctions),
			len(current.footprintRelations), len(current.footprintRules),
			len(current.footprintFunctions), len(current.footprintProbes),
		)
	}
	if !reflect.DeepEqual(current.relations[:len(prior.relations)], prior.relations) {
		t.Fatal("029 did not carry forward the exact 028 relation contract")
	}
	if !reflect.DeepEqual(current.rules[:len(prior.rules)], prior.rules) {
		t.Fatal("029 did not carry forward the exact 028 rule contract")
	}
	if !reflect.DeepEqual(current.fingerprintTables, prior.fingerprintTables) ||
		!reflect.DeepEqual(current.fingerprintFunctions, prior.fingerprintFunctions) {
		t.Fatal("029 did not carry forward inherited fingerprint contracts")
	}

	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", "029_provider_settlement_receipts.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migrationSQL := string(data)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS provider_settlement_orders",
		"CREATE TABLE IF NOT EXISTS provider_settlement_checkout_sessions",
		"CREATE TABLE IF NOT EXISTS provider_settlement_payment_receipts",
		"enforce_provider_settlement_order",
		"provider_settlement_order_enforced",
		"enforce_provider_settlement_checkout_session",
		"provider_settlement_checkout_session_enforced",
		"enforce_provider_settlement_payment_receipt",
		"provider_settlement_payment_receipt_enforced",
		"provider_settlement_orders_no_update",
		"provider_settlement_checkout_sessions_no_update",
		"provider_settlement_payment_receipts_no_update",
		"Stripe-signed webhook path proves a paid Checkout Session",
	} {
		if !strings.Contains(migrationSQL, required) {
			t.Fatalf("029 provider settlement contract is missing %q", required)
		}
	}
}

func TestProtectedMigrationStateFailsClosed(t *testing.T) {
	migration := migrationFile{name: "019_provider_exchange.sql", sha256: strings.Repeat("a", 64)}
	for _, test := range []struct {
		name  string
		state protectedMigrationState
		want  string
	}{
		{name: "clean", state: protectedMigrationState{}, want: ""},
		{name: "ambiguous footprint", state: protectedMigrationState{anyFootprint: true}, want: "ambiguous_prior_019"},
		{name: "orphan ledger", state: protectedMigrationState{ledgerExists: true}, want: "without its required receipt"},
		{name: "checksum drift", state: protectedMigrationState{receiptExists: true, receiptSHA256: strings.Repeat("b", 64), receiptSchemaSHA256: strings.Repeat("c", 64), currentSchemaSHA256: strings.Repeat("c", 64), complete: true}, want: "checksum drift"},
		{name: "schema drift", state: protectedMigrationState{receiptExists: true, receiptSHA256: strings.Repeat("a", 64)}, want: "schema drift"},
		{name: "definition drift", state: protectedMigrationState{receiptExists: true, receiptSHA256: strings.Repeat("a", 64), receiptSchemaSHA256: strings.Repeat("b", 64), currentSchemaSHA256: strings.Repeat("c", 64), complete: true}, want: "schema fingerprint drift"},
		{name: "exact receipt", state: protectedMigrationState{receiptExists: true, receiptSHA256: strings.Repeat("a", 64), receiptSchemaSHA256: strings.Repeat("c", 64), currentSchemaSHA256: strings.Repeat("c", 64), complete: true}, want: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateProtectedMigrationState(migration, test.state, true)
			if test.want == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
	future := migrationFile{name: "020_future.sql", sha256: strings.Repeat("b", 64)}
	if err := validateProtectedMigrationState(future, protectedMigrationState{ledgerExists: true}, false); err != nil {
		t.Fatalf("future protected migration could not use existing ledger: %v", err)
	}
	superseded := protectedMigrationState{
		receiptExists: true, receiptSHA256: strings.Repeat("a", 64), complete: true,
		receiptSchemaSHA256: strings.Repeat("b", 64), currentSchemaSHA256: strings.Repeat("c", 64),
	}
	if err := validateProtectedMigrationState(migration, superseded, false); err != nil {
		t.Fatalf("superseded protected receipt still owned latest schema fingerprint: %v", err)
	}
	if err := validateProtectedMigrationState(future, protectedMigrationState{anyFootprint: true}, false); err == nil || !strings.Contains(err.Error(), "ambiguous_prior_020") {
		t.Fatalf("future ambiguous footprint error = %v", err)
	}
}

func TestProtectedMigrationRequiresExactReleaseRevision(t *testing.T) {
	valid := "0123456789abcdef0123456789abcdef01234567"
	if got, err := protectedMigrationRevision(valid); err != nil || got != valid {
		t.Fatalf("valid release revision = %q, err=%v", got, err)
	}
	for _, invalid := range []string{"", "development", strings.Repeat("g", 40), valid[:39]} {
		if _, err := protectedMigrationRevision(invalid); err == nil {
			t.Fatalf("invalid release revision %q was accepted", invalid)
		}
	}
}

func TestRequiredProtectedMigrationSetIsExactThrough029(t *testing.T) {
	migrations, err := loadMigrations(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateRequiredProtectedMigrationSet(migrations); err != nil {
		t.Fatalf("repository protected migration set: %v", err)
	}
	withoutTerminal := make([]migrationFile, 0, len(migrations)-1)
	for _, migration := range migrations {
		if migration.name != "029_provider_settlement_receipts.sql" {
			withoutTerminal = append(withoutTerminal, migration)
		}
	}
	if err := validateRequiredProtectedMigrationSet(withoutTerminal); err == nil ||
		!strings.Contains(err.Error(), "required protected migration is missing: 029_provider_settlement_receipts.sql") {
		t.Fatalf("missing terminal protected migration error=%v", err)
	}
	withUnknown := append(append([]migrationFile(nil), migrations...), migrationFile{name: "030_unreviewed.sql"})
	if err := validateRequiredProtectedMigrationSet(withUnknown); err == nil ||
		!strings.Contains(err.Error(), "protected migration set is not exact through 029_provider_settlement_receipts.sql") {
		t.Fatalf("extra protected migration error=%v", err)
	}
}

func TestProtectedMigrationSpecsMustBeCumulative(t *testing.T) {
	base := protectedMigrationSpecs["019_provider_exchange.sql"]
	future := base
	future.allObjectsAreFootprint = false
	future.footprintRelations = nil
	future.footprintRules = nil
	protectedMigrationSpecs["020_future.sql"] = future
	t.Cleanup(func() { delete(protectedMigrationSpecs, "020_future.sql") })
	migrations := []migrationFile{
		{name: "019_provider_exchange.sql"},
		{name: "020_future.sql"},
	}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("cumulative future migration contract rejected: %v", err)
	}

	future.relations = append([]migrationRelation(nil), base.relations[:len(base.relations)-1]...)
	protectedMigrationSpecs["020_future.sql"] = future
	if err := validateProtectedMigrationSpecChain(migrations); err == nil || !strings.Contains(err.Error(), "does not carry forward relation contract") {
		t.Fatalf("non-cumulative future migration contract error = %v", err)
	}
}

func TestProtectedMigrationFootprintsAreExactNewDelta(t *testing.T) {
	base := protectedMigrationSpecs["020_action_interest_receipts.sql"]
	migrations := []migrationFile{
		{name: "019_provider_exchange.sql"},
		{name: "020_action_interest_receipts.sql"},
		{name: "021_future.sql"},
	}

	valid := protectedMigrationSpec{
		relations: append(append([]migrationRelation(nil), base.relations...),
			migrationRelation{name: "future_table", relkind: "r"}),
		rules:              append([]migrationRule(nil), base.rules...),
		footprintRelations: map[string]bool{"future_table": true},
		footprintRules:     map[string]bool{},
	}
	protectedMigrationSpecs["021_future.sql"] = valid
	t.Cleanup(func() { delete(protectedMigrationSpecs, "021_future.sql") })
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("exact future delta rejected: %v", err)
	}

	markedInherited := valid
	markedInherited.footprintRelations = map[string]bool{
		"future_table":             true,
		"action_interest_receipts": true,
	}
	protectedMigrationSpecs["021_future.sql"] = markedInherited
	if err := validateProtectedMigrationSpecChain(migrations); err == nil || !strings.Contains(err.Error(), "marks inherited relation") {
		t.Fatalf("inherited footprint marker error = %v", err)
	}

	omittedNew := valid
	omittedNew.footprintRelations = map[string]bool{}
	protectedMigrationSpecs["021_future.sql"] = omittedNew
	if err := validateProtectedMigrationSpecChain(migrations); err == nil || !strings.Contains(err.Error(), "omits new relation") {
		t.Fatalf("omitted new footprint error = %v", err)
	}
}

func TestProtectedMigrationFingerprintFunctionsAreCumulativeAndExactNewDelta(t *testing.T) {
	const baseName = "900_test_function_base.sql"
	const futureName = "901_test_function_future.sql"
	const inheritedFunction = "provider_pilot_review_snapshot_sha256(uuid,text,uuid)"
	const futureFunction = "future_snapshot_sha256(uuid)"
	base := protectedMigrationSpec{
		allObjectsAreFootprint: true,
		fingerprintFunctions:   []string{inheritedFunction},
		footprintFunctions:     map[string]bool{inheritedFunction: true},
	}
	valid := protectedMigrationSpec{
		relations:            append([]migrationRelation(nil), base.relations...),
		rules:                append([]migrationRule(nil), base.rules...),
		fingerprintTables:    append([]string(nil), base.fingerprintTables...),
		fingerprintFunctions: append(append([]string(nil), base.fingerprintFunctions...), futureFunction),
		footprintRelations:   map[string]bool{},
		footprintRules:       map[string]bool{},
		footprintFunctions:   map[string]bool{futureFunction: true},
	}
	protectedMigrationSpecs[baseName] = base
	protectedMigrationSpecs[futureName] = valid
	t.Cleanup(func() {
		delete(protectedMigrationSpecs, baseName)
		delete(protectedMigrationSpecs, futureName)
	})
	migrations := []migrationFile{{name: baseName}, {name: futureName}}
	if err := validateProtectedMigrationSpecChain(migrations); err != nil {
		t.Fatalf("exact future standalone-function delta rejected: %v", err)
	}

	markedInherited := valid
	markedInherited.footprintFunctions = map[string]bool{
		futureFunction:    true,
		inheritedFunction: true,
	}
	protectedMigrationSpecs[futureName] = markedInherited
	if err := validateProtectedMigrationSpecChain(migrations); err == nil || !strings.Contains(err.Error(), "marks inherited fingerprint function") {
		t.Fatalf("inherited standalone-function footprint marker error = %v", err)
	}

	omittedNew := valid
	omittedNew.footprintFunctions = map[string]bool{}
	protectedMigrationSpecs[futureName] = omittedNew
	if err := validateProtectedMigrationSpecChain(migrations); err == nil || !strings.Contains(err.Error(), "omits new fingerprint function") {
		t.Fatalf("omitted standalone-function footprint error = %v", err)
	}

	droppedInherited := valid
	droppedInherited.fingerprintFunctions = []string{futureFunction}
	protectedMigrationSpecs[futureName] = droppedInherited
	if err := validateProtectedMigrationSpecChain(migrations); err == nil || !strings.Contains(err.Error(), "does not carry forward fingerprint function") {
		t.Fatalf("dropped inherited standalone-function error = %v", err)
	}
}
