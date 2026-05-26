package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

var validScopes = map[string]bool{"project": true, "global": true}
var validPurposes = map[string]bool{"detect": true, "skills": true, "config": true, "commands": true}

// ProviderRegistryService returns the provider registry with override and user-settings support.
type ProviderRegistryService struct {
	repo             ProviderRegistryRepo
	overrideRepo     ProviderOverrideRepo
	userSettingsRepo ProviderUserSettingsRepo
}

func NewProviderRegistryService(repo ProviderRegistryRepo, overrideRepo ProviderOverrideRepo, userSettingsRepo ProviderUserSettingsRepo) *ProviderRegistryService {
	return &ProviderRegistryService{repo: repo, overrideRepo: overrideRepo, userSettingsRepo: userSettingsRepo}
}

func (s *ProviderRegistryService) List(ctx context.Context) ([]domain.ProviderRegistryEntry, error) {
	entries, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load provider registry", err.Error())
	}

	overrides, err := s.overrideRepo.ListAll(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load provider path overrides", err.Error())
	}

	userSettings, err := s.userSettingsRepo.ListAll(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load provider user settings", err.Error())
	}

	// Build a lookup map: providerDefinitionID → user-set enabled value.
	userEnabled := make(map[int64]bool, len(userSettings))
	for _, us := range userSettings {
		userEnabled[us.ProviderDefinitionID] = us.Enabled
	}

	if len(overrides) > 0 {
		entries = mergeOverrides(entries, overrides)
	} else {
		for i := range entries {
			for j := range entries[i].Candidates {
				entries[i].Candidates[j].Source = "builtin"
			}
		}
	}

	// Stamp IsEnabled and CanToggle on each entry.
	// IsEnabled is clamped to false when canToggle is false regardless of any stored user setting,
	// so that a provider that became unsupported after being enabled never reports isEnabled=true.
	for i := range entries {
		canToggle := deriveCanToggle(entries[i].Definition.Status)
		entries[i].CanToggle = canToggle
		if !canToggle {
			entries[i].IsEnabled = false
		} else if v, ok := userEnabled[entries[i].Definition.ID]; ok {
			entries[i].IsEnabled = v
		} else {
			entries[i].IsEnabled = true
		}
	}

	return entries, nil
}

// SetEnabled persists the user's enabled preference for the given provider.
// Returns validation_error if the provider is unknown or if enabled=true is requested
// for a provider that cannot be toggled (unsupported/disabled status).
func (s *ProviderRegistryService) SetEnabled(ctx context.Context, providerKey string, enabled bool) error {
	if providerKey == "" {
		return domain.NewValidationError("Provider key is required", "providerKey must not be empty")
	}

	def, err := s.repo.GetByKey(ctx, providerKey)
	if err != nil {
		return domain.NewDatabaseError("Could not look up provider", err.Error())
	}
	if def == nil {
		return domain.NewValidationError("Unknown provider", fmt.Sprintf("provider key %q not found", providerKey))
	}

	if enabled && !deriveCanToggle(def.Status) {
		return domain.NewValidationError(
			"Provider cannot be enabled",
			fmt.Sprintf("provider %q has status %q and cannot be toggled on", providerKey, def.Status),
		)
	}

	if err := s.userSettingsRepo.Upsert(ctx, def.ID, enabled); err != nil {
		return domain.NewDatabaseError("Could not save provider user setting", err.Error())
	}
	return nil
}

// deriveCanToggle returns true for supported and experimental providers.
func deriveCanToggle(status domain.ProviderStatus) bool {
	return status == domain.ProviderStatusSupported || status == domain.ProviderStatusExperimental
}

