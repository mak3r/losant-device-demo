package commands

import "testing"

func TestBaseNodeCount(t *testing.T) {
	cases := []struct {
		module string
		want   int
	}{
		{"aws-k3s-ha", 3},
		{"gcp-k3s-ha", 3},
		{"gcp-k3s-single", 1},
		{"aws-k3s-single", 1},
		{"unknown-module", 1},
	}
	for _, tc := range cases {
		if got := baseNodeCount(tc.module); got != tc.want {
			t.Errorf("baseNodeCount(%q) = %d, want %d", tc.module, got, tc.want)
		}
	}
}
