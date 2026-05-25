package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// dashFake implements all six dashboard interfaces in one struct.
type dashFake struct {
	// settings
	settings    *domain.AppSettings
	settingsErr error

	// host
	host    *domain.SkillHostFolder
	hostErr error

	// skill
	skillCount    int
	skillCountErr error

	// project
	projectCount    int
	projectCountErr error

	// install
	installCounts    domain.InstallModeCounts
	installCountsErr error

	// warning
	warnCounts    domain.WarningSeverityCounts
	warnCountsErr error
	warnings      []domain.Warning
	warningsErr   error
	capturedLimit int
}

func (f *dashFake) Get(_ context.Context) (*domain.AppSettings, error) {
	return f.settings, f.settingsErr
}

func (f *dashFake) GetByID(_ context.Context, _ int64) (*domain.SkillHostFolder, error) {
	return f.host, f.hostErr
}

func (f *dashFake) CountByHost(_ context.Context, _ int64) (int, error) {
	return f.skillCount, f.skillCountErr
}

func (f *dashFake) CountActive(_ context.Context) (int, error) {
	return f.projectCount, f.projectCountErr
}

func (f *dashFake) CountByModeActive(_ context.Context) (domain.InstallModeCounts, error) {
	return f.installCounts, f.installCountsErr
}

func (f *dashFake) CountActiveBySeverity(_ context.Context) (domain.WarningSeverityCounts, error) {
	return f.warnCounts, f.warnCountsErr
}

func (f *dashFake) ListActive(_ context.Context, limit int) ([]domain.Warning, error) {
	f.capturedLimit = limit
	return f.warnings, f.warningsErr
}

// helper: build a DashboardService from a single dashFake.
func newDashSvc(f *dashFake) *DashboardService {
	return NewDashboardService(f, f, f, f, f, f)
}

// defaultFake returns a dashFake with sensible defaults (no active host).
func defaultFake() *dashFake {
	return &dashFake{
		settings: &domain.AppSettings{
			ID:                      1,
			ActiveSkillHostFolderID: nil,
			DefaultInstallMode:      "symlink",
			DatabaseVersion:         1,
		},
		projectCount: 3,
		installCounts: domain.InstallModeCounts{Symlink: 5, RsyncCopy: 2, Direct: 1},
		warnCounts:   domain.WarningSeverityCounts{Info: 1, Warning: 2, Error: 0, Blocking: 0},
		warnings: []domain.Warning{
			{ID: 1, Code: "test.warn", Message: "msg", Severity: domain.WarningSeverityWarning, ScopeType: domain.WarningScopeApp},
		},
	}
}

func TestDashboardService_NoActiveHost(t *testing.T) {
	f := defaultFake()
	svc := newDashSvc(f)
	view, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if view.ActiveHost != nil {
		t.Errorf("expected nil ActiveHost, got %+v", view.ActiveHost)
	}
	if view.Summary.Skills != 0 {
		t.Errorf("expected Skills=0, got %d", view.Summary.Skills)
	}
	if view.Summary.Projects != 3 {
		t.Errorf("expected Projects=3, got %d", view.Summary.Projects)
	}
	if view.InstallsByMode.Symlink != 5 {
		t.Errorf("expected Symlink=5, got %d", view.InstallsByMode.Symlink)
	}
	if view.WarningsBySeverity.Warning != 2 {
		t.Errorf("expected WarningsBySeverity.Warning=2, got %d", view.WarningsBySeverity.Warning)
	}
}

func TestDashboardService_ActiveHostPresent(t *testing.T) {
	f := defaultFake()
	hostID := int64(7)
	f.settings.ActiveSkillHostFolderID = &hostID

	scannedAt := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	f.host = &domain.SkillHostFolder{
		ID:            hostID,
		Path:          "/tmp/host",
		SkillsPath:    "/tmp/host/.agents/skills",
		Status:        domain.SkillHostStatusActive,
		LastScannedAt: &scannedAt,
	}
	f.skillCount = 12

	svc := newDashSvc(f)
	view, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if view.ActiveHost == nil {
		t.Fatal("expected non-nil ActiveHost")
	}
	if view.ActiveHost.Path != "/tmp/host" {
		t.Errorf("Path: got %q want /tmp/host", view.ActiveHost.Path)
	}
	if view.Summary.Skills != 12 {
		t.Errorf("Skills: got %d want 12", view.Summary.Skills)
	}
	if view.ActiveHost.LastScannedAt == nil {
		t.Fatal("expected LastScannedAt to be set")
	}
	if *view.ActiveHost.LastScannedAt != "2025-06-15T12:00:00Z" {
		t.Errorf("LastScannedAt: got %q want 2025-06-15T12:00:00Z", *view.ActiveHost.LastScannedAt)
	}
}

func TestDashboardService_HostRowMissing(t *testing.T) {
	f := defaultFake()
	hostID := int64(99999)
	f.settings.ActiveSkillHostFolderID = &hostID
	f.host = nil // GetByID returns nil, nil

	svc := newDashSvc(f)
	view, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if view.ActiveHost != nil {
		t.Errorf("expected nil ActiveHost for missing host row, got %+v", view.ActiveHost)
	}
	if view.Summary.Skills != 0 {
		t.Errorf("expected Skills=0 when host row missing, got %d", view.Summary.Skills)
	}
}

func TestDashboardService_SummaryWarningsEqualsTotal(t *testing.T) {
	f := defaultFake()
	f.warnCounts = domain.WarningSeverityCounts{Info: 3, Warning: 5, Error: 2, Blocking: 1}

	svc := newDashSvc(f)
	view, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if view.Summary.Warnings != view.WarningsBySeverity.Total() {
		t.Errorf("Summary.Warnings=%d != WarningsBySeverity.Total()=%d",
			view.Summary.Warnings, view.WarningsBySeverity.Total())
	}
}

func TestDashboardService_ListActiveCalledWithLimit50(t *testing.T) {
	f := defaultFake()
	svc := newDashSvc(f)
	_, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if f.capturedLimit != 50 {
		t.Errorf("ListActive called with limit=%d, want 50", f.capturedLimit)
	}
}

func TestDashboardService_SettingsError(t *testing.T) {
	f := defaultFake()
	f.settingsErr = errors.New("db connection failed")

	svc := newDashSvc(f)
	view, err := svc.Get(context.Background())
	if view != nil {
		t.Errorf("expected nil view on settings error, got %+v", view)
	}
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *domain.AppError, got %T: %v", err, err)
	}
	if appErr.Code != domain.CodeDatabase {
		t.Errorf("code: got %s want %s", appErr.Code, domain.CodeDatabase)
	}
}
