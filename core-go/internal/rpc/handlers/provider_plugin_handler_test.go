package handlers_test

import (
	"context"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/rpc/handlers"
)

// ---- stub services ----

type stubPluginScanGlobalSvc struct {
	opID int64
	err  error
}

func (s *stubPluginScanGlobalSvc) ScanGlobal(_ context.Context) (int64, error) {
	return s.opID, s.err
}

type stubPluginListSvc struct {
	global   domain.GlobalPluginView
	globals  []domain.GlobalPluginView
	projects []domain.ProjectPluginView
	err      error
}

func (s *stubPluginListSvc) List(_ context.Context) (domain.GlobalPluginView, []domain.ProjectPluginView, error) {
	return s.global, s.projects, s.err
}

func (s *stubPluginListSvc) ListAll(_ context.Context) ([]domain.GlobalPluginView, []domain.ProjectPluginView, error) {
	if s.globals != nil {
		return s.globals, s.projects, s.err
	}
	return []domain.GlobalPluginView{s.global}, s.projects, s.err
}

// ---- scanGlobal tests ----

func TestProviderPluginScanGlobalHandler_Success(t *testing.T) {
	svc := &stubPluginScanGlobalSvc{opID: 42}
	cli := startServer(t, handler.Map{"providerPlugin.scanGlobal": handlers.NewProviderPluginScanGlobalHandler(svc)})

	var resp struct {
		OperationID int64 `json:"operationId"`
	}
	if err := cli.CallResult(context.Background(), "providerPlugin.scanGlobal", nil, &resp); err != nil {
		t.Fatalf("scanGlobal: %v", err)
	}
	if resp.OperationID != 42 {
		t.Errorf("operationId: got %d want 42", resp.OperationID)
	}
}

