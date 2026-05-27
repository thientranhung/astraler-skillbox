package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/operations"
	"github.com/astraler/skillbox/core-go/internal/providers"
	"github.com/astraler/skillbox/core-go/internal/repositories"
)

type pluginDefRepo interface {
	GetByKey(ctx context.Context, key string) (*domain.ProviderDefinition, error)
}

type pluginProjectRepo interface {
	GetByID(ctx context.Context, id int64) (*domain.Project, error)
}

// ProviderPluginService handles scanning and listing provider plugin declarations.
type ProviderPluginService struct {
	repo         *repositories.ProviderPluginRepo
	pdRepo       pluginDefRepo
	projRepo     pluginProjectRepo
	runner       OperationRunner
	pluginWriter func(filePath, allowedDir, pluginName, marketplaceName string, enabled bool) error
}

func NewProviderPluginService(
	repo *repositories.ProviderPluginRepo,
	pdRepo pluginDefRepo,
	projRepo pluginProjectRepo,
	runner OperationRunner,
) *ProviderPluginService {
	return &ProviderPluginService{
		repo:         repo,
		pdRepo:       pdRepo,
		projRepo:     projRepo,
		runner:       runner,
		pluginWriter: providers.WriteJSONPluginEnabled,
	}
}

// ScanGlobal starts an async scan of configured provider user layers.
// Returns the operation ID immediately.
func (s *ProviderPluginService) ScanGlobal(ctx context.Context) (int64, error) {
	defs, err := s.pluginProviderDefs(ctx)
	if err != nil {
		return 0, err
	}
	target := operations.Target{Type: "provider_plugin_global", ID: 0}
	opID, err := s.runner.Start(ctx, target, domain.OperationTypeScan,
		func(opCtx context.Context, progress operations.ProgressFn) (any, error) {
			return nil, s.scanGlobalInternal(opCtx, defs, progress)
		})
	if err != nil {
		if _, ok := err.(*domain.AppError); ok {
			return 0, err
		}
		return 0, domain.NewDatabaseError("Could not start plugin global scan operation", err.Error())
	}
	return opID, nil
}

// ScanProject starts an async scan of the project and local settings layers for a project.
// Returns the operation ID immediately.
func (s *ProviderPluginService) ScanProject(ctx context.Context, projectID int64) (int64, error) {
	project, err := s.projRepo.GetByID(ctx, projectID)
	if err != nil {
		return 0, domain.NewDatabaseError("Could not fetch project", err.Error())
	}
	if project == nil {
		return 0, domain.NewValidationError("Project not found", fmt.Sprintf("projectId %d does not exist", projectID))
	}
	defs, err := s.pluginProviderDefs(ctx)
	if err != nil {
		return 0, err
	}
	target := operations.Target{Type: "provider_plugin_project", ID: projectID}
	opID, err := s.runner.Start(ctx, target, domain.OperationTypeScan,
		func(opCtx context.Context, progress operations.ProgressFn) (any, error) {
			return nil, s.scanProjectInternal(opCtx, project, defs, progress)
		})
	if err != nil {
		if _, ok := err.(*domain.AppError); ok {
			return 0, err
		}
		return 0, domain.NewDatabaseError("Could not start plugin project scan operation", err.Error())
	}
	return opID, nil
}

// List returns the current global plugin view and per-project plugin views from persisted scan data.
func (s *ProviderPluginService) List(ctx context.Context) (domain.GlobalPluginView, []domain.ProjectPluginView, error) {
	globals, projects, err := s.ListAll(ctx)
	if err != nil {
		return domain.GlobalPluginView{}, nil, err
	}
	if len(globals) == 0 {
		homeDir, _ := os.UserHomeDir()
		return domain.GlobalPluginView{
			ProviderKey:       "claude",
			UserLayerPath:     filepath.Join(homeDir, ".claude", "settings.json"),
			ManagedOutOfScope: true,
		}, projects, nil
	}
	return globals[0], projects, nil
}

