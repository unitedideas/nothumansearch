package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

type scriptedProviderTXTResolver struct {
	records map[string][]string
	errors  map[string]error
}

func (r scriptedProviderTXTResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	if err := r.errors[name]; err != nil {
		return nil, err
	}
	return r.records[name], nil
}

func TestProviderChallengeCandidatesAreExactBoundedAndDeduplicated(t *testing.T) {
	t.Parallel()
	valid := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	records := []string{
		"google-site-verification=unrelated",
		providerDNSChallengePrefix + valid,
		" " + providerDNSChallengePrefix + valid + " ",
		providerDNSChallengePrefix + "too-short",
		"prefix-" + providerDNSChallengePrefix + valid,
	}
	if got, want := providerChallengeCandidates(records), []string{valid}; !reflect.DeepEqual(got, want) {
		t.Fatalf("providerChallengeCandidates = %#v, want %#v", got, want)
	}
}

func TestReverifyDueProviderClaimsRecordsMatchesAndLookupFailures(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	valid := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	leases := []models.ProviderClaimDNSLease{
		{
			ClaimID:                "123e4567-e89b-42d3-a456-426614174000",
			LeaseID:                "223e4567-e89b-42d3-a456-426614174000",
			VerificationRecordName: "_nothumansearch.match.example",
			LeaseUntil:             now.Add(time.Minute),
		},
		{
			ClaimID:                "323e4567-e89b-42d3-a456-426614174000",
			LeaseID:                "423e4567-e89b-42d3-a456-426614174000",
			VerificationRecordName: "_nothumansearch.error.example",
			LeaseUntil:             now.Add(time.Minute),
		},
	}
	completed := map[string][]string{}
	h := &ProviderExchangeHandler{
		DB: &sql.DB{},
		TXTResolver: scriptedProviderTXTResolver{
			records: map[string][]string{
				"_nothumansearch.match.example": {providerDNSChallengePrefix + valid},
			},
			errors: map[string]error{
				"_nothumansearch.error.example": errors.New("temporary DNS failure"),
			},
		},
		dnsNow: func() time.Time { return now },
		leaseDNSChecks: func(_ *sql.DB, gotNow time.Time, limit int) ([]models.ProviderClaimDNSLease, error) {
			if !gotNow.Equal(now) || limit != 10 {
				t.Fatalf("lease inputs = %s, %d", gotNow, limit)
			}
			return leases, nil
		},
		completeDNSCheck: func(_ *sql.DB, claimID, _ string, candidates []string, checkedAt time.Time) (*models.ProviderClaimDNSCheckResult, error) {
			if !checkedAt.Equal(now) {
				t.Fatalf("completion time = %s, want %s", checkedAt, now)
			}
			completed[claimID] = append([]string(nil), candidates...)
			if claimID == leases[0].ClaimID {
				return &models.ProviderClaimDNSCheckResult{ClaimID: claimID, Matched: true}, nil
			}
			return &models.ProviderClaimDNSCheckResult{ClaimID: claimID, Revoked: true}, nil
		},
	}

	stats, err := h.ReverifyDueProviderClaims(context.Background(), 10)
	if err != nil {
		t.Fatalf("ReverifyDueProviderClaims: %v", err)
	}
	if stats != (ProviderDNSReverificationStats{Leased: 2, Matched: 1, Failed: 1, Revoked: 1}) {
		t.Fatalf("stats = %+v", stats)
	}
	if got := completed[leases[0].ClaimID]; !reflect.DeepEqual(got, []string{valid}) {
		t.Fatalf("matched candidates = %#v", got)
	}
	if got := completed[leases[1].ClaimID]; len(got) != 0 {
		t.Fatalf("DNS error produced proof candidates: %#v", got)
	}
	encoded, err := json.Marshal(leases[0])
	if err != nil {
		t.Fatalf("marshal internal lease: %v", err)
	}
	if strings.Contains(string(encoded), valid) || string(encoded) != "{}" {
		t.Fatalf("internal DNS lease is publicly serializable: %s", encoded)
	}
}

func TestReverifyDueProviderClaimsDoesNotCountCallerCancellationAsDNSFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	completed := false
	h := &ProviderExchangeHandler{
		DB:     &sql.DB{},
		dnsNow: time.Now,
		leaseDNSChecks: func(*sql.DB, time.Time, int) ([]models.ProviderClaimDNSLease, error) {
			return []models.ProviderClaimDNSLease{{
				ClaimID:                "123e4567-e89b-42d3-a456-426614174000",
				LeaseID:                "223e4567-e89b-42d3-a456-426614174000",
				VerificationRecordName: "_nothumansearch.example.com",
			}}, nil
		},
		completeDNSCheck: func(*sql.DB, string, string, []string, time.Time) (*models.ProviderClaimDNSCheckResult, error) {
			completed = true
			return nil, nil
		},
	}
	stats, err := h.ReverifyDueProviderClaims(ctx, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if completed || stats.Failed != 0 {
		t.Fatalf("canceled check completed=%t stats=%+v", completed, stats)
	}
}
