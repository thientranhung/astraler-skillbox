package testhooks

import (
	"testing"
	"time"
)

func TestScanDelayDuration_Unset(t *testing.T) {
	t.Setenv("SKILLBOX_SCAN_DELAY_MS", "")
	if d := ScanDelayDuration(); d != 0 {
		t.Errorf("unset: got %v, want 0", d)
	}
}

func TestScanDelayDuration_Zero(t *testing.T) {
	t.Setenv("SKILLBOX_SCAN_DELAY_MS", "0")
	if d := ScanDelayDuration(); d != 0 {
		t.Errorf("zero: got %v, want 0", d)
	}
}

func TestScanDelayDuration_Negative(t *testing.T) {
	t.Setenv("SKILLBOX_SCAN_DELAY_MS", "-500")
	if d := ScanDelayDuration(); d != 0 {
		t.Errorf("negative: got %v, want 0", d)
	}
}

func TestScanDelayDuration_Invalid(t *testing.T) {
	for _, val := range []string{"abc", "1.5", "", "NaN"} {
		t.Setenv("SKILLBOX_SCAN_DELAY_MS", val)
		if d := ScanDelayDuration(); d != 0 {
			t.Errorf("invalid %q: got %v, want 0", val, d)
		}
	}
}

func TestScanDelayDuration_Valid(t *testing.T) {
	t.Setenv("SKILLBOX_SCAN_DELAY_MS", "200")
	if d := ScanDelayDuration(); d != 200*time.Millisecond {
		t.Errorf("200ms: got %v, want 200ms", d)
	}
}

func TestScanDelayDuration_Cap(t *testing.T) {
	t.Setenv("SKILLBOX_SCAN_DELAY_MS", "99999")
	if d := ScanDelayDuration(); d != 5000*time.Millisecond {
		t.Errorf("cap: got %v, want 5000ms", d)
	}
}

func TestScanDelayDuration_ExactCap(t *testing.T) {
	t.Setenv("SKILLBOX_SCAN_DELAY_MS", "5000")
	if d := ScanDelayDuration(); d != 5000*time.Millisecond {
		t.Errorf("exact cap: got %v, want 5000ms", d)
	}
}

func TestScanDelayDuration_OneBelowCap(t *testing.T) {
	t.Setenv("SKILLBOX_SCAN_DELAY_MS", "4999")
	if d := ScanDelayDuration(); d != 4999*time.Millisecond {
		t.Errorf("4999ms: got %v, want 4999ms", d)
	}
}
