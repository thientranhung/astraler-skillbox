package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/services"
)

type dashboardService interface {
	Get(ctx context.Context) (*services.DashboardView, error)
}

type dashboardActiveHostResponse struct {
	HostID     int64   `json:"hostId"`
	Path       string  `json:"path"`
	SkillsPath string  `json:"skillsPath"`
	Status     string  `json:"status"`
	LastScanAt *string `json:"lastScanAt"`
}

type dashboardSummaryResponse struct {
	Skills   int `json:"skills"`
	Projects int `json:"projects"`
	Warnings int `json:"warnings"`
}

type dashboardInstallsByModeResponse struct {
	Symlink   int `json:"symlink"`
	RsyncCopy int `json:"rsyncCopy"`
	Direct    int `json:"direct"`
}

type dashboardWarningsBySeverityResponse struct {
	Info     int `json:"info"`
	Warning  int `json:"warning"`
	Error    int `json:"error"`
	Blocking int `json:"blocking"`
}

type dashboardWarningResponse struct {
	Code      string  `json:"code"`
	Message   string  `json:"message"`
	Severity  string  `json:"severity"`
	ScopeType string  `json:"scopeType"`
	ScopeID   *int64  `json:"scopeId"`
	ActionKey *string `json:"actionKey"`
}

type dashboardGetResponse struct {
	ActiveHost         *dashboardActiveHostResponse        `json:"activeHost"`
	Summary            dashboardSummaryResponse            `json:"summary"`
	InstallsByMode     dashboardInstallsByModeResponse     `json:"installsByMode"`
	WarningsBySeverity dashboardWarningsBySeverityResponse `json:"warningsBySeverity"`
	Warnings           []dashboardWarningResponse          `json:"warnings"`
}

func NewDashboardGetHandler(svc dashboardService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		view, err := svc.Get(ctx)
		if err != nil {
			return nil, wrapError(err)
		}

		resp := dashboardGetResponse{
			Summary: dashboardSummaryResponse{
				Skills:   view.Summary.Skills,
				Projects: view.Summary.Projects,
				Warnings: view.Summary.Warnings,
			},
			InstallsByMode: dashboardInstallsByModeResponse{
				Symlink:   view.InstallsByMode.Symlink,
				RsyncCopy: view.InstallsByMode.RsyncCopy,
				Direct:    view.InstallsByMode.Direct,
			},
			WarningsBySeverity: dashboardWarningsBySeverityResponse{
				Info:     view.WarningsBySeverity.Info,
				Warning:  view.WarningsBySeverity.Warning,
				Error:    view.WarningsBySeverity.Error,
				Blocking: view.WarningsBySeverity.Blocking,
			},
			Warnings: make([]dashboardWarningResponse, 0, len(view.Warnings)),
		}

		if view.ActiveHost != nil {
			resp.ActiveHost = &dashboardActiveHostResponse{
				HostID:     view.ActiveHost.HostID,
				Path:       view.ActiveHost.Path,
				SkillsPath: view.ActiveHost.SkillsPath,
				Status:     string(view.ActiveHost.Status),
				LastScanAt: view.ActiveHost.LastScannedAt,
			}
		}

		for _, item := range view.Warnings {
			resp.Warnings = append(resp.Warnings, dashboardWarningResponse{
				Code:      item.Code,
				Message:   item.Message,
				Severity:  string(item.Severity),
				ScopeType: string(item.ScopeType),
				ScopeID:   item.ScopeID,
				ActionKey: item.ActionKey,
			})
		}

		return resp, nil
	})
}
