package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/handlers"
)

const (
	preflightContract = "nhs-provider-cutover-preflight-v2"
	preflightTimeout  = 90 * time.Second
)

// releaseRevision is injected into the exact container binary at build time.
// The operator-supplied revision is only an expected value and cannot relabel a
// preflight binary built from different source.
var releaseRevision = "development"

type sessionReport struct {
	CandidateSessions        int  `json:"candidate_sessions"`
	OtherTaggedNHSSessions   int  `json:"other_tagged_nhs_sessions"`
	DatabaseInternalSessions int  `json:"database_internal_sessions"`
	UnattributedSessions     int  `json:"unattributed_sessions"`
	OldWriterSessionsZero    bool `json:"old_writer_sessions_zero"`
}

type cutoverReport struct {
	Contract                string                                   `json:"contract"`
	CheckedAt               time.Time                                `json:"checked_at"`
	CandidateRevision       string                                   `json:"candidate_revision"`
	BinaryRevision          string                                   `json:"binary_revision"`
	Mode                    string                                   `json:"mode"`
	DatabaseIdentitySHA256  string                                   `json:"database_identity_sha256"`
	Migrations              *database.MigrationReadinessReport       `json:"migrations"`
	LivePreHandoffTickets   int                                      `json:"live_pre_handoff_tickets"`
	Signing                 *handlers.ProviderSigningRetentionReport `json:"signing"`
	Sessions                sessionReport                            `json:"sessions"`
	ReadyForQuiescedCutover bool                                     `json:"ready_for_quiesced_cutover"`
}

type cutoverReceipt struct {
	Contract     string         `json:"contract"`
	ReportSHA256 string         `json:"report_sha256"`
	Report       *cutoverReport `json:"report"`
}

func main() {
	revision := flag.String("revision", "", "exact 40-character candidate commit")
	mode := flag.String("mode", "", "provider exchange mode: pilot or disabled")
	migrationsDir := flag.String("migrations-dir", "", "candidate migrations directory")
	flag.Parse()

	*revision = strings.ToLower(strings.TrimSpace(*revision))
	*mode = strings.ToLower(strings.TrimSpace(*mode))
	compiledRevision := strings.ToLower(strings.TrimSpace(releaseRevision))
	if !validRevision(*revision) || (*mode != "pilot" && *mode != "disabled") {
		fail("invalid_arguments")
	}
	if !validRevision(compiledRevision) || compiledRevision != *revision {
		fail("candidate_revision_mismatch")
	}
	if *migrationsDir == "" {
		root := os.Getenv("APP_ROOT")
		if root == "" {
			executable, err := os.Executable()
			if err != nil {
				fail("candidate_root_unavailable")
			}
			root = filepath.Dir(executable)
		}
		*migrationsDir = filepath.Join(root, "migrations")
	}

	ctx, cancel := context.WithTimeout(context.Background(), preflightTimeout)
	defer cancel()
	if err := database.ConnectWithReleaseRevisionContext(ctx, *revision); err != nil {
		fail("database_connection_failed")
	}
	defer func() {
		_ = database.DB.Close()
		database.DB = nil
	}()

	migrations, err := database.InspectMigrationReadiness(ctx, *migrationsDir, *revision)
	if err != nil {
		fail("migration_readiness_failed")
	}
	databaseIdentity, err := databaseIdentitySHA256(ctx)
	if err != nil {
		fail("database_identity_failed")
	}
	liveTickets, err := countLivePreHandoffTickets(ctx)
	if err != nil {
		fail("live_ticket_preflight_failed")
	}
	signing, err := handlers.ValidateProviderExchangeSigningRetentionReadOnlyContext(
		ctx, database.DB, *mode == "pilot",
	)
	if err != nil {
		fail("signing_retention_failed")
	}
	sessions, err := inspectSessions(ctx, *revision)
	if err != nil {
		fail("session_preflight_failed")
	}

	report := &cutoverReport{
		Contract:               preflightContract,
		CheckedAt:              time.Now().UTC(),
		CandidateRevision:      *revision,
		BinaryRevision:         compiledRevision,
		Mode:                   *mode,
		DatabaseIdentitySHA256: databaseIdentity,
		Migrations:             migrations,
		LivePreHandoffTickets:  liveTickets,
		Signing:                signing,
		Sessions:               sessions,
	}
	report.ReadyForQuiescedCutover = liveTickets == 0 && sessions.OldWriterSessionsZero &&
		(*mode == "disabled" || signing.ConfigurationValidated)
	encoded, err := json.Marshal(report)
	if err != nil {
		fail("receipt_encoding_failed")
	}
	digest := sha256.Sum256(encoded)
	receipt := cutoverReceipt{
		Contract:     preflightContract,
		ReportSHA256: hex.EncodeToString(digest[:]),
		Report:       report,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(true)
	if err := encoder.Encode(receipt); err != nil {
		os.Exit(1)
	}
	if !report.ReadyForQuiescedCutover {
		os.Exit(1)
	}
}

func validRevision(value string) bool {
	if len(value) != 40 || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 20
}

func databaseIdentitySHA256(ctx context.Context) (string, error) {
	var databaseName, databaseUser, serverAddress string
	var serverPort int
	err := database.DB.QueryRowContext(ctx, `
		SELECT current_database(), current_user,
		       COALESCE(inet_server_addr()::text,'local'),
		       COALESCE(inet_server_port(),0)`).Scan(
		&databaseName, &databaseUser, &serverAddress, &serverPort,
	)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf(
		"%s\x00%s\x00%s\x00%d", databaseName, databaseUser, serverAddress, serverPort,
	)))
	return hex.EncodeToString(digest[:]), nil
}

