package handlers

import "testing"

func TestContract_ProjectAdd_Response(t *testing.T) {
	schema := loadSchema(t, "methods/project.add.json")
	resp := projectAddResponse{
		ProjectID: 1,
		Name:      "myproject",
		Path:      "/home/user/myproject",
		Status:    "active",
	}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_ProjectList_Response(t *testing.T) {
	schema := loadSchema(t, "methods/project.list.json")
	ts := "2025-05-25T00:00:00Z"
	resp := projectListResponse{
		Projects: []projectListItem{
			{
				ID:     1,
				Name:   "alpha",
				Path:   "/home/user/alpha",
				Status: "active",
				Providers: []projectListProviderSummary{
					{
						Key:             "generic_agents",
						DisplayName:     "Shared Agent Skills",
						ProviderStatus:  "supported",
						DetectionStatus: "detected",
					},
				},
				SkillCount:    2,
				WarningCount:  0,
				LastScannedAt: &ts,
			},
		},
	}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_ProjectList_EmptyResponse(t *testing.T) {
	schema := loadSchema(t, "methods/project.list.json")
	resp := projectListResponse{
		Projects: []projectListItem{},
	}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_ProjectGet_Response(t *testing.T) {
	schema := loadSchema(t, "methods/project.get.json")
	resp := projectGetResponse{
		Project: projectGetProject{
			ID:            1,
			Name:          "alpha",
			Path:          "/home/user/alpha",
			Status:        "active",
			LastScannedAt: nil,
		},
		Providers: []projectGetProvider{
			{
				ProjectProviderID: 10,
				ProviderKey:       "generic_agents",
				DisplayName:       "Shared Agent Skills",
				ProviderStatus:    "supported",
				DetectionStatus:   "detected",
				DetectedPath:      nil,
				SkillsPath:        nil,
				EntryCount:        0,
			},
		},
		Entries:  []projectGetEntry{},
		Warnings: []projectGetWarning{},
	}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_ProjectGet_WithEntry(t *testing.T) {
	schema := loadSchema(t, "methods/project.get.json")
	resp := projectGetResponse{
		Project: projectGetProject{
			ID:            1,
			Name:          "alpha",
			Path:          "/home/user/alpha",
			Status:        "active",
			LastScannedAt: nil,
		},
		Providers: []projectGetProvider{
			{
				ProjectProviderID: 10,
				ProviderKey:       "generic_agents",
				DisplayName:       "Shared Agent Skills",
				ProviderStatus:    "supported",
				DetectionStatus:   "detected",
				DetectedPath:      nil,
				SkillsPath:        nil,
				EntryCount:        1,
			},
		},
		Entries: []projectGetEntry{
			{
				ID:                1,
				ProjectProviderID: 10,
				ProviderKey:       "generic_agents",
				Name:              "my-skill",
				Mode:              "symlink",
				Status:            "current",
				ProjectSkillPath:  "/home/user/alpha/.agents/skills/my-skill",
				SymlinkTargetPath: nil,
				SkillID:           nil,
			},
		},
		Warnings: []projectGetWarning{},
	}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_ProjectScan_Response(t *testing.T) {
	schema := loadSchema(t, "methods/project.scan.json")
	resp := projectScanResponse{OperationID: 1}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_ProjectRemove_Response(t *testing.T) {
	schema := loadSchema(t, "methods/project.remove.json")
	resp := projectRemoveResponse{Removed: true}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_InstallSkill_Response(t *testing.T) {
	schema := loadSchema(t, "methods/install.skill.json")
	resp := installSkillResponse{OperationID: 42}
	validateAgainstSchema(t, schema, resp)
}

func TestContract_RemoveSkill_Response(t *testing.T) {
	schema := loadSchema(t, "methods/remove.skill.json")
	resp := removeSkillResponse{OperationID: 51}
	validateAgainstSchema(t, schema, resp)
}
