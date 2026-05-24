package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type hostScanService interface {
	ScanHost(ctx context.Context, hostID int64) (int64, error)
}

type hostScanRequest struct {
	HostID int64 `json:"hostId"`
}

type hostScanResponse struct {
	OperationID int64 `json:"operationId"`
}

func NewHostScanHandler(svc hostScanService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p hostScanRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, domain.NewValidationError("Invalid params", err.Error())
		}

		opID, err := svc.ScanHost(ctx, p.HostID)
		if err != nil {
			return nil, err
		}
		return hostScanResponse{OperationID: opID}, nil
	})
}
