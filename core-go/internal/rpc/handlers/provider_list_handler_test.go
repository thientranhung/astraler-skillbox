package handlers_test

import (
	"context"
	"testing"

	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/rpc/handlers"
)

type stubProviderRegistrySvc struct {
	entries []domain.ProviderRegistryEntry
	err     error
}

func (s *stubProviderRegistrySvc) List(_ context.Context) ([]domain.ProviderRegistryEntry, error) {
	return s.entries, s.err
}

func makeRegistryEntry(key string, status domain.ProviderStatus, hasGlobal bool) domain.ProviderRegistryEntry {
	iconKey := key
	return domain.ProviderRegistryEntry{
		Definition: domain.ProviderDefinition{
			Key:            key,
			DisplayName:    key,
			ProviderType:   key,
			IconKey:        &iconKey,
			Status:         status,
			HasGlobalLevel: hasGlobal,
		},
		Candidates: []domain.ProviderPathCandidate{
			{RelativePath: "." + key, Scope: "project", Purpose: "detect", Priority: 10, VerificationStatus: "assumed"},
		},
	}
}

func TestProviderListHandler_ReturnsProviders(t *testing.T) {
	svc := &stubProviderRegistrySvc{
		entries: []domain.ProviderRegistryEntry{
			makeRegistryEntry("generic_agents", domain.ProviderStatusSupported, true),
			makeRegistryEntry("claude", domain.ProviderStatusExperimental, true),
		},
	}
	cli := startServer(t, handler.Map{"provider.list": handlers.NewProviderListHandler(svc)})

	var resp struct {
		Providers []struct {
			Key            string `json:"key"`
			IsAvailable    bool   `json:"isAvailable"`
			HasGlobalLevel bool   `json:"hasGlobalLevel"`
		} `json:"providers"`
	}
	if err := cli.CallResult(context.Background(), "provider.list", nil, &resp); err != nil {
		t.Fatalf("provider.list: %v", err)
	}
	if len(resp.Providers) != 2 {
		t.Errorf("provider count: got %d want 2", len(resp.Providers))
	}
	if resp.Providers[0].Key != "generic_agents" {
		t.Errorf("first key: got %q want generic_agents", resp.Providers[0].Key)
	}
	if !resp.Providers[0].IsAvailable {
		t.Error("supported provider should have isAvailable=true")
	}
	if !resp.Providers[0].HasGlobalLevel {
		t.Error("generic_agents should have hasGlobalLevel=true")
	}
}

func TestProviderListHandler_UnsupportedIsAvailable_False(t *testing.T) {
	svc := &stubProviderRegistrySvc{
		entries: []domain.ProviderRegistryEntry{
			makeRegistryEntry("opencode", domain.ProviderStatusUnsupported, false),
		},
	}
	cli := startServer(t, handler.Map{"provider.list": handlers.NewProviderListHandler(svc)})

	var resp struct {
		Providers []struct {
			IsAvailable bool `json:"isAvailable"`
		} `json:"providers"`
	}
	if err := cli.CallResult(context.Background(), "provider.list", nil, &resp); err != nil {
		t.Fatalf("provider.list: %v", err)
	}
	if len(resp.Providers) == 0 {
		t.Fatal("expected at least one provider")
	}
	if resp.Providers[0].IsAvailable {
		t.Error("unsupported provider should have isAvailable=false")
	}
}

func TestProviderListHandler_EmptyProviders(t *testing.T) {
	svc := &stubProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{}}
	cli := startServer(t, handler.Map{"provider.list": handlers.NewProviderListHandler(svc)})

	var resp struct {
		Providers []interface{} `json:"providers"`
	}
	if err := cli.CallResult(context.Background(), "provider.list", nil, &resp); err != nil {
		t.Fatalf("provider.list: %v", err)
	}
	if resp.Providers == nil {
		t.Error("providers must not be nil")
	}
}

func TestProviderListHandler_SourceAlwaysBuiltin(t *testing.T) {
	svc := &stubProviderRegistrySvc{
		entries: []domain.ProviderRegistryEntry{
			makeRegistryEntry("claude", domain.ProviderStatusExperimental, false),
		},
	}
	cli := startServer(t, handler.Map{"provider.list": handlers.NewProviderListHandler(svc)})

	var resp struct {
		Providers []struct {
			Candidates []struct {
				Source string `json:"source"`
			} `json:"candidates"`
		} `json:"providers"`
	}
	if err := cli.CallResult(context.Background(), "provider.list", nil, &resp); err != nil {
		t.Fatalf("provider.list: %v", err)
	}
	if len(resp.Providers) == 0 || len(resp.Providers[0].Candidates) == 0 {
		t.Fatal("expected at least one candidate")
	}
	if resp.Providers[0].Candidates[0].Source != "builtin" {
		t.Errorf("source: got %q want builtin", resp.Providers[0].Candidates[0].Source)
	}
}
