package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/services"
)

type hostChooseService interface {
	ChooseHost(ctx context.Context, path string) (*services.ChooseHostResult, error)
}

type hostChooseRequest struct {
	Path string `json:"path"`
}

type hostChooseResponse struct {
	HostID      int64  `json:"hostId"`
	Path        string `json:"path"`
	SkillsPath  string `json:"skillsPath"`
	Initialized bool   `json:"initialized"`
	Status      string `json:"status"`
}

func NewHostChooseHandler(svc hostChooseService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p hostChooseRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, domain.NewValidationError("Invalid params", err.Error())
		}
		if p.Path == "" {
			return nil, domain.NewValidationError("path is required", "path field missing")
		}

		result, err := svc.ChooseHost(ctx, p.Path)
		if err != nil {
			return nil, err
		}

		return hostChooseResponse{
			HostID:      result.HostID,
			Path:        result.Path,
			SkillsPath:  result.SkillsPath,
			Initialized: result.Initialized,
			Status:      string(result.Status),
		}, nil
	})
}
