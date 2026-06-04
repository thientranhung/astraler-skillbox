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
	globalRepo             GlobalRepo
	scanRepo               GlobalScanWriter
	settingsRepo           AppSettingsRepo
	hostLister             SkillHostLister
	skillsByHost           SkillsByHostLister
	registry               ProviderRegistry
	fs                     GlobalFilesystem
	runner                 OperationRunner
	pathResolver           GlobalProviderPathResolver
	enabledReader          ProviderEnabledReader // optional; nil → treat all providers as enabled
	providerRegistryLister ProviderRegistryLister // optional; when set, scan iterates all registry providers
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

// WithEnabledReader injects the source of provider enabled state. When set, disabled
// providers are skipped during scan and their stale global entries are cleared.
func (s *GlobalSkillsService) WithEnabledReader(r ProviderEnabledReader) *GlobalSkillsService {
	s.enabledReader = r
	return s
}

// WithProviderRegistryLister injects the provider registry lister. When set, the scan
// iterates all DB registry providers and emits an explicit state for every row — providers
// without a global adapter receive no_global_skills, providers excluded from the resolver
// receive not_configured. Silent omission is a bug; this enforces full coverage.
func (s *GlobalSkillsService) WithProviderRegistryLister(l ProviderRegistryLister) *GlobalSkillsService {
	s.providerRegistryLister = l
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

	// Load provider enabled state once; nil map means all providers are enabled.
	var enabledMap map[string]bool
	if s.enabledReader != nil {
		enabledMap, err = s.enabledReader.EnabledMap(ctx)
		if err != nil {
			return nil, domain.NewDatabaseError("Could not load provider enabled state", err.Error())
		}
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

	if s.providerRegistryLister != nil {
		// Registry-driven: iterate every DB provider entry so that every registry row
		// has an explicit state in global_provider_locations. Silent omission is a bug.
		registryEntries, regErr := s.providerRegistryLister.ListAll(ctx)
		if regErr != nil {
			return nil, domain.NewDatabaseError("Could not load provider registry for global scan", regErr.Error())
		}

		// Build a key → GlobalProviderAdapter map from the adapter registry.
		adapterMap := make(map[string]providers.GlobalProviderAdapter, len(registryEntries))
		for _, a := range s.registry.All() {
			if ga, ok := a.(providers.GlobalProviderAdapter); ok {
				adapterMap[a.Key()] = ga
			}
		}

		for _, regEntry := range registryEntries {
			key := regEntry.Definition.Key
			defID := regEntry.Definition.ID

			if err := s.scanOneProvider(ctx, progress, key, defID, homeDir, enabledMap, globalPathsMap, adapterMap, hostSummaries, summary); err != nil {
				return nil, err
			}
		}
	} else {
		// Legacy adapter-driven iteration: only processes adapters that implement
		// GlobalProviderAdapter. Providers without a global adapter are silently skipped.
		// Use WithProviderRegistryLister to enable full registry coverage.
		for _, adapter := range s.registry.All() {
			ga, ok := adapter.(providers.GlobalProviderAdapter)
			if !ok {
				continue
			}

			defID, _, _, lookupErr := s.globalRepo.ProviderDefByKey(ctx, adapter.Key())
			if lookupErr != nil {
				return nil, domain.NewDatabaseError("Could not look up provider definition for "+adapter.Key(), lookupErr.Error())
			}
			if defID == 0 {
				continue
			}

			adapterMap := map[string]providers.GlobalProviderAdapter{adapter.Key(): ga}
			if err := s.scanOneProvider(ctx, progress, adapter.Key(), defID, homeDir, enabledMap, globalPathsMap, adapterMap, hostSummaries, summary); err != nil {
				return nil, err
			}
		}
	}

	progress("done", 0, 0, "")
	return summary, nil
}

// scanOneProvider handles one provider key during a global scan, writing an explicit
// state to global_provider_locations regardless of global-adapter availability.
func (s *GlobalSkillsService) scanOneProvider(
	ctx context.Context,
	progress operations.ProgressFn,
	key string,
	defID int64,
	homeDir string,
	enabledMap map[string]bool,
	globalPathsMap map[string]providers.GlobalScopePaths,
	adapterMap map[string]providers.GlobalProviderAdapter,
	hostSummaries []HostSummary,
	summary *globalScanSummary,
) error {
	// Disabled provider: commit disabled, clear stale entries.
	if enabled, exists := enabledMap[key]; exists && !enabled {
		return s.scanRepo.CommitGlobalScan(ctx, defID, nil, nil,
			domain.GlobalLocationStatusDisabled,
			[]repositories.GlobalInstallScanResult{}, []domain.Warning{},
			time.Now().UTC())
	}

	ga, hasGlobal := adapterMap[key]
	if !hasGlobal {
		// Provider is in the registry but has no global adapter → no global skills support.
		return s.scanRepo.CommitGlobalScan(ctx, defID, nil, nil,
			domain.GlobalLocationStatusNoGlobalSkills,
			[]repositories.GlobalInstallScanResult{}, []domain.Warning{},
			time.Now().UTC())
	}

	// Resolve effective paths.
	var paths providers.GlobalScopePaths
	if globalPathsMap != nil {
		p, found := globalPathsMap[key]
		if !found {
			// Path resolver excludes this provider: not configured for global skills.
			return s.scanRepo.CommitGlobalScan(ctx, defID, nil, nil,
				domain.GlobalLocationStatusNotConfigured,
				[]repositories.GlobalInstallScanResult{}, []domain.Warning{},
				time.Now().UTC())
		}
		paths = p
	} else {
		paths = ga.DefaultGlobalPaths()
	}

	res, detectErr := ga.DetectGlobal(homeDir, paths, s.fs)
	if detectErr != nil {
		return domain.NewFilesystemError("Could not detect global skills for "+key, detectErr.Error())
	}

	progress("classifying_entries", 0, 0, "")

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
		return domain.NewDatabaseError("Could not commit global scan for "+key, commitErr.Error())
	}

	summary.EntriesFound += len(installs)
	summary.WarningsCreated += len(locWarnings)
	for _, inst := range installs {
		if inst.Warning != nil {
			summary.WarningsCreated++
		}
	}
	return nil
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
