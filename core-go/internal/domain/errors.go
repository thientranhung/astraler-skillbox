package domain

import "fmt"

// Error codes matching M2 error taxonomy (JSON-RPC codes 1001-1099).
const (
	CodeValidation         = "validation_error"
	CodeFilesystem         = "filesystem_error"
	CodeProvider           = "provider_error"
	CodeDatabase           = "database_error"
	CodeAuth               = "auth_error"
	CodeNetwork            = "network_error"
	CodeConflict           = "conflict_error"
	CodeOperationCancelled = "operation_cancelled"
	CodeUserCancelled      = "user_cancelled"
	CodeUnknown            = "unknown_error"
)

var rpcCodes = map[string]int{
	CodeValidation:         1001,
	CodeFilesystem:         1002,
	CodeProvider:           1003,
	CodeDatabase:           1004,
	CodeConflict:           1005,
	CodeUserCancelled:      1006,
	CodeOperationCancelled: 1007,
	CodeUnknown:            1099,
}

// AppError is a structured domain error with user-facing and technical messages.
type AppError struct {
	Code             string `json:"code"`
	UserMessage      string `json:"userMessage"`
	TechnicalMessage string `json:"technicalMessage"`
	OperationID      *int64 `json:"operationId,omitempty"`
	EntityRef        string `json:"entityRef,omitempty"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.UserMessage, e.TechnicalMessage)
}

// RPCCode returns the JSON-RPC error code for this error.
func (e *AppError) RPCCode() int {
	if c, ok := rpcCodes[e.Code]; ok {
		return c
	}
	return rpcCodes[CodeUnknown]
}

func NewValidationError(userMsg, techMsg string) *AppError {
	return &AppError{Code: CodeValidation, UserMessage: userMsg, TechnicalMessage: techMsg}
}

func NewFilesystemError(userMsg, techMsg string) *AppError {
	return &AppError{Code: CodeFilesystem, UserMessage: userMsg, TechnicalMessage: techMsg}
}

func NewDatabaseError(userMsg, techMsg string) *AppError {
	return &AppError{Code: CodeDatabase, UserMessage: userMsg, TechnicalMessage: techMsg}
}

func NewConflictError(userMsg, techMsg string) *AppError {
	return &AppError{Code: CodeConflict, UserMessage: userMsg, TechnicalMessage: techMsg}
}

func NewUnknownError(userMsg, techMsg string) *AppError {
	return &AppError{Code: CodeUnknown, UserMessage: userMsg, TechnicalMessage: techMsg}
}
