package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type providerPluginSetEnabledSvc interface {
	SetPluginEnabled(ctx context.Context, providerKey, pluginName, marketplaceName string, enabled bool) (int64, error)
}

type providerPluginSetEnabledRequest struct {
	ProviderKey     string `json:"providerKey"`
	PluginName      string `json:"pluginName"`
	MarketplaceName string `json:"marketplaceName"`
	Layer           string `json:"layer"`
	Enabled         bool   `json:"enabled"`
}

type providerPluginSetEnabledResponse struct {
	OperationID int64 `json:"operationId"`
}

func NewProviderPluginSetEnabledHandler(svc providerPluginSetEnabledSvc) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p providerPluginSetEnabledRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, wrapError(domain.NewValidationError("Invalid params", err.Error()))
		}
		if p.ProviderKey == "" {
			return nil, wrapError(domain.NewValidationError("providerKey is required", "providerKey field is empty"))
		}
		if p.PluginName == "" {
			return nil, wrapError(domain.NewValidationError("pluginName is required", "pluginName field is empty"))
		}
		if p.MarketplaceName == "" {
			return nil, wrapError(domain.NewValidationError("marketplaceName is required", "marketplaceName field is empty"))
		}
		if p.Layer != "user" {
			return nil, wrapError(domain.NewValidationError(
				"Only user-layer plugin writes are supported",
				"only layer=user is supported in this version; project and local layer writes are not yet available",
			))
		}

		opID, err := svc.SetPluginEnabled(ctx, p.ProviderKey, p.PluginName, p.MarketplaceName, p.Enabled)
		if err != nil {
			return nil, wrapError(err)
		}
		return providerPluginSetEnabledResponse{OperationID: opID}, nil
	})
}