// ListAll returns current global plugin views and per-project plugin views for all plugin-capable providers.
func (s *ProviderPluginService) ListAll(ctx context.Context) ([]domain.GlobalPluginView, []domain.ProjectPluginView, error) {
	defs, err := s.pluginProviderDefsAllowMissing(ctx)
	if err != nil {
		return nil, nil, err
	}

	var globals []domain.GlobalPluginView
	var allProjects []domain.ProjectPluginView
	for _, def := range defs {
		global, projects, err := s.listProvider(ctx, def)
		if err != nil {
			return nil, nil, err
		}
		globals = append(globals, global)
		allProjects = append(allProjects, projects...)
	}
	return globals, allProjects, nil
}

func (s *ProviderPluginService) listProvider(ctx context.Context, def pluginProviderDef) (domain.GlobalPluginView, []domain.ProjectPluginView, error) {
	scans, err := s.repo.ListLayerScansForProvider(ctx, def.Provider.ID)
	if err != nil {
		return domain.GlobalPluginView{}, nil, domain.NewDatabaseError("Could not load plugin layer scans", err.Error())
	}
	// Load entries and marketplaces for all scans upfront.
	entryMap := make(map[int64][]domain.PluginEntry, len(scans))
	marketplaceMap := make(map[int64][]domain.PluginMarketplace, len(scans))
	for _, sc := range scans {
		entries, err := s.repo.ListEntriesForScan(ctx, sc.ID)
		if err != nil {
			return domain.GlobalPluginView{}, nil, domain.NewDatabaseError("Could not load plugin entries", err.Error())
		}
		entryMap[sc.ID] = entries

		mps, err := s.repo.ListMarketplacesForScan(ctx, sc.ID)
		if err != nil {
			return domain.GlobalPluginView{}, nil, domain.NewDatabaseError("Could not load plugin marketplaces", err.Error())
		}
		marketplaceMap[sc.ID] = mps
	}

	global := domain.GlobalPluginView{
		ProviderKey:       def.Provider.Key,
		UserLayerPath:     def.UserFilePath(),
		ManagedOutOfScope: true,
	}
	var userScan *domain.PluginLayerScan
	for i := range scans {
		if scans[i].SettingsLayer == domain.PluginLayerUser && scans[i].ProjectID == nil {
			sc := scans[i]
			global.Scan = &sc
			userScan = &sc
			if sc.ScanStatus == domain.PluginLayerScanOK {
				global.Plugins = entryMap[sc.ID]
				global.Marketplaces = marketplaceMap[sc.ID]
			}
			break
		}
	}

	// Build project views: group project/local scans by project_id.
	projectScanMap := make(map[int64][]domain.PluginLayerScan)
	for _, sc := range scans {
		if sc.ProjectID != nil {
			pid := *sc.ProjectID
			projectScanMap[pid] = append(projectScanMap[pid], sc)
		}
	}

	var projectViews []domain.ProjectPluginView
	for projectID, projectScans := range projectScanMap {
		view := buildProjectPluginView(projectID, def.Provider.Key, projectScans, userScan, entryMap, marketplaceMap)
		projectViews = append(projectViews, view)
	}

	return global, projectViews, nil
}

// SetPluginEnabled writes the enabled/disabled state of a plugin to the user-layer
// settings file for the given provider, then rescans that layer so the persisted
// view reflects the change. Only Claude and Antigravity CLI are supported in this
// slice; Codex writes return a validation_error. Only the "user" (global) layer is
// supported; project and local layer writes are deferred.
func (s *ProviderPluginService) SetPluginEnabled(
	ctx context.Context,
	providerKey, pluginName, marketplaceName string,
	enabled bool,
) (int64, error) {
	const supportedLayer = "user"

	// Validate provider — only JSON-format providers supported in this slice.
	switch providerKey {
	case "claude", "antigravity_cli":
		// OK
	case "codex":
		return 0, domain.NewValidationError(
			"Codex plugin writes not supported",
			"Codex uses TOML format; write support is deferred to a future slice",
		)
	default:
		return 0, domain.NewValidationError(
			"Unknown provider",
			fmt.Sprintf("providerKey %q does not support plugin writes", providerKey),
		)
	}

	if pluginName == "" || marketplaceName == "" {
		return 0, domain.NewValidationError("Plugin name and marketplace are required", "pluginName and marketplaceName must be non-empty")
	}

	// Load the provider definition (must exist in DB).
	defs, err := s.pluginProviderDefsAllowMissing(ctx)
	if err != nil {
		return 0, err
	}
	var targetDef *pluginProviderDef
	for i := range defs {
		if defs[i].Provider.Key == providerKey {
			targetDef = &defs[i]
			break
		}
	}
	if targetDef == nil {
		return 0, domain.NewValidationError(
			"Provider not configured",
			fmt.Sprintf("provider %q not found in database", providerKey),
		)
	}

	def := *targetDef

	target := operations.Target{Type: "provider_plugin_global", ID: 0}
	opID, err := s.runner.Start(ctx, target, domain.OperationTypeScan,
		func(opCtx context.Context, progress operations.ProgressFn) (any, error) {
			return nil, s.setPluginEnabledInternal(opCtx, def, pluginName, marketplaceName, enabled, progress)
		})
	if err != nil {
		if _, ok := err.(*domain.AppError); ok {
			return 0, err
		}
		return 0, domain.NewDatabaseError("Could not start plugin write operation", err.Error())
	}
	return opID, nil
}

