package network

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// UpdateCheckResult holds the outcome of a single git ls-remote call.
type UpdateCheckResult struct {
	SourceURL  string
	SourceRef  string
	RemoteSHA  string // empty on error or not-found
	Error      string // non-empty on failure
	DurationMS int64
}

// UpdateCheckClient queries remote plugin source refs.
// Implementations: GitLsRemoteClient (real), NoopClient (stub when setting OFF).
type UpdateCheckClient interface {
	LsRemote(ctx context.Context, sourceURL, ref string) UpdateCheckResult
}

// NoopClient is the boot-time stub when network.update_check.enabled=false.
// It satisfies the interface but never contacts any host.
type NoopClient struct{}

func (NoopClient) LsRemote(_ context.Context, sourceURL, ref string) UpdateCheckResult {
	return UpdateCheckResult{SourceURL: sourceURL, SourceRef: ref, Error: "update_check_disabled"}
}

// GitLsRemoteClient shells out to system git with a stripped environment.
type GitLsRemoteClient struct {
	timeout time.Duration // per-request (default 8s)
}

func NewGitLsRemoteClient() *GitLsRemoteClient {
	return &GitLsRemoteClient{timeout: 8 * time.Second}
}

func (c *GitLsRemoteClient) LsRemote(ctx context.Context, sourceURL, ref string) UpdateCheckResult {
	start := time.Now()
	res := UpdateCheckResult{SourceURL: sourceURL, SourceRef: ref}

	// Larry-1: HTTPS-only — reject non-https schemes before any subprocess.
	u, err := url.Parse(sourceURL)
	if err != nil || u.Scheme != "https" {
		res.Error = fmt.Sprintf("non_https_scheme_rejected: %q", sourceURL)
		res.DurationMS = time.Since(start).Milliseconds()
		return res
	}

	// Larry-3: git-not-found — degrade gracefully.
	gitPath, err := exec.LookPath("git")
	if err != nil {
		res.Error = "git_not_found"
		res.DurationMS = time.Since(start).Milliseconds()
		return res
	}

	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"ls-remote", "--", sourceURL}
	if ref != "" {
		args = append(args, ref)
	}
	cmd := exec.CommandContext(reqCtx, gitPath, args...)

	// Larry-2: env stripping — only PATH + GIT_TERMINAL_PROMPT=0.
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"GIT_TERMINAL_PROMPT=0",
	}

	out, err := cmd.Output()
	res.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		if reqCtx.Err() != nil {
			res.Error = "timeout"
		} else {
			res.Error = "git_ls_remote_failed"
		}
		return res
	}

	// Output format: "<sha>\t<refname>\n..."  — take the first SHA found.
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 && parts[0] != "" {
			res.RemoteSHA = parts[0]
			break
		}
	}
	if res.RemoteSHA == "" {
		res.Error = "ref_not_found"
	}
	return res
}
