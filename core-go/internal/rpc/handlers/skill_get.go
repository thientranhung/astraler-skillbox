package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/services"
)

type skillGetService interface {
	GetSkillDetail(ctx context.Context, skillID int64) (*services.SkillDetailView, error)
}

type skillGetRequest struct {
	SkillID int64 `json:"skillId"`
}

type skillGetSkill struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	RelativePath  string  `json:"relativePath"`
	AbsolutePath  string  `json:"absolutePath"`
	Status        string  `json:"status"`
	SourceLabel   *string `json:"sourceLabel"`
	HostPath      string  `json:"hostPath"`
	LastScannedAt *string `json:"lastScannedAt"`
}

type skillGetProjectInstall struct {
	ProjectID           int64  `json:"projectId"`
	ProjectName         string `json:"projectName"`
	ProjectProviderID   int64  `json:"projectProviderId"`
	ProviderKey         string `json:"providerKey"`
	ProviderDisplayName string `json:"providerDisplayName"`
	Mode                string `json:"mode"`
	Status              string `json:"status"`
	ProjectSkillPath    string `json:"projectSkillPath"`
}

type skillGetResponse struct {
	Skill    skillGetSkill            `json:"skill"`
	Projects []skillGetProjectInstall `json:"projects"`
}

func NewSkillGetHandler(svc skillGetService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p skillGetRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, wrapError(domain.NewValidationError("Invalid params", err.Error()))
		}

		if p.SkillID <= 0 {
			return nil, wrapError(domain.NewValidationError("Skill not found", "skillId must be > 0"))
		}

		view, err := svc.GetSkillDetail(ctx, p.SkillID)
		if err != nil {
			return nil, wrapError(err)
		}

		resp := skillGetResponse{
			Skill: skillGetSkill{
				ID:            view.Skill.ID,
				Name:          view.Skill.Name,
				RelativePath:  view.Skill.RelativePath,
				AbsolutePath:  view.Skill.AbsolutePath,
				Status:        string(view.Skill.Status),
				SourceLabel:   view.Skill.SourceLabel,
				HostPath:      view.Skill.HostPath,
				LastScannedAt: view.Skill.LastScannedAt,
			},
			Projects: make([]skillGetProjectInstall, 0, len(view.Projects)),
		}

		for _, u := range view.Projects {
			resp.Projects = append(resp.Projects, skillGetProjectInstall{
				ProjectID:           u.ProjectID,
				ProjectName:         u.ProjectName,
				ProjectProviderID:   u.ProjectProviderID,
				ProviderKey:         u.ProviderKey,
				ProviderDisplayName: u.ProviderDisplayName,
				Mode:                u.Mode,
				Status:              u.Status,
				ProjectSkillPath:    u.ProjectSkillPath,
			})
		}

		return resp, nil
	})
}
