package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// MarketplacePluginSource holds the parsed source section of a marketplace.json plugin entry.
type MarketplacePluginSource struct {
	PluginName string
	SourceType string  // "git-subdir" | "url" | "local" | "unknown"
	URL        string  // HTTPS URL for git-subdir and url types; empty for local
	Ref        string  // tag or branch (may be empty for url type)
	SHA        string  // marketplace-pinned SHA (may differ from installed)
}

// ReadClaudeMarketplacePluginSources reads a single marketplace.json file and returns
// source metadata for each plugin. Only entries with object-typed sources are included;
// string/local sources are skipped.
// allowedDir must contain marketplacePath to satisfy path-confinement.
func ReadClaudeMarketplacePluginSources(marketplacePath, allowedDir string) ([]MarketplacePluginSource, error) {
	cleanFile := filepath.Clean(marketplacePath)
	cleanDir := filepath.Clean(allowedDir)
	if cleanFile != cleanDir && !strings.HasPrefix(cleanFile, cleanDir+string(os.PathSeparator)) {
		return nil, nil // path escape — skip silently
	}

	fi, err := os.Lstat(cleanFile)
	if err != nil || fi.Mode()&os.ModeSymlink != 0 {
		return nil, nil
	}
	if fi.Size() > ClaudeSettingsMaxFileSize {
		return nil, nil
	}

	data, err := os.ReadFile(cleanFile)
	if err != nil {
		return nil, nil
	}

	var doc struct {
		Plugins []json.RawMessage `json:"plugins"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, nil
	}

	out := make([]MarketplacePluginSource, 0, len(doc.Plugins))
	for _, raw := range doc.Plugins {
		var entry struct {
			Name   string          `json:"name"`
			Source json.RawMessage `json:"source"`
		}
		if err := json.Unmarshal(raw, &entry); err != nil || entry.Name == "" {
			continue
		}

		// source may be a string (local path) or an object
		var srcObj struct {
			Source string `json:"source"`
			URL    string `json:"url"`
			Ref    string `json:"ref"`
			SHA    string `json:"sha"`
		}
		if err := json.Unmarshal(entry.Source, &srcObj); err != nil {
			// string source or malformed — skip
			continue
		}

		src := MarketplacePluginSource{
			PluginName: entry.Name,
			SourceType: srcObj.Source,
			URL:        srcObj.URL,
			Ref:        srcObj.Ref,
			SHA:        srcObj.SHA,
		}
		if src.URL == "" {
			continue
		}
		out = append(out, src)
	}
	return out, nil
}

// ScanMarketplaceSources walks the Claude marketplaces directory and returns all plugin sources.
// claudeConfigDir is typically ~/.claude; all marketplace paths are confined within it.
func ScanMarketplaceSources(claudeConfigDir string) map[string]MarketplacePluginSource {
	out := make(map[string]MarketplacePluginSource)
	marketplacesDir := filepath.Join(claudeConfigDir, "plugins", "marketplaces")

	entries, err := os.ReadDir(marketplacesDir)
	if err != nil {
		return out
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		marketplaceName := e.Name()
		mPath := filepath.Join(marketplacesDir, marketplaceName, ".claude-plugin", "marketplace.json")
		sources, _ := ReadClaudeMarketplacePluginSources(mPath, claudeConfigDir)
		for _, s := range sources {
			key := s.PluginName + "@" + marketplaceName
			out[key] = s
		}
	}
	return out
}
