package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type appResetAllResponse struct {
	Restarting bool `json:"restarting"`
}

// NewAppResetAllHandler returns a handler for app.resetAll. resetFn must truncate
// user data tables and reset settings to defaults. Electron triggers app.relaunch()
// upon receiving the response; the Go process is killed shortly after.
func NewAppResetAllHandler(resetFn func() error) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		if err := resetFn(); err != nil {
			return nil, domain.NewFilesystemError("Could not reset application data", err.Error())
		}
		return appResetAllResponse{Restarting: true}, nil
	})
}
