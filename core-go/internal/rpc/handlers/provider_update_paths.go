package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
)

type providerPathsService interface {
	UpdatePaths(ctx context.Context, providerKey, scope, purpose string, paths []string) error
	ResetPaths(ctx context.Context, providerKey, scope, purpose string) (bool, error)
}

type providerUpdatePathsRequest struct {
	ProviderKey string   `json:"providerKey"`
	Scope       string   `json:"scope"`
	Purpose     string   `json:"purpose"`
	Paths       []string `json:"paths"`
}

type providerUpdatePathsResponse struct {
	Updated bool `json:"updated"`
}

func NewProviderUpdatePathsHandler(svc providerPathsService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p providerUpdatePathsRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, err
		}
		if p.ProviderKey == "" {
			return nil, wrapError(newValidationError("providerKey is required"))
		}
		if err := svc.UpdatePaths(ctx, p.ProviderKey, p.Scope, p.Purpose, p.Paths); err != nil {
			return nil, wrapError(err)
		}
		return providerUpdatePathsResponse{Updated: true}, nil
	})
}
