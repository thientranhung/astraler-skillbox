package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/network"
	"github.com/astraler/skillbox/core-go/internal/providers"
	"github.com/astraler/skillbox/core-go/internal/repositories"
)

const (
	updateCheckConcurrency = 4
	updateCheckBatchDeadline = 30 * time.Second
	updateCheckHostFailThreshold = 3
)

type UpdateCheckService struct {
	cacheRepo       *repositories.UpdateCheckCacheRepo
	client          network.UpdateCheckClient
	claudeConfigDir string
	// BatchDeadline overrides updateCheckBatchDeadline when non-zero; set in tests.
	BatchDeadline time.Duration
}

func NewUpdateCheckService(
	cacheRepo *repositories.UpdateCheckCacheRepo,
	client network.UpdateCheckClient,
	claudeConfigDir string,
) *UpdateCheckService {
	return &UpdateCheckService{
		cacheRepo:       cacheRepo,
		client:          client,
		claudeConfigDir: claudeConfigDir,
	}
}

// RunResult is the full response from RunUpdateCheck.
type RunResult struct {
	Status  string                          // "ok" | "git_not_found" | "error"
	Plugins []domain.UpdateCheckPluginResult
}

// RunUpdateCheck performs the manual update check. Always-on (ADR-0002): there is
// no opt-in gate. Network contact only happens when the user triggers this.
func (s *UpdateCheckService) RunUpdateCheck(ctx context.Context) (RunResult, error) {
	// Larry-3: git-not-found check before any work.
	if _, err := exec.LookPath("git"); err != nil {
		return RunResult{Status: "git_not_found"}, nil
	}

	// Derive allowlist from disk on every call (ADR §11 / Larry-1 allowlist-from-disk).
	sources := providers.ScanMarketplaceSources(s.claudeConfigDir)

	// Read installed_plugins.json for gitCommitSha values.
	installPath := filepath.Join(s.claudeConfigDir, "plugins", "installed_plugins.json")
	installScan := providers.ScanClaudeInstalledPluginsFile(installPath, s.claudeConfigDir)
	shaMap := providers.BuildSHAMap(installScan)

	// Build the list of remotes to query (only HTTPS git sources with installed SHA).
	type workItem struct {
		providerKey     string
		pluginName      string
		marketplaceName string
		sourceURL       string
		sourceRef       string
		installedSHA    string
		installedVersion string
	}

	var items []workItem
	for key, src := range sources {
		if src.URL == "" {
			continue
		}
		// Parse "pluginName@marketplaceName" key
		pluginName, marketplaceName := src.PluginName, ""
		for i := len(key) - 1; i >= 0; i-- {
			if key[i] == '@' {
				pluginName = key[:i]
				marketplaceName = key[i+1:]
				break
			}
		}
		if marketplaceName == "" {
			continue
		}

		ref := src.Ref
		if ref == "" {
			ref = "HEAD"
		}

		var installedSHA, installedVersion string
		if sha := shaMap[key]; sha != nil {
			installedSHA = *sha
		}

		items = append(items, workItem{
			providerKey:     "claude",
			pluginName:      pluginName,
			marketplaceName: marketplaceName,
			sourceURL:       src.URL,
			sourceRef:       ref,
			installedSHA:    installedSHA,
			installedVersion: installedVersion,
		})
	}

	// Run with concurrency cap + batch deadline.
	deadline := s.BatchDeadline
	if deadline == 0 {
		deadline = updateCheckBatchDeadline
	}
	batchCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	sem := semaphore.NewWeighted(updateCheckConcurrency)
	hostFails := make(map[string]int)
	var mu sync.Mutex

	results := make([]domain.UpdateCheckPluginResult, 0, len(items))
	resCh := make(chan domain.UpdateCheckPluginResult, len(items))

	var wg sync.WaitGroup
	for i, item := range items {
		item := item
		if err := sem.Acquire(batchCtx, 1); err != nil {
			// Emit a terminal timeout result for every work item that never started.
			for _, ti := range items[i:] {
				resCh <- domain.UpdateCheckPluginResult{
					ProviderKey:     ti.providerKey,
					PluginName:      ti.pluginName,
					MarketplaceName: ti.marketplaceName,
					Error:           "timeout",
				}
			}
			break
		}
		wg.Add(1)
		go func() {
			defer sem.Release(1)
			defer wg.Done()

			host := hostFromURL(item.sourceURL)
			mu.Lock()
			fails := hostFails[host]
			mu.Unlock()
			if fails >= updateCheckHostFailThreshold {
				resCh <- domain.UpdateCheckPluginResult{
					ProviderKey:     item.providerKey,
					PluginName:      item.pluginName,
					MarketplaceName: item.marketplaceName,
					Error:           "host_backoff",
				}
				return
			}

			lsResult := s.client.LsRemote(batchCtx, item.sourceURL, item.sourceRef)

			var updateAvailable *bool
			if lsResult.Error == "" && item.installedSHA != "" {
				ua := item.installedSHA != lsResult.RemoteSHA
				updateAvailable = &ua
			}

			now := time.Now().UTC()
			nowStr := now.Format(time.RFC3339)
			cacheEntry := domain.UpdateCheckCacheEntry{
				ProviderKey:      item.providerKey,
				PluginName:       item.pluginName,
				MarketplaceName:  item.marketplaceName,
				SourceURL:        item.sourceURL,
				SourceRef:        item.sourceRef,
				InstalledSHA:     item.installedSHA,
				InstalledVersion: item.installedVersion,
				RemoteSHA:        lsResult.RemoteSHA,
				UpdateAvailable:  updateAvailable,
				CheckedAt:        now,
				Error:            lsResult.Error,
			}
			_ = s.cacheRepo.Upsert(batchCtx, cacheEntry)

			if lsResult.Error != "" {
				mu.Lock()
				hostFails[host]++
				mu.Unlock()
			}

			res := domain.UpdateCheckPluginResult{
				ProviderKey:     item.providerKey,
				PluginName:      item.pluginName,
				MarketplaceName: item.marketplaceName,
				UpdateAvailable: updateAvailable,
				LastCheckedAt:   &nowStr,
				Error:           lsResult.Error,
			}
			resCh <- res
		}()
	}

	wg.Wait()
	close(resCh)
	for r := range resCh {
		results = append(results, r)
	}

	return RunResult{Status: "ok", Plugins: results}, nil
}

