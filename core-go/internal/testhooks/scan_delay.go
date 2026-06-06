package testhooks

import (
	"os"
	"strconv"
	"time"
)

const maxScanDelayMS int64 = 5000

// ScanDelayDuration reads SKILLBOX_SCAN_DELAY_MS and returns a clamped duration.
// Returns 0 when unset, <= 0, or unparseable - no delay applied.
// Caps at 5000 ms so an accidental set env variable cannot hang indefinitely.
func ScanDelayDuration() time.Duration {
	s := os.Getenv("SKILLBOX_SCAN_DELAY_MS")
	if s == "" {
		return 0
	}
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil || ms <= 0 {
		return 0
	}
	if ms > maxScanDelayMS {
		ms = maxScanDelayMS
	}
	return time.Duration(ms) * time.Millisecond
}
