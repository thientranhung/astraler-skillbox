package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/services"
)

type projectAddService interface {
	AddProject(ctx context.Context, path string) (*services.AddProjectResult, error)
}

type projectAddRequest struct {
	Path string `json:"path"`
}

type projectAddResponse struct {
	ProjectID int64  `json:"projectId"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Status    string `json:"status"`
}

func NewProjectAddHandler(svc projectAddService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p projectAddRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, wrapError(domain.NewValidationError("Invalid params", err.Error()))
		}
		if p.Path == "" {
			return nil, wrapError(domain.NewValidationError("path is required", "path field missing"))
		}

		result, err := svc.AddProject(ctx, p.Path)
		if err != nil {
			return nil, wrapError(err)
		}

		return projectAddResponse{
			ProjectID: result.ProjectID,
			Name:      result.Name,
			Path:      result.Path,
			Status:    string(result.Status),
		}, nil
	})
}
