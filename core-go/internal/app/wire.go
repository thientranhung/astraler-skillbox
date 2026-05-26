package app

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	rpchandlers "github.com/astraler/skillbox/core-go/internal/rpc/handlers"
	"github.com/astraler/skillbox/core-go/internal/operations"
	"github.com/astraler/skillbox/core-go/internal/services"
)

// App holds the registered JSON-RPC method map.
type App struct {
	methods handler.Map
}

// New builds the composition root and registers all RPC handlers.
func New(
	hostSvc *services.SkillHostService,
	libSvc *services.SkillLibraryService,
	settingsSvc *services.SettingsService,
	runner *operations.Runner,
	projectSvc *services.ProjectService,
	dashboardSvc *services.DashboardService,
	globalSvc *services.GlobalSkillsService,
	providerRegistrySvc *services.ProviderRegistryService,
) *App {
	a := &App{
		methods: handler.Map{
			"ping":             handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) { return rpchandlers.Ping(), nil }),
			"host.choose":      rpchandlers.NewHostChooseHandler(hostSvc),
			"host.scan":        rpchandlers.NewHostScanHandler(hostSvc),
			"skill.list":       rpchandlers.NewSkillListHandler(libSvc),
			"skill.get":        rpchandlers.NewSkillGetHandler(libSvc),
			"settings.get":     rpchandlers.NewSettingsGetHandler(settingsSvc),
			"operation.cancel": rpchandlers.NewOperationCancelHandler(runner),
			"project.add":      rpchandlers.NewProjectAddHandler(projectSvc),
			"project.list":     rpchandlers.NewProjectListHandler(projectSvc),
			"project.get":      rpchandlers.NewProjectGetHandler(projectSvc),
			"project.scan":     rpchandlers.NewProjectScanHandler(projectSvc),
			"project.remove":   rpchandlers.NewProjectRemoveHandler(projectSvc),
			"install.skill":    rpchandlers.NewInstallSkillHandler(projectSvc),
			"remove.skill":     rpchandlers.NewRemoveSkillHandler(projectSvc),
			"dashboard.get":    rpchandlers.NewDashboardGetHandler(dashboardSvc),
			"global.scan":      rpchandlers.NewGlobalScanHandler(globalSvc),
			"global.list":      rpchandlers.NewGlobalListHandler(globalSvc),
			"provider.list":       rpchandlers.NewProviderListHandler(providerRegistrySvc),
			"provider.updatePaths": rpchandlers.NewProviderUpdatePathsHandler(providerRegistrySvc),
			"provider.resetPaths":  rpchandlers.NewProviderResetPathsHandler(providerRegistrySvc),
			"provider.setEnabled":  rpchandlers.NewProviderSetEnabledHandler(providerRegistrySvc),
		},
	}
	return a
}

// Assigner returns the handler map for use with jrpc2.NewServer.
func (a *App) Assigner() jrpc2.Assigner {
	return a.methods
}

// HasMethod reports whether method is registered.
func (a *App) HasMethod(method string) bool {
	return a.methods[method] != nil
}