func (s *ProviderPluginService) setPluginEnabledInternal(
	ctx context.Context,
	def pluginProviderDef,
	pluginName, marketplaceName string,
	enabled bool,
	progress operations.ProgressFn,
) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return domain.NewValidationError("Cannot resolve home directory", err.Error())
	}
	allowedDir := filepath.Join(homeDir, def.GlobalDir)
	filePath := def.UserFilePath()

	progress("writing_plugin_setting", 0, 1, "")
	if err := s.pluginWriter(filePath, allowedDir, pluginName, marketplaceName, enabled); err != nil {
		return domain.NewFilesystemError("Could not write plugin setting", err.Error())
	}
	progress("writing_plugin_setting", 1, 1, def.Provider.Key)

	// Rescan the user layer so the persisted view reflects the new state.
	return s.scanGlobalInternal(ctx, []pluginProviderDef{def}, progress)
}

// ---- internal scan logic ----

type pluginProviderDef struct {
	Provider    *domain.ProviderDefinition
	GlobalDir   string
	UserFile    string
	ProjectDir  string
	ProjectFile string
	LocalFile   string
	Scanner     func(filePath, allowedDir string) providers.ClaudeSettingsScan
}

func (d pluginProviderDef) UserFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, d.GlobalDir, d.UserFile)
}

func (d pluginProviderDef) ProjectAllowedDir(projectPath string) string {
	return filepath.Join(projectPath, d.ProjectDir)
}

func (d pluginProviderDef) ProjectFilePath(projectPath string) string {
	return filepath.Join(d.ProjectAllowedDir(projectPath), d.ProjectFile)
}

func (d pluginProviderDef) LocalFilePath(projectPath string) string {
	if d.LocalFile == "" {
		return ""
	}
	return filepath.Join(d.ProjectAllowedDir(projectPath), d.LocalFile)
}

func (s *ProviderPluginService) pluginProviderDefs(ctx context.Context) ([]pluginProviderDef, error) {
	defs, err := s.pluginProviderDefsAllowMissing(ctx)
	if err != nil {
		return nil, err
	}
	if len(defs) == 0 {
		return nil, domain.NewValidationError("Provider plugins unavailable", "no plugin-capable provider definitions found")
	}
	return defs, nil
}

