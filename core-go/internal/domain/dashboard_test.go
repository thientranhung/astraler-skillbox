package domain

import "testing"

func TestWarningSeverityCounts_Total(t *testing.T) {
	cases := []struct {
		name string
		c    WarningSeverityCounts
		want int
	}{
		{"all zero", WarningSeverityCounts{}, 0},
		{"all set", WarningSeverityCounts{Info: 1, Warning: 2, Error: 3, Blocking: 4}, 10},
	}
	for _, tc := range cases {
		if got := tc.c.Total(); got != tc.want {
			t.Errorf("%s: Total() = %d, want %d", tc.name, got, tc.want)
		}
	}
}
