package providers_test

import (
	"reflect"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/providers"
)

func TestInstallTargets(t *testing.T) {
	targets := providers.InstallTargets()
	if len(targets) != 2 {
		t.Fatalf("target count: got %d want 2", len(targets))
	}

	shared := targets[0]
	if shared.ID != "shared_agents" {
		t.Errorf("shared ID: got %q", shared.ID)
	}
	if shared.ProviderKey != providers.GenericAgentsKey {
		t.Errorf("shared provider key: got %q", shared.ProviderKey)
	}
	if shared.DisplayName != "Shared Agent Skills" {
		t.Errorf("shared display name: got %q", shared.DisplayName)
	}
	if shared.RelativeSkillsPath != providers.GenericAgentsSkillsPath {
		t.Errorf("shared path: got %q", shared.RelativeSkillsPath)
	}
	if !reflect.DeepEqual(shared.CompatibleLabels, []string{"Codex", "Antigravity", "compatible agents"}) {
		t.Errorf("shared compatible labels: got %#v", shared.CompatibleLabels)
	}

	claude := targets[1]
	if claude.ID != providers.ClaudeKey {
		t.Errorf("claude ID: got %q", claude.ID)
	}
	if claude.ProviderKey != providers.ClaudeKey {
		t.Errorf("claude provider key: got %q", claude.ProviderKey)
	}
	if claude.DisplayName != "Claude" {
		t.Errorf("claude display name: got %q", claude.DisplayName)
	}
	if claude.RelativeSkillsPath != providers.ClaudeSkillsPath {
		t.Errorf("claude path: got %q", claude.RelativeSkillsPath)
	}
}

func TestInstallTargetByProviderKey(t *testing.T) {
	target, ok := providers.InstallTargetByProviderKey(providers.GenericAgentsKey)
	if !ok {
		t.Fatal("expected target for generic_agents")
	}
	if target.ID != "shared_agents" {
		t.Errorf("target ID: got %q want shared_agents", target.ID)
	}

	if _, ok := providers.InstallTargetByProviderKey("codex"); ok {
		t.Fatal("codex must not be exposed as a detected provider target in Slice 2E")
	}
}
