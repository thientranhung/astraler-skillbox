package handlers

import (
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// TestContract_ProviderList_Response validates the handler output against the JSON schema contract.
func TestContract_ProviderList_Response(t *testing.T) {
	schema := loadSchema(t, "methods/provider.list.json")

	iconKey := "claude"
	resp := providerListResponse{
		Providers: []providerListProvider{
			{
				Key:                "claude",
				DisplayName:        "Claude",
				ProviderType:       "claude",
				IconKey:            &iconKey,
				Status:             "experimental",
				IsAvailable:        true,
				IsEnabled:          true,
				CanToggle:          true,
				CanCreateStructure: false,
				HasGlobalLevel:     true,
				Candidates: []providerListPathCandidate{
					{
						RelativePath:       ".claude",
						Scope:              "project",
						Purpose:            "detect",
						Priority:           10,
						Source:             "builtin",
						VerificationStatus: "assumed",
					},
					{
						RelativePath:       "~/.claude/skills",
						Scope:              "global",
						Purpose:            "skills",
						Priority:           10,
						Source:             "builtin",
						VerificationStatus: "assumed",
					},
				},
			},
		},
	}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_ProviderList_EmptyResponse(t *testing.T) {
	schema := loadSchema(t, "methods/provider.list.json")
	resp := providerListResponse{Providers: []providerListProvider{}}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_ProviderList_NullIconKey(t *testing.T) {
	schema := loadSchema(t, "methods/provider.list.json")
	resp := providerListResponse{
		Providers: []providerListProvider{
			{
				Key:                "generic_agents",
				DisplayName:        "Shared Agent Skills",
				ProviderType:       "generic_agents",
				IconKey:            nil,
				Status:             "supported",
				IsAvailable:        true,
				IsEnabled:          true,
				CanToggle:          true,
				CanCreateStructure: false,
				HasGlobalLevel:     true,
				Candidates:         []providerListPathCandidate{},
			},
		},
	}
	validateAgainstSchema(t, schema, resp)
}

func TestDeriveIsAvailable_SupportedAndExperimental(t *testing.T) {
	cases := []struct {
		status      domain.ProviderStatus
		isAvailable bool
	}{
		{domain.ProviderStatusSupported, true},
		{domain.ProviderStatusExperimental, true},
		{domain.ProviderStatusUnsupported, false},
		{domain.ProviderStatusDisabled, false},
	}
	for _, c := range cases {
		got := deriveIsAvailable(c.status)
		if got != c.isAvailable {
			t.Errorf("deriveIsAvailable(%q): got %v want %v", c.status, got, c.isAvailable)
		}
	}
}
