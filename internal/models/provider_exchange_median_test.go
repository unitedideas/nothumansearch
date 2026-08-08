package models

import "testing"

func TestProviderProofMedianSeconds(t *testing.T) {
	for name, test := range map[string]struct {
		values []int64
		want   int64
	}{
		"empty": {values: nil, want: 0},
		"odd":   {values: []int64{9, 1, 5}, want: 5},
		"even":  {values: []int64{10, 2, 8, 4}, want: 6},
	} {
		t.Run(name, func(t *testing.T) {
			if got := providerProofMedianSeconds(test.values); got != test.want {
				t.Fatalf("median = %d, want %d", got, test.want)
			}
		})
	}
}
