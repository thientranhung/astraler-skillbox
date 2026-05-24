package filesystem

import (
	"errors"
	"io/fs"
	"os"
)

// ValidateHostPath checks that path is absolute, exists, is a directory,
// and is writable. Returns a FilesystemError with appropriate code on failure.
func ValidateHostPath(path string) error {
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

	if err := checkWritable(path); err != nil {
		return newErr(ErrNotWritable, path, err.Error())
	}

	return nil
}

func isAbs(path string) bool {
	return len(path) > 0 && path[0] == '/'
}

func checkWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".skillbox-write-check-*")
	if err != nil {
		return err
	}
	f.Close()
	os.Remove(f.Name())
	return nil
}
