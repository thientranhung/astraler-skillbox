package app

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
	rpchandlers "github.com/astraler/skillbox/core-go/internal/rpc/handlers"
)

// App holds the registered JSON-RPC method map.
type App struct {
	methods handler.Map
}

// New builds the composition root and registers all RPC handlers.
func New() *App {
	a := &App{
		methods: handler.Map{
			"ping": handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
				return rpchandlers.Ping(), nil
			}),
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
