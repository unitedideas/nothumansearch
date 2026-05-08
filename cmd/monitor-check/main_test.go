package main

import (
	"strings"
	"testing"
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
