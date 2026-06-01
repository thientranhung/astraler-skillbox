package services

import (
	"context"
	"os"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// ActiveHostView is embedded in SettingsView.
type ActiveHostView struct {
	HostID        int64
	Path          string
	SkillsPath    string
	Status        domain.SkillHostStatus
	LastScannedAt *string
}

// SettingsView is the view model for settings.get.
type SettingsView struct {
	ActiveSkillHostFolderID *int64
	DefaultInstallMode      string
	DatabaseVersion         int
	ActiveHost              *ActiveHostView
}

// SettingsService returns the app settings view model.
type SettingsService struct {
	appSettingsRepo AppSettingsRepo
	hostRepo        HostRepo
}

func NewSettingsService(appSettingsRepo AppSettingsRepo, hostRepo HostRepo) *SettingsService {
	return &SettingsService{appSettingsRepo: appSettingsRepo, hostRepo: hostRepo}
}

func (s *SettingsService) Get(ctx context.Context) (*SettingsView, error) {
	settings, err := s.appSettingsRepo.Get(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not read settings", err.Error())
	}

	view := &SettingsView{
		ActiveSkillHostFolderID: settings.ActiveSkillHostFolderID,
		DefaultInstallMode:      settings.DefaultInstallMode,
		DatabaseVersion:         settings.DatabaseVersion,
	}

	if settings.ActiveSkillHostFolderID != nil {
		host, err := s.hostRepo.GetByID(ctx, *settings.ActiveSkillHostFolderID)
		if err == nil && host != nil {
			effectiveStatus := host.Status
			if _, statErr := os.Stat(host.Path); os.IsNotExist(statErr) {
				effectiveStatus = domain.SkillHostStatusMissing
			}
			ah := &ActiveHostView{
				HostID:     host.ID,
				Path:       host.Path,
				SkillsPath: host.SkillsPath,
				Status:     effectiveStatus,
			}
			if host.LastScannedAt != nil {
				ts := host.LastScannedAt.UTC().Format("2006-01-02T15:04:05Z")
				ah.LastScannedAt = &ts
			}
			view.ActiveHost = ah
		}
		// If host row is missing, activeHost remains nil (defensive).
	}

	return view, nil
}
