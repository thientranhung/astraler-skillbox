package handlers_test

import (
	"context"
	"testing"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/rpc/handlers"
	"github.com/astraler/skillbox/core-go/internal/services"
)

// --- stubs ---

type stubProjectAdd struct {
	result *services.AddProjectResult
	err    error
}

func (s *stubProjectAdd) AddProject(_ context.Context, _ string) (*services.AddProjectResult, error) {
	return s.result, s.err
}

type stubProjectList struct {
	items []services.ProjectListItem
	err   error
}

func (s *stubProjectList) ListProjects(_ context.Context) ([]services.ProjectListItem, error) {
	return s.items, s.err
}

type stubProjectGet struct {
	result *services.ProjectDetailView
	err    error
}

func (s *stubProjectGet) GetProject(_ context.Context, _ int64) (*services.ProjectDetailView, error) {
	return s.result, s.err
}

type stubProjectScan struct {
	opID int64
	err  error
}

func (s *stubProjectScan) ScanProject(_ context.Context, _ int64) (int64, error) {
	return s.opID, s.err
}

// --- project.add ---

func TestProjectAddHandler_Success(t *testing.T) {
	svc := &stubProjectAdd{result: &services.AddProjectResult{
		ProjectID: 3,
		Name:      "myproject",
		Path:      "/home/user/myproject",
		Status:    domain.ProjectStatusActive,
	}}
	cli := startServer(t, handler.Map{"project.add": handlers.NewProjectAddHandler(svc)})

	var resp struct {
		ProjectID int64  `json:"projectId"`
		Name      string `json:"name"`
		Status    string `json:"status"`
	}
	if err := cli.CallResult(context.Background(), "project.add", map[string]string{"path": "/home/user/myproject"}, &resp); err != nil {
		t.Fatalf("project.add: %v", err)
	}
	if resp.ProjectID != 3 {
		t.Errorf("projectId: got %d want 3", resp.ProjectID)
	}
	if resp.Name != "myproject" {
		t.Errorf("name: got %q want myproject", resp.Name)
	}
	if resp.Status != "active" {
		t.Errorf("status: got %q want active", resp.Status)
	}
}

func TestProjectAddHandler_MissingPath_ReturnsValidationError(t *testing.T) {
	svc := &stubProjectAdd{}
	cli := startServer(t, handler.Map{"project.add": handlers.NewProjectAddHandler(svc)})

	err := cli.CallResult(context.Background(), "project.add", map[string]string{"path": ""}, nil)
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("payload code: got %q want %q", we.ae.Code, domain.CodeValidation)
	}
}

