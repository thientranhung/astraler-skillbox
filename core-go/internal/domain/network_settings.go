package domain

import "time"

type NetworkSettings struct {
	ID                 int
	UpdateCheckEnabled bool
	CacheTTLHours      int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
