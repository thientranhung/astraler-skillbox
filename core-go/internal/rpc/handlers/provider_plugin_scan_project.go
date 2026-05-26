package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
)

type providerPluginScanProjectSvc interface {
	ScanProject(ctx context.Context, projectID int64) (int64, error)
}

type providerPluginScanProjectRequest struct {
	ProjectID int64 `json:"projectId"`
}

type providerPluginScanProjectResponse struct {
	OperationID int64 `json:"operationId"`
}

func NewProviderPluginScanProjectHandler(svc providerPluginScanProjectSvc) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p providerPluginScanProjectRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, jrpc2.Errorf(jrpc2.InvalidParams, "invalid params: %v", err)
		}
		if p.ProjectID <= 0 {
			return nil, jrpc2.Errorf(jrpc2.InvalidParams, "projectId must be a positive integer")
		}
		opID, err := svc.ScanProject(ctx, p.ProjectID)
		if err != nil {
			return nil, wrapError(err)
		}
		return providerPluginScanProjectResponse{OperationID: opID}, nil
	})
}
