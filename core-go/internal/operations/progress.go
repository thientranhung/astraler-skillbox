package operations

import "encoding/json"

// ProgressEvent is emitted by a running operation to report its progress.
type ProgressEvent struct {
	OperationID int64
	Status      string
	Phase       string
	Processed   *int
	Total       *int
	Message     *string
	Metadata    json.RawMessage
}

// ProgressFn is the callback an operation work function calls to report progress.
type ProgressFn func(phase string, processed, total int, msg string)
