package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/rpc/handlers"
	"github.com/astraler/skillbox/core-go/internal/services"
)

// startServer creates an in-process server and returns a connected client.
func startServer(t *testing.T, methods handler.Map) *jrpc2.Client {
	t.Helper()
	cch, sch := channel.Direct()
	srv := jrpc2.NewServer(methods, nil)
	srv.Start(sch)
	t.Cleanup(func() { srv.Stop(); srv.Wait() })
	return jrpc2.NewClient(cch, nil)
}

// extractAppError unmarshals the jrpc2 error Data field into domain.AppError.
func extractAppError(t *testing.T, err error) *domain.AppError {
	t.Helper()
	var rpcErr *jrpc2.Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *jrpc2.Error, got %T: %v", err, err)
	}
	var ae domain.AppError
	if err2 := json.Unmarshal(rpcErr.Data, &ae); err2 != nil {
		t.Fatalf("unmarshal error data: %v (raw: %s)", err2, rpcErr.Data)
	}
	return &ae
}

// -- stubs --

type stubHostChoose struct {
	result *services.ChooseHostResult
	err    error
}

func (s *stubHostChoose) ChooseHost(_ context.Context, _ string) (*services.ChooseHostResult, error) {
	return s.result, s.err
}

type stubHostScan struct {
	opID int64
	err  error
}

func (s *stubHostScan) ScanHost(_ context.Context, _ int64) (int64, error) {
	return s.opID, s.err
}

type stubRunner struct {
	acked bool
	err   error
}

func (s *stubRunner) Cancel(_ context.Context, _ int64) (bool, error) {
	return s.acked, s.err
}

// -- tests --

func TestHostChooseHandler_Success(t *testing.T) {
	svc := &stubHostChoose{result: &services.ChooseHostResult{
		HostID:      7,
		Path:        "/tmp/host",
		SkillsPath:  "/tmp/host/.agents/skills",
		Initialized: true,
		Status:      domain.SkillHostStatusActive,
	}}
	cli := startServer(t, handler.Map{"host.choose": handlers.NewHostChooseHandler(svc)})

	var resp struct {
		HostID      int64  `json:"hostId"`
		Initialized bool   `json:"initialized"`
		Status      string `json:"status"`
	}
	if err := cli.CallResult(context.Background(), "host.choose", map[string]string{"path": "/tmp/host"}, &resp); err != nil {
		t.Fatalf("host.choose: %v", err)
	}
	if resp.HostID != 7 {
		t.Errorf("hostId: got %d want 7", resp.HostID)
	}
	if !resp.Initialized {
		t.Error("expected initialized=true")
	}
	if resp.Status != "active" {
		t.Errorf("status: got %q want active", resp.Status)
	}
}

func TestHostChooseHandler_ValidationError_MapsToJRPCError(t *testing.T) {
	svc := &stubHostChoose{err: domain.NewValidationError("Host not found", "path invalid")}
	cli := startServer(t, handler.Map{"host.choose": handlers.NewHostChooseHandler(svc)})

	err := cli.CallResult(context.Background(), "host.choose", map[string]string{"path": "/bad"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	ae := extractAppError(t, err)
	if ae.Code != domain.CodeValidation {
		t.Errorf("code: got %q want %q", ae.Code, domain.CodeValidation)
	}
	if ae.RPCCode() != 1001 {
		t.Errorf("rpcCode: got %d want 1001", ae.RPCCode())
	}
}

func TestHostChooseHandler_MissingPath_ReturnsBadRequest(t *testing.T) {
	svc := &stubHostChoose{}
	cli := startServer(t, handler.Map{"host.choose": handlers.NewHostChooseHandler(svc)})

	err := cli.CallResult(context.Background(), "host.choose", map[string]string{"path": ""}, nil)
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	ae := extractAppError(t, err)
	if ae.Code != domain.CodeValidation {
		t.Errorf("code: got %q want %q", ae.Code, domain.CodeValidation)
	}
}

func TestOperationCancelHandler_Nonexistent_ReturnsValidationError(t *testing.T) {
	runner := &stubRunner{err: domain.NewValidationError("Operation not found", "operationId 9999 does not exist")}
	cli := startServer(t, handler.Map{"operation.cancel": handlers.NewOperationCancelHandler(runner)})

	err := cli.CallResult(context.Background(), "operation.cancel", map[string]int64{"operationId": 9999}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	ae := extractAppError(t, err)
	if ae.Code != domain.CodeValidation {
		t.Errorf("code: got %q want validation_error", ae.Code)
	}
}

func TestOperationCancelHandler_Acknowledged(t *testing.T) {
	runner := &stubRunner{acked: true}
	cli := startServer(t, handler.Map{"operation.cancel": handlers.NewOperationCancelHandler(runner)})

	var resp struct{ Acknowledged bool `json:"acknowledged"` }
	if err := cli.CallResult(context.Background(), "operation.cancel", map[string]int64{"operationId": 42}, &resp); err != nil {
		t.Fatalf("operation.cancel: %v", err)
	}
	if !resp.Acknowledged {
		t.Error("expected acknowledged=true")
	}
}

func TestHostScanHandler_ReturnsOperationID(t *testing.T) {
	svc := &stubHostScan{opID: 55}
	cli := startServer(t, handler.Map{"host.scan": handlers.NewHostScanHandler(svc)})

	var resp struct{ OperationID int64 `json:"operationId"` }
	if err := cli.CallResult(context.Background(), "host.scan", map[string]int64{"hostId": 1}, &resp); err != nil {
		t.Fatalf("host.scan: %v", err)
	}
	if resp.OperationID != 55 {
		t.Errorf("operationId: got %d want 55", resp.OperationID)
	}
}

func TestHostScanHandler_ConflictError_MapsToJRPCError(t *testing.T) {
	svc := &stubHostScan{err: domain.NewConflictError("scan already running", "target locked")}
	cli := startServer(t, handler.Map{"host.scan": handlers.NewHostScanHandler(svc)})

	err := cli.CallResult(context.Background(), "host.scan", map[string]int64{"hostId": 1}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	ae := extractAppError(t, err)
	if ae.Code != domain.CodeConflict {
		t.Errorf("code: got %q want conflict_error", ae.Code)
	}
}
