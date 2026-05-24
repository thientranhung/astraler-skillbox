package filesystem

import "fmt"

type FilesystemErrorCode string

const (
	ErrPathNotFound   FilesystemErrorCode = "path_not_found"
	ErrNotADirectory  FilesystemErrorCode = "not_a_directory"
	ErrNotWritable    FilesystemErrorCode = "not_writable"
	ErrNotAbsolute    FilesystemErrorCode = "not_absolute"
	ErrPermission     FilesystemErrorCode = "permission_denied"
)

type FilesystemError struct {
	Code    FilesystemErrorCode
	Path    string
	Message string
}

func (e *FilesystemError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Path, e.Message)
}

func newErr(code FilesystemErrorCode, path, msg string) *FilesystemError {
	return &FilesystemError{Code: code, Path: path, Message: msg}
}
