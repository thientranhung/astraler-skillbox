package handlers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/rpc/handlers"
)

type stubProviderEnablementSvc struct {
	setErr error
}

func (s *stubProviderEnablementSvc) SetEnabled(_ context.Context, _ string, _ bool) error {
	return s.setErr
}

func TestProviderSetEnabledHandler_EnableSuccess(t *testing.T) {
	svc := &stubProviderEnablementSvc{}
	cli := startServer(t, handler.Map{"provider.setEnabled": handlers.NewProviderSetEnabledHandler(svc)})

	var resp struct{ Updated bool `json:"updated"` }
	err := cli.CallResult(context.Background(), "provider.setEnabled", map[string]interface{}{
		"providerKey": "claude",
		"enabled":     true,
	}, &resp)
	if err != nil {
		t.Fatalf("provider.setEnabled: %v", err)
	}
	if !resp.Updated {
		t.Error("expected updated=true")
	}
}

func TestProviderSetEnabledHandler_DisableSuccess(t *testing.T) {
	svc := &stubProviderEnablementSvc{}
	cli := startServer(t, handler.Map{"provider.setEnabled": handlers.NewProviderSetEnabledHandler(svc)})

	var resp struct{ Updated bool `json:"updated"` }
	err := cli.CallResult(context.Background(), "provider.setEnabled", map[string]interface{}{
		"providerKey": "generic_agents",
		"enabled":     false,
	}, &resp)
	if err != nil {
		t.Fatalf("provider.setEnabled: %v", err)
	}
	if !resp.Updated {
		t.Error("expected updated=true")
	}
}

func TestProviderSetEnabledHandler_UnknownProvider(t *testing.T) {
	svc := &stubProviderEnablementSvc{setErr: domain.NewValidationError("Unknown provider", "key not found")}
	cli := startServer(t, handler.Map{"provider.setEnabled": handlers.NewProviderSetEnabledHandler(svc)})

	err := cli.CallResult(context.Background(), "provider.setEnabled", map[string]interface{}{
		"providerKey": "no_such",
		"enabled":     true,
	}, nil)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("code: got %q want validation_error", we.ae.Code)
	}
}

func TestProviderSetEnabledHandler_UnsupportedEnable(t *testing.T) {
	svc := &stubProviderEnablementSvc{
		setErr: domain.NewValidationError("Provider cannot be enabled", "status is unsupported"),
	}
	cli := startServer(t, handler.Map{"provider.setEnabled": handlers.NewProviderSetEnabledHandler(svc)})

	err := cli.CallResult(context.Background(), "provider.setEnabled", map[string]interface{}{
		"providerKey": "opencode",
		"enabled":     true,
	}, nil)
	if err == nil {
		t.Fatal("expected error when enabling unsupported provider")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("code: got %q want validation_error", we.ae.Code)
	}
}

func TestProviderSetEnabledHandler_EmptyKeyRejected(t *testing.T) {
	svc := &stubProviderEnablementSvc{
		setErr: domain.NewValidationError("Provider key is required", "providerKey must not be empty"),
	}
	cli := startServer(t, handler.Map{"provider.setEnabled": handlers.NewProviderSetEnabledHandler(svc)})

	err := cli.CallResult(context.Background(), "provider.setEnabled", map[string]interface{}{
		"providerKey": "",
		"enabled":     true,
	}, nil)
	if err == nil {
		t.Fatal("expected error for empty providerKey")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("code: got %q want validation_error", we.ae.Code)
	}
}

func TestProviderSetEnabledHandler_MissingEnabled_ReturnsInvalidParams(t *testing.T) {
	// Regression: {"providerKey":"claude"} with no enabled field must not silently become enabled=false.
	svc := &stubProviderEnablementSvc{}
	cli := startServer(t, handler.Map{"provider.setEnabled": handlers.NewProviderSetEnabledHandler(svc)})

	err := cli.CallResult(context.Background(), "provider.setEnabled", map[string]interface{}{
		"providerKey": "claude",
		// enabled is intentionally absent
	}, nil)
	if err == nil {
		t.Fatal("expected error for missing enabled field")
	}
	var rpcErr *jrpc2.Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *jrpc2.Error, got %T", err)
	}
	if rpcErr.Code != jrpc2.InvalidParams {
		t.Errorf("wire code: got %d want InvalidParams (%d)", rpcErr.Code, jrpc2.InvalidParams)
	}
}

