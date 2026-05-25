package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type installSkillService interface {
	InstallSkills(ctx context.Context, projectID int64, providerKey string, skillIDs []int64) (int64, error)
}

type installSkillRequest struct {
	ProjectID   int64   `json:"projectId"`
	ProviderKey string  `json:"providerKey"`
	SkillIDs    []int64 `json:"skillIds"`
}

type installSkillResponse struct {
	OperationID int64 `json:"operationId"`
}

func NewInstallSkillHandler(svc installSkillService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p installSkillRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, wrapError(domain.NewValidationError("Invalid params", err.Error()))
		}

		opID, err := svc.InstallSkills(ctx, p.ProjectID, p.ProviderKey, p.SkillIDs)
		if err != nil {
			return nil, wrapError(err)
		}
		return installSkillResponse{OperationID: opID}, nil
	})
}
