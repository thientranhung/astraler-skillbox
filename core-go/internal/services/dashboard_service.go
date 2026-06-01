package services

import (
	"context"
	"os"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// Local narrow interfaces — NOT added to interfaces.go to avoid breaking existing mocks.

type dashboardSettingsRepo interface {
	Get(ctx context.Context) (*domain.AppSettings, error)
}

type dashboardHostRepo interface {
	GetByID(ctx context.Context, id int64) (*domain.SkillHostFolder, error)
}

type dashboardSkillRepo interface {
	CountByHost(ctx context.Context, hostID int64) (int, error)
}

type dashboardProjectRepo interface {
	CountActive(ctx context.Context) (int, error)
}

type dashboardInstallRepo interface {
	CountByModeActive(ctx context.Context) (domain.InstallModeCounts, error)
}

type dashboardWarningRepo interface {
	CountActiveBySeverity(ctx context.Context) (domain.WarningSeverityCounts, error)
	ListActive(ctx context.Context, limit int) ([]domain.Warning, error)
}

// View structs — exported, no JSON tags (JSON tags live on handler response structs).

type DashboardActiveHost struct {
	HostID        int64
	Path          string
	SkillsPath    string
	Status        domain.SkillHostStatus
	LastScannedAt *string // pre-formatted UTC string or nil
}

type DashboardSummary struct {
	Skills   int
	Projects int
	Warnings int
}

type DashboardWarningItem struct {
	Code      string
	Message   string
	Severity  domain.WarningSeverity
	ScopeType domain.WarningScopeType
	ScopeID   *int64
	ActionKey *string
}

type DashboardView struct {
	ActiveHost         *DashboardActiveHost
	Summary            DashboardSummary
	InstallsByMode     domain.InstallModeCounts
	WarningsBySeverity domain.WarningSeverityCounts
	Warnings           []DashboardWarningItem
}

const dashboardWarningLimit = 50

type DashboardService struct {
	settings dashboardSettingsRepo
	host     dashboardHostRepo
	skill    dashboardSkillRepo
	project  dashboardProjectRepo
	install  dashboardInstallRepo
	warning  dashboardWarningRepo
}

func NewDashboardService(
	settings dashboardSettingsRepo,
	host dashboardHostRepo,
	skill dashboardSkillRepo,
	project dashboardProjectRepo,
	install dashboardInstallRepo,
	warning dashboardWarningRepo,
) *DashboardService {
	return &DashboardService{
		settings: settings,
		host:     host,
		skill:    skill,
		project:  project,
		install:  install,
		warning:  warning,
	}
}

func (s *DashboardService) Get(ctx context.Context) (*DashboardView, error) {
	// 1. Read settings.
	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not read settings", err.Error())
	}

	var view DashboardView

	// 2. Active host + skill count.
	if settings.ActiveSkillHostFolderID != nil {
		host, err := s.host.GetByID(ctx, *settings.ActiveSkillHostFolderID)
		if err != nil {
			return nil, domain.NewDatabaseError("Could not read skill host folder", err.Error())
		}
		if host != nil {
			effectiveStatus := host.Status
			if _, statErr := os.Stat(host.Path); os.IsNotExist(statErr) {
				effectiveStatus = domain.SkillHostStatusMissing
			}
			ah := &DashboardActiveHost{
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

			skillCount, err := s.skill.CountByHost(ctx, host.ID)
			if err != nil {
				return nil, domain.NewDatabaseError("Could not count skills", err.Error())
			}
			view.Summary.Skills = skillCount
		}
		// If host row is missing, ActiveHost remains nil (defensive).
	}

	// 3. Project count.
	projectCount, err := s.project.CountActive(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not count projects", err.Error())
	}
	view.Summary.Projects = projectCount

	// 4. Installs by mode.
	installCounts, err := s.install.CountByModeActive(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not count installs", err.Error())
	}
	view.InstallsByMode = installCounts

	// 5. Warnings by severity.
	warnCounts, err := s.warning.CountActiveBySeverity(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not count warnings", err.Error())
	}
	view.WarningsBySeverity = warnCounts

	// 6. Summary.Warnings = total warning count.
	view.Summary.Warnings = view.WarningsBySeverity.Total()

	// 7. Warning list.
	warnings, err := s.warning.ListActive(ctx, dashboardWarningLimit)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not list warnings", err.Error())
	}
	items := make([]DashboardWarningItem, 0, len(warnings))
	for _, w := range warnings {
		items = append(items, DashboardWarningItem{
			Code:      w.Code,
			Message:   w.Message,
			Severity:  w.Severity,
			ScopeType: w.ScopeType,
			ScopeID:   w.ScopeID,
			ActionKey: w.ActionKey,
		})
	}
	view.Warnings = items

	return &view, nil
}
