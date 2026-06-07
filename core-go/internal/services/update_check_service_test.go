package services_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/astraler/skillbox/core-go/internal/network"
	"github.com/astraler/skillbox/core-go/internal/repositories"
	"github.com/astraler/skillbox/core-go/internal/services"
)

// openTestDB opens a SQLite DB at a temp path with all migrations applied.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := repositories.OpenDatabase(path)
	if err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// spyClient records calls; returns a configurable result.
type spyClient struct {
	calls   []spyCall
	result  network.UpdateCheckResult
}

type spyCall struct {
	URL string
	Ref string
}

func (c *spyClient) LsRemote(_ context.Context, url, ref string) network.UpdateCheckResult {
	c.calls = append(c.calls, spyCall{URL: url, Ref: ref})
	res := c.result
	res.SourceURL = url
	res.SourceRef = ref
	return res
}

// TestUpdateCheck_AlwaysOn_MockClient verifies the always-on path (ADR-0002):
// updateCheck.run returns results from the client with no opt-in setting required.
func TestUpdateCheck_AlwaysOn_MockClient(t *testing.T) {
	db := openTestDB(t)
	cacheRepo := repositories.NewUpdateCheckCacheRepo(db)

	// Create a real marketplace dir structure so the service can find plugin sources.
	claudeDir := t.TempDir()
	mktDir := filepath.Join(claudeDir, "plugins", "marketplaces", "test-market", ".claude-plugin")
	if err := os.MkdirAll(mktDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mktDir, "marketplace.json"), []byte(`{
		"name": "test-market",
		"plugins": [
			{
				"name": "my-plugin",
				"source": {
					"source": "git-subdir",
					"url": "https://github.com/example/plugins.git",
					"ref": "v1.0.0",
					"sha": "aabbcc"
				}
			}
		]
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create installed_plugins.json with a gitCommitSha.
	pluginsDir := filepath.Join(claudeDir, "plugins")
	oldSHA := "aaaaaabbbbbbccccccddddddeeeeeeffffffff00"
	newSHA := "1111111111111111111111111111111111111111"
	if err := os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), []byte(`{
		"version": 2,
		"plugins": {
			"my-plugin@test-market": [
				{
					"scope": "user",
					"version": "v1.0.0",
					"gitCommitSha": "`+oldSHA+`"
				}
			]
		}
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	spy := &spyClient{
		result: network.UpdateCheckResult{RemoteSHA: newSHA},
	}

	ctx := context.Background()
	svc := services.NewUpdateCheckService(cacheRepo, spy, claudeDir)
	result, err := svc.RunUpdateCheck(ctx)
	if err != nil {
		t.Fatalf("RunUpdateCheck: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("status: got %q want %q", result.Status, "ok")
	}
	if len(spy.calls) == 0 {
		t.Fatal("expected at least one LsRemote call, got 0")
	}
	if spy.calls[0].URL != "https://github.com/example/plugins.git" {
		t.Errorf("LsRemote URL: got %q", spy.calls[0].URL)
	}

	// Find the my-plugin result.
	var found *struct {
		UpdateAvailable *bool
	}
	for _, p := range result.Plugins {
		if p.PluginName == "my-plugin" && p.MarketplaceName == "test-market" {
			found = &struct{ UpdateAvailable *bool }{p.UpdateAvailable}
		}
	}
	if found == nil {
		t.Fatal("my-plugin result not found")
	}
	if found.UpdateAvailable == nil || !*found.UpdateAvailable {
		t.Errorf("updateAvailable: expected true (oldSHA != newSHA), got %v", found.UpdateAvailable)
	}

	// Cache should be persisted.
	cached, err := cacheRepo.GetByPlugin(ctx, "claude", "my-plugin", "test-market")
	if err != nil || cached == nil {
		t.Fatalf("cache not persisted: %v", err)
	}
	if cached.RemoteSHA != newSHA {
		t.Errorf("cached RemoteSHA: got %q want %q", cached.RemoteSHA, newSHA)
	}
}

// blockingLsClient blocks each LsRemote call until the context is done,
// simulating a slow network. Used to exercise the batch-deadline path.
type blockingLsClient struct{}

func (c *blockingLsClient) LsRemote(ctx context.Context, url, _ string) network.UpdateCheckResult {
	<-ctx.Done()
	return network.UpdateCheckResult{SourceURL: url, Error: "context_cancelled"}
}

// TestUpdateCheck_BatchDeadline_NotStartedItemsGetTimeoutResult verifies TC-PLUGIN-009:
// when the batch deadline fires before all queued work items can start, every item that
// never acquired the semaphore is returned with Error == "timeout", so the renderer
// sees a complete, accurate result set.
func TestUpdateCheck_BatchDeadline_NotStartedItemsGetTimeoutResult(t *testing.T) {
	const pluginCount = 6 // > updateCheckConcurrency (4); items 4-5 will never start

	db := openTestDB(t)
	cacheRepo := repositories.NewUpdateCheckCacheRepo(db)

	claudeDir := t.TempDir()
	mktDir := filepath.Join(claudeDir, "plugins", "marketplaces", "slow-market", ".claude-plugin")
	if err := os.MkdirAll(mktDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Build marketplace.json with pluginCount plugins.
	pluginsJSON := `{"name":"slow-market","plugins":[`
	for i := 0; i < pluginCount; i++ {
		if i > 0 {
			pluginsJSON += ","
		}
		pluginsJSON += fmt.Sprintf(`{"name":"slow-plugin-%d","source":{"source":"git-subdir","url":"https://github.com/example/plugins-%d.git","ref":"main","sha":"abc%d"}}`, i, i, i)
	}
	pluginsJSON += `]}`
	if err := os.WriteFile(filepath.Join(mktDir, "marketplace.json"), []byte(pluginsJSON), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(claudeDir, "plugins", "installed_plugins.json"),
		[]byte(`{"version":2,"plugins":{}}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	svc := services.NewUpdateCheckService(cacheRepo, &blockingLsClient{}, claudeDir)
	svc.BatchDeadline = 50 * time.Millisecond

	result, err := svc.RunUpdateCheck(context.Background())
	if err != nil {
		t.Fatalf("RunUpdateCheck: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("status: got %q want %q", result.Status, "ok")
	}
	if len(result.Plugins) != pluginCount {
		t.Errorf("plugin count: got %d want %d - not-started items are missing terminal results", len(result.Plugins), pluginCount)
	}

	timeoutCount := 0
	for _, p := range result.Plugins {
		if p.Error == "" {
			t.Errorf("plugin %q/%q has no error but all items should have failed (deadline or timeout)", p.PluginName, p.MarketplaceName)
		}
		if p.Error == "timeout" {
			timeoutCount++
		}
	}

	// Items beyond the concurrency cap (pluginCount - updateCheckConcurrency) should be "timeout".
	// updateCheckConcurrency is an unexported constant (4); at least 2 items must be timeout.
	if timeoutCount < pluginCount-4 {
		t.Errorf("expected at least %d timeout results (not-started items), got %d", pluginCount-4, timeoutCount)
	}
}

// TestUpdateCheck_HTTPSOnlyFiltered verifies non-https URLs are rejected by the real client.
// Uses GitLsRemoteClient directly (no real network call - rejected before any subprocess).
func TestUpdateCheck_HTTPSOnlyFiltered(t *testing.T) {
	db := openTestDB(t)
	cacheRepo := repositories.NewUpdateCheckCacheRepo(db)

	claudeDir := t.TempDir()
	mktDir := filepath.Join(claudeDir, "plugins", "marketplaces", "bad-market", ".claude-plugin")
	if err := os.MkdirAll(mktDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mktDir, "marketplace.json"), []byte(`{
		"name": "bad-market",
		"plugins": [
			{
				"name": "bad-plugin",
				"source": {
					"source": "git",
					"url": "git://github.com/example/plugins.git",
					"ref": "main"
				}
			}
		]
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	pluginsDir := filepath.Join(claudeDir, "plugins")
	if err := os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), []byte(`{
		"version": 2, "plugins": {}
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Use the real client - HTTPS check fires before any subprocess (no network contact).
	realClient := network.NewGitLsRemoteClient()

	ctx := context.Background()
	svc := services.NewUpdateCheckService(cacheRepo, realClient, claudeDir)
	result, err := svc.RunUpdateCheck(ctx)
	if err != nil {
		t.Fatalf("RunUpdateCheck: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("status: got %q", result.Status)
	}
	// bad-plugin result should have a non-empty error (non_https_scheme_rejected).
	foundBadPlugin := false
	for _, p := range result.Plugins {
		if p.PluginName == "bad-plugin" {
			foundBadPlugin = true
			if p.Error == "" {
				t.Error("bad-plugin with git:// URL should have error set (non_https_scheme_rejected)")
			}
		}
	}
	if !foundBadPlugin {
		t.Error("bad-plugin missing from results")
	}
}

// newAppCheckSvc creates a minimal UpdateCheckService for CheckAppUpdate tests.
// It has no real DB or claudeConfigDir (not needed for app update checks).
func newAppCheckSvc(t *testing.T, srv *httptest.Server) *services.UpdateCheckService {
	t.Helper()
	db := openTestDB(t)
	cacheRepo := repositories.NewUpdateCheckCacheRepo(db)
	svc := services.NewUpdateCheckService(cacheRepo, nil, "")
	svc.AppCheckURL = srv.URL
	svc.HTTPClient = srv.Client()
	return svc
}

// TestCheckAppUpdate_UpToDate verifies the core correctness requirement from FB-004:
// when GitHub returns the same version tag as the running binary, updateAvailable is false.
func TestCheckAppUpdate_UpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"tag_name":"v0.1.2","html_url":"https://github.com/example/releases/tag/v0.1.2"}`)
	}))
	defer srv.Close()

	svc := newAppCheckSvc(t, srv)
	result, err := svc.CheckAppUpdate(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error != nil {
		t.Errorf("expected no error, got %q", *result.Error)
	}
	if result.UpdateAvailable {
		t.Error("expected updateAvailable=false when current version == latest version")
	}
	if result.CurrentVersion != "0.1.2" {
		t.Errorf("currentVersion: got %q, want %q", result.CurrentVersion, "0.1.2")
	}
	if result.LatestVersion == nil || *result.LatestVersion != "0.1.2" {
		t.Errorf("latestVersion: got %v, want 0.1.2", result.LatestVersion)
	}
	if result.ReleaseURL == nil || *result.ReleaseURL == "" {
		t.Error("releaseUrl should be present even when up to date")
	}
}

// TestCheckAppUpdate_Available verifies that a newer GitHub tag sets updateAvailable=true.
func TestCheckAppUpdate_Available(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"tag_name":"v1.0.0","html_url":"https://github.com/example/releases/tag/v1.0.0"}`)
	}))
	defer srv.Close()

	svc := newAppCheckSvc(t, srv)
	result, err := svc.CheckAppUpdate(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error != nil {
		t.Errorf("unexpected error field: %q", *result.Error)
	}
	if !result.UpdateAvailable {
		t.Error("expected updateAvailable=true when latest > current")
	}
	if result.LatestVersion == nil || *result.LatestVersion != "1.0.0" {
		t.Errorf("latestVersion: got %v, want 1.0.0", result.LatestVersion)
	}
}

// TestCheckAppUpdate_NoReleases verifies 404 -> error="no_releases" and updateAvailable=false.
func TestCheckAppUpdate_NoReleases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc := newAppCheckSvc(t, srv)
	result, err := svc.CheckAppUpdate(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error == nil || *result.Error != "no_releases" {
		t.Errorf("error: got %v, want no_releases", result.Error)
	}
	if result.UpdateAvailable {
		t.Error("updateAvailable must be false on error")
	}
}

// TestCheckAppUpdate_HTTPError verifies non-200/non-404 -> error="http_error".
func TestCheckAppUpdate_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := newAppCheckSvc(t, srv)
	result, err := svc.CheckAppUpdate(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error == nil || *result.Error != "http_error" {
		t.Errorf("error: got %v, want http_error", result.Error)
	}
}

// TestCheckAppUpdate_ParseError verifies malformed JSON -> error="parse_error".
func TestCheckAppUpdate_ParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `not valid json {`)
	}))
	defer srv.Close()

	svc := newAppCheckSvc(t, srv)
	result, err := svc.CheckAppUpdate(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error == nil || *result.Error != "parse_error" {
		t.Errorf("error: got %v, want parse_error", result.Error)
	}
}

// TestCheckAppUpdate_NetworkError verifies that a closed server -> error="network_error".
func TestCheckAppUpdate_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	svc := newAppCheckSvc(t, srv)
	srv.Close() // close before calling to simulate network failure

	result, err := svc.CheckAppUpdate(context.Background(), "0.1.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error == nil || *result.Error != "network_error" {
		t.Errorf("error: got %v, want network_error", result.Error)
	}
	if result.UpdateAvailable {
		t.Error("updateAvailable must be false on network_error")
	}
}
