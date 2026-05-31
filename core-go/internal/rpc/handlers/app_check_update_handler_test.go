package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/creachadair/jrpc2/handler"

	rpchandlers "github.com/astraler/skillbox/core-go/internal/rpc/handlers"
	"github.com/astraler/skillbox/core-go/internal/services"
)

type stubAppCheckUpdateSvc struct {
	result services.AppCheckUpdateResult
	err    error
}

func (s *stubAppCheckUpdateSvc) CheckAppUpdate(_ context.Context, currentVersion string) (services.AppCheckUpdateResult, error) {
	s.result.CurrentVersion = currentVersion
	return s.result, s.err
}

func ptrStr(s string) *string { return &s }

func TestAppCheckUpdateHandler_UpdateAvailable(t *testing.T) {
	latestVer := "1.2.3"
	releaseURL := "https://github.com/thientranhung/astraler-skillbox/releases/tag/v1.2.3"

	svc := &stubAppCheckUpdateSvc{
		result: services.AppCheckUpdateResult{
			LatestVersion:   &latestVer,
			UpdateAvailable: true,
			ReleaseURL:      &releaseURL,
		},
	}
	h := rpchandlers.NewAppCheckUpdateHandler(svc, "0.1.0")
	cli := startServer(t, handler.Map{"app.checkUpdate": h})

	var raw json.RawMessage
	if err := cli.CallResult(context.Background(), "app.checkUpdate", nil, &raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp struct {
		CurrentVersion  string  `json:"currentVersion"`
		LatestVersion   *string `json:"latestVersion"`
		UpdateAvailable bool    `json:"updateAvailable"`
		ReleaseURL      *string `json:"releaseUrl"`
		Error           *string `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.CurrentVersion != "0.1.0" {
		t.Errorf("currentVersion: got %q, want %q", resp.CurrentVersion, "0.1.0")
	}
	if resp.LatestVersion == nil || *resp.LatestVersion != "1.2.3" {
		t.Errorf("latestVersion: got %v", resp.LatestVersion)
	}
	if !resp.UpdateAvailable {
		t.Error("expected updateAvailable=true")
	}
	if resp.Error != nil {
		t.Errorf("expected error=null, got %q", *resp.Error)
	}
}

func TestAppCheckUpdateHandler_ErrorSurfaced(t *testing.T) {
	// app.checkUpdate is always-on (ADR-0002): failures surface in the error
	// field, not as RPC errors. "network_error" is a valid runtime error path.
	errStr := "network_error"
	svc := &stubAppCheckUpdateSvc{
		result: services.AppCheckUpdateResult{
			Error: &errStr,
		},
	}
	h := rpchandlers.NewAppCheckUpdateHandler(svc, "0.1.0")
	cli := startServer(t, handler.Map{"app.checkUpdate": h})

	var raw json.RawMessage
	if err := cli.CallResult(context.Background(), "app.checkUpdate", nil, &raw); err != nil {
		t.Fatalf("unexpected RPC error: %v", err)
	}

	var resp struct {
		Error           *string `json:"error"`
		UpdateAvailable bool    `json:"updateAvailable"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error == nil || *resp.Error != "network_error" {
		t.Errorf("expected error=network_error, got %v", resp.Error)
	}
	if resp.UpdateAvailable {
		t.Error("expected updateAvailable=false on error")
	}
}
