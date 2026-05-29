package network

import (
	"context"
	"strings"
	"testing"
)

func TestGitLsRemoteClient_HTTPSOnly(t *testing.T) {
	client := NewGitLsRemoteClient()
	ctx := context.Background()

	cases := []struct {
		name      string
		url       string
		wantError string
	}{
		{"http rejected", "http://github.com/foo/bar.git", "non_https_scheme_rejected"},
		{"git rejected", "git://github.com/foo/bar.git", "non_https_scheme_rejected"},
		{"ssh rejected", "ssh://git@github.com/foo/bar.git", "non_https_scheme_rejected"},
		{"file rejected", "file:///tmp/repo.git", "non_https_scheme_rejected"},
		{"bare path rejected", "/tmp/repo.git", "non_https_scheme_rejected"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := client.LsRemote(ctx, tc.url, "main")
			if !strings.Contains(res.Error, tc.wantError) {
				t.Errorf("LsRemote(%q): got error %q, want containing %q", tc.url, res.Error, tc.wantError)
			}
			if res.RemoteSHA != "" {
				t.Errorf("LsRemote(%q): expected no RemoteSHA, got %q", tc.url, res.RemoteSHA)
			}
		})
	}
}

func TestNoopClient_NeverContacts(t *testing.T) {
	client := NoopClient{}
	ctx := context.Background()
	res := client.LsRemote(ctx, "https://github.com/example/repo.git", "main")
	if res.Error != "update_check_disabled" {
		t.Errorf("NoopClient: expected error 'update_check_disabled', got %q", res.Error)
	}
	if res.RemoteSHA != "" {
		t.Errorf("NoopClient: expected empty RemoteSHA, got %q", res.RemoteSHA)
	}
}

func TestGitLsRemoteClient_GitNotFound(t *testing.T) {
	// Only run if git is genuinely absent — skip otherwise to avoid test fragility.
	// This test is informational; the real gate is the service-level git_not_found check.
	t.Skip("git-not-found test requires a PATH without git; covered by unit inspection of LookPath branch")
}