// GetCachedResults returns the current cache, used by providerPlugin.list for optional fields.
func (s *UpdateCheckService) GetCachedResults(ctx context.Context) (map[string]*domain.UpdateCheckCacheEntry, error) {
	entries, err := s.cacheRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*domain.UpdateCheckCacheEntry, len(entries))
	for i := range entries {
		e := entries[i]
		key := e.ProviderKey + "/" + e.PluginName + "@" + e.MarketplaceName
		out[key] = &e
	}
	return out, nil
}

func hostFromURL(rawURL string) string {
	// Best-effort: extract host without url.Parse overhead in hot path.
	// Format: "https://<host>/..."
	s := rawURL
	if len(s) > 8 && s[:8] == "https://" {
		s = s[8:]
	}
	if i := indexOf(s, '/'); i >= 0 {
		return s[:i]
	}
	return s
}

func indexOf(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// ClaudeConfigDirFromHomeDir returns the default Claude config dir (~/.claude).
func ClaudeConfigDirFromHomeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

const (
	githubReleasesURL  = "https://api.github.com/repos/thientranhung/astraler-skillbox/releases/latest"
	appCheckHTTPTimeout = 8 * time.Second
	appCheckBodyLimit  = 64 * 1024
)

// AppCheckUpdateResult is returned by CheckAppUpdate.
type AppCheckUpdateResult struct {
	CurrentVersion  string
	LatestVersion   *string
	UpdateAvailable bool
	ReleaseURL      *string
	Error           *string // "network_error" | "no_releases" | "http_error" | "parse_error"
}

// CheckAppUpdate fetches the latest GitHub release and compares it with currentVersion.
// Always runs — no opt-in gate. App version check is standard behavior.
func (s *UpdateCheckService) CheckAppUpdate(ctx context.Context, currentVersion string) (AppCheckUpdateResult, error) {
	httpCtx, cancel := context.WithTimeout(ctx, appCheckHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, githubReleasesURL, nil)
	if err != nil {
		errStr := "network_error"
		return AppCheckUpdateResult{CurrentVersion: currentVersion, Error: &errStr}, nil
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Skillbox/"+currentVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		errStr := "network_error"
		return AppCheckUpdateResult{CurrentVersion: currentVersion, Error: &errStr}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		errStr := "no_releases"
		return AppCheckUpdateResult{CurrentVersion: currentVersion, Error: &errStr}, nil
	}
	if resp.StatusCode != http.StatusOK {
		errStr := "http_error"
		return AppCheckUpdateResult{CurrentVersion: currentVersion, Error: &errStr}, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, appCheckBodyLimit))
	if err != nil {
		errStr := "network_error"
		return AppCheckUpdateResult{CurrentVersion: currentVersion, Error: &errStr}, nil
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		errStr := "parse_error"
		return AppCheckUpdateResult{CurrentVersion: currentVersion, Error: &errStr}, nil
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	releaseURL := release.HTMLURL
	updateAvailable := latestVersion != currentVersion

	return AppCheckUpdateResult{
		CurrentVersion:  currentVersion,
		LatestVersion:   &latestVersion,
		UpdateAvailable: updateAvailable,
		ReleaseURL:      &releaseURL,
	}, nil
}
