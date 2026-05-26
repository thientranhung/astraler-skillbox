package services

import (
	"context"
	"strconv"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// SkillsLibraryView is the view model returned by skill.list.
type SkillsLibraryView struct {
	HostPath   string
	Skills     []SkillItem
	Totals     SkillTotals
	LastScanAt *string
	Warnings   []WarningItem
}

type SkillItem struct {
	ID                 int64
	Name               string
	RelativePath       string
	Status             domain.SkillStatus
	SourceLabel        *string
	LastScannedAt      *string
	ProjectsUsingCount int
}

type SkillTotals struct {
	Available     int
	Missing       int
	Unreadable    int
	LocalModified int
	Unknown       int
}

type WarningItem struct {
	Code     string
	Message  string
	ScopeRef *string
}

type SkillDetailItem struct {
	ID           int64
	Name         string
	RelativePath string
	AbsolutePath string
	Status       domain.SkillStatus
	SourceLabel  *string
	HostPath     string
	LastScannedAt *string
}

type SkillDetailView struct {
	Skill    SkillDetailItem
	Projects []domain.SkillProjectUsage
}

// SkillLibraryService returns the skills library view model.
type SkillLibraryService struct {
	skillRepo   SkillRepo
	hostRepo    HostRepo
	warningRepo WarningRepo
}

func NewSkillLibraryService(skillRepo SkillRepo, hostRepo HostRepo, warningRepo WarningRepo) *SkillLibraryService {
	return &SkillLibraryService{
		skillRepo:   skillRepo,
		hostRepo:    hostRepo,
		warningRepo: warningRepo,
	}
}

func (s *SkillLibraryService) List(ctx context.Context, hostID int64) (*SkillsLibraryView, error) {
	host, err := s.hostRepo.GetByID(ctx, hostID)
	if err != nil || host == nil {
		return nil, domain.NewValidationError("Host not found", "hostId not in database")
	}

	skills, err := s.skillRepo.ListByHost(ctx, hostID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not list skills", err.Error())
	}

	counts, err := s.skillRepo.CountProjectsPerSkillByHost(ctx, hostID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not count projects per skill", err.Error())
	}

	warnings, err := s.warningRepo.ListByScope(ctx, domain.WarningScopeSkillHostFolder, hostID, false)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not list warnings", err.Error())
	}

	view := &SkillsLibraryView{
		HostPath: host.Path,
	}

	// Convert skills.
	for _, sk := range skills {
		item := SkillItem{
			ID:                 sk.ID,
			Name:               sk.Name,
			RelativePath:       sk.RelativePath,
			Status:             sk.Status,
			ProjectsUsingCount: counts[sk.ID],
		}
		if sk.LastScannedAt != nil {
			ts := sk.LastScannedAt.UTC().Format("2006-01-02T15:04:05Z")
			item.LastScannedAt = &ts
		}
		view.Skills = append(view.Skills, item)

		switch sk.Status {
		case domain.SkillStatusAvailable:
			view.Totals.Available++
		case domain.SkillStatusMissing:
			view.Totals.Missing++
		case domain.SkillStatusUnreadable:
			view.Totals.Unreadable++
		case domain.SkillStatusLocalModified:
			view.Totals.LocalModified++
		default:
			view.Totals.Unknown++
		}
	}

	// Last scan at from host.
	if host.LastScannedAt != nil {
		ts := host.LastScannedAt.UTC().Format("2006-01-02T15:04:05Z")
		view.LastScanAt = &ts
	}

	// Convert warnings.
	for _, w := range warnings {
		item := WarningItem{Code: w.Code, Message: w.Message}
		if w.ScopeID != nil {
			ref := w.ScopeType.String() + ":" + int64ToString(*w.ScopeID)
			item.ScopeRef = &ref
		}
		view.Warnings = append(view.Warnings, item)
	}

	return view, nil
}

func int64ToString(n int64) string {
	return strconv.FormatInt(n, 10)
}

// GetSkillDetail returns metadata and per-project installs for the given skill.
// Returns validation_error if the skill is not found or id <= 0.
func (s *SkillLibraryService) GetSkillDetail(ctx context.Context, skillID int64) (*SkillDetailView, error) {
	if skillID <= 0 {
		return nil, domain.NewValidationError("Skill not found", "skillId must be > 0")
	}

	sk, err := s.skillRepo.GetByID(ctx, skillID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not fetch skill", err.Error())
	}
	if sk == nil {
		return nil, domain.NewValidationError("Skill not found", "no skill with that id")
	}

	host, err := s.hostRepo.GetByID(ctx, sk.SkillHostFolderID)
	if err != nil || host == nil {
		return nil, domain.NewDatabaseError("Could not fetch skill host", "")
	}

	usages, err := s.skillRepo.ProjectsUsingSkill(ctx, skillID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not fetch project usages", err.Error())
	}
	if usages == nil {
		usages = []domain.SkillProjectUsage{}
	}

	detail := SkillDetailItem{
		ID:           sk.ID,
		Name:         sk.Name,
		RelativePath: sk.RelativePath,
		AbsolutePath: sk.AbsolutePath,
		Status:       sk.Status,
		HostPath:     host.Path,
	}
	if sk.LastScannedAt != nil {
		ts := sk.LastScannedAt.UTC().Format("2006-01-02T15:04:05Z")
		detail.LastScannedAt = &ts
	}

	return &SkillDetailView{Skill: detail, Projects: usages}, nil
}
