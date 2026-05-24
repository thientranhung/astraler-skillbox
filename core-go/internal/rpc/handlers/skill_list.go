package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/services"
)

type skillListService interface {
	List(ctx context.Context, hostID int64) (*services.SkillsLibraryView, error)
}

type skillListRequest struct {
	HostID int64 `json:"hostId"`
}

type skillListSkill struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	RelativePath  string  `json:"relativePath"`
	Status        string  `json:"status"`
	SourceLabel   *string `json:"sourceLabel"`
	LastScannedAt *string `json:"lastScannedAt"`
}

type skillListTotals struct {
	Available     int `json:"available"`
	Missing       int `json:"missing"`
	Unreadable    int `json:"unreadable"`
	LocalModified int `json:"local_modified"`
	Unknown       int `json:"unknown"`
}

type skillListWarning struct {
	Code     string  `json:"code"`
	Message  string  `json:"message"`
	ScopeRef *string `json:"scopeRef"`
}

type skillListResponse struct {
	HostPath   string             `json:"hostPath"`
	Skills     []skillListSkill   `json:"skills"`
	Totals     skillListTotals    `json:"totals"`
	LastScanAt *string            `json:"lastScanAt"`
	Warnings   []skillListWarning `json:"warnings"`
}

func NewSkillListHandler(svc skillListService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p skillListRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, domain.NewValidationError("Invalid params", err.Error())
		}

		view, err := svc.List(ctx, p.HostID)
		if err != nil {
			return nil, err
		}

		resp := skillListResponse{
			HostPath:   view.HostPath,
			LastScanAt: view.LastScanAt,
			Totals: skillListTotals{
				Available:     view.Totals.Available,
				Missing:       view.Totals.Missing,
				Unreadable:    view.Totals.Unreadable,
				LocalModified: view.Totals.LocalModified,
				Unknown:       view.Totals.Unknown,
			},
		}

		for _, s := range view.Skills {
			resp.Skills = append(resp.Skills, skillListSkill{
				ID:            s.ID,
				Name:          s.Name,
				RelativePath:  s.RelativePath,
				Status:        string(s.Status),
				SourceLabel:   s.SourceLabel,
				LastScannedAt: s.LastScannedAt,
			})
		}
		if resp.Skills == nil {
			resp.Skills = []skillListSkill{}
		}

		for _, w := range view.Warnings {
			resp.Warnings = append(resp.Warnings, skillListWarning{
				Code:     w.Code,
				Message:  w.Message,
				ScopeRef: w.ScopeRef,
			})
		}
		if resp.Warnings == nil {
			resp.Warnings = []skillListWarning{}
		}

		return resp, nil
	})
}
