package domain

import "time"

type OperationStatus string

const (
	OperationStatusQueued    OperationStatus = "queued"
	OperationStatusRunning   OperationStatus = "running"
	OperationStatusSuccess   OperationStatus = "success"
	OperationStatusFailed    OperationStatus = "failed"
	OperationStatusCancelled OperationStatus = "cancelled"
	OperationStatusPartial   OperationStatus = "partial"
)

type OperationType string

const (
	OperationTypeScan                  OperationType = "scan"
	OperationTypeChangeSkillHostFolder OperationType = "change_skill_host_folder"
	OperationTypeInstallSkill          OperationType = "install_skill"
)

type Operation struct {
	ID           int64
	OperationType OperationType
	TargetType   string
	TargetID     *int64
	Status       OperationStatus
	StartedAt    *time.Time
	FinishedAt   *time.Time
	ErrorMessage *string
	MetadataJSON *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
