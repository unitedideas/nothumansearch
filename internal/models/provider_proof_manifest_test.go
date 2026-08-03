package models

import (
	"errors"
	"testing"

	"github.com/lib/pq"
)

func TestProviderProofManifestIssueRetryIsNarrow(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "serialization failure",
			err:  &pq.Error{Code: "40001"},
			want: true,
		},
		{
			name: "exact pilot uniqueness race",
			err: &pq.Error{
				Code:       "23505",
				Constraint: "provider_commercial_proof_manifests_provider_pilot_epoch_id_key",
			},
			want: true,
		},
		{
			name: "manifest id collision",
			err: &pq.Error{
				Code:       "23505",
				Constraint: "provider_commercial_proof_manifests_pkey",
			},
			want: false,
		},
		{
			name: "unrelated unique conflict",
			err: &pq.Error{
				Code:       "23505",
				Constraint: "other_unique_key",
			},
			want: false,
		},
		{
			name: "business conflict",
			err:  ErrProviderProofManifestRequestConflict,
			want: false,
		},
		{
			name: "wrapped serialization failure",
			err:  errors.Join(errors.New("issue manifest"), &pq.Error{Code: "40001"}),
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := retryableProviderProofManifestIssue(test.err); got != test.want {
				t.Fatalf("retryableProviderProofManifestIssue(%v)=%t, want %t", test.err, got, test.want)
			}
		})
	}
}
