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

// -- stubs --

type stubProviderUpdateResetSvc struct {
	updateErr error
	resetVal  bool
	resetErr  error
}

func (s *stubProviderUpdateResetSvc) UpdatePaths(_ context.Context, _, _, _ string, _ []string) error {
	return s.updateErr
}

func (s *stubProviderUpdateResetSvc) ResetPaths(_ context.Context, _, _, _ string) (bool, error) {
	return s.resetVal, s.resetErr
}

// -- updatePaths tests --

func TestProviderUpdatePathsHandler_Success(t *testing.T) {
	svc := &stubProviderUpdateResetSvc{}
	cli := startServer(t, handler.Map{"provider.updatePaths": handlers.NewProviderUpdatePathsHandler(svc)})

	var resp struct{ Updated bool `json:"updated"` }
	err := cli.CallResult(context.Background(), "provider.updatePaths", map[string]interface{}{
		"providerKey": "claude",
		"scope":       "project",
		"purpose":     "detect",
		"paths":       []string{".custom"},
	}, &resp)
	if err != nil {
		t.Fatalf("provider.updatePaths: %v", err)
	}
	if !resp.Updated {
		t.Error("expected updated=true on success")
	}
}

func TestProviderUpdatePathsHandler_MissingKey(t *testing.T) {
	svc := &stubProviderUpdateResetSvc{}
	cli := startServer(t, handler.Map{"provider.updatePaths": handlers.NewProviderUpdatePathsHandler(svc)})

	err := cli.CallResult(context.Background(), "provider.updatePaths", map[string]interface{}{
		"scope": "project", "purpose": "detect", "paths": []string{".custom"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for missing providerKey")
	}
	var rpcErr *jrpc2.Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *jrpc2.Error, got %T", err)
	}
}

func TestProviderUpdatePathsHandler_ServiceValidationError(t *testing.T) {
	svc := &stubProviderUpdateResetSvc{updateErr: domain.NewValidationError("Unknown provider", "key not found")}
	cli := startServer(t, handler.Map{"provider.updatePaths": handlers.NewProviderUpdatePathsHandler(svc)})

	err := cli.CallResult(context.Background(), "provider.updatePaths", map[string]interface{}{
		"providerKey": "no_such",
		"scope":       "project",
		"purpose":     "detect",
		"paths":       []string{".custom"},
	}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("code: got %q want validation_error", we.ae.Code)
	}
}

// -- resetPaths tests --

func TestProviderResetPathsHandler_ExistingOverride(t *testing.T) {
	svc := &stubProviderUpdateResetSvc{resetVal: true}
	cli := startServer(t, handler.Map{"provider.resetPaths": handlers.NewProviderResetPathsHandler(svc)})

	var resp struct{ Reset bool `json:"reset"` }
	err := cli.CallResult(context.Background(), "provider.resetPaths", map[string]interface{}{
		"providerKey": "claude",
		"scope":       "project",
		"purpose":     "detect",
	}, &resp)
	if err != nil {
		t.Fatalf("provider.resetPaths: %v", err)
	}
	if !resp.Reset {
		t.Error("expected reset=true")
	}
}

func TestProviderResetPathsHandler_NoOverride(t *testing.T) {
	svc := &stubProviderUpdateResetSvc{resetVal: false}
	cli := startServer(t, handler.Map{"provider.resetPaths": handlers.NewProviderResetPathsHandler(svc)})

	var resp struct{ Reset bool `json:"reset"` }
	err := cli.CallResult(context.Background(), "provider.resetPaths", map[string]interface{}{
		"providerKey": "claude",
		"scope":       "project",
		"purpose":     "detect",
	}, &resp)
	if err != nil {
		t.Fatalf("provider.resetPaths: %v", err)
	}
	if resp.Reset {
		t.Error("expected reset=false when no override existed")
	}
}

func TestProviderResetPathsHandler_UnknownProvider(t *testing.T) {
	svc := &stubProviderUpdateResetSvc{resetErr: domain.NewValidationError("Unknown provider", "key not found")}
	cli := startServer(t, handler.Map{"provider.resetPaths": handlers.NewProviderResetPathsHandler(svc)})

	err := cli.CallResult(context.Background(), "provider.resetPaths", map[string]interface{}{
		"providerKey": "no_such",
		"scope":       "project",
		"purpose":     "detect",
	}, nil)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	we := extractWireError(t, err, jrpc2.Code(1001))
	if we.ae.Code != domain.CodeValidation {
		t.Errorf("code: got %q want validation_error", we.ae.Code)
	}
}

func TestProviderResetPathsHandler_MissingProviderKey(t *testing.T) {
	svc := &stubProviderUpdateResetSvc{}
	cli := startServer(t, handler.Map{"provider.resetPaths": handlers.NewProviderResetPathsHandler(svc)})

	err := cli.CallResult(context.Background(), "provider.resetPaths", map[string]interface{}{
		"scope": "project", "purpose": "detect",
	}, nil)
	if err == nil {
		t.Fatal("expected error for missing providerKey")
	}
	var rpcErr *jrpc2.Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *jrpc2.Error, got %T", err)
	}
}
