package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
)

type providerPluginScanGlobalSvc interface {
	ScanGlobal(ctx context.Context) (int64, error)
}

type providerPluginScanGlobalResponse struct {
	OperationID int64 `json:"operationId"`
}

func NewProviderPluginScanGlobalHandler(svc providerPluginScanGlobalSvc) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		opID, err := svc.ScanGlobal(ctx)
		if err != nil {
			return nil, wrapError(err)
		}
		return providerPluginScanGlobalResponse{OperationID: opID}, nil
	})
}
