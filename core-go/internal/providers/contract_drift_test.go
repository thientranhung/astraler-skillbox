package providers_test

import (
	"fmt"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
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

// uniformFS is a minimal FsReader that returns the same PathInfo for every path.
// ListSkillEntries always returns an empty list.
type uniformFS struct {
	pi filesystem.PathInfo
}

func (f *uniformFS) PathInfo(_ string) (filesystem.PathInfo, error) { return f.pi, nil }
func (f *uniformFS) ListSkillEntries(_ string) ([]filesystem.ProjectEntry, error) {
	return nil, nil
}

var fixtureFS = map[string]*uniformFS{
	"missing":  {pi: filesystem.PathInfo{Exists: false}},
	"detected": {pi: filesystem.PathInfo{Exists: true, IsDir: true, Readable: true}},
	"invalid":  {pi: filesystem.PathInfo{Exists: true, IsDir: false, Readable: true}},
}

// TestContractDrift_AdapterStatusesAreContractAllowed invokes every adapter in
// NewDefaultRegistry() against missing/detected/invalid filesystem fixtures and
// asserts that each returned DetectionStatus is present in the contract enum.
// Adding a new adapter to NewDefaultRegistry() automatically extends coverage.
func TestContractDrift_AdapterStatusesAreContractAllowed(t *testing.T) {
	reg := providers.NewDefaultRegistry()

	for _, adapter := range reg.All() {
		for fixtureName, fs := range fixtureFS {
			name := fmt.Sprintf("%s/%s", adapter.Key(), fixtureName)
			result, err := adapter.Detect("/project", adapter.DefaultProjectPaths(), fs)
			if err != nil {
				t.Errorf("%s: Detect returned error: %v", name, err)
				continue
			}
			if !contractAllowedStatuses[result.DetectionStatus] {
				t.Errorf("%s: DetectionStatus %q is not in the contract enum", name, result.DetectionStatus)
			}
		}
	}
}

// TestContractDrift_RegisteredAdapterKeys verifies that each registered adapter key
// has a corresponding exported key constant. Uses NewDefaultRegistry so this test
// and main.go always reflect the same adapter set.
func TestContractDrift_RegisteredAdapterKeys(t *testing.T) {
	reg := providers.NewDefaultRegistry()
	wantKeys := map[string]bool{
		providers.GenericAgentsKey:  true,
		providers.ClaudeKey:         true,
		providers.CodexKey:          true,
		providers.AntigravityCLIKey: true,
	}
	for key := range wantKeys {
		if _, ok := reg.Get(key); !ok {
			t.Errorf("adapter key %q not found in registry", key)
		}
	}
	if len(reg.All()) != len(wantKeys) {
		t.Errorf("registry has %d adapters, want %d", len(reg.All()), len(wantKeys))
	}
}
