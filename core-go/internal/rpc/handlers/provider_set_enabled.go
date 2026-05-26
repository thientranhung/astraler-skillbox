package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
)

type providerEnablementService interface {
	SetEnabled(ctx context.Context, providerKey string, enabled bool) error
}

type providerSetEnabledParams struct {
	ProviderKey string `json:"providerKey"`
	Enabled     *bool  `json:"enabled"`
}

type providerSetEnabledResponse struct {
	Updated bool `json:"updated"`
}

func NewProviderSetEnabledHandler(svc providerEnablementService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var params providerSetEnabledParams
		if err := req.UnmarshalParams(&params); err != nil {
			return nil, jrpc2.Errorf(jrpc2.InvalidParams, "invalid params: %v", err)
		}
		if params.Enabled == nil {
			return nil, jrpc2.Errorf(jrpc2.InvalidParams, "invalid params: enabled is required")
		}
		if err := svc.SetEnabled(ctx, params.ProviderKey, *params.Enabled); err != nil {
			return nil, wrapError(err)
		}
		return providerSetEnabledResponse{Updated: true}, nil
	})
}
