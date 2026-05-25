package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type removeSkillService interface {
	RemoveSkill(ctx context.Context, projectID int64, installID int64) (int64, error)
}

type removeSkillRequest struct {
	ProjectID int64 `json:"projectId"`
	InstallID int64 `json:"installId"`
}

type removeSkillResponse struct {
	OperationID int64 `json:"operationId"`
}

func NewRemoveSkillHandler(svc removeSkillService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p removeSkillRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, wrapError(domain.NewValidationError("Invalid params", err.Error()))
		}

		opID, err := svc.RemoveSkill(ctx, p.ProjectID, p.InstallID)
		if err != nil {
			return nil, wrapError(err)
		}
		return removeSkillResponse{OperationID: opID}, nil
	})
}
