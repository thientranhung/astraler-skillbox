package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/services"
)

type projectListService interface {
	ListProjects(ctx context.Context) ([]services.ProjectListItem, error)
}

type projectListProviderSummary struct {
	Key             string `json:"key"`
	DisplayName     string `json:"displayName"`
	ProviderStatus  string `json:"providerStatus"`
	DetectionStatus string `json:"detectionStatus"`
}

type projectListItem struct {
	ID            int64                        `json:"id"`
	Name          string                       `json:"name"`
	Path          string                       `json:"path"`
	Status        string                       `json:"status"`
	Providers     []projectListProviderSummary `json:"providers"`
	SkillCount    int                          `json:"skillCount"`
	WarningCount  int                          `json:"warningCount"`
	LastScannedAt *string                      `json:"lastScannedAt"`
}

type projectListResponse struct {
	Projects []projectListItem `json:"projects"`
}

func NewProjectListHandler(svc projectListService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		items, err := svc.ListProjects(ctx)
		if err != nil {
			return nil, wrapError(err)
		}

		resp := projectListResponse{
			Projects: make([]projectListItem, 0, len(items)),
		}
		for _, item := range items {
			providers := make([]projectListProviderSummary, 0, len(item.Providers))
			for _, p := range item.Providers {
				providers = append(providers, projectListProviderSummary{
					Key:             p.ProviderKey,
					DisplayName:     p.ProviderDisplayName,
					ProviderStatus:  string(p.ProviderStatus),
					DetectionStatus: string(p.DetectionStatus),
				})
			}
			resp.Projects = append(resp.Projects, projectListItem{
				ID:            item.ID,
				Name:          item.Name,
				Path:          item.Path,
				Status:        string(item.Status),
				Providers:     providers,
				SkillCount:    item.SkillCount,
				WarningCount:  item.WarningCount,
				LastScannedAt: formatTimePtr(item.LastScannedAt),
			})
		}

		return resp, nil
	})
}
