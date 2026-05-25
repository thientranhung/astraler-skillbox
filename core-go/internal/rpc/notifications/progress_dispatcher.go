package notifications

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/creachadair/jrpc2"

	"github.com/astraler/skillbox/core-go/internal/operations"
)

// ProgressParams is the JSON payload for operation.progress notifications.
type ProgressParams struct {
	OperationID int64           `json:"operationId"`
	Status      string          `json:"status"`
	Phase       string          `json:"phase"`
	Processed   *int            `json:"processed"`
	Total       *int            `json:"total"`
	Message     *string         `json:"message"`
	Metadata    json.RawMessage `json:"metadata"`
}

// StartDispatcher reads ProgressEvents from ch and pushes operation.progress
// notifications via srv. It runs until ch is closed.
func StartDispatcher(ctx context.Context, srv *jrpc2.Server, ch <-chan operations.ProgressEvent) {
	go func() {
		for {
			select {
			case evt, ok := <-ch:
				if !ok {
					return
				}
				params := ProgressParams{
					OperationID: evt.OperationID,
					Status:      evt.Status,
					Phase:       evt.Phase,
					Processed:   evt.Processed,
					Total:       evt.Total,
					Message:     evt.Message,
					Metadata:    evt.Metadata,
				}
				if err := srv.Notify(ctx, "operation.progress", params); err != nil {
					slog.Warn("progress notification failed", "operationId", evt.OperationID, "err", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