func (s *ProviderPluginService) pluginProviderDefsAllowMissing(ctx context.Context) ([]pluginProviderDef, error) {
	templates := []struct {
		key         string
		globalDir   string
		userFile    string
		projectDir  string
		projectFile string
		localFile   string
		scanner     func(filePath, allowedDir string) providers.ClaudeSettingsScan
	}{
		{"claude", ".claude", "settings.json", ".claude", "settings.json", "settings.local.json", providers.ScanClaudeSettingsFile},
		{"codex", ".codex", "config.toml", ".codex", "config.toml", "", providers.ScanCodexConfigFile},
		{"antigravity_cli", filepath.Join(".gemini", "antigravity-cli"), "settings.json", filepath.Join(".gemini", "antigravity-cli"), "settings.json", "", providers.ScanAntigravityCLISettingsFile},
	}
	defs := make([]pluginProviderDef, 0, len(templates))
	for _, tmpl := range templates {
		pd, err := s.pdRepo.GetByKey(ctx, tmpl.key)
		if err != nil {
			return nil, domain.NewDatabaseError(fmt.Sprintf("Could not load %s provider definition", tmpl.key), err.Error())
		}
		if pd == nil {
			continue
		}
		defs = append(defs, pluginProviderDef{
			Provider: pd, GlobalDir: tmpl.globalDir, UserFile: tmpl.userFile,
			ProjectDir: tmpl.projectDir, ProjectFile: tmpl.projectFile, LocalFile: tmpl.localFile,
			Scanner: tmpl.scanner,
		})
	}
	return defs, nil
}

func (s *ProviderPluginService) scanGlobalInternal(ctx context.Context, defs []pluginProviderDef, progress operations.ProgressFn) error {
	total := len(defs)
	progress("scanning_user_layer", 0, total, "")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return domain.NewValidationError("Cannot resolve home directory", err.Error())
	}
	for i, def := range defs {
		allowedDir := filepath.Join(homeDir, def.GlobalDir)
		filePath := filepath.Join(allowedDir, def.UserFile)
		scanResult := def.Scanner(filePath, allowedDir)
		scan := &domain.PluginLayerScan{
			ProviderDefinitionID: def.Provider.ID,
			SettingsLayer:        domain.PluginLayerUser,
			ScanStatus:           domain.PluginLayerScanStatus(scanResult.Status),
			SettingsFilePath:     filePath,
			LastScannedAt:        time.Now().UTC(),
			Warnings:             sanitizeWarnings(scanResult.Warnings),
		}
		entries, mps := pluginScanToEntries(scanResult)
		if err := s.repo.CommitLayerScan(ctx, scan, entries, mps); err != nil {
			return domain.NewDatabaseError("Could not commit global plugin scan", err.Error())
		}
		progress("scanning_user_layer", i+1, total, def.Provider.Key)
	}
	return nil
}

func (s *ProviderPluginService) scanProjectInternal(ctx context.Context, project *domain.Project, defs []pluginProviderDef, progress operations.ProgressFn) error {
	total := len(defs)
	for i, def := range defs {
		allowedDir := def.ProjectAllowedDir(project.Path)
		projectFilePath := def.ProjectFilePath(project.Path)
		projectResult := def.Scanner(projectFilePath, allowedDir)
		projectScan := &domain.PluginLayerScan{
			ProviderDefinitionID: def.Provider.ID,
			ProjectID:            &project.ID,
			SettingsLayer:        domain.PluginLayerProject,
			ScanStatus:           domain.PluginLayerScanStatus(projectResult.Status),
			SettingsFilePath:     projectFilePath,
			LastScannedAt:        time.Now().UTC(),
			Warnings:             sanitizeWarnings(projectResult.Warnings),
		}
		pe, pm := pluginScanToEntries(projectResult)
		if err := s.repo.CommitLayerScan(ctx, projectScan, pe, pm); err != nil {
			return domain.NewDatabaseError("Could not commit project layer scan", err.Error())
		}

		if localFilePath := def.LocalFilePath(project.Path); localFilePath != "" {
			localResult := def.Scanner(localFilePath, allowedDir)
			localScan := &domain.PluginLayerScan{
				ProviderDefinitionID: def.Provider.ID,
				ProjectID:            &project.ID,
				SettingsLayer:        domain.PluginLayerLocal,
				ScanStatus:           domain.PluginLayerScanStatus(localResult.Status),
				SettingsFilePath:     localFilePath,
				LastScannedAt:        time.Now().UTC(),
				Warnings:             sanitizeWarnings(localResult.Warnings),
			}
			le, lm := pluginScanToEntries(localResult)
			if err := s.repo.CommitLayerScan(ctx, localScan, le, lm); err != nil {
				return domain.NewDatabaseError("Could not commit local layer scan", err.Error())
			}
		}
		progress("scanning_project_layer", i+1, total, def.Provider.Key)
	}
	return nil
}

