package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/astraler/skillbox/core-go/internal/app"
	corerpc "github.com/astraler/skillbox/core-go/internal/rpc"
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

	a := app.New()
	srv := corerpc.New(a.Assigner(), os.Stdin, os.Stdout)

	ctx := context.Background()
	if err := srv.Notify(ctx, "server.ready", map[string]interface{}{
		"version":      "0.1.0-m1",
		"pid":          os.Getpid(),
		"capabilities": []string{"ping"},
	}); err != nil {
		slog.Error("failed to send server.ready", "err", err)
		os.Exit(1)
	}

	slog.Info("skillbox-core started", "pid", os.Getpid())

	if err := srv.Wait(); err != nil {
		slog.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}
