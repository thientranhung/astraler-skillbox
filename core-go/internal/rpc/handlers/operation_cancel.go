package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type operationRunner interface {
	Cancel(operationID int64) bool
}

type operationCancelRequest struct {
	OperationID int64 `json:"operationId"`
}

type operationCancelResponse struct {
	Acknowledged bool `json:"acknowledged"`
}

func NewOperationCancelHandler(runner operationRunner) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p operationCancelRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, domain.NewValidationError("Invalid params", err.Error())
		}
		if p.OperationID <= 0 {
			return nil, domain.NewValidationError("operationId is required", "operationId must be positive")
		}
		acked := runner.Cancel(p.OperationID)
		return operationCancelResponse{Acknowledged: acked}, nil
	})
}
