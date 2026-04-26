package entity

import (
	"encoding/json"
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
	PayloadTemplate string
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

func NewScheduledTriggerPayload(taskID int64, payloadTemplate string) string {
	base := map[string]interface{}{
		"trigger": "schedule",
		"taskId":  strconv.FormatInt(taskID, 10),
	}

	if payloadTemplate == "" {
		b, _ := json.Marshal(base)
		return string(b)
	}

	var tpl struct {
		Metadata string                 `json:"metadata"`
		Body     map[string]interface{} `json:"body"`
	}
	if err := json.Unmarshal([]byte(payloadTemplate), &tpl); err != nil || tpl.Body == nil {
		b, _ := json.Marshal(base)
		return string(b)
	}

	merged := make(map[string]interface{})
	for k, v := range tpl.Body {
		merged[k] = v
	}
	for k, v := range base {
		merged[k] = v
	}

	b, _ := json.Marshal(merged)
	return string(b)
}