// pluginScanToEntries converts scanner output to domain types.
// Entries and marketplaces are only populated when scan status is ok.
func pluginScanToEntries(r providers.ClaudeSettingsScan) ([]domain.PluginEntry, []domain.PluginMarketplace) {
	if r.Status != "ok" {
		return nil, nil
	}
	entries := make([]domain.PluginEntry, 0, len(r.Plugins))
	for _, p := range r.Plugins {
		decl := domain.PluginDeclarationEnabled
		if !p.Enabled {
			decl = domain.PluginDeclarationDisabled
		}
		entries = append(entries, domain.PluginEntry{
			PluginName:      p.PluginName,
			MarketplaceName: p.MarketplaceName,
			Declaration:     decl,
		})
	}
	mps := make([]domain.PluginMarketplace, 0, len(r.Marketplaces))
	for _, m := range r.Marketplaces {
		mps = append(mps, domain.PluginMarketplace{
			MarketplaceName: m.MarketplaceName,
			SourceType:      m.SourceType,
			SourceSummary:   m.SourceSummary,
		})
	}
	return entries, mps
}

// ---- effective state resolution ----

type pluginKey struct {
	PluginName      string
	MarketplaceName string
}

func buildProjectPluginView(
	projectID int64,
	providerKey string,
	projectScans []domain.PluginLayerScan,
	userScan *domain.PluginLayerScan,
	entryMap map[int64][]domain.PluginEntry,
	marketplaceMap map[int64][]domain.PluginMarketplace,
) domain.ProjectPluginView {
	var localScan, projectLayerScan *domain.PluginLayerScan
	for i := range projectScans {
		sc := &projectScans[i]
		switch sc.SettingsLayer {
		case domain.PluginLayerLocal:
			localScan = sc
		case domain.PluginLayerProject:
			projectLayerScan = sc
		}
	}

	// Layer scans ordered: local, project, user (omit absent ones)
	var layerScans []domain.PluginLayerScan
	if localScan != nil {
		layerScans = append(layerScans, *localScan)
	}
	if projectLayerScan != nil {
		layerScans = append(layerScans, *projectLayerScan)
	}
	if userScan != nil {
		layerScans = append(layerScans, *userScan)
	}

	// Collect all distinct plugin keys across all layers
	allKeys := map[pluginKey]struct{}{}
	addKeys := func(scanID int64) {
		for _, e := range entryMap[scanID] {
			allKeys[pluginKey{e.PluginName, e.MarketplaceName}] = struct{}{}
		}
	}
	if localScan != nil {
		addKeys(localScan.ID)
	}
	if projectLayerScan != nil {
		addKeys(projectLayerScan.ID)
	}
	if userScan != nil {
		addKeys(userScan.ID)
	}

	// Compute effective state for each plugin key
	var effectivePlugins []domain.PluginEffectiveEntry
	for key := range allKeys {
		effective := resolveEffectivePlugin(key.PluginName, key.MarketplaceName, localScan, projectLayerScan, userScan, entryMap)
		// Absent plugins (all layers absent) are not surfaced in project view
		if effective.EffectiveStatus != domain.PluginEffectiveAbsent {
			effectivePlugins = append(effectivePlugins, effective)
		}
	}

	// Collect marketplaces (union from all layers, deduplicated by name)
	mpSeen := map[string]struct{}{}
	var marketplaces []domain.PluginMarketplace
	addMPs := func(scanID int64) {
		for _, m := range marketplaceMap[scanID] {
			if _, seen := mpSeen[m.MarketplaceName]; !seen {
				mpSeen[m.MarketplaceName] = struct{}{}
				marketplaces = append(marketplaces, m)
			}
		}
	}
	if localScan != nil {
		addMPs(localScan.ID)
	}
	if projectLayerScan != nil {
		addMPs(projectLayerScan.ID)
	}
	if userScan != nil {
		addMPs(userScan.ID)
	}

	return domain.ProjectPluginView{
		ProjectID:         projectID,
		ProviderKey:       providerKey,
		LayerScans:        layerScans,
		Plugins:           effectivePlugins,
		Marketplaces:      marketplaces,
		ManagedOutOfScope: true,
	}
}

