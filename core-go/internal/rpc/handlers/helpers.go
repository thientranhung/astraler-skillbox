package handlers

import (
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func newValidationError(detail string) error {
	return domain.NewValidationError(detail, detail)
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format("2006-01-02T15:04:05Z")
	return &s
}
