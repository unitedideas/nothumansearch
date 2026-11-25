package models

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/testpostgres"
)

func TestSearchTextRelevancePostgres(t *testing.T) {
	dsn := testpostgres.DSN(t, "NHS_RANKING_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set NHS_EMBEDDED_POSTGRES=1 for disposable PostgreSQL")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	schema, err := os.ReadFile("../../migrations/001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`ALTER TABLE sites ADD COLUMN search_vector tsvector, ADD COLUMN has_favicon boolean DEFAULT false, ADD COLUMN favicon_url text DEFAULT ''`); err != nil {
		t.Fatal(err)
	}
	// Public catalog vectors preserve lexical evidence from the October and November audits.
	data, err := os.ReadFile("testdata/ranking_catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Domain      string `json:"domain"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Score       int    `json:"agentic_score"`
		Vector      string `json:"vector"`
	}
	if err = json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, f := range fixtures {
		_, err = db.Exec(`INSERT INTO sites(domain,url,name,description,agentic_score,search_vector,has_structured_api,crawl_status) VALUES($1,$2,$3,$4,$5,$6::tsvector,true,'success')`, f.Domain, "https://"+f.Domain, f.Name, f.Description, f.Score, f.Vector)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct{ query, first, before, after string }{
		{"llms.txt generator", "llms-text.com", "", ""},
		{"LLMS.TXT generator", "llms-text.com", "", ""},
		{"llms.txt. generator", "llms-text.com", "", ""},
		{"api.notion.com docs", "api.notion.com", "", ""},
		{"API.NOTION.COM docs", "api.notion.com", "", ""},
		{"api.notion.com.", "api.notion.com", "", ""},
		{"Notion API", "api.notion.com", "", ""},
		{"OpenRouter free models", "openrouter.ai", "", ""},
		{"browser automation API", "roveapi.com", "", ""},
		{"validate email address", "mailboxvalidator.com", "", ""},
		{"historical weather data", "", "weatherbit.io", "exchange-rate.nimbus-api.com"},
		{"vector similarity search", "", "pinecone.io", "memestack.ai"},
		{"subscription billing API", "developer.paddle.com", "stripe.com", "zlurp.ai"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			sites, total, err := SearchSites(db, SearchParams{Query: tc.query, Limit: 100})
			if err != nil {
				t.Fatal(err)
			}
			if len(sites) == 0 || total != len(sites) {
				t.Fatalf("missing results: total=%d rows=%d", total, len(sites))
			}
			if tc.first != "" && sites[0].Domain != tc.first {
				t.Errorf("first=%s, want %s", sites[0].Domain, tc.first)
			}
			if tc.before != "" {
				positions := map[string]int{}
				for i, s := range sites {
					positions[s.Domain] = i
				}
				before, ok1 := positions[tc.before]
				after, ok2 := positions[tc.after]
				if !ok1 || !ok2 || before >= after {
					t.Errorf("want %s before %s; positions=%v", tc.before, tc.after, positions)
				}
			}
		})
	}

	assertExactHostAbsent := func(t *testing.T, params SearchParams) {
		t.Helper()
		params.Query = "api.notion.com docs"
		params.Limit = 100
		sites, total, err := SearchSites(db, params)
		if err != nil {
			t.Fatal(err)
		}
		if total != len(sites) {
			t.Fatalf("count and result rows differ: total=%d rows=%d", total, len(sites))
		}
		for _, site := range sites {
			if site.Domain == "api.notion.com" {
				t.Fatal("exact hostname bypassed a requested filter or eligibility")
			}
		}
	}
	t.Run("exact host respects requested filters", func(t *testing.T) {
		var score int
		if err := db.QueryRow(`SELECT agentic_score FROM sites WHERE domain='api.notion.com'`).Scan(&score); err != nil {
			t.Fatal(err)
		}
		for _, params := range []SearchParams{
			{MinScore: score + 1},
			{Category: "finance"},
			{HasMCP: true},
			{HasOpenAPI: true},
		} {
			assertExactHostAbsent(t, params)
		}
	})
	for _, tc := range []struct{ name, change, restore string }{
		{"exact host requires successful crawl", `UPDATE sites SET crawl_status='failed' WHERE domain='api.notion.com'`, `UPDATE sites SET crawl_status='success' WHERE domain='api.notion.com'`},
		{"exact host requires a hard agent signal", `UPDATE sites SET has_structured_api=false WHERE domain='api.notion.com'`, `UPDATE sites SET has_structured_api=true WHERE domain='api.notion.com'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := db.Exec(tc.change); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if _, err := db.Exec(tc.restore); err != nil {
					t.Error(err)
				}
			}()
			assertExactHostAbsent(t, SearchParams{})
		})
	}
}
