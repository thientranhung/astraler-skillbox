package services

import (
	"context"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// ProviderRegistryService returns the read-only provider registry for the UI.
type ProviderRegistryService struct {
	repo ProviderRegistryRepo
}

func NewProviderRegistryService(repo ProviderRegistryRepo) *ProviderRegistryService {
	return &ProviderRegistryService{repo: repo}
}

func (s *ProviderRegistryService) List(ctx context.Context) ([]domain.ProviderRegistryEntry, error) {
	entries, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load provider registry", err.Error())
	}
	return entries, nil
}
