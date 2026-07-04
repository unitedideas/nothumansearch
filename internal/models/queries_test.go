package models

import "testing"

func TestPinDomainOrderClauseNormalizesDomain(t *testing.T) {
	clause, domain, ok := pinDomainOrderClause(7, " https://www.BringYour.ai/path?utm_source=test ")
	if !ok {
		t.Fatal("pinDomainOrderClause returned ok=false")
	}
	if domain != "bringyour.ai" {
		t.Fatalf("domain = %q, want bringyour.ai", domain)
	}
	wantClause := "CASE WHEN lower(domain) = $7 THEN 1 ELSE 0 END DESC, "
	if clause != wantClause {
		t.Fatalf("clause = %q, want %q", clause, wantClause)
	}
}

func TestPinDomainOrderClauseIgnoresEmptyDomain(t *testing.T) {
	clause, domain, ok := pinDomainOrderClause(3, "  ")
	if ok {
		t.Fatalf("ok = true, want false")
	}
	if clause != "" || domain != "" {
		t.Fatalf("clause/domain = %q/%q, want empty", clause, domain)
	}
}
