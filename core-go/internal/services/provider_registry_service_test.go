package services

import (
	"context"
	"errors"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// -- mock --

type mockProviderRegistryRepo struct {
	entries []domain.ProviderRegistryEntry
	err     error
}

func (m *mockProviderRegistryRepo) ListAll(_ context.Context) ([]domain.ProviderRegistryEntry, error) {
	return m.entries, m.err
}

func makeTestEntry(key, status string) domain.ProviderRegistryEntry {
	iconKey := key
	return domain.ProviderRegistryEntry{
		Definition: domain.ProviderDefinition{
			Key:         key,
			DisplayName: key,
			ProviderType: key,
			IconKey:     &iconKey,
			Status:      domain.ProviderStatus(status),
		},
		Candidates: []domain.ProviderPathCandidate{
			{RelativePath: "." + key, Scope: "project", Purpose: "detect", Priority: 10, VerificationStatus: "assumed"},
		},
	}
}

// -- tests --

func TestProviderRegistryService_List_ReturnsEntries(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{
		makeTestEntry("generic_agents", "supported"),
		makeTestEntry("claude", "experimental"),
	}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{entries: entries})

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len: got %d want 2", len(got))
	}
	if got[0].Definition.Key != "generic_agents" {
		t.Errorf("first key: got %q want generic_agents", got[0].Definition.Key)
	}
}

func TestProviderRegistryService_List_RepoErrorWrapped(t *testing.T) {
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{err: errors.New("db gone")})

	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *domain.AppError, got %T: %v", err, err)
	}
	if appErr.Code != domain.CodeDatabase {
		t.Errorf("error code: got %q want database_error", appErr.Code)
	}
}

func TestProviderRegistryService_List_EmptyIsNotNil(t *testing.T) {
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{entries: []domain.ProviderRegistryEntry{}})

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
}
