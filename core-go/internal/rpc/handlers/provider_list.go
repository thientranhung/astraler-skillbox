package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type providerRegistryService interface {
	List(ctx context.Context) ([]domain.ProviderRegistryEntry, error)
}

type providerListPathCandidate struct {
	RelativePath       string `json:"relativePath"`
	Scope              string `json:"scope"`
	Purpose            string `json:"purpose"`
	Priority           int    `json:"priority"`
	Source             string `json:"source"`
	VerificationStatus string `json:"verificationStatus"`
}

type providerListProvider struct {
	Key                string                      `json:"key"`
	DisplayName        string                      `json:"displayName"`
	ProviderType       string                      `json:"providerType"`
	IconKey            *string                     `json:"iconKey"`
	Status             string                      `json:"status"`
	IsAvailable        bool                        `json:"isAvailable"`
	IsEnabled          bool                        `json:"isEnabled"`
	CanToggle          bool                        `json:"canToggle"`
	CanCreateStructure bool                        `json:"canCreateStructure"`
	HasGlobalLevel     bool                        `json:"hasGlobalLevel"`
	Candidates         []providerListPathCandidate `json:"candidates"`
}

type providerListResponse struct {
	Providers []providerListProvider `json:"providers"`
}

func NewProviderListHandler(svc providerRegistryService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		entries, err := svc.List(ctx)
		if err != nil {
			return nil, wrapError(err)
		}

		providers := make([]providerListProvider, len(entries))
		for i, e := range entries {
			d := e.Definition
			candidates := make([]providerListPathCandidate, len(e.Candidates))
			for j, c := range e.Candidates {
				source := c.Source
				if source == "" {
					source = "builtin"
				}
				candidates[j] = providerListPathCandidate{
					RelativePath:       c.RelativePath,
					Scope:              c.Scope,
					Purpose:            c.Purpose,
					Priority:           c.Priority,
					Source:             source,
					VerificationStatus: c.VerificationStatus,
				}
			}
			providers[i] = providerListProvider{
				Key:                d.Key,
				DisplayName:        d.DisplayName,
				ProviderType:       d.ProviderType,
				IconKey:            d.IconKey,
				Status:             string(d.Status),
				IsAvailable:        deriveIsAvailable(d.Status),
				IsEnabled:          e.IsEnabled,
				CanToggle:          e.CanToggle,
				CanCreateStructure: d.CanCreateStructure,
				HasGlobalLevel:     d.HasGlobalLevel,
				Candidates:         candidates,
			}
		}

		return providerListResponse{Providers: providers}, nil
	})
}

// deriveIsAvailable returns true for supported and experimental providers.
// This reflects Skillbox support status, not user preference — override storage is deferred.
func deriveIsAvailable(status domain.ProviderStatus) bool {
	return status == domain.ProviderStatusSupported || status == domain.ProviderStatusExperimental
}
