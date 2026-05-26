package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type globalListService interface {
	ListGlobal(ctx context.Context) ([]domain.GlobalLocationView, error)
}

// Response structs mirror global.list.json definitions exactly.

type globalListWarningResponse struct {
	Code      string  `json:"code"`
	Severity  string  `json:"severity"`
	ScopeType string  `json:"scopeType"`
	ScopeID   *int64  `json:"scopeId"`
	ActionKey *string `json:"actionKey"`
	Message   string  `json:"message"`
}

type globalListEntryResponse struct {
	GlobalInstallID   int64   `json:"globalInstallId"`
	SkillName         string  `json:"skillName"`
	SkillID           *int64  `json:"skillId"`
	Mode              string  `json:"mode"`
	Status            string  `json:"status"`
	GlobalSkillPath   string  `json:"globalSkillPath"`
	SourceSkillPath   *string `json:"sourceSkillPath"`
	SymlinkTargetPath *string `json:"symlinkTargetPath"`
}

type globalListLocationResponse struct {
	GlobalProviderLocationID int64                       `json:"globalProviderLocationId"`
	ProviderKey              string                      `json:"providerKey"`
	ProviderDisplayName      string                      `json:"providerDisplayName"`
	ProviderStatus           string                      `json:"providerStatus"`
	Path                     *string                     `json:"path"`
	SkillsPath               *string                     `json:"skillsPath"`
	Status                   string                      `json:"status"`
	LastScannedAt            *string                     `json:"lastScannedAt"`
	Entries                  []globalListEntryResponse   `json:"entries"`
	Warnings                 []globalListWarningResponse `json:"warnings"`
}

type globalListResponse struct {
	Locations []globalListLocationResponse `json:"locations"`
}

func NewGlobalListHandler(svc globalListService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		locs, err := svc.ListGlobal(ctx)
		if err != nil {
			return nil, wrapError(err)
		}
		return mapGlobalListResponse(locs), nil
	})
}

func mapGlobalListResponse(locs []domain.GlobalLocationView) globalListResponse {
	locations := make([]globalListLocationResponse, 0, len(locs))
	for _, loc := range locs {
		entries := make([]globalListEntryResponse, 0, len(loc.Entries))
		for _, e := range loc.Entries {
			entries = append(entries, globalListEntryResponse{
				GlobalInstallID:   e.GlobalInstallID,
				SkillName:         e.SkillName,
				SkillID:           e.SkillID,
				Mode:              string(e.Mode),
				Status:            string(e.Status),
				GlobalSkillPath:   e.GlobalSkillPath,
				SourceSkillPath:   e.SourceSkillPath,
				SymlinkTargetPath: e.SymlinkTargetPath,
			})
		}

		warnings := make([]globalListWarningResponse, 0, len(loc.Warnings))
		for _, w := range loc.Warnings {
			warnings = append(warnings, globalListWarningResponse{
				Code:      w.Code,
				Severity:  string(w.Severity),
				ScopeType: string(w.ScopeType),
				ScopeID:   w.ScopeID,
				ActionKey: w.ActionKey,
				Message:   w.Message,
			})
		}

		locations = append(locations, globalListLocationResponse{
			GlobalProviderLocationID: loc.GlobalProviderLocationID,
			ProviderKey:              loc.ProviderKey,
			ProviderDisplayName:      loc.ProviderDisplayName,
			ProviderStatus:           loc.ProviderStatus,
			Path:                     loc.Path,
			SkillsPath:               loc.SkillsPath,
			Status:                   string(loc.Status),
			LastScannedAt:            loc.LastScannedAt,
			Entries:                  entries,
			Warnings:                 warnings,
		})
	}
	return globalListResponse{Locations: locations}
}
