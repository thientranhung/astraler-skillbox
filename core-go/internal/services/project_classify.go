package services

import (
	"strings"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

// HostSummary holds the minimal host info needed to classify project entries.
// Callers should place active hosts before inactive ones so that when a skills
// path is shared (degenerate case), the active host wins.
type HostSummary struct {
	ID         int64
	SkillsPath string
	IsActive   bool
	Skills     []domain.Skill
}

// ClassifiedEntry is the outcome of classifying one provider adapter entry.
type ClassifiedEntry struct {
	Mode                      domain.InstallMode
	Status                    domain.InstallStatus
	SkillID                   *int64
	InstalledFromHostFolderID *int64
	SourceSkillPath           *string
	SymlinkTargetPath         *string
}

// ClassifyAdapterEntry derives install mode and status for a single provider
// adapter entry by comparing it against known skill host folders.
//
// Classification rules (Slice 2A):
//   - Broken symlink                         → symlink / broken_symlink
//   - Symlink with resolve error              → symlink / error
//   - Symlink resolving into active host      → symlink / current  (+ skillID if known)
//   - Symlink resolving into inactive host    → symlink / old_host
//   - Symlink resolving outside all hosts     → symlink / external_symlink
//   - Non-symlink directory or file           → direct  / current
func ClassifyAdapterEntry(entry providers.AdapterEntry, hosts []HostSummary) ClassifiedEntry {
	if entry.IsSymlink {
		return classifySymlinkEntry(entry, hosts)
	}
	return ClassifiedEntry{
		Mode:   domain.InstallModeDirect,
		Status: domain.InstallStatusCurrent,
	}
}

func classifySymlinkEntry(entry providers.AdapterEntry, hosts []HostSummary) ClassifiedEntry {
	raw := ptrStr(entry.SymlinkTargetRaw)

	if entry.Broken {
		return ClassifiedEntry{
			Mode:              domain.InstallModeSymlink,
			Status:            domain.InstallStatusBrokenSymlink,
			SymlinkTargetPath: raw,
		}
	}
	if entry.ResolveError != nil {
		return ClassifiedEntry{
			Mode:              domain.InstallModeSymlink,
			Status:            domain.InstallStatusError,
			SymlinkTargetPath: raw,
		}
	}

	resolved := entry.ResolvedTarget
	for i := range hosts {
		h := &hosts[i]
		if !isUnderSkillsPath(resolved, h.SkillsPath) {
			continue
		}

		result := ClassifiedEntry{
			Mode:                      domain.InstallModeSymlink,
			InstalledFromHostFolderID: &h.ID,
			SourceSkillPath:           &resolved,
			SymlinkTargetPath:         raw,
		}
		if h.IsActive {
			result.Status = domain.InstallStatusCurrent
		} else {
			result.Status = domain.InstallStatusOldHost
		}
		for _, sk := range h.Skills {
			if sk.AbsolutePath == resolved {
				id := sk.ID
				result.SkillID = &id
				break
			}
		}
		return result
	}

	return ClassifiedEntry{
		Mode:              domain.InstallModeSymlink,
		Status:            domain.InstallStatusExternalSymlink,
		SymlinkTargetPath: raw,
	}
}

// isUnderSkillsPath reports whether child is a direct or indirect descendant of
// parent. The check uses a "/" suffix to prevent false positives where parent is
// a prefix of a sibling directory name (e.g. /a/skillsX vs /a/skills).
func isUnderSkillsPath(child, parent string) bool {
	if parent == "" || child == "" {
		return false
	}
	return strings.HasPrefix(child, parent+"/")
}

// ptrStr returns a pointer to s, or nil when s is empty.
func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
