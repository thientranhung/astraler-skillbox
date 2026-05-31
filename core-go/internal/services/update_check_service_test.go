package services_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

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

// TestUpdateCheck_HTTPSOnlyFiltered verifies non-https URLs are rejected by the real client.
// Uses GitLsRemoteClient directly (no real network call — rejected before any subprocess).
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

	// Use the real client — HTTPS check fires before any subprocess (no network contact).
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
