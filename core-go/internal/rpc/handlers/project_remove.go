package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/services"
)

type projectRemoveService interface {
	RemoveProject(ctx context.Context, projectID int64) (*services.ProjectRemoveResult, error)
}

type projectRemoveRequest struct {
	ProjectID int64 `json:"projectId"`
}

type projectRemoveResponse struct {
	Removed bool `json:"removed"`
}

func NewProjectRemoveHandler(svc projectRemoveService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p projectRemoveRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, wrapError(domain.NewValidationError("Invalid params", err.Error()))
		}
		if p.ProjectID <= 0 {
			return nil, wrapError(domain.NewValidationError("projectId is required", "projectId must be >= 1"))
		}

		result, err := svc.RemoveProject(ctx, p.ProjectID)
		if err != nil {
			return nil, wrapError(err)
		}

		return projectRemoveResponse{Removed: result.Removed}, nil
	})
}
