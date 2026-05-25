package domain

import "testing"

func TestProjectStatusValues(t *testing.T) {
	cases := []struct {
		s    ProjectStatus
		want string
	}{
		{ProjectStatusActive, "active"},
		{ProjectStatusMissing, "missing"},
		{ProjectStatusUnreadable, "unreadable"},
		{ProjectStatusRemoved, "removed"},
	}
	for _, tc := range cases {
		if string(tc.s) != tc.want {
			t.Errorf("ProjectStatus: got %q want %q", string(tc.s), tc.want)
		}
	}
}

func TestProviderStatusValues(t *testing.T) {
	cases := []struct {
		s    ProviderStatus
		want string
	}{
		{ProviderStatusSupported, "supported"},
		{ProviderStatusExperimental, "experimental"},
		{ProviderStatusUnsupported, "unsupported"},
		{ProviderStatusDisabled, "disabled"},
	}
	for _, tc := range cases {
		if string(tc.s) != tc.want {
			t.Errorf("ProviderStatus: got %q want %q", string(tc.s), tc.want)
		}
	}
}

func TestDetectionStatusValues(t *testing.T) {
	cases := []struct {
		s    DetectionStatus
		want string
	}{
		{DetectionStatusDetected, "detected"},
		{DetectionStatusConfigured, "configured"},
		{DetectionStatusMissing, "missing"},
		{DetectionStatusUnsupported, "unsupported"},
		{DetectionStatusInvalidStructure, "invalid_structure"},
		{DetectionStatusFormatUnknown, "format_unknown"},
	}
	for _, tc := range cases {
		if string(tc.s) != tc.want {
			t.Errorf("DetectionStatus: got %q want %q", string(tc.s), tc.want)
		}
	}
}

func TestInstallModeValues(t *testing.T) {
	cases := []struct {
		m    InstallMode
		want string
	}{
		{InstallModeSymlink, "symlink"},
		{InstallModeRsyncCopy, "rsync_copy"},
		{InstallModeDirect, "direct"},
	}
	for _, tc := range cases {
		if string(tc.m) != tc.want {
			t.Errorf("InstallMode: got %q want %q", string(tc.m), tc.want)
		}
	}
}

func TestInstallStatusValues(t *testing.T) {
	cases := []struct {
		s    InstallStatus
		want string
	}{
		{InstallStatusCurrent, "current"},
		{InstallStatusOutdated, "outdated"},
		{InstallStatusMissing, "missing"},
		{InstallStatusBrokenSymlink, "broken_symlink"},
		{InstallStatusOldHost, "old_host"},
		{InstallStatusExternalSymlink, "external_symlink"},
		{InstallStatusConflict, "conflict"},
		{InstallStatusNeedsSync, "needs_sync"},
		{InstallStatusError, "error"},
	}
	for _, tc := range cases {
		if string(tc.s) != tc.want {
			t.Errorf("InstallStatus: got %q want %q", string(tc.s), tc.want)
		}
	}
}

func TestWarningScopeProjectValues(t *testing.T) {
	cases := []struct {
		s    WarningScopeType
		want string
	}{
		{WarningScopeProject, "project"},
		{WarningScopeProjectProvider, "project_provider"},
		{WarningScopeInstall, "install"},
	}
	for _, tc := range cases {
		if string(tc.s) != tc.want {
			t.Errorf("WarningScopeType: got %q want %q", string(tc.s), tc.want)
		}
	}
}

func TestNewProviderError_RPCCode(t *testing.T) {
	e := NewProviderError("provider failed", "adapter returned error")
	if got := e.RPCCode(); got != 1003 {
		t.Fatalf("NewProviderError RPCCode: got %d want 1003", got)
	}
	if e.Code != CodeProvider {
		t.Fatalf("NewProviderError Code: got %q want %q", e.Code, CodeProvider)
	}
}
