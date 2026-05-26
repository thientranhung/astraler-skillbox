package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
)

type globalScanService interface {
	ScanGlobal(ctx context.Context) (int64, error)
}

type globalScanResponse struct {
	OperationID int64 `json:"operationId"`
}

func NewGlobalScanHandler(svc globalScanService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		opID, err := svc.ScanGlobal(ctx)
		if err != nil {
			return nil, wrapError(err)
		}
		return globalScanResponse{OperationID: opID}, nil
	})
}

