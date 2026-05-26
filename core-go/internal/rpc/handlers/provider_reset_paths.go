package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
)

type providerResetPathsRequest struct {
	ProviderKey string `json:"providerKey"`
	Scope       string `json:"scope"`
	Purpose     string `json:"purpose"`
}

type providerResetPathsResponse struct {
	Reset bool `json:"reset"`
}

func NewProviderResetPathsHandler(svc providerPathsService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p providerResetPathsRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, err
		}
		if p.ProviderKey == "" {
			return nil, wrapError(newValidationError("providerKey is required"))
		}
		reset, err := svc.ResetPaths(ctx, p.ProviderKey, p.Scope, p.Purpose)
		if err != nil {
			return nil, wrapError(err)
		}
		return providerResetPathsResponse{Reset: reset}, nil
	})
}
