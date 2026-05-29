package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/services"
)

type updateCheckSvc interface {
	RunUpdateCheck(ctx context.Context) (services.RunResult, error)
}

type updateCheckRunResponse struct {
	Status  string                          `json:"status"`
	Plugins []domain.UpdateCheckPluginResult `json:"plugins"`
}

func NewUpdateCheckRunHandler(svc updateCheckSvc) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		result, err := svc.RunUpdateCheck(ctx)
		if err != nil {
			return nil, wrapError(err)
		}
		plugins := result.Plugins
		if plugins == nil {
			plugins = []domain.UpdateCheckPluginResult{}
		}
		return updateCheckRunResponse{
			Status:  result.Status,
			Plugins: plugins,
		}, nil
	})
}
