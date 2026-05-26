package providers_test

import (
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

func TestConventionalProviderAdapters_DetectFolders(t *testing.T) {
	cases := []struct {
		name      string
		adapter   providers.ProviderAdapter
		key       string
		rootPath  string
		skillPath string
	}{
		{"codex", providers.NewCodexAdapter(), providers.CodexKey, "/project/.codex", "/project/.codex/skills"},
		{"gemini", providers.NewGeminiAdapter(), providers.GeminiKey, "/project/.gemini", "/project/.gemini/skills"},
		{"antigravity", providers.NewAntigravityCLIAdapter(), providers.AntigravityCLIKey, "/project/.antigravity-cli", "/project/.antigravity-cli/skills"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mfs := newMockFS()
			mfs.setDir(c.rootPath)
			mfs.setDir(c.skillPath)
			mfs.entries[c.skillPath] = []filesystem.ProjectEntry{
				{Name: "skill-a", Path: c.skillPath + "/skill-a", IsDir: true},
			}

			result, err := c.adapter.Detect("/project", mfs)
			if err != nil {
				t.Fatalf("Detect: %v", err)
			}
			if c.adapter.Key() != c.key {
				t.Errorf("Key: got %q want %q", c.adapter.Key(), c.key)
			}
			if !result.Present {
				t.Fatal("Present: want true")
			}
			if result.DetectionStatus != domain.DetectionStatusDetected {
				t.Errorf("DetectionStatus: got %q want detected", result.DetectionStatus)
			}
			if result.DetectedPath != c.rootPath {
				t.Errorf("DetectedPath: got %q want %q", result.DetectedPath, c.rootPath)
			}
			if result.SkillsPath != c.skillPath {
				t.Errorf("SkillsPath: got %q want %q", result.SkillsPath, c.skillPath)
			}
			if len(result.Entries) != 1 || result.Entries[0].Name != "skill-a" {
				t.Errorf("Entries: got %#v", result.Entries)
			}
		})
	}
}

func TestConventionalProviderAdapters_MissingDoesNotWarn(t *testing.T) {
	for _, adapter := range []providers.ProviderAdapter{
		providers.NewCodexAdapter(),
		providers.NewGeminiAdapter(),
		providers.NewAntigravityCLIAdapter(),
	} {
		result, err := adapter.Detect("/project", newMockFS())
		if err != nil {
			t.Fatalf("%s Detect: %v", adapter.Key(), err)
		}
		if result.Present {
			t.Errorf("%s Present: want false", adapter.Key())
		}
		if result.DetectionStatus != domain.DetectionStatusMissing {
			t.Errorf("%s DetectionStatus: got %q want missing", adapter.Key(), result.DetectionStatus)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("%s Warnings: got %d want 0", adapter.Key(), len(result.Warnings))
		}
	}
}