func TestProjectAddHandler_ServiceError_MapsToJRPCError(t *testing.T) {
	svc := &stubProjectAdd{err: domain.NewValidationError("path is not a directory", "not a dir")}
	cli := startServer(t, handler.Map{"project.add": handlers.NewProjectAddHandler(svc)})

	err := cli.CallResult(context.Background(), "project.add", map[string]string{"path": "/bad"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	extractWireError(t, err, jrpc2.Code(1001))
}

// --- project.list ---

func TestProjectListHandler_EmptyList(t *testing.T) {
	svc := &stubProjectList{items: []services.ProjectListItem{}}
	cli := startServer(t, handler.Map{"project.list": handlers.NewProjectListHandler(svc)})

	var resp struct {
		Projects []interface{} `json:"projects"`
	}
	if err := cli.CallResult(context.Background(), "project.list", map[string]interface{}{}, &resp); err != nil {
		t.Fatalf("project.list: %v", err)
	}
	if len(resp.Projects) != 0 {
		t.Errorf("expected empty projects, got %d", len(resp.Projects))
	}
}

func TestProjectListHandler_WithProjects(t *testing.T) {
	now := time.Now().UTC()
	svc := &stubProjectList{items: []services.ProjectListItem{
		{
			ID:     1,
			Name:   "alpha",
			Path:   "/home/user/alpha",
			Status: domain.ProjectStatusActive,
			Providers: []domain.ProjectProviderSummary{
				{
					ProjectProviderID:   10,
					ProviderKey:         "generic_agents",
					ProviderDisplayName: "Shared Agent Skills",
					ProviderStatus:      domain.ProviderStatusSupported,
					DetectionStatus:     domain.DetectionStatusDetected,
					EntryCount:          2,
				},
			},
			SkillCount:    2,
			WarningCount:  1,
			LastScannedAt: &now,
		},
	}}
	cli := startServer(t, handler.Map{"project.list": handlers.NewProjectListHandler(svc)})

	var resp struct {
		Projects []struct {
			ID           int64  `json:"id"`
			Name         string `json:"name"`
			SkillCount   int    `json:"skillCount"`
			WarningCount int    `json:"warningCount"`
			Providers    []struct {
				Key         string `json:"key"`
				DisplayName string `json:"displayName"`
				EntryCount  int    `json:"entryCount"`
			} `json:"providers"`
		} `json:"projects"`
	}
	if err := cli.CallResult(context.Background(), "project.list", map[string]interface{}{}, &resp); err != nil {
		t.Fatalf("project.list: %v", err)
	}
	if len(resp.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(resp.Projects))
	}
	if resp.Projects[0].ID != 1 {
		t.Errorf("id: got %d want 1", resp.Projects[0].ID)
	}
	if resp.Projects[0].SkillCount != 2 {
		t.Errorf("skillCount: got %d want 2", resp.Projects[0].SkillCount)
	}
	if resp.Projects[0].WarningCount != 1 {
		t.Errorf("warningCount: got %d want 1", resp.Projects[0].WarningCount)
	}
	if resp.Projects[0].Providers[0].Key != "generic_agents" {
		t.Errorf("provider key: got %q want generic_agents", resp.Projects[0].Providers[0].Key)
	}
	if resp.Projects[0].Providers[0].DisplayName != "Shared Agent Skills" {
		t.Errorf("provider displayName: got %q", resp.Projects[0].Providers[0].DisplayName)
	}
	if resp.Projects[0].Providers[0].EntryCount != 2 {
		t.Errorf("provider entryCount: got %d want 2", resp.Projects[0].Providers[0].EntryCount)
	}
}

// --- project.get ---

func TestProjectGetHandler_Success(t *testing.T) {
	now := time.Now().UTC()
	svc := &stubProjectGet{result: &services.ProjectDetailView{
		Project: domain.Project{
			ID:            5,
			Name:          "beta",
			Path:          "/home/user/beta",
			Status:        domain.ProjectStatusActive,
			LastScannedAt: &now,
		},
		Providers: []domain.ProjectProviderSummary{
			{
				ProjectProviderID:   20,
				ProviderKey:         "generic_agents",
				ProviderDisplayName: "Shared Agent Skills",
				ProviderStatus:      domain.ProviderStatusSupported,
				DetectionStatus:     domain.DetectionStatusDetected,
			},
		},
		Entries:  []domain.Install{},
		Warnings: []domain.Warning{},
	}}
	cli := startServer(t, handler.Map{"project.get": handlers.NewProjectGetHandler(svc)})

	var resp struct {
		Project struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"project"`
		Providers []struct {
			ProviderKey string `json:"providerKey"`
			DisplayName string `json:"displayName"`
		} `json:"providers"`
		Entries  []interface{} `json:"entries"`
		Warnings []interface{} `json:"warnings"`
	}
	if err := cli.CallResult(context.Background(), "project.get", map[string]int64{"projectId": 5}, &resp); err != nil {
		t.Fatalf("project.get: %v", err)
	}
	if resp.Project.ID != 5 {
		t.Errorf("project.id: got %d want 5", resp.Project.ID)
	}
	if resp.Project.Name != "beta" {
		t.Errorf("project.name: got %q want beta", resp.Project.Name)
	}
	if len(resp.Providers) != 1 {
		t.Errorf("providers: got %d want 1", len(resp.Providers))
	}
	if resp.Providers[0].ProviderKey != "generic_agents" {
		t.Errorf("provider key: got %q want generic_agents", resp.Providers[0].ProviderKey)
	}
	if resp.Providers[0].DisplayName != "Shared Agent Skills" {
		t.Errorf("provider displayName: got %q", resp.Providers[0].DisplayName)
	}
}

func TestProjectGetHandler_NotFound_ReturnsValidationError(t *testing.T) {
	svc := &stubProjectGet{err: domain.NewValidationError("Project not found", "projectId 99 does not exist")}
	cli := startServer(t, handler.Map{"project.get": handlers.NewProjectGetHandler(svc)})

	err := cli.CallResult(context.Background(), "project.get", map[string]int64{"projectId": 99}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("payload code: got %q want %q", we.ae.Code, domain.CodeValidation)
	}
}

func TestProjectGetHandler_EntryProviderKeyDerived(t *testing.T) {
	svc := &stubProjectGet{result: &services.ProjectDetailView{
		Project: domain.Project{ID: 1, Name: "p", Path: "/p", Status: domain.ProjectStatusActive},
		Providers: []domain.ProjectProviderSummary{
			{
				ProjectProviderID:   42,
				ProviderKey:         "cursor",
				ProviderDisplayName: "Cursor",
				ProviderStatus:      domain.ProviderStatusSupported,
				DetectionStatus:     domain.DetectionStatusDetected,
			},
		},
		Entries: []domain.Install{
			{
				ID:                1,
				ProjectProviderID: 42,
				SkillName:         "my-skill",
				InstallMode:       domain.InstallModeSymlink,
				InstallStatus:     domain.InstallStatusCurrent,
				ProjectSkillPath:  "/p/.cursor/skills/my-skill",
			},
		},
		Warnings: []domain.Warning{},
	}}
	cli := startServer(t, handler.Map{"project.get": handlers.NewProjectGetHandler(svc)})

	var resp struct {
		Entries []struct {
			ProviderKey string `json:"providerKey"`
			Name        string `json:"name"`
		} `json:"entries"`
	}
	if err := cli.CallResult(context.Background(), "project.get", map[string]int64{"projectId": 1}, &resp); err != nil {
		t.Fatalf("project.get: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("entries: got %d want 1", len(resp.Entries))
	}
	if resp.Entries[0].ProviderKey != "cursor" {
		t.Errorf("entry.providerKey: got %q want cursor", resp.Entries[0].ProviderKey)
	}
	if resp.Entries[0].Name != "my-skill" {
		t.Errorf("entry.name: got %q want my-skill", resp.Entries[0].Name)
	}
}

// --- project.scan ---

func TestProjectScanHandler_ReturnsOperationID(t *testing.T) {
	svc := &stubProjectScan{opID: 77}
	cli := startServer(t, handler.Map{"project.scan": handlers.NewProjectScanHandler(svc)})

	var resp struct {
		OperationID int64 `json:"operationId"`
	}
	if err := cli.CallResult(context.Background(), "project.scan", map[string]int64{"projectId": 1}, &resp); err != nil {
		t.Fatalf("project.scan: %v", err)
	}
	if resp.OperationID != 77 {
		t.Errorf("operationId: got %d want 77", resp.OperationID)
	}
}

func TestProjectScanHandler_ConflictError_MapsTo1005(t *testing.T) {
	svc := &stubProjectScan{err: domain.NewConflictError("scan already running", "target locked")}
	cli := startServer(t, handler.Map{"project.scan": handlers.NewProjectScanHandler(svc)})

	err := cli.CallResult(context.Background(), "project.scan", map[string]int64{"projectId": 1}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	we := extractWireError(t, err, jrpc2.Code(1005))
	if we.ae.Code != domain.CodeConflict {
		t.Errorf("payload code: got %q want conflict_error", we.ae.Code)
	}
}

func TestProjectScanHandler_NotFound_ReturnsValidationError(t *testing.T) {
	svc := &stubProjectScan{err: domain.NewValidationError("Project not found", "projectId 999 does not exist")}
	cli := startServer(t, handler.Map{"project.scan": handlers.NewProjectScanHandler(svc)})

	err := cli.CallResult(context.Background(), "project.scan", map[string]int64{"projectId": 999}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("payload code: got %q want validation_error", we.ae.Code)
	}
}

// --- project.remove ---

type stubProjectRemove struct {
	result *services.ProjectRemoveResult
	err    error
}

func (s *stubProjectRemove) RemoveProject(_ context.Context, _ int64) (*services.ProjectRemoveResult, error) {
	return s.result, s.err
}

func TestProjectRemoveHandler_Success(t *testing.T) {
	svc := &stubProjectRemove{result: &services.ProjectRemoveResult{Removed: true}}
	cli := startServer(t, handler.Map{"project.remove": handlers.NewProjectRemoveHandler(svc)})

	var resp struct {
		Removed bool `json:"removed"`
	}
	if err := cli.CallResult(context.Background(), "project.remove", map[string]int64{"projectId": 1}, &resp); err != nil {
		t.Fatalf("project.remove: %v", err)
	}
	if !resp.Removed {
		t.Error("expected removed=true")
	}
}

func TestProjectRemoveHandler_ZeroID_ReturnsValidationError(t *testing.T) {
	svc := &stubProjectRemove{}
	cli := startServer(t, handler.Map{"project.remove": handlers.NewProjectRemoveHandler(svc)})

	err := cli.CallResult(context.Background(), "project.remove", map[string]int64{"projectId": 0}, nil)
	if err == nil {
		t.Fatal("expected error for projectId=0")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("payload code: got %q want validation_error", we.ae.Code)
	}
}

func TestProjectRemoveHandler_NotFound_ReturnsValidationError(t *testing.T) {
	svc := &stubProjectRemove{err: domain.NewValidationError("Project not found", "projectId 99 does not exist or is already removed")}
	cli := startServer(t, handler.Map{"project.remove": handlers.NewProjectRemoveHandler(svc)})

	err := cli.CallResult(context.Background(), "project.remove", map[string]int64{"projectId": 99}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("payload code: got %q want validation_error", we.ae.Code)
	}
}

// --- install.skill ---

type stubInstallSkill struct {
	opID int64
	err  error
}

func (s *stubInstallSkill) InstallSkills(_ context.Context, _ int64, _ string, _ []int64) (int64, error) {
	return s.opID, s.err
}

func TestInstallSkillHandler_ReturnsOperationID(t *testing.T) {
	svc := &stubInstallSkill{opID: 42}
	cli := startServer(t, handler.Map{"install.skill": handlers.NewInstallSkillHandler(svc)})

	var resp struct {
		OperationID int64 `json:"operationId"`
	}
	params := map[string]interface{}{
		"projectId":   int64(1),
		"providerKey": "generic_agents",
		"skillIds":    []int64{10, 20},
	}
	if err := cli.CallResult(context.Background(), "install.skill", params, &resp); err != nil {
		t.Fatalf("install.skill: %v", err)
	}
	if resp.OperationID != 42 {
		t.Errorf("operationId: got %d want 42", resp.OperationID)
	}
}

func TestInstallSkillHandler_BadParams_ReturnsValidationError(t *testing.T) {
	svc := &stubInstallSkill{}
	cli := startServer(t, handler.Map{"install.skill": handlers.NewInstallSkillHandler(svc)})

	// Pass a string where an int64 is expected to trigger UnmarshalParams failure.
	err := cli.CallResult(context.Background(), "install.skill", map[string]interface{}{
		"projectId":   "not-a-number",
		"providerKey": "generic_agents",
		"skillIds":    []int64{1},
	}, nil)
	if err == nil {
		t.Fatal("expected error for bad params")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("payload code: got %q want validation_error", we.ae.Code)
	}
}

func TestInstallSkillHandler_ServiceError_MapsToJRPCError(t *testing.T) {
	svc := &stubInstallSkill{err: domain.NewValidationError("Project not found", "projectId 99 does not exist")}
	cli := startServer(t, handler.Map{"install.skill": handlers.NewInstallSkillHandler(svc)})

	params := map[string]interface{}{
		"projectId":   int64(99),
		"providerKey": "generic_agents",
		"skillIds":    []int64{1},
	}
	err := cli.CallResult(context.Background(), "install.skill", params, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("payload code: got %q want validation_error", we.ae.Code)
	}
}

func TestInstallSkillHandler_ConflictError_MapsTo1005(t *testing.T) {
	svc := &stubInstallSkill{err: domain.NewConflictError("install already running", "target locked")}
	cli := startServer(t, handler.Map{"install.skill": handlers.NewInstallSkillHandler(svc)})

	params := map[string]interface{}{
		"projectId":   int64(1),
		"providerKey": "generic_agents",
		"skillIds":    []int64{1},
	}
	err := cli.CallResult(context.Background(), "install.skill", params, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	we := extractWireError(t, err, jrpc2.Code(1005))
	if we.ae.Code != domain.CodeConflict {
		t.Errorf("payload code: got %q want conflict_error", we.ae.Code)
	}
}

// --- remove.skill ---

type stubRemoveSkill struct {
	opID int64
	err  error
}

func (s *stubRemoveSkill) RemoveSkill(_ context.Context, _ int64, _ int64) (int64, error) {
	return s.opID, s.err
}

func TestRemoveSkillHandler_ReturnsOperationID(t *testing.T) {
	svc := &stubRemoveSkill{opID: 51}
	cli := startServer(t, handler.Map{"remove.skill": handlers.NewRemoveSkillHandler(svc)})

	params := map[string]interface{}{"projectId": 1, "installId": 1001}
	var resp struct {
		OperationID int64 `json:"operationId"`
	}
	if err := cli.CallResult(context.Background(), "remove.skill", params, &resp); err != nil {
		t.Fatalf("remove.skill: %v", err)
	}
	if resp.OperationID != 51 {
		t.Errorf("operationId: got %d want 51", resp.OperationID)
	}
}

func TestRemoveSkillHandler_BadParams_ReturnsValidationError(t *testing.T) {
	svc := &stubRemoveSkill{}
	cli := startServer(t, handler.Map{"remove.skill": handlers.NewRemoveSkillHandler(svc)})

	err := cli.CallResult(context.Background(), "remove.skill", map[string]interface{}{
		"projectId": "not-a-number",
	}, nil)
	if err == nil {
		t.Fatal("expected error for bad params")
	}
}

func TestRemoveSkillHandler_ConflictError_MapsTo1005(t *testing.T) {
	svc := &stubRemoveSkill{err: domain.NewConflictError("project busy", "target locked")}
	cli := startServer(t, handler.Map{"remove.skill": handlers.NewRemoveSkillHandler(svc)})

	params := map[string]interface{}{"projectId": 1, "installId": 1001}
	err := cli.CallResult(context.Background(), "remove.skill", params, nil)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if jerr, ok := err.(*jrpc2.Error); ok {
		if jerr.Code != 1005 {
			t.Errorf("code: got %d want 1005", jerr.Code)
		}
	} else {
		t.Fatalf("expected *jrpc2.Error, got %T", err)
	}
}
