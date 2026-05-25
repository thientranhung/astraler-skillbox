package repositories

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func TestProviderDefinitionRepo_GetByKey_Seeded(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProviderDefinitionRepo(db)
	ctx := context.Background()

	pd, err := repo.GetByKey(ctx, "generic_agents")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if pd == nil {
		t.Fatal("expected generic_agents definition, got nil")
	}
	if pd.Key != "generic_agents" {
		t.Errorf("key: got %q want generic_agents", pd.Key)
	}
	if pd.Status != domain.ProviderStatusSupported {
		t.Errorf("status: got %q want supported", pd.Status)
	}
	if pd.DisplayName == "" {
		t.Error("display_name should not be empty")
	}
}

func TestProviderDefinitionRepo_GetByKey_Missing(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProviderDefinitionRepo(db)

	pd, err := repo.GetByKey(context.Background(), "nonexistent_provider")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if pd != nil {
		t.Errorf("expected nil for missing key, got %v", pd)
	}
}