// mergeOverrides replaces builtin candidates for each (providerID, scope, purpose)
// slot that has an override. Override candidates carry Source="override".
func mergeOverrides(entries []domain.ProviderRegistryEntry, overrides []domain.ProviderPathOverride) []domain.ProviderRegistryEntry {
	result := make([]domain.ProviderRegistryEntry, len(entries))
	for i, e := range entries {
		// Collect overridden slots for this provider.
		overriddenSlots := map[string][]string{} // "scope:purpose" → paths
		for _, o := range overrides {
			if o.ProviderDefinitionID == e.Definition.ID {
				key := o.Scope + ":" + o.Purpose
				overriddenSlots[key] = o.Paths
			}
		}

		var newCands []domain.ProviderPathCandidate

		// Add override candidates first.
		for slotKey, paths := range overriddenSlots {
			parts := strings.SplitN(slotKey, ":", 2)
			scope, purpose := parts[0], parts[1]
			for _, p := range paths {
				newCands = append(newCands, domain.ProviderPathCandidate{
					ProviderDefinitionID: e.Definition.ID,
					RelativePath:         p,
					Scope:                scope,
					Purpose:              purpose,
					Priority:             10,
					VerificationStatus:   "assumed",
					Source:               "override",
				})
			}
		}

		// Add builtin candidates for non-overridden slots.
		for _, c := range e.Candidates {
			key := c.Scope + ":" + c.Purpose
			if _, overridden := overriddenSlots[key]; !overridden {
				c.Source = "builtin"
				newCands = append(newCands, c)
			}
		}

		result[i] = domain.ProviderRegistryEntry{
			Definition: e.Definition,
			Candidates: newCands,
		}
	}
	return result
}

// UpdatePaths validates and persists a path override for the given (providerKey, scope, purpose).
func (s *ProviderRegistryService) UpdatePaths(ctx context.Context, providerKey, scope, purpose string, paths []string) error {
	if providerKey == "" {
		return domain.NewValidationError("Provider key is required", "providerKey must not be empty")
	}
	if !validScopes[scope] {
		return domain.NewValidationError("Invalid scope", fmt.Sprintf("scope must be 'project' or 'global', got %q", scope))
	}
	if !validPurposes[purpose] {
		return domain.NewValidationError("Invalid purpose", fmt.Sprintf("purpose must be one of detect/skills/config/commands, got %q", purpose))
	}
	if len(paths) == 0 {
		return domain.NewValidationError("Paths must not be empty", "provide at least one path, or use resetPaths to restore defaults")
	}
	for _, p := range paths {
		if err := validatePath(p, scope); err != nil {
			return err
		}
	}

	provID, err := s.overrideRepo.GetProviderIDByKey(ctx, providerKey)
	if err != nil {
		return domain.NewDatabaseError("Could not look up provider", err.Error())
	}
	if provID == 0 {
		return domain.NewValidationError("Unknown provider", fmt.Sprintf("provider key %q not found", providerKey))
	}

	if err := s.overrideRepo.Upsert(ctx, domain.ProviderPathOverride{
		ProviderDefinitionID: provID,
		Scope:                scope,
		Purpose:              purpose,
		Paths:                paths,
	}); err != nil {
		return domain.NewDatabaseError("Could not save path override", err.Error())
	}
	return nil
}

// ResetPaths removes the user override for (providerKey, scope, purpose), restoring builtin defaults.
// Returns true if an override was removed, false if none existed.
func (s *ProviderRegistryService) ResetPaths(ctx context.Context, providerKey, scope, purpose string) (bool, error) {
	if providerKey == "" {
		return false, domain.NewValidationError("Provider key is required", "providerKey must not be empty")
	}
	if !validScopes[scope] {
		return false, domain.NewValidationError("Invalid scope", fmt.Sprintf("scope must be 'project' or 'global', got %q", scope))
	}
	if !validPurposes[purpose] {
		return false, domain.NewValidationError("Invalid purpose", fmt.Sprintf("purpose must be one of detect/skills/config/commands, got %q", purpose))
	}

	provID, err := s.overrideRepo.GetProviderIDByKey(ctx, providerKey)
	if err != nil {
		return false, domain.NewDatabaseError("Could not look up provider", err.Error())
	}
	if provID == 0 {
		return false, domain.NewValidationError("Unknown provider", fmt.Sprintf("provider key %q not found", providerKey))
	}

	deleted, err := s.overrideRepo.Delete(ctx, provID, scope, purpose)
	if err != nil {
		return false, domain.NewDatabaseError("Could not reset path override", err.Error())
	}
	return deleted, nil
}

