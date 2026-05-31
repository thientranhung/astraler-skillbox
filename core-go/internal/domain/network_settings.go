package domain

import "time"

type NetworkSettings struct {
	ID            int
	CacheTTLHours int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
