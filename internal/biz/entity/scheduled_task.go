package entity

import (
	"strconv"
	"time"
)

const (
	ScheduledTaskRunStatusSuccess int32 = 1
	ScheduledTaskRunStatusFailed  int32 = 2
)

type ScheduledTask struct {
	ID             int64
	Name           string
	Description    string
	RuleChainID    string
	CronExpr       string
	ScheduleType   string
	ScheduleConfig string
	Disabled       bool
	LastRunAt      *time.Time
	LastStatus     int32
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type ScheduledTaskRun struct {
	ID             int64
	TaskID         int64
	RuleChainID    string
	Status         int32
	TriggerPayload string
	ErrorMessage   string
	StartedAt      time.Time
	FinishedAt     time.Time
	CreatedAt      time.Time
}

func NewScheduledTriggerPayload(taskID int64) string {
	return `{"trigger":"schedule","taskId":"` + strconv.FormatInt(taskID, 10) + `"}`
}
