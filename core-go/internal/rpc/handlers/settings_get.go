package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/services"
)

type settingsService interface {
	Get(ctx context.Context) (*services.SettingsView, error)
}

type settingsActiveHost struct {
	HostID        int64   `json:"hostId"`
	Path          string  `json:"path"`
	SkillsPath    string  `json:"skillsPath"`
	Status        string  `json:"status"`
	LastScannedAt *string `json:"lastScannedAt"`
}

type settingsGetResponse struct {
	ActiveSkillHostFolderID *int64              `json:"activeSkillHostFolderId"`
	DefaultInstallMode      string              `json:"defaultInstallMode"`
	DatabaseVersion         int                 `json:"databaseVersion"`
	ActiveHost              *settingsActiveHost `json:"activeHost"`
}

func NewSettingsGetHandler(svc settingsService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		view, err := svc.Get(ctx)
		if err != nil {
			if ae, ok := err.(*domain.AppError); ok {
				return nil, ae
			}
			return nil, domain.NewDatabaseError("Could not read settings", err.Error())
		}

		resp := settingsGetResponse{
			ActiveSkillHostFolderID: view.ActiveSkillHostFolderID,
			DefaultInstallMode:      view.DefaultInstallMode,
			DatabaseVersion:         view.DatabaseVersion,
		}

		if view.ActiveHost != nil {
			resp.ActiveHost = &settingsActiveHost{
				HostID:        view.ActiveHost.HostID,
				Path:          view.ActiveHost.Path,
				SkillsPath:    view.ActiveHost.SkillsPath,
				Status:        string(view.ActiveHost.Status),
				LastScannedAt: view.ActiveHost.LastScannedAt,
			}
		}

		return resp, nil
	})
}
