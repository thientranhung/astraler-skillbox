package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/services"
)

type appCheckUpdateSvc interface {
	CheckAppUpdate(ctx context.Context, currentVersion string) (services.AppCheckUpdateResult, error)
}

type appCheckUpdateResponse struct {
	CurrentVersion  string  `json:"currentVersion"`
	LatestVersion   *string `json:"latestVersion"`
	UpdateAvailable bool    `json:"updateAvailable"`
	ReleaseURL      *string `json:"releaseUrl"`
	Error           *string `json:"error"`
}

func NewAppCheckUpdateHandler(svc appCheckUpdateSvc, currentVersion string) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		result, err := svc.CheckAppUpdate(ctx, currentVersion)
		if err != nil {
			return nil, wrapError(err)
		}
		return appCheckUpdateResponse{
			CurrentVersion:  result.CurrentVersion,
			LatestVersion:   result.LatestVersion,
			UpdateAvailable: result.UpdateAvailable,
			ReleaseURL:      result.ReleaseURL,
			Error:           result.Error,
		}, nil
	})
}
