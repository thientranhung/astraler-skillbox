package services

import (
	"context"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/operations"
	"github.com/astraler/skillbox/core-go/internal/providers"
	"github.com/astraler/skillbox/core-go/internal/repositories"
)

// Stable singleton lock for the global scan — used before any global_provider_locations row exists.
const GlobalScanTargetType = "global_scan"
const GlobalScanTargetID int64 = 0

type globalScanSummary struct {
	EntriesFound    int
	WarningsCreated int
}

// GlobalSkillsService handles global skills read operations.
type GlobalSkillsService struct {
	globalRepo   GlobalRepo
	scanRepo     GlobalScanWriter
	settingsRepo AppSettingsRepo
	hostLister   SkillHostLister
	skillsByHost SkillsByHostLister
	registry     ProviderRegistry
	fs           GlobalFilesystem
	runner       OperationRunner
	pathResolver GlobalProviderPathResolver
}

func NewGlobalSkillsService(
	globalRepo GlobalRepo,
	scanRepo GlobalScanWriter,
	settingsRepo AppSettingsRepo,
	hostLister SkillHostLister,
	skillsByHost SkillsByHostLister,
	registry ProviderRegistry,
	fs GlobalFilesystem,
	runner OperationRunner,
) *GlobalSkillsService {
	return &GlobalSkillsService{
		globalRepo:   globalRepo,
		scanRepo:     scanRepo,
		settingsRepo: settingsRepo,
		hostLister:   hostLister,
		skillsByHost: skillsByHost,
		registry:     registry,
		fs:           fs,
		runner:       runner,
	}
}

// WithGlobalPathResolver injects an optional path resolver that overrides builtin paths.
func (s *GlobalSkillsService) WithGlobalPathResolver(r GlobalProviderPathResolver) *GlobalSkillsService {
	s.pathResolver = r
	return s
}

// ScanGlobal starts an async read-only global scan under the stable singleton lock.
// Returns the operationId; returns conflict_error if a scan is already running.
func (s *GlobalSkillsService) ScanGlobal(ctx context.Context) (int64, error) {
	target := operations.Target{Type: GlobalScanTargetType, ID: GlobalScanTargetID}
	opID, err := s.runner.Start(ctx, target, domain.OperationTypeScanGlobalSkills,
		func(opCtx context.Context, progress operations.ProgressFn) (any, error) {
			return s.scanGlobalInternal(opCtx, progress)
		})
	if err != nil {
		if ae, ok := err.(*domain.AppError); ok {
			return 0, ae
		}
		return 0, domain.NewDatabaseError("Could not start global scan", err.Error())
	}
	return opID, nil
}

