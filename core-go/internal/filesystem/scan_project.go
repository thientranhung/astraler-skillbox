package filesystem

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// PathInfo holds basic filesystem facts about a path (follows symlinks via os.Stat).
type PathInfo struct {
	Exists   bool
	IsDir    bool
	Readable bool
}

// ProjectEntry describes one top-level entry in a project's skills directory.
// Unlike HostEntry it captures raw symlink facts without host classification.
type ProjectEntry struct {
	Name             string
	Path             string
	IsDir            bool
	IsSymlink        bool
	SymlinkTargetRaw string // raw target exactly as returned by os.Readlink (may be relative)
	ResolvedTarget   string // canonical path via EvalSymlinks; empty when Broken or non-symlink
	Broken           bool   // EvalSymlinks returned ErrNotExist
	ResolveError     error  // non-nil for other EvalSymlinks errors (loop, IO, …)
}

// ValidateProjectPath checks that path is absolute, exists, and is a directory.
// It does NOT check writability — project folders are read-only in 2A.
func ValidateProjectPath(path string) error {
	if !isAbs(path) {
		return newErr(ErrNotAbsolute, path, "path must be absolute")
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return newErr(ErrPathNotFound, path, "path does not exist")
		}
		return newErr(ErrPermission, path, err.Error())
	}
	if !info.IsDir() {
		return newErr(ErrNotADirectory, path, "path is not a directory")
	}
	return nil
}

// StatPathInfo returns filesystem facts about path, following symlinks (os.Stat).
// ENOENT is not an error — it returns PathInfo{Exists: false}.
func StatPathInfo(path string) (PathInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return PathInfo{Exists: false}, nil
		}
		return PathInfo{}, err
	}
	pi := PathInfo{Exists: true, IsDir: info.IsDir()}
	if info.IsDir() {
		f, openErr := os.Open(path)
		if openErr != nil {
			pi.Readable = false
		} else {
			f.Close()
			pi.Readable = true
		}
	} else {
		pi.Readable = true
	}
	return pi, nil
}

// ScanProjectSkills reads the direct children of skillsPath and records raw
// filesystem facts for each entry. Does NOT classify entries against hosts.
// Returns a FilesystemError if skillsPath cannot be read.
func ScanProjectSkills(skillsPath string) ([]ProjectEntry, error) {
	entries, err := os.ReadDir(skillsPath)
	if err != nil {
		return nil, &FilesystemError{Code: ErrPermission, Path: skillsPath, Message: err.Error()}
	}

	result := make([]ProjectEntry, 0, len(entries))
	for _, de := range entries {
		name := de.Name()
		absPath := filepath.Join(skillsPath, name)

		entry := ProjectEntry{Name: name, Path: absPath}

		if de.Type()&os.ModeSymlink != 0 {
			entry.IsSymlink = true
			rawTarget, readErr := os.Readlink(absPath)
			if readErr != nil {
				entry.Broken = true
			} else {
				// Preserve the raw target exactly as returned by os.Readlink.
				entry.SymlinkTargetRaw = rawTarget

				resolved, evalErr := filepath.EvalSymlinks(absPath)
				if evalErr != nil {
					if errors.Is(evalErr, fs.ErrNotExist) {
						entry.Broken = true
					} else {
						entry.ResolveError = evalErr
					}
				} else {
					entry.ResolvedTarget = resolved
					// Determine IsDir from the resolved target's actual type.
					if resolvedInfo, statErr := os.Stat(resolved); statErr == nil {
						entry.IsDir = resolvedInfo.IsDir()
					}
				}
			}
		} else {
			if fi, statErr := de.Info(); statErr == nil {
				entry.IsDir = fi.IsDir()
			}
		}

		result = append(result, entry)
	}
	return result, nil
}
