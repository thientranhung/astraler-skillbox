package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type projectScanService interface {
	ScanProject(ctx context.Context, projectID int64) (int64, error)
}

type projectScanRequest struct {
	ProjectID int64 `json:"projectId"`
}

type projectScanResponse struct {
	OperationID int64 `json:"operationId"`
}

func NewProjectScanHandler(svc projectScanService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p projectScanRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, wrapError(domain.NewValidationError("Invalid params", err.Error()))
		}

		opID, err := svc.ScanProject(ctx, p.ProjectID)
		if err != nil {
			return nil, wrapError(err)
		}
		return projectScanResponse{OperationID: opID}, nil
	})
}
