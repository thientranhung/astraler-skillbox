package domain

import "testing"

func TestWarningSeverityCounts_Total(t *testing.T) {
	c := WarningSeverityCounts{
		Info:     1,
		Warning:  2,
		Error:    3,
		Blocking: 4,
	}
	if got := c.Total(); got != 10 {
		t.Errorf("Total() = %d, want 10", got)
	}
}
