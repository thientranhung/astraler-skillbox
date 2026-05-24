package filesystem

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// EnsureAgentsSkills creates <hostPath>/.agents/skills if it doesn't exist.
// Returns (true, nil) if created, (false, nil) if it already existed.
func EnsureAgentsSkills(hostPath string) (created bool, err error) {
	skillsPath := filepath.Join(hostPath, ".agents", "skills")
	_, statErr := os.Stat(skillsPath)
	if statErr == nil {
		return false, nil
	}
	if !errors.Is(statErr, fs.ErrNotExist) {
		return false, &FilesystemError{Code: ErrPermission, Path: skillsPath, Message: statErr.Error()}
	}

	if mkErr := os.MkdirAll(skillsPath, 0o755); mkErr != nil {
		return false, &FilesystemError{Code: ErrNotWritable, Path: skillsPath, Message: mkErr.Error()}
	}
	return true, nil
}
