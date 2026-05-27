package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type providerPluginRemoveOverrideSvc interface {
	RemoveOverride(ctx context.Context, providerKey, pluginName, marketplaceName, layer string, projectID int64) (int64, error)
}

type providerPluginRemoveOverrideRequest struct {
	ProviderKey     string `json:"providerKey"`
	PluginName      string `json:"pluginName"`
	MarketplaceName string `json:"marketplaceName"`
	Layer           string `json:"layer"`
	ProjectID       int64  `json:"projectId"`
}

type providerPluginRemoveOverrideResponse struct {
	OperationID int64 `json:"operationId"`
}

func NewProviderPluginRemoveOverrideHandler(svc providerPluginRemoveOverrideSvc) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p providerPluginRemoveOverrideRequest
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
		if p.Layer != "project" {
			return nil, wrapError(domain.NewValidationError(
				"Invalid layer",
				"layer must be project",
			))
		}
		if p.ProjectID == 0 {
			return nil, wrapError(domain.NewValidationError(
				"projectId is required for project-layer removes",
				"projectId must be non-zero when layer=project",
			))
		}

		opID, err := svc.RemoveOverride(ctx, p.ProviderKey, p.PluginName, p.MarketplaceName, p.Layer, p.ProjectID)
		if err != nil {
			return nil, wrapError(err)
		}
		return providerPluginRemoveOverrideResponse{OperationID: opID}, nil
	})
}
