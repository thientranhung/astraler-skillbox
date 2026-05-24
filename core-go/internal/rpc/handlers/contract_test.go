package handlers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qri-io/jsonschema"
)

// contractsRoot resolves the shared/api-contracts dir from this package.
// Package lives at: core-go/internal/rpc/handlers/ → 4 levels up to repo root.
func contractsRoot() string {
	dir, _ := os.Getwd()
	// dir = .../astraler-skillbox/core-go/internal/rpc/handlers
	return filepath.Join(dir, "../../../../shared/api-contracts")
}

// resolveRefs walks JSON value and replaces {"$ref":"#/definitions/X"} nodes with
// the inlined definition. qri-io/jsonschema v0.2.1 does not resolve $ref without a
// document URI, so we pre-process the schema before parsing.
func resolveRefs(data json.RawMessage, defs map[string]json.RawMessage) json.RawMessage {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err == nil {
		if refRaw, hasRef := obj["$ref"]; hasRef {
			var ref string
			if json.Unmarshal(refRaw, &ref) == nil && strings.HasPrefix(ref, "#/definitions/") {
				name := strings.TrimPrefix(ref, "#/definitions/")
				if def, ok := defs[name]; ok {
					return resolveRefs(def, defs)
				}
			}
		}
		resolved := make(map[string]json.RawMessage, len(obj))
		for k, v := range obj {
			resolved[k] = resolveRefs(v, defs)
		}
		out, _ := json.Marshal(resolved)
		return out
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err == nil {
		for i, v := range arr {
			arr[i] = resolveRefs(v, defs)
		}
		out, _ := json.Marshal(arr)
		return out
	}
	return data
}

func loadSchema(t *testing.T, relPath string) *jsonschema.Schema {
	t.Helper()
	full := filepath.Join(contractsRoot(), relPath)
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("load schema %s: %v", relPath, err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse schema doc %s: %v", relPath, err)
	}
	var defs map[string]json.RawMessage
	if raw, ok := doc["definitions"]; ok {
		_ = json.Unmarshal(raw, &defs)
	}
	resolved := resolveRefs(data, defs)

	rs := &jsonschema.Schema{}
	if err := json.Unmarshal(resolved, rs); err != nil {
		t.Fatalf("parse schema %s: %v", relPath, err)
	}
	return rs
}

func validateAgainstSchema(t *testing.T, schema *jsonschema.Schema, value interface{}) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	errs, err := schema.ValidateBytes(context.Background(), data)
	if err != nil {
		t.Fatalf("schema.Validate: %v", err)
	}
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("schema violation: %s", e.Message)
		}
	}
}

// -- mock services for contract tests --

type mockChooseHostSvc struct{}

func (m *mockChooseHostSvc) ChooseHost(_ context.Context, _ string) (*chooseHostResult, error) {
	status := "active"
	return &chooseHostResult{
		HostID:      1,
		Path:        "/tmp/test-host",
		SkillsPath:  "/tmp/test-host/.agents/skills",
		Initialized: true,
		Status:      status,
	}, nil
}

type chooseHostResult = struct {
	HostID      int64
	Path        string
	SkillsPath  string
	Initialized bool
	Status      string
}

func TestContract_HostChoose_Response(t *testing.T) {
	resp := hostChooseResponse{
		HostID:      1,
		Path:        "/tmp/test-host",
		SkillsPath:  "/tmp/test-host/.agents/skills",
		Initialized: true,
		Status:      "active",
	}

	schema := loadSchema(t, "methods/host.choose.json")
	validateAgainstSchema(t, schema, resp)
}

func TestContract_Settings_Response(t *testing.T) {
	schema := loadSchema(t, "methods/settings.get.json")

	resp := settingsGetResponse{
		ActiveSkillHostFolderID: nil,
		DefaultInstallMode:      "symlink",
		DatabaseVersion:         1,
		ActiveHost:              nil,
	}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_SkillList_Response(t *testing.T) {
	schema := loadSchema(t, "methods/skill.list.json")

	resp := skillListResponse{
		HostPath:   "/tmp/host",
		Skills:     []skillListSkill{},
		Totals:     skillListTotals{},
		LastScanAt: nil,
		Warnings:   []skillListWarning{},
	}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_OperationCancel_Response(t *testing.T) {
	schema := loadSchema(t, "methods/operation.cancel.json")
	resp := operationCancelResponse{Acknowledged: true}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_HostScan_Response(t *testing.T) {
	schema := loadSchema(t, "methods/host.scan.json")
	resp := hostScanResponse{OperationID: 1}
	validateAgainstSchema(t, schema, resp)
}