// ProjectPaths returns effective project-scope detect and skills relative paths for every
// known provider: override.Paths[0] if a project-scope override exists, else the builtin
// candidate from the DB. Callers use this map to pass resolved paths into adapter.Detect.
// Global-scope overrides are intentionally excluded.
func (s *ProviderRegistryService) ProjectPaths(ctx context.Context) (map[string]providers.ProjectScopePaths, error) {
	entries, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load provider registry", err.Error())
	}
	overrides, err := s.overrideRepo.ListAll(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load provider path overrides", err.Error())
	}

	// Index project-scope overrides: providerDefinitionID → purpose → first path.
	type overrideKey struct {
		provID  int64
		purpose string
	}
	overrideFirst := make(map[overrideKey]string)
	for _, o := range overrides {
		if o.Scope != "project" || len(o.Paths) == 0 {
			continue
		}
		k := overrideKey{o.ProviderDefinitionID, o.Purpose}
		if _, seen := overrideFirst[k]; !seen {
			overrideFirst[k] = o.Paths[0]
		}
	}

	result := make(map[string]providers.ProjectScopePaths, len(entries))
	for _, e := range entries {
		var p providers.ProjectScopePaths
		// Resolve detect rel.
		if v, ok := overrideFirst[overrideKey{e.Definition.ID, "detect"}]; ok {
			p.DetectRel = v
		} else {
			for _, c := range e.Candidates {
				if c.Scope == "project" && c.Purpose == "detect" {
					p.DetectRel = c.RelativePath
					break
				}
			}
		}
		// Resolve skills rel.
		if v, ok := overrideFirst[overrideKey{e.Definition.ID, "skills"}]; ok {
			p.SkillsRel = v
		} else {
			for _, c := range e.Candidates {
				if c.Scope == "project" && c.Purpose == "skills" {
					p.SkillsRel = c.RelativePath
					break
				}
			}
		}
		result[e.Definition.Key] = p
	}
	return result, nil
}

// GlobalPaths returns effective global-scope detect and skills relative paths for every
// provider that has has_global_level=true: override.Paths[0] if a global-scope override
// exists, else the builtin candidate from the DB.
// Providers with has_global_level=false are excluded from the returned map.
func (s *ProviderRegistryService) GlobalPaths(ctx context.Context) (map[string]providers.GlobalScopePaths, error) {
	entries, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load provider registry", err.Error())
	}
	overrides, err := s.overrideRepo.ListAll(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load provider path overrides", err.Error())
	}

	// Index global-scope overrides: providerDefinitionID → purpose → first path.
	type overrideKey struct {
		provID  int64
		purpose string
	}
	overrideFirst := make(map[overrideKey]string)
	for _, o := range overrides {
		if o.Scope != "global" || len(o.Paths) == 0 {
			continue
		}
		k := overrideKey{o.ProviderDefinitionID, o.Purpose}
		if _, seen := overrideFirst[k]; !seen {
			overrideFirst[k] = o.Paths[0]
		}
	}

	result := make(map[string]providers.GlobalScopePaths)
	for _, e := range entries {
		if !e.Definition.HasGlobalLevel {
			continue
		}
		var p providers.GlobalScopePaths
		// Resolve detect rel.
		if v, ok := overrideFirst[overrideKey{e.Definition.ID, "detect"}]; ok {
			p.DetectRel = v
		} else {
			for _, c := range e.Candidates {
				if c.Scope == "global" && c.Purpose == "detect" {
					p.DetectRel = c.RelativePath
					break
				}
			}
		}
		// Resolve skills rel.
		if v, ok := overrideFirst[overrideKey{e.Definition.ID, "skills"}]; ok {
			p.SkillsRel = v
		} else {
			for _, c := range e.Candidates {
				if c.Scope == "global" && c.Purpose == "skills" {
					p.SkillsRel = c.RelativePath
					break
				}
			}
		}
		// Skip providers whose effective detect or skills path resolved to empty.
		// This guards against future providers that have has_global_level=true but
		// no seeded global candidates, which would cause expandGlobalPath to return homeDir.
		if p.DetectRel == "" || p.SkillsRel == "" {
			continue
		}
		result[e.Definition.Key] = p
	}
	return result, nil
}

func validatePath(p, scope string) error {
	if p == "" {
		return domain.NewValidationError("Empty path", "path must not be empty")
	}
	switch scope {
	case "project":
		if strings.HasPrefix(p, "/") {
			return domain.NewValidationError("Invalid project path", fmt.Sprintf("project path must be relative, got absolute path %q", p))
		}
		clean := filepath.Clean(p)
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return domain.NewValidationError("Invalid project path", fmt.Sprintf("project path must not escape via .., got %q", p))
		}
	case "global":
		if !strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "~/") {
			return domain.NewValidationError("Invalid global path", fmt.Sprintf("global path must start with / or ~/, got %q", p))
		}
	}
	return nil
}
