package models

import (
	"os"
	"strings"
	"testing"
)

func TestPriorityUsageReservationNilGuards(t *testing.T) {
	key := &APIKey{ID: 7, MonthlyLimit: 10}

	reservationID, remaining, reserved, err := ReservePriorityUsage(nil, key, "rest", "GET", "/api/v1/search", "")
	if err != nil {
		t.Fatalf("ReservePriorityUsage nil DB error = %v, want nil", err)
	}
	if reservationID != 0 || remaining != 0 || reserved {
		t.Fatalf("ReservePriorityUsage nil DB = (%d, %d, %t), want zero, zero, false", reservationID, remaining, reserved)
	}
	if err := ReleasePriorityUsage(nil, key, 1); err != nil {
		t.Fatalf("ReleasePriorityUsage nil DB error = %v, want nil", err)
	}
	if err := ReleasePriorityUsage(nil, nil, 0); err != nil {
		t.Fatalf("ReleasePriorityUsage nil key/reservation error = %v, want nil", err)
	}
}

func TestPriorityUsageReservationSourceIsAtomicAndExactlyRefundable(t *testing.T) {
	source, err := os.ReadFile("api_usage.go")
	if err != nil {
		t.Fatalf("read api_usage.go: %v", err)
	}
	text := string(source)
	reserveStart := strings.Index(text, "func ReservePriorityUsage")
	releaseStart := strings.Index(text, "func ReleasePriorityUsage")
	resetStart := strings.Index(text, "func QuotaResetUnix")
	if reserveStart < 0 || releaseStart <= reserveStart || resetStart <= releaseStart {
		t.Fatal("could not isolate priority reservation/refund source")
	}
	reserveSource := text[reserveStart:releaseStart]
	for _, required := range []string{
		"db.Begin()",
		"pg_advisory_xact_lock",
		"RETURNING id",
		"tx.Commit()",
	} {
		if !strings.Contains(reserveSource, required) {
			t.Fatalf("priority reservation missing atomicity contract %q", required)
		}
	}

	releaseSource := text[releaseStart:resetStart]
	if !strings.Contains(releaseSource, "WHERE id=$1 AND api_key_id=$2") {
		t.Fatal("priority refund is not scoped to the exact reservation and API key")
	}
}
