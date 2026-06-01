package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/astraler/skillbox/core-go/internal/app"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/network"
	"github.com/astraler/skillbox/core-go/internal/operations"
	"github.com/astraler/skillbox/core-go/internal/providers"
	"github.com/astraler/skillbox/core-go/internal/repositories"
	corerpc "github.com/astraler/skillbox/core-go/internal/rpc"
	"github.com/astraler/skillbox/core-go/internal/rpc/notifications"
	"github.com/astraler/skillbox/core-go/internal/services"
)

func main() {
	// All logs go to stderr; stdout is reserved for JSON-RPC protocol bytes.
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in main", "err", r)
			os.Exit(1)
		}
	}()

	dbPath := resolveDBPath()
	slog.Info("opening database", "path", dbPath)

	db, err := repositories.OpenDatabase(dbPath)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	fs := filesystem.NewGateway()

	hostRepo := repositories.NewSkillHostFolderRepo(db)
	skillRepo := repositories.NewSkillRepo(db)
	warningRepo := repositories.NewWarningRepo(db)
	appSettingsRepo := repositories.NewAppSettingsRepo(db)
	operationRepo := repositories.NewOperationRepo(db)
	scanWriter := repositories.NewScanRepo(db)
	projectRepo := repositories.NewProjectRepo(db)
	ppRepo := repositories.NewProjectProviderRepo(db)
	installRepo := repositories.NewInstallRepo(db)
	projectScanRepo := repositories.NewProjectScanRepo(db)
	pdRepo := repositories.NewProviderDefinitionRepo(db)
	overrideRepo := repositories.NewProviderOverrideRepo(db)
	providerUserSettingsRepo := repositories.NewProviderUserSettingsRepo(db)

	progressCh := make(chan operations.ProgressEvent, 64)
	runner := operations.NewRunner(operationRepo, progressCh)

	// Clean up any operations left in queued/running state from a prior crash.
	ctx := context.Background()
	if err := operationRepo.MarkStaleAsFailed(ctx, "process restarted"); err != nil {
		slog.Warn("could not mark stale operations failed", "err", err)
	}

	hostSvc := services.NewSkillHostService(hostRepo, appSettingsRepo, fs, runner, scanWriter)
	libSvc := services.NewSkillLibraryService(skillRepo, hostRepo, warningRepo)
	settingsSvc := services.NewSettingsService(appSettingsRepo, hostRepo)

	providerRegistry := providers.NewDefaultRegistry()
	providerRegistrySvc := services.NewProviderRegistryService(pdRepo, overrideRepo, providerUserSettingsRepo)

	projectSvc := services.NewProjectService(projectRepo, ppRepo, warningRepo, installRepo, fs).
		WithScanDeps(runner, projectScanRepo).
		WithProviderDeps(providerRegistry, pdRepo, hostRepo, skillRepo).
		WithInstallDeps(fs, hostRepo, skillRepo).
		WithRemoveDeps(fs, installRepo).
		WithPathResolver(providerRegistrySvc)

	dashboardSvc := services.NewDashboardService(appSettingsRepo, hostRepo, skillRepo, projectRepo, installRepo, warningRepo)

	globalScanRepo := repositories.NewGlobalScanRepo(db)
	globalLocationRepo := repositories.NewGlobalLocationRepo(db)
	globalSvc := services.NewGlobalSkillsService(globalLocationRepo, globalScanRepo, appSettingsRepo, hostRepo, skillRepo, providerRegistry, fs, runner).
		WithGlobalPathResolver(providerRegistrySvc).
		WithEnabledReader(providerRegistrySvc)

	providerPluginRepo := repositories.NewProviderPluginRepo(db)
	providerPluginSvc := services.NewProviderPluginService(providerPluginRepo, pdRepo, projectRepo, providerRegistrySvc, runner)
	projectSvc.WithPluginDeps(providerPluginSvc, providerPluginSvc)

	updateCheckCacheRepo := repositories.NewUpdateCheckCacheRepo(db)
	// Plugin update-check is always-on (ADR-0002). Wire the real git ls-remote
	// client; network contact only occurs when the user triggers updateCheck.run.
	updateCheckClient := network.NewGitLsRemoteClient()
	claudeConfigDir := services.ClaudeConfigDirFromHomeDir()
	updateCheckSvc := services.NewUpdateCheckService(updateCheckCacheRepo, updateCheckClient, claudeConfigDir)

	resetFn := func() error {
		return repositories.ResetAllData(context.Background(), db)
	}
	a := app.New(hostSvc, libSvc, settingsSvc, runner, projectSvc, dashboardSvc, globalSvc, providerRegistrySvc, providerPluginSvc, updateCheckSvc, resetFn, AppVersion)

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	srv := corerpc.New(a.Assigner(), os.Stdin, os.Stdout)
	notifications.StartDispatcher(sigCtx, srv, progressCh)

	if err := srv.Notify(sigCtx, "server.ready", map[string]interface{}{
		"version":      "0.1.0-m3",
		"pid":          os.Getpid(),
		"capabilities": []string{"ping", "host.choose", "host.scan", "skill.list", "skill.get", "settings.get", "operation.cancel", "project.add", "project.list", "project.get", "project.scan", "project.remove", "install.skill", "remove.skill", "dashboard.get", "global.scan", "global.list", "provider.list", "provider.updatePaths", "provider.resetPaths", "provider.setEnabled", "providerPlugin.scanGlobal", "providerPlugin.list", "providerPlugin.setEnabled"},
	}); err != nil {
		slog.Error("failed to send server.ready", "err", err)
		os.Exit(1)
	}

	slog.Info("skillbox-core started", "pid", os.Getpid())

	go func() {
		<-sigCtx.Done()
		slog.Info("shutdown signal received, stopping server")
		// Cancel in-memory operations before stopping so goroutines exit cleanly.
		runner.MarkAllRunningAsFailed("server shutting down")
		srv.Stop()
	}()

	if err := srv.Wait(); err != nil {
		slog.Info("server stopped", "reason", err)
	}
}

func resolveDBPath() string {
	if p := os.Getenv("SKILLBOX_DB_PATH"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "skillbox.db"
	}
	dir := filepath.Join(home, "Library", "Application Support", "Astraler Skillbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Warn("could not create data dir, falling back to cwd", "err", err)
		return "skillbox.db"
	}
	return filepath.Join(dir, "skillbox.db")
}
