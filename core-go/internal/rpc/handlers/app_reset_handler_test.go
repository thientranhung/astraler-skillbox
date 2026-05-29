package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	rpchandlers "github.com/astraler/skillbox/core-go/internal/rpc/handlers"
)

func TestAppResetAllHandler_Success(t *testing.T) {
	called := false
	h := rpchandlers.NewAppResetAllHandler(func() error {
		called = true
		return nil
	})

	cli := startServer(t, handler.Map{"app.resetAll": h})

	var result json.RawMessage
	if err := cli.CallResult(context.Background(), "app.resetAll", nil, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("resetFn was not called")
	}

	var resp struct {
		Restarting bool `json:"restarting"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.Restarting {
		t.Error("expected restarting=true")
	}
}

func TestAppResetAllHandler_ResetError(t *testing.T) {
	h := rpchandlers.NewAppResetAllHandler(func() error {
		return errors.New("disk full")
	})

	cli := startServer(t, handler.Map{"app.resetAll": h})

	var result json.RawMessage
	err := cli.CallResult(context.Background(), "app.resetAll", nil, &result)
	if err == nil {
		t.Fatal("expected error from failing resetFn, got nil")
	}

	var rpcErr *jrpc2.Error
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *jrpc2.Error, got %T: %v", err, err)
	}
}
