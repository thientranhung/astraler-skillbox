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
// user data tables and reset settings to defaults. On success the renderer clears
// its query cache and navigates to /setup; no process restart occurs.
func NewAppResetAllHandler(resetFn func() error) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		if err := resetFn(); err != nil {
			return nil, domain.NewFilesystemError("Could not reset application data", err.Error())
		}
		return appResetAllResponse{Restarting: true}, nil
	})
}
