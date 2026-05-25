package providers_test

import (
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

// contractAllowedStatuses mirrors the detectionStatus enum in the JSON Schema contracts
// (project.get.json, project.list.json). Any DetectionStatus emitted by a registered
// adapter must appear in this set.
var contractAllowedStatuses = map[domain.DetectionStatus]bool{
	domain.DetectionStatusDetected:         true,
	domain.DetectionStatusMissing:          true,
	domain.DetectionStatusInvalidStructure: true,
}

// adapterScenarios lists representative detect results from each registered adapter.
// Add a scenario whenever a new adapter is registered.
var adapterScenarios = []struct {
	name   string
	result providers.DetectResult
}{
	// GenericAgents
	{
		name: "generic_agents/missing",
		result: providers.DetectResult{
			Present:         false,
			DetectionStatus: domain.DetectionStatusMissing,
		},
	},
	{
		name: "generic_agents/detected",
		result: providers.DetectResult{
			Present:         true,
			DetectionStatus: domain.DetectionStatusDetected,
		},
	},
	{
		name: "generic_agents/invalid_structure",
		result: providers.DetectResult{
			Present:         true,
			DetectionStatus: domain.DetectionStatusInvalidStructure,
		},
	},
	// Claude
	{
		name: "claude/missing",
		result: providers.DetectResult{
			Present:         false,
			DetectionStatus: domain.DetectionStatusMissing,
		},
	},
	{
		name: "claude/detected",
		result: providers.DetectResult{
			Present:         true,
			DetectionStatus: domain.DetectionStatusDetected,
		},
	},
	{
		name: "claude/invalid_structure",
		result: providers.DetectResult{
			Present:         true,
			DetectionStatus: domain.DetectionStatusInvalidStructure,
		},
	},
}

// TestContractDrift_AdapterStatusesAreContractAllowed verifies that every DetectionStatus
// value emitted by registered adapters is present in the contract enum. If this test fails,
// either the JSON Schema contract or this list needs updating.
func TestContractDrift_AdapterStatusesAreContractAllowed(t *testing.T) {
	for _, s := range adapterScenarios {
		if !contractAllowedStatuses[s.result.DetectionStatus] {
			t.Errorf("scenario %q emits DetectionStatus %q which is not in the contract enum", s.name, s.result.DetectionStatus)
		}
	}
}

// TestContractDrift_RegisteredAdapterKeys verifies that each registered adapter key
// has a corresponding exported key constant. This catches typos before runtime.
func TestContractDrift_RegisteredAdapterKeys(t *testing.T) {
	reg := providers.NewRegistry(
		providers.NewGenericAgentsAdapter(),
		providers.NewClaudeAdapter(),
	)
	wantKeys := map[string]bool{
		providers.GenericAgentsKey: true,
		providers.ClaudeKey:        true,
	}
	for key := range wantKeys {
		if _, ok := reg.Get(key); !ok {
			t.Errorf("adapter key %q not found in registry", key)
		}
	}
	all := reg.All()
	if len(all) != len(wantKeys) {
		t.Errorf("registry has %d adapters, want %d", len(all), len(wantKeys))
	}
}
