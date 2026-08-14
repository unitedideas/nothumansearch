package models

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/unitedideas/nothumansearch/internal/testpostgres"
)

func TestMCPAnalyticsDetailFollowthroughPostgres(t *testing.T) {
	dsn := testpostgres.DSN(t, "NHS_QUERY_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set NHS_QUERY_TEST_POSTGRES_DSN to an isolated disposable PostgreSQL database or set NHS_EMBEDDED_POSTGRES=1")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE mcp_requests (
			method text NOT NULL,
			tool_name text,
			arguments jsonb,
			result_count integer,
			user_agent text,
			ip_hash text,
			duration_ms integer,
			created_at timestamptz NOT NULL DEFAULT NOW()
		)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO mcp_requests (method, tool_name, arguments, user_agent, ip_hash) VALUES
		('tools/call', 'get_site_details', '{"synthetic_test":false}', 'client', 'a'),
		('tools/call', 'get_site_details', '{"search_receipt_supplied":true,"selection_recorded":false,"synthetic_test":false}', 'client', 'b'),
		('tools/call', 'get_site_details', '{"search_receipt_supplied":true,"selection_recorded":true,"synthetic_test":false}', 'client', 'c'),
		('tools/call', 'get_site_details', '{"search_receipt_supplied":true,"selection_recorded":true,"synthetic_test":true}', 'smoke', 'd'),
		('tools/call', 'search_agents', '{"demand_topics":["developer-tools"],"synthetic_test":false}', 'client', 'e')
	`); err != nil {
		t.Fatal(err)
	}

	report, err := GetMCPAnalytics(db, 1)
	if err != nil {
		t.Fatal(err)
	}
	followthrough, ok := report["detail_followthrough"].(map[string]any)
	if !ok {
		t.Fatalf("detail_followthrough = %#v", report["detail_followthrough"])
	}
	for key, want := range map[string]int{
		"detail_calls":                  3,
		"search_receipt_supplied_calls": 2,
		"selection_recorded_calls":      1,
	} {
		if got := followthrough[key]; got != want {
			t.Fatalf("%s = %#v, want %d", key, got, want)
		}
	}
}
