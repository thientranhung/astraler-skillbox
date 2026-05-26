package repositories

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func TestProviderDefinitionRepo_ListAll_ReturnsAllBuiltins(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProviderDefinitionRepo(db)
	ctx := context.Background()

	entries, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	wantKeys := []string{"generic_agents", "claude", "codex", "gemini", "antigravity_cli", "opencode"}
	if len(entries) != len(wantKeys) {
		t.Errorf("entry count: got %d want %d", len(entries), len(wantKeys))
	}

	byKey := make(map[string]domain.ProviderRegistryEntry, len(entries))
	for _, e := range entries {
		byKey[e.Definition.Key] = e
	}

	for _, k := range wantKeys {
		if _, ok := byKey[k]; !ok {
			t.Errorf("provider %q missing from ListAll", k)
		}
	}
}

func TestProviderDefinitionRepo_ListAll_IconKeySeeded(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProviderDefinitionRepo(db)
	ctx := context.Background()

	entries, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	byKey := make(map[string]domain.ProviderRegistryEntry, len(entries))
	for _, e := range entries {
		byKey[e.Definition.Key] = e
	}

	cases := []struct {
		key     string
		iconKey string
	}{
		{"claude", "claude"},
		{"codex", "codex"},
		{"gemini", "gemini"},
		{"antigravity_cli", "antigravity"},
		{"opencode", "opencode"},
	}
	for _, c := range cases {
		e, ok := byKey[c.key]
		if !ok {
			t.Errorf("provider %q not found", c.key)
			continue
		}
		if e.Definition.IconKey == nil || *e.Definition.IconKey != c.iconKey {
			got := "<nil>"
			if e.Definition.IconKey != nil {
				got = *e.Definition.IconKey
			}
			t.Errorf("%s icon_key: got %q want %q", c.key, got, c.iconKey)
		}
	}
}

func TestProviderDefinitionRepo_ListAll_CandidatesGrouped(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProviderDefinitionRepo(db)
	ctx := context.Background()

	entries, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	byKey := make(map[string]domain.ProviderRegistryEntry, len(entries))
	for _, e := range entries {
		byKey[e.Definition.Key] = e
	}

	ga := byKey["generic_agents"]
	if len(ga.Candidates) < 4 {
		t.Errorf("generic_agents should have at least 4 candidates (2 project + 2 global), got %d", len(ga.Candidates))
	}

	var projectDetect, globalSkills int
	for _, c := range ga.Candidates {
		if c.Scope == "project" && c.Purpose == "detect" {
			projectDetect++
		}
		if c.Scope == "global" && c.Purpose == "skills" {
			globalSkills++
		}
	}
	if projectDetect == 0 {
		t.Error("generic_agents: no project detect candidate found")
	}
	if globalSkills == 0 {
		t.Error("generic_agents: no global skills candidate found")
	}
}

func TestProviderDefinitionRepo_ListAll_EmptyCandidatesNeverNil(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProviderDefinitionRepo(db)
	ctx := context.Background()

	entries, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	for _, e := range entries {
		if e.Candidates == nil {
			t.Errorf("provider %q: Candidates must not be nil (should be empty slice)", e.Definition.Key)
		}
	}
}