func TestProviderPluginScanGlobalHandler_ConflictError(t *testing.T) {
	svc := &stubPluginScanGlobalSvc{err: domain.NewConflictError("busy", "target locked")}
	cli := startServer(t, handler.Map{"providerPlugin.scanGlobal": handlers.NewProviderPluginScanGlobalHandler(svc)})

	err := cli.CallResult(context.Background(), "providerPlugin.scanGlobal", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var rpcErr *jrpc2.Error
	if ok := isJRPCError(err, &rpcErr); !ok {
		t.Fatalf("expected jrpc2.Error, got %T: %v", err, err)
	}
}

// ---- list tests ----

func TestProviderPluginListHandler_EmptyGlobal(t *testing.T) {
	svc := &stubPluginListSvc{
		global: domain.GlobalPluginView{
			ProviderKey:       "claude",
			UserLayerPath:     "/home/user/.claude/settings.json",
			ManagedOutOfScope: true,
		},
	}
	cli := startServer(t, handler.Map{"providerPlugin.list": handlers.NewProviderPluginListHandler(svc)})

	var resp struct {
		Globals []struct {
			ProviderKey string `json:"providerKey"`
		} `json:"globals"`
		Global struct {
			ProviderKey       string        `json:"providerKey"`
			UserLayerPath     string        `json:"userLayerPath"`
			UserLayerStatus   interface{}   `json:"userLayerStatus"`
			Plugins           []interface{} `json:"plugins"`
			Marketplaces      []interface{} `json:"marketplaces"`
			ManagedOutOfScope bool          `json:"managedOutOfScope"`
		} `json:"global"`
		Projects []interface{} `json:"projects"`
	}
	if err := cli.CallResult(context.Background(), "providerPlugin.list", nil, &resp); err != nil {
		t.Fatalf("list: %v", err)
	}
	if resp.Global.ProviderKey != "claude" {
		t.Errorf("providerKey: got %q want claude", resp.Global.ProviderKey)
	}
	if len(resp.Globals) != 1 || resp.Globals[0].ProviderKey != "claude" {
		t.Errorf("globals: got %+v want one claude entry", resp.Globals)
	}
	if !resp.Global.ManagedOutOfScope {
		t.Error("managedOutOfScope: want true")
	}
	if resp.Global.UserLayerStatus != nil {
		t.Errorf("userLayerStatus: want nil (never scanned), got %v", resp.Global.UserLayerStatus)
	}
	if resp.Global.Plugins == nil {
		t.Error("plugins: should be empty array, not null")
	}
}

func TestProviderPluginListHandler_WithScannedUserLayer(t *testing.T) {
	enabled := domain.PluginDeclarationEnabled
	svc := &stubPluginListSvc{
		global: domain.GlobalPluginView{
			ProviderKey:   "claude",
			UserLayerPath: "/home/.claude/settings.json",
			Scan: &domain.PluginLayerScan{
				ScanStatus:    domain.PluginLayerScanOK,
				SettingsLayer: domain.PluginLayerUser,
			},
			Plugins: []domain.PluginEntry{
				{PluginName: "foo", MarketplaceName: "npm", Declaration: enabled},
			},
			ManagedOutOfScope: true,
		},
	}
	cli := startServer(t, handler.Map{"providerPlugin.list": handlers.NewProviderPluginListHandler(svc)})

	var resp struct {
		Global struct {
			UserLayerStatus string `json:"userLayerStatus"`
			Plugins         []struct {
				PluginName string `json:"pluginName"`
				Status     string `json:"status"`
			} `json:"plugins"`
		} `json:"global"`
	}
	if err := cli.CallResult(context.Background(), "providerPlugin.list", nil, &resp); err != nil {
		t.Fatalf("list: %v", err)
	}
	if resp.Global.UserLayerStatus != "ok" {
		t.Errorf("userLayerStatus: got %q want ok", resp.Global.UserLayerStatus)
	}
	if len(resp.Global.Plugins) != 1 {
		t.Fatalf("plugins: got %d want 1", len(resp.Global.Plugins))
	}
	if resp.Global.Plugins[0].PluginName != "foo" {
		t.Errorf("plugin name: got %q want foo", resp.Global.Plugins[0].PluginName)
	}
	if resp.Global.Plugins[0].Status != "enabled" {
		t.Errorf("plugin status: got %q want enabled", resp.Global.Plugins[0].Status)
	}
}

func TestProviderPluginListHandler_MultipleGlobals(t *testing.T) {
	svc := &stubPluginListSvc{
		globals: []domain.GlobalPluginView{
			{ProviderKey: "claude", UserLayerPath: "/home/.claude/settings.json", ManagedOutOfScope: true},
			{ProviderKey: "codex", UserLayerPath: "/home/.codex/config.toml", ManagedOutOfScope: true},
		},
	}
	cli := startServer(t, handler.Map{"providerPlugin.list": handlers.NewProviderPluginListHandler(svc)})

	var resp struct {
		Globals []struct {
			ProviderKey   string `json:"providerKey"`
			UserLayerPath string `json:"userLayerPath"`
		} `json:"globals"`
		Global struct {
			ProviderKey string `json:"providerKey"`
		} `json:"global"`
	}
	if err := cli.CallResult(context.Background(), "providerPlugin.list", nil, &resp); err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(resp.Globals) != 2 {
		t.Fatalf("globals: got %d want 2", len(resp.Globals))
	}
	if resp.Globals[1].ProviderKey != "codex" || resp.Globals[1].UserLayerPath != "/home/.codex/config.toml" {
		t.Errorf("codex global: got %+v", resp.Globals[1])
	}
	if resp.Global.ProviderKey != "claude" {
		t.Errorf("legacy global providerKey: got %q want claude", resp.Global.ProviderKey)
	}
}

func TestProviderPluginListHandler_ProjectViewWithProvenance(t *testing.T) {
	localLayer := domain.PluginLayerLocal
	localDecl := domain.PluginDeclarationEnabled

	svc := &stubPluginListSvc{
		global: domain.GlobalPluginView{ProviderKey: "claude", UserLayerPath: "/h/.claude/s.json", ManagedOutOfScope: true},
		projects: []domain.ProjectPluginView{
			{
				ProjectID:   5,
				ProviderKey: "claude",
				LayerScans: []domain.PluginLayerScan{
					{SettingsLayer: domain.PluginLayerLocal, ScanStatus: domain.PluginLayerScanOK, SettingsFilePath: "/proj/.claude/settings.local.json"},
				},
				Plugins: []domain.PluginEffectiveEntry{
					{
						PluginName:      "bar",
						MarketplaceName: "npm",
						EffectiveStatus: domain.PluginEffectiveEnabled,
						ProvenanceLayer: &localLayer,
						LayerBreakdown: []domain.PluginLayerBreakdown{
							{Layer: domain.PluginLayerLocal, ScanStatus: domain.PluginLayerScanOK, Declaration: &localDecl},
						},
					},
				},
				ManagedOutOfScope: true,
			},
		},
	}
	cli := startServer(t, handler.Map{"providerPlugin.list": handlers.NewProviderPluginListHandler(svc)})

	var resp struct {
		Projects []struct {
			ProjectID int64 `json:"projectId"`
			Plugins   []struct {
				PluginName      string `json:"pluginName"`
				EffectiveStatus string `json:"effectiveStatus"`
				ProvenanceLayer string `json:"provenanceLayer"`
				LayerBreakdown  []struct {
					Layer       string  `json:"layer"`
					Declaration *string `json:"declaration"`
				} `json:"layerBreakdown"`
			} `json:"plugins"`
			LayerStatuses []struct {
				Layer      string `json:"layer"`
				ScanStatus string `json:"scanStatus"`
			} `json:"layerStatuses"`
		} `json:"projects"`
	}
	if err := cli.CallResult(context.Background(), "providerPlugin.list", nil, &resp); err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(resp.Projects) != 1 {
		t.Fatalf("projects: got %d want 1", len(resp.Projects))
	}
	proj := resp.Projects[0]
	if proj.ProjectID != 5 {
		t.Errorf("projectId: got %d want 5", proj.ProjectID)
	}
	if len(proj.Plugins) != 1 {
		t.Fatalf("plugins: got %d want 1", len(proj.Plugins))
	}
	if proj.Plugins[0].EffectiveStatus != "enabled" {
		t.Errorf("effectiveStatus: got %q want enabled", proj.Plugins[0].EffectiveStatus)
	}
	if proj.Plugins[0].ProvenanceLayer != "local" {
		t.Errorf("provenanceLayer: got %q want local", proj.Plugins[0].ProvenanceLayer)
	}
	if len(proj.Plugins[0].LayerBreakdown) != 1 {
		t.Fatalf("layerBreakdown: got %d want 1", len(proj.Plugins[0].LayerBreakdown))
	}
	if proj.Plugins[0].LayerBreakdown[0].Declaration == nil || *proj.Plugins[0].LayerBreakdown[0].Declaration != "enabled" {
		t.Errorf("breakdown declaration: got %v want enabled", proj.Plugins[0].LayerBreakdown[0].Declaration)
	}
}

func TestProviderPluginListHandler_ServiceError(t *testing.T) {
	svc := &stubPluginListSvc{err: domain.NewDatabaseError("db error", "details")}
	cli := startServer(t, handler.Map{"providerPlugin.list": handlers.NewProviderPluginListHandler(svc)})

	err := cli.CallResult(context.Background(), "providerPlugin.list", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---- helpers ----

func isJRPCError(err error, out **jrpc2.Error) bool {
	if e, ok := err.(*jrpc2.Error); ok {
		*out = e
		return true
	}
	return false
}

// ---- stub for removeOverride ----

type stubPluginRemoveOverrideSvc struct {
	opID     int64
	err      error
	lastCall removeOverrideCall
}

type removeOverrideCall struct {
	providerKey     string
	pluginName      string
	marketplaceName string
	layer           string
	projectID       int64
}

func (s *stubPluginRemoveOverrideSvc) RemoveOverride(_ context.Context, providerKey, pluginName, marketplaceName, layer string, projectID int64) (int64, error) {
	s.lastCall = removeOverrideCall{providerKey, pluginName, marketplaceName, layer, projectID}
	return s.opID, s.err
}

// ---- removeOverride tests ----

func TestProviderPluginRemoveOverrideHandler_Success(t *testing.T) {
	svc := &stubPluginRemoveOverrideSvc{opID: 99}
	cli := startServer(t, handler.Map{"providerPlugin.removeOverride": handlers.NewProviderPluginRemoveOverrideHandler(svc)})

	var resp struct {
		OperationID int64 `json:"operationId"`
	}
	params := map[string]interface{}{
		"providerKey":     "claude",
		"pluginName":      "test-plugin",
		"marketplaceName": "npm",
		"layer":           "project",
		"projectId":       5,
	}
	if err := cli.CallResult(context.Background(), "providerPlugin.removeOverride", params, &resp); err != nil {
		t.Fatalf("removeOverride: %v", err)
	}
	if resp.OperationID != 99 {
		t.Errorf("operationId: got %d want 99", resp.OperationID)
	}
	if svc.lastCall.providerKey != "claude" {
		t.Errorf("providerKey: got %q want claude", svc.lastCall.providerKey)
	}
}

func TestProviderPluginRemoveOverrideHandler_MissingProviderKey(t *testing.T) {
	svc := &stubPluginRemoveOverrideSvc{}
	cli := startServer(t, handler.Map{"providerPlugin.removeOverride": handlers.NewProviderPluginRemoveOverrideHandler(svc)})

	params := map[string]interface{}{
		"pluginName":      "test",
		"marketplaceName": "npm",
		"layer":           "project",
		"projectId":       5,
	}
	err := cli.CallResult(context.Background(), "providerPlugin.removeOverride", params, nil)
	if err == nil {
		t.Fatal("expected error for missing providerKey")
	}
}

func TestProviderPluginRemoveOverrideHandler_InvalidLayer(t *testing.T) {
	svc := &stubPluginRemoveOverrideSvc{}
	cli := startServer(t, handler.Map{"providerPlugin.removeOverride": handlers.NewProviderPluginRemoveOverrideHandler(svc)})

	params := map[string]interface{}{
		"providerKey":     "claude",
		"pluginName":      "test",
		"marketplaceName": "npm",
		"layer":           "user",
		"projectId":       5,
	}
	err := cli.CallResult(context.Background(), "providerPlugin.removeOverride", params, nil)
	if err == nil {
		t.Fatal("expected error for non-project layer")
	}
}

func TestProviderPluginRemoveOverrideHandler_MissingProjectId(t *testing.T) {
	svc := &stubPluginRemoveOverrideSvc{}
	cli := startServer(t, handler.Map{"providerPlugin.removeOverride": handlers.NewProviderPluginRemoveOverrideHandler(svc)})

	params := map[string]interface{}{
		"providerKey":     "claude",
		"pluginName":      "test",
		"marketplaceName": "npm",
		"layer":           "project",
	}
	err := cli.CallResult(context.Background(), "providerPlugin.removeOverride", params, nil)
	if err == nil {
		t.Fatal("expected error for missing projectId")
	}
}

func TestProviderPluginRemoveOverrideHandler_MissingPluginName(t *testing.T) {
	svc := &stubPluginRemoveOverrideSvc{}
	cli := startServer(t, handler.Map{"providerPlugin.removeOverride": handlers.NewProviderPluginRemoveOverrideHandler(svc)})

	params := map[string]interface{}{
		"providerKey":     "claude",
		"marketplaceName": "npm",
		"layer":           "project",
		"projectId":       5,
	}
	err := cli.CallResult(context.Background(), "providerPlugin.removeOverride", params, nil)
	if err == nil {
		t.Fatal("expected error for missing pluginName")
	}
}

func TestProviderPluginRemoveOverrideHandler_MissingMarketplaceName(t *testing.T) {
	svc := &stubPluginRemoveOverrideSvc{}
	cli := startServer(t, handler.Map{"providerPlugin.removeOverride": handlers.NewProviderPluginRemoveOverrideHandler(svc)})

	params := map[string]interface{}{
		"providerKey": "claude",
		"pluginName":  "test",
		"layer":       "project",
		"projectId":   5,
	}
	err := cli.CallResult(context.Background(), "providerPlugin.removeOverride", params, nil)
	if err == nil {
		t.Fatal("expected error for missing marketplaceName")
	}
}