const (
	maxScanWarnings   = 20
	maxScanWarningLen = 512
)

// sanitizeWarnings caps the warning list and each string length before storage.
// Keeps raw settings content out of persisted data.
func sanitizeWarnings(warnings []string) []string {
	if len(warnings) == 0 {
		return nil
	}
	if len(warnings) > maxScanWarnings {
		warnings = warnings[:maxScanWarnings]
	}
	result := make([]string, len(warnings))
	for i, w := range warnings {
		if len(w) > maxScanWarningLen {
			w = w[:maxScanWarningLen]
		}
		result[i] = w
	}
	return result
}

// resolveEffectivePlugin computes the effective status for one plugin across layers (local > project > user).
// A missing settings file is treated as absent at that layer and does not block inheritance.
// Other non-ok statuses (malformed, unreadable, too_large, symlink, path_escape) block inheritance.
func resolveEffectivePlugin(
	pluginName, marketplaceName string,
	localScan, projectScan, userScan *domain.PluginLayerScan,
	entryMap map[int64][]domain.PluginEntry,
) domain.PluginEffectiveEntry {
	var breakdown []domain.PluginLayerBreakdown

	checkLayer := func(sc *domain.PluginLayerScan, layer domain.PluginSettingsLayer) (done bool, entry domain.PluginEffectiveEntry) {
		if sc == nil {
			return false, domain.PluginEffectiveEntry{}
		}
		// Missing file = no entries at this layer; continue to next layer.
		if sc.ScanStatus == domain.PluginLayerScanMissing {
			breakdown = append(breakdown, domain.PluginLayerBreakdown{Layer: layer, ScanStatus: sc.ScanStatus})
			return false, domain.PluginEffectiveEntry{}
		}
		// Any other non-ok status (malformed, unreadable, too_large, symlink, path_escape) blocks inheritance.
		if sc.ScanStatus != domain.PluginLayerScanOK {
			breakdown = append(breakdown, domain.PluginLayerBreakdown{Layer: layer, ScanStatus: sc.ScanStatus})
			return true, domain.PluginEffectiveEntry{
				PluginName: pluginName, MarketplaceName: marketplaceName,
				EffectiveStatus: domain.PluginEffectiveUnknown, LayerBreakdown: breakdown,
			}
		}
		if decl := findPluginDecl(entryMap[sc.ID], pluginName, marketplaceName); decl != nil {
			breakdown = append(breakdown, domain.PluginLayerBreakdown{Layer: layer, ScanStatus: domain.PluginLayerScanOK, Declaration: decl})
			prov := layer
			return true, domain.PluginEffectiveEntry{
				PluginName: pluginName, MarketplaceName: marketplaceName,
				EffectiveStatus: declToEffective(*decl),
				ProvenanceLayer: &prov, LayerBreakdown: breakdown,
			}
		}
		breakdown = append(breakdown, domain.PluginLayerBreakdown{Layer: layer, ScanStatus: domain.PluginLayerScanOK})
		return false, domain.PluginEffectiveEntry{}
	}

	if done, result := checkLayer(localScan, domain.PluginLayerLocal); done {
		return result
	}
	if done, result := checkLayer(projectScan, domain.PluginLayerProject); done {
		return result
	}
	if done, result := checkLayer(userScan, domain.PluginLayerUser); done {
		return result
	}

	return domain.PluginEffectiveEntry{
		PluginName: pluginName, MarketplaceName: marketplaceName,
		EffectiveStatus: domain.PluginEffectiveAbsent, LayerBreakdown: breakdown,
	}
}

func findPluginDecl(entries []domain.PluginEntry, pluginName, marketplaceName string) *domain.PluginDeclaration {
	for _, e := range entries {
		if e.PluginName == pluginName && e.MarketplaceName == marketplaceName {
			d := e.Declaration
			return &d
		}
	}
	return nil
}

func declToEffective(decl domain.PluginDeclaration) domain.PluginEffectiveStatus {
	if decl == domain.PluginDeclarationEnabled {
		return domain.PluginEffectiveEnabled
	}
	return domain.PluginEffectiveDisabled
}
