package handlers

import (
	"context"
	"strconv"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/services"
)

type projectGetService interface {
	GetProject(ctx context.Context, projectID int64) (*services.ProjectDetailView, error)
}

type projectGetRequest struct {
	ProjectID int64 `json:"projectId"`
}

type projectGetProject struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Path          string  `json:"path"`
	Status        string  `json:"status"`
	LastScannedAt *string `json:"lastScannedAt"`
}

type projectGetProvider struct {
	ProjectProviderID int64   `json:"projectProviderId"`
	ProviderKey       string  `json:"providerKey"`
	DisplayName       string  `json:"displayName"`
	ProviderStatus    string  `json:"providerStatus"`
	DetectionStatus   string  `json:"detectionStatus"`
	DetectedPath      *string `json:"detectedPath"`
	SkillsPath        *string `json:"skillsPath"`
	EntryCount        int     `json:"entryCount"`
}

type projectGetEntry struct {
	ID                int64   `json:"id"`
	ProjectProviderID int64   `json:"projectProviderId"`
	ProviderKey       string  `json:"providerKey"`
	Name              string  `json:"name"`
	Mode              string  `json:"mode"`
	Status            string  `json:"status"`
	ProjectSkillPath  string  `json:"projectSkillPath"`
	SymlinkTargetPath *string `json:"symlinkTargetPath"`
	SkillID           *int64  `json:"skillId"`
}

type projectGetWarning struct {
	Code      string  `json:"code"`
	Severity  string  `json:"severity"`
	Message   string  `json:"message"`
	ScopeType string  `json:"scopeType"`
	ScopeRef  *string `json:"scopeRef"`
	ActionKey *string `json:"actionKey"`
}

type projectGetResponse struct {
	Project   projectGetProject    `json:"project"`
	Providers []projectGetProvider `json:"providers"`
	Entries   []projectGetEntry    `json:"entries"`
	Warnings  []projectGetWarning  `json:"warnings"`
}

func NewProjectGetHandler(svc projectGetService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p projectGetRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, wrapError(domain.NewValidationError("Invalid params", err.Error()))
		}
		if p.ProjectID == 0 {
			return nil, wrapError(domain.NewValidationError("projectId is required", "projectId field missing or zero"))
		}

		view, err := svc.GetProject(ctx, p.ProjectID)
		if err != nil {
			return nil, wrapError(err)
		}

		// Build providerKey lookup for entry mapping.
		keyByPPID := make(map[int64]string, len(view.Providers))
		for _, pp := range view.Providers {
			keyByPPID[pp.ProjectProviderID] = pp.ProviderKey
		}

		providers := make([]projectGetProvider, 0, len(view.Providers))
		for _, pp := range view.Providers {
			providers = append(providers, projectGetProvider{
				ProjectProviderID: pp.ProjectProviderID,
				ProviderKey:       pp.ProviderKey,
				DisplayName:       pp.ProviderDisplayName,
				ProviderStatus:    string(pp.ProviderStatus),
				DetectionStatus:   string(pp.DetectionStatus),
				DetectedPath:      pp.DetectedPath,
				SkillsPath:        pp.SkillsPath,
				EntryCount:        pp.EntryCount,
			})
		}

		entries := make([]projectGetEntry, 0, len(view.Entries))
		for _, e := range view.Entries {
			entries = append(entries, projectGetEntry{
				ID:                e.ID,
				ProjectProviderID: e.ProjectProviderID,
				ProviderKey:       keyByPPID[e.ProjectProviderID],
				Name:              e.SkillName,
				Mode:              string(e.InstallMode),
				Status:            string(e.InstallStatus),
				ProjectSkillPath:  e.ProjectSkillPath,
				SymlinkTargetPath: e.SymlinkTargetPath,
				SkillID:           e.SkillID,
			})
		}

		warnings := make([]projectGetWarning, 0, len(view.Warnings))
		for _, w := range view.Warnings {
			var scopeRef *string
			if w.ScopeID != nil {
				ref := string(w.ScopeType) + ":" + strconv.FormatInt(*w.ScopeID, 10)
				scopeRef = &ref
			}
			warnings = append(warnings, projectGetWarning{
				Code:      w.Code,
				Severity:  string(w.Severity),
				Message:   w.Message,
				ScopeType: string(w.ScopeType),
				ScopeRef:  scopeRef,
				ActionKey: w.ActionKey,
			})
		}

		return projectGetResponse{
			Project: projectGetProject{
				ID:            view.Project.ID,
				Name:          view.Project.Name,
				Path:          view.Project.Path,
				Status:        string(view.Project.Status),
				LastScannedAt: formatTimePtr(view.Project.LastScannedAt),
			},
			Providers: providers,
			Entries:   entries,
			Warnings:  warnings,
		}, nil
	})
}
