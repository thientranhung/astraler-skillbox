package handlers

import (
	"context"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type providerPluginListSvc interface {
	List(ctx context.Context) (domain.GlobalPluginView, []domain.ProjectPluginView, error)
}

// -- response types --

type ppListResponse struct {
	Global   ppGlobalView    `json:"global"`
	Projects []ppProjectView `json:"projects"`
}

type ppGlobalView struct {
	ProviderKey       string          `json:"providerKey"`
	UserLayerPath     string          `json:"userLayerPath"`
	UserLayerStatus   *string         `json:"userLayerStatus"`
	LastScannedAt     *string         `json:"lastScannedAt"`
	ScanWarnings      []string        `json:"scanWarnings"`
	Plugins           []ppGlobalEntry `json:"plugins"`
	Marketplaces      []ppMarketplace `json:"marketplaces"`
	ManagedOutOfScope bool            `json:"managedOutOfScope"`
}

type ppGlobalEntry struct {
	PluginName      string `json:"pluginName"`
	MarketplaceName string `json:"marketplaceName"`
	Status          string `json:"status"` // enabled | disabled
}

type ppProjectView struct {
	ProjectID         int64            `json:"projectId"`
	ProviderKey       string           `json:"providerKey"`
	LayerStatuses     []ppLayerStatus  `json:"layerStatuses"`
	Plugins           []ppProjectEntry `json:"plugins"`
	Marketplaces      []ppMarketplace  `json:"marketplaces"`
	ManagedOutOfScope bool             `json:"managedOutOfScope"`
}

type ppLayerStatus struct {
	Layer        string   `json:"layer"`
	ScanStatus   string   `json:"scanStatus"`
	FilePath     string   `json:"filePath"`
	ScannedAt    *string  `json:"lastScannedAt"`
	ScanWarnings []string `json:"scanWarnings"`
}

type ppProjectEntry struct {
	PluginName      string          `json:"pluginName"`
	MarketplaceName string          `json:"marketplaceName"`
	EffectiveStatus string          `json:"effectiveStatus"`
	ProvenanceLayer *string         `json:"provenanceLayer"`
	LayerBreakdown  []ppLayerDetail `json:"layerBreakdown"`
}

type ppLayerDetail struct {
	Layer       string  `json:"layer"`
	ScanStatus  string  `json:"scanStatus"`
	Declaration *string `json:"declaration"`
}

type ppMarketplace struct {
	MarketplaceName string `json:"marketplaceName"`
	SourceType      string `json:"sourceType"`
	SourceSummary   string `json:"sourceSummary"`
}

func NewProviderPluginListHandler(svc providerPluginListSvc) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		global, projects, err := svc.List(ctx)
		if err != nil {
			return nil, wrapError(err)
		}
		return ppListResponse{
			Global:   mapPPGlobalView(global),
			Projects: mapPPProjectViews(projects),
		}, nil
	})
}

func mapPPGlobalView(g domain.GlobalPluginView) ppGlobalView {
	view := ppGlobalView{
		ProviderKey:       g.ProviderKey,
		UserLayerPath:     g.UserLayerPath,
		ScanWarnings:      []string{},
		Plugins:           []ppGlobalEntry{},
		Marketplaces:      []ppMarketplace{},
		ManagedOutOfScope: g.ManagedOutOfScope,
	}
	if g.Scan != nil {
		s := string(g.Scan.ScanStatus)
		view.UserLayerStatus = &s
		t := g.Scan.LastScannedAt.UTC().Format(time.RFC3339)
		view.LastScannedAt = &t
		if len(g.Scan.Warnings) > 0 {
			view.ScanWarnings = g.Scan.Warnings
		}
	}
	for _, e := range g.Plugins {
		view.Plugins = append(view.Plugins, ppGlobalEntry{
			PluginName:      e.PluginName,
			MarketplaceName: e.MarketplaceName,
			Status:          string(e.Declaration),
		})
	}
	for _, m := range g.Marketplaces {
		view.Marketplaces = append(view.Marketplaces, ppMarketplace{
			MarketplaceName: m.MarketplaceName,
			SourceType:      m.SourceType,
			SourceSummary:   m.SourceSummary,
		})
	}
	return view
}

func mapPPProjectViews(projects []domain.ProjectPluginView) []ppProjectView {
	result := make([]ppProjectView, 0, len(projects))
	for _, p := range projects {
		view := ppProjectView{
			ProjectID:         p.ProjectID,
			ProviderKey:       p.ProviderKey,
			LayerStatuses:     []ppLayerStatus{},
			Plugins:           []ppProjectEntry{},
			Marketplaces:      []ppMarketplace{},
			ManagedOutOfScope: p.ManagedOutOfScope,
		}
		for _, sc := range p.LayerScans {
			t := sc.LastScannedAt.UTC().Format(time.RFC3339)
			warnings := sc.Warnings
			if warnings == nil {
				warnings = []string{}
			}
			view.LayerStatuses = append(view.LayerStatuses, ppLayerStatus{
				Layer:        string(sc.SettingsLayer),
				ScanStatus:   string(sc.ScanStatus),
				FilePath:     sc.SettingsFilePath,
				ScannedAt:    &t,
				ScanWarnings: warnings,
			})
		}
		for _, e := range p.Plugins {
			var provLayer *string
			if e.ProvenanceLayer != nil {
				s := string(*e.ProvenanceLayer)
				provLayer = &s
			}
			pe := ppProjectEntry{
				PluginName:      e.PluginName,
				MarketplaceName: e.MarketplaceName,
				EffectiveStatus: string(e.EffectiveStatus),
				ProvenanceLayer: provLayer,
				LayerBreakdown:  []ppLayerDetail{},
			}
			for _, bd := range e.LayerBreakdown {
				d := ppLayerDetail{
					Layer:      string(bd.Layer),
					ScanStatus: string(bd.ScanStatus),
				}
				if bd.Declaration != nil {
					s := string(*bd.Declaration)
					d.Declaration = &s
				}
				pe.LayerBreakdown = append(pe.LayerBreakdown, d)
			}
			view.Plugins = append(view.Plugins, pe)
		}
		for _, m := range p.Marketplaces {
			view.Marketplaces = append(view.Marketplaces, ppMarketplace{
				MarketplaceName: m.MarketplaceName,
				SourceType:      m.SourceType,
				SourceSummary:   m.SourceSummary,
			})
		}
		result = append(result, view)
	}
	return result
}