func (s *GlobalSkillsService) scanGlobalInternal(ctx context.Context, progress operations.ProgressFn) (any, error) {
	progress("reading_global_location", 0, 0, "")

	homeDir, err := s.fs.HomeDir()
	if err != nil {
		return nil, domain.NewFilesystemError("Could not resolve home directory", err.Error())
	}

	// Resolve effective global paths (override ?? builtin).
	var globalPathsMap map[string]providers.GlobalScopePaths
	if s.pathResolver != nil {
		globalPathsMap, err = s.pathResolver.GlobalPaths(ctx)
		if err != nil {
			return nil, domain.NewDatabaseError("Could not resolve global provider paths", err.Error())
		}
	}

	// Build host summaries for classification (active host first).
	hosts, err := s.hostLister.ListAll(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load skill host folders", err.Error())
	}
	hostSummaries, err := s.buildHostSummaries(ctx, hosts)
	if err != nil {
		return nil, err
	}

	summary := &globalScanSummary{}

	// Iterate all registered adapters; only process those that implement GlobalProviderAdapter.
	for _, adapter := range s.registry.All() {
		ga, ok := adapter.(providers.GlobalProviderAdapter)
		if !ok {
			continue
		}

		// Look up provider definition — gating: skip if not seeded.
		defID, _, _, lookupErr := s.globalRepo.ProviderDefByKey(ctx, adapter.Key())
		if lookupErr != nil {
			// Provider not found in DB → skip (not seeded or has_global_level=0).
			continue
		}

		// Resolve effective paths: resolver result ?? adapter defaults.
		var paths providers.GlobalScopePaths
		if globalPathsMap != nil {
			if p, found := globalPathsMap[adapter.Key()]; found {
				paths = p
			} else {
				// Not in resolver map means has_global_level=false for this provider → skip.
				continue
			}
		} else {
			paths = ga.DefaultGlobalPaths()
		}

		res, detectErr := ga.DetectGlobal(homeDir, paths, s.fs)
		if detectErr != nil {
			return nil, domain.NewFilesystemError("Could not detect global skills for "+adapter.Key(), detectErr.Error())
		}

		progress("classifying_entries", 0, 0, "")

		// Classify entries using the existing project-install semantics.
		rescan := "rescan"
		installs := make([]repositories.GlobalInstallScanResult, 0, len(res.Entries))
		for _, entry := range res.Entries {
			c := ClassifyAdapterEntry(entry, hostSummaries)
			inst := repositories.GlobalInstallScanResult{
				SkillID:                   c.SkillID,
				SkillName:                 entry.Name,
				InstallMode:               c.Mode,
				InstallStatus:             c.Status,
				GlobalSkillPath:           entry.Path,
				SourceSkillPath:           c.SourceSkillPath,
				SymlinkTargetPath:         c.SymlinkTargetPath,
				InstalledFromHostFolderID: c.InstalledFromHostFolderID,
			}

			switch c.Status {
			case domain.InstallStatusBrokenSymlink:
				w := domain.Warning{
					ScopeType: domain.WarningScopeGlobalInstall,
					Severity:  domain.WarningSeverityWarning,
					Code:      "broken_symlink",
					Message:   "Global skill " + entry.Name + " has a broken symlink",
					ActionKey: &rescan,
				}
				inst.Warning = &w
			case domain.InstallStatusExternalSymlink:
				w := domain.Warning{
					ScopeType: domain.WarningScopeGlobalInstall,
					Severity:  domain.WarningSeverityWarning,
					Code:      "external_symlink",
					Message:   "Global skill " + entry.Name + " is a symlink to an external location",
					ActionKey: &rescan,
				}
				inst.Warning = &w
			case domain.InstallStatusOldHost:
				w := domain.Warning{
					ScopeType: domain.WarningScopeGlobalInstall,
					Severity:  domain.WarningSeverityWarning,
					Code:      "old_host_symlink",
					Message:   "Global skill " + entry.Name + " is a symlink to an old host folder",
					ActionKey: &rescan,
				}
				inst.Warning = &w
			}

			installs = append(installs, inst)
		}

		// Convert adapter location-scoped warnings to domain.Warning.
		locWarnings := make([]domain.Warning, 0, len(res.Warnings))
		for _, aw := range res.Warnings {
			w := domain.Warning{
				ScopeType: domain.WarningScopeGlobalProviderLocation,
				Severity:  aw.Severity,
				Code:      aw.Code,
				Message:   aw.Message,
				ActionKey: &rescan,
			}
			locWarnings = append(locWarnings, w)
		}

		var pathPtr, skillsPtr *string
		if res.GlobalPath != "" {
			pathPtr = &res.GlobalPath
		}
		if res.GlobalSkillsPath != "" {
			skillsPtr = &res.GlobalSkillsPath
		}

		if commitErr := s.scanRepo.CommitGlobalScan(ctx, defID, pathPtr, skillsPtr,
			res.Status, installs, locWarnings, time.Now().UTC()); commitErr != nil {
			return nil, domain.NewDatabaseError("Could not commit global scan for "+adapter.Key(), commitErr.Error())
		}

		summary.EntriesFound += len(installs)
		summary.WarningsCreated += len(locWarnings)
		for _, inst := range installs {
			if inst.Warning != nil {
				summary.WarningsCreated++
			}
		}
	}

	progress("done", 0, 0, "")
	return summary, nil
}

// buildHostSummaries loads skills for each host, active hosts first.
func (s *GlobalSkillsService) buildHostSummaries(ctx context.Context, hosts []domain.SkillHostFolder) ([]HostSummary, error) {
	summaries := make([]HostSummary, 0, len(hosts))
	for _, active := range []bool{true, false} {
		for _, h := range hosts {
			isActive := h.Status == domain.SkillHostStatusActive
			if isActive != active {
				continue
			}
			skills, err := s.skillsByHost.ListByHost(ctx, h.ID)
			if err != nil {
				return nil, domain.NewDatabaseError("Could not load skills for host", err.Error())
			}
			summaries = append(summaries, HostSummary{
				ID:         h.ID,
				SkillsPath: h.SkillsPath,
				IsActive:   isActive,
				Skills:     skills,
			})
		}
	}
	return summaries, nil
}

// ListGlobal returns the persisted global location views (read-only).
func (s *GlobalSkillsService) ListGlobal(ctx context.Context) ([]domain.GlobalLocationView, error) {
	locs, err := s.globalRepo.ListForView(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not list global skills", err.Error())
	}
	return locs, nil
}
