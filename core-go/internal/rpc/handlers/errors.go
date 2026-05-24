package handlers

import (
	"encoding/json"
	"errors"

	"github.com/creachadair/jrpc2"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// wrapError converts *domain.AppError to *jrpc2.Error so the client receives
// an integer error code and a structured data payload that matches
// shared/api-contracts/shared/error.json.
// Non-AppError values are returned as-is.
func wrapError(err error) error {
	var ae *domain.AppError
	if errors.As(err, &ae) {
		data, _ := json.Marshal(ae)
		return &jrpc2.Error{
			Code:    jrpc2.Code(ae.RPCCode()),
			Message: ae.UserMessage,
			Data:    json.RawMessage(data),
		}
	}
	return err
}