func countLivePreHandoffTickets(ctx context.Context) (int, error) {
	var tableExists bool
	if err := database.DB.QueryRowContext(ctx,
		`SELECT to_regclass('public.action_tickets') IS NOT NULL`).Scan(&tableExists); err != nil {
		return 0, err
	}
	if !tableExists {
		return 0, nil
	}
	var count int
	err := database.DB.QueryRowContext(ctx, `
		SELECT COUNT(*)::int
		FROM action_tickets
		WHERE status IN ('created','redirected','accepted','activated')
		  AND expires_at > clock_timestamp()
		  AND authorization_revoked_at IS NULL`).Scan(&count)
	return count, err
}

func inspectSessions(ctx context.Context, revision string) (sessionReport, error) {
	report := sessionReport{}
	applicationName := "nhs-server:" + revision
	rows, err := database.DB.QueryContext(ctx, `
		SELECT usename, application_name, COALESCE(client_addr::text,''),
		       COALESCE(inet_server_addr()::text,''), state
		FROM pg_stat_activity
		WHERE datname=current_database()
		  AND pid<>pg_backend_pid()
		  AND backend_type='client backend'`)
	if err != nil {
		return report, err
	}
	defer rows.Close()
	for rows.Next() {
		var user, app, clientAddress, serverAddress, state string
		if err := rows.Scan(&user, &app, &clientAddress, &serverAddress, &state); err != nil {
			return report, err
		}
		switch {
		case app == applicationName:
			report.CandidateSessions++
		case strings.HasPrefix(app, "nhs-server:"):
			report.OtherTaggedNHSSessions++
		case databaseInternalAdministrativeSession(user, app, clientAddress, serverAddress, state):
			report.DatabaseInternalSessions++
		default:
			report.UnattributedSessions++
		}
	}
	if err := rows.Err(); err != nil {
		return report, err
	}
	// The current preflight backend is excluded above. Every other client
	// backend is a possible writer, including another candidate-tagged server
	// and clients that use a nonempty third-party application_name.
	report.OldWriterSessionsZero = report.CandidateSessions == 0 &&
		report.OtherTaggedNHSSessions == 0 && report.UnattributedSessions == 0
	return report, nil
}

// postgres-flex keeps one idle flypgadmin connection from the database
// machine to each database. It is not an application writer. The exception is
// deliberately exact and counted in the receipt: a remote, active, named, or
// differently owned session remains unattributed and blocks the cutover.
func databaseInternalAdministrativeSession(user, app, clientAddress, serverAddress, state string) bool {
	return user == "flypgadmin" && app == "" && state == "idle" &&
		clientAddress != "" && clientAddress == serverAddress
}

func fail(code string) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
		"contract": preflightContract,
		"ok":       false,
		"error":    code,
	})
	os.Exit(1)
}
