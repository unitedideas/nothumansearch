package testpostgres

import (
	"net/url"
	"testing"
)

func TestWithSSLDisabled(t *testing.T) {
	dsn, err := withSSLDisabled("postgres://postgres:postgres@127.0.0.1:5444/nhs_release_test?application_name=test")
	if err != nil {
		t.Fatalf("normalize DSN: %v", err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse normalized DSN: %v", err)
	}
	if got := parsed.Query().Get("sslmode"); got != "disable" {
		t.Fatalf("sslmode = %q, want disable", got)
	}
	if got := parsed.Query().Get("application_name"); got != "test" {
		t.Fatalf("application_name = %q, want test", got)
	}
}
