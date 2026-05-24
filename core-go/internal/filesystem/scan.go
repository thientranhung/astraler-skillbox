package filesystem

import (
	"os"
	"path/filepath"
)

// HostEntry describes one direct child of the skills path.
type HostEntry struct {
	Name          string
	RelativePath  string // relative to the scanned directory parent (skillsPath)
	AbsolutePath  string
	IsDir         bool
	IsSymlink     bool
	SymlinkTarget string // resolved target path; empty for non-symlinks
	Broken        bool   // symlink target does not exist
	External      bool   // symlink target is outside skillsPath
}

// ScanHostFolder reads the direct children of skillsPath and classifies each
// entry. Only one level deep — skill entries are top-level items in skillsPath.
func ScanHostFolder(skillsPath string) ([]HostEntry, error) {
	entries, err := os.ReadDir(skillsPath)
	if err != nil {
		return nil, &FilesystemError{Code: ErrPermission, Path: skillsPath, Message: err.Error()}
	}

	var result []HostEntry
	for _, de := range entries {
		name := de.Name()
		absPath := filepath.Join(skillsPath, name)
		relPath := filepath.Join(".agents", "skills", name)

		entry := HostEntry{
			Name:         name,
			RelativePath: relPath,
			AbsolutePath: absPath,
		}

		if de.Type()&os.ModeSymlink != 0 {
			entry.IsSymlink = true
			target, err := os.Readlink(absPath)
			if err == nil {
				// Resolve to absolute path.
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(absPath), target)
				}
				target = filepath.Clean(target)
				entry.SymlinkTarget = target

				// Check if target exists.
				if _, statErr := os.Stat(absPath); statErr != nil {
					entry.Broken = true
				} else {
					entry.IsDir = true
					// External: target not inside skillsPath.
					if !isUnder(target, skillsPath) {
						entry.External = true
					}
				}
			} else {
				entry.Broken = true
			}
		} else {
			info, err := de.Info()
			if err == nil {
				entry.IsDir = info.IsDir()
			}
		}

		result = append(result, entry)
	}
	return result, nil
}

// isUnder returns true if path is inside or equal to root.
func isUnder(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return len(rel) > 0 && rel[0] != '.'
}
