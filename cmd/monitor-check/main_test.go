package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestMonitorLogIdentityRedactsEmail(t *testing.T) {
	got := monitorLogIdentity("Owner+Monitor@Example.COM")
	if strings.Contains(got, "Owner") || strings.Contains(got, "owner") || strings.Contains(got, "Monitor") || strings.Contains(got, "@") {
		t.Fatalf("monitorLogIdentity leaked raw email data: %q", got)
	}
	if !strings.Contains(got, "email_domain=example.com") {
		t.Fatalf("monitorLogIdentity missing email domain context: %q", got)
	}
	if !strings.Contains(got, "email_hash=") {
		t.Fatalf("monitorLogIdentity missing stable redacted hash: %q", got)
	}
}

func TestFirstCheckQuarantineReason(t *testing.T) {
	tests := []struct {
		name      string
		lastScore *int
		site      *models.Site
		cerr      error
		wantOK    bool
		want      string
	}{
		{
			name:   "first crawl failure",
			cerr:   errors.New("dial failed"),
			wantOK: true,
			want:   firstCheckFailedQuarantineReason,
		},
		{
			name:   "first zero score",
			site:   &models.Site{AgenticScore: 0},
			wantOK: true,
			want:   firstCheckZeroScoreQuarantineReason,
		},
		{
			name:   "first valid score remains active",
			site:   &models.Site{AgenticScore: 25, HasLLMsTxt: true},
			wantOK: false,
		},
		{
			name:      "previously valid monitor can alert instead",
			lastScore: intPtr(80),
			site:      &models.Site{AgenticScore: 0},
			wantOK:    false,
		},
		{
			name:      "legacy zero baseline still quarantines",
			lastScore: intPtr(0),
			site:      &models.Site{AgenticScore: 0},
			wantOK:    true,
			want:      firstCheckZeroScoreQuarantineReason,
		},
		{
			name:      "legacy zero baseline can recover",
			lastScore: intPtr(0),
			site:      &models.Site{AgenticScore: 20, HasLLMsTxt: true},
			wantOK:    false,
		},
	}
	for _, tc := range tests {
		got, ok := firstCheckQuarantineReason(&models.Monitor{LastScore: tc.lastScore}, tc.site, tc.cerr)
		if ok != tc.wantOK {
			t.Fatalf("%s: ok = %v, want %v", tc.name, ok, tc.wantOK)
		}
		if got != tc.want {
			t.Fatalf("%s: reason = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func intPtr(v int) *int {
	return &v
}
