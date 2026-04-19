package entity

import "time"

// RunStatus 表示运行实例的状态。
type RunStatus string

const (
	RunStatusPending         RunStatus = "pending"
	RunStatusReady           RunStatus = "ready"
	RunStatusRunning         RunStatus = "running"
	RunStatusWaitingRecovery RunStatus = "waiting_recovery"
	RunStatusCompleted       RunStatus = "completed"
	RunStatusFailed          RunStatus = "failed"
	RunStatusCancelled       RunStatus = "cancelled"
)

// StepStatus 表示运行步骤的状态。
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusReady     StepStatus = "ready"
	StepStatusRunning   StepStatus = "running"
	StepStatusSucceeded StepStatus = "succeeded"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// StepKind 表示计划步骤类型。
type StepKind string

const (
	StepKindRoute    StepKind = "route"
	StepKindAgent    StepKind = "agent"
	StepKindParallel StepKind = "parallel"
	StepKindReview   StepKind = "review"
	StepKindHandoff  StepKind = "handoff"
	StepKindFinalize StepKind = "finalize"
)

// RecoveryActionType 表示失败后的恢复动作类型。
type RecoveryActionType string

const (
	RecoveryActionRetryStep           RecoveryActionType = "retry_step"
	RecoveryActionRerouteStep         RecoveryActionType = "reroute_step"
	RecoveryActionSkipStep            RecoveryActionType = "skip_step"
	RecoveryActionRetryFromCheckpoint RecoveryActionType = "retry_from_checkpoint"
)

// ExecutionPlan 表示一次协作运行的静态执行计划。
type ExecutionPlan struct {
	PlanID       string         `json:"planId"`
	PlanVersion  int            `json:"planVersion"`
	SourceMode   string         `json:"sourceMode"`
	EntryStepIDs []string       `json:"entryStepIds,omitempty"`
	Steps        []*PlanStep    `json:"steps,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// PlanStep 表示计划中的单个步骤定义。
type PlanStep struct {
	StepID       string         `json:"stepId"`
	Kind         StepKind       `json:"kind"`
	Name         string         `json:"name"`
	DependsOn    []string       `json:"dependsOn,omitempty"`
	AgentBinding string         `json:"agentBinding,omitempty"`
	InputRefs    []string       `json:"inputRefs,omitempty"`
	OutputRef    string         `json:"outputRef,omitempty"`
	Config       map[string]any `json:"config,omitempty"`
}

// PlaygroundRun 表示一次计划运行实例。
type PlaygroundRun struct {
	RunID            string         `json:"runId"`
	SchemeID         string         `json:"schemeId"`
	PlanID           string         `json:"planId"`
	Status           RunStatus      `json:"status"`
	InputArtifactID  string         `json:"inputArtifactId,omitempty"`
	CurrentStepIDs   []string       `json:"currentStepIds,omitempty"`
	LastCheckpointID string         `json:"lastCheckpointId,omitempty"`
	FailureSummary   string         `json:"failureSummary,omitempty"`
	StartedAt        *time.Time     `json:"startedAt,omitempty"`
	FinishedAt       *time.Time     `json:"finishedAt,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

// RuntimeStep 表示运行期的步骤状态快照。
type RuntimeStep struct {
	RunID          string     `json:"runId"`
	StepID         string     `json:"stepId"`
	Kind           StepKind   `json:"kind"`
	Name           string     `json:"name"`
	Status         StepStatus `json:"status"`
	AgentBinding   string     `json:"agentBinding,omitempty"`
	FailureSummary string     `json:"failureSummary,omitempty"`
	InputRefs      []string   `json:"inputRefs,omitempty"`
	OutputRef      string     `json:"outputRef,omitempty"`
	// CheckpointAfter 表示该步骤成功结束后应写入恢复检查点。
	CheckpointAfter bool           `json:"checkpointAfter,omitempty"`
	StartedAt       *time.Time     `json:"startedAt,omitempty"`
	FinishedAt      *time.Time     `json:"finishedAt,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// RuntimeArtifact 表示运行期结构化产物。
type RuntimeArtifact struct {
	ArtifactID     string         `json:"artifactId"`
	RunID          string         `json:"runId"`
	Type           string         `json:"type"`
	ProducerStepID string         `json:"producerStepId,omitempty"`
	SchemaVersion  int            `json:"schemaVersion"`
	Payload        map[string]any `json:"payload,omitempty"`
	Summary        string         `json:"summary,omitempty"`
	CreatedAt      *time.Time     `json:"createdAt,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// RecoveryAction 表示失败后允许的恢复操作。
type RecoveryAction struct {
	ID     string             `json:"id"`
	RunID  string             `json:"runId"`
	StepID string             `json:"stepId,omitempty"`
	Type   RecoveryActionType `json:"type"`
	// TargetRef 表示恢复动作指向的目标标识，例如 agent ID 或 checkpoint ID。
	TargetRef string         `json:"targetRef,omitempty"`
	Reason    string         `json:"reason,omitempty"`
	CreatedAt *time.Time     `json:"createdAt,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
