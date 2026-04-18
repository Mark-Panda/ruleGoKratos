package entity

import (
	"crypto/rand"
	"fmt"
	"time"
)

// TraceEventType Trace 事件类型
type TraceEventType string

const (
	TraceEventWorkflowStart       TraceEventType = "WORKFLOW_START"       // 工作流开始
	TraceEventPlanSummary         TraceEventType = "PLAN_SUMMARY"         // 规划执行：已确定的子任务顺序（可读摘要）
	TraceEventStepOutput          TraceEventType = "STEP_OUTPUT"          // 某执行步骤完成后的产出摘要
	TraceEventTaskAssigned        TraceEventType = "TASK_ASSIGNED"        // 任务分配
	TraceEventAgentEnterWorker    TraceEventType = "AGENT_ENTER_WORKER"   // Agent 进入工作状态
	TraceEventAgentExitWorker     TraceEventType = "AGENT_EXIT_WORKER"    // Agent 退出工作状态
	TraceEventWorkerDelegated     TraceEventType = "WORKER_DELEGATED"     // 任务委派
	TraceEventThinking            TraceEventType = "THINKING"             // 思考中
	TraceEventToolCall            TraceEventType = "TOOL_CALL"            // 工具调用
	TraceEventToolResult          TraceEventType = "TOOL_RESULT"          // 工具结果
	TraceEventHandoff             TraceEventType = "HANDOFF"              // 任务交接
	TraceEventSupervisorCheck     TraceEventType = "SUPERVISOR_CHECK"     // 监督者检查
	TraceEventSupervisorIntervene TraceEventType = "SUPERVISOR_INTERVENE" // 监督者干预
	TraceEventError               TraceEventType = "ERROR"                // 错误
	TraceEventWorkflowEnd         TraceEventType = "WORKFLOW_END"         // 工作流结束
)

// TraceEvent Trace 事件
type TraceEvent struct {
	ID        string                 `json:"id"`
	RunID     string                 `json:"runId"`     // 运行的 Run ID
	Timestamp int64                  `json:"timestamp"` // 毫秒时间戳
	Type      TraceEventType         `json:"type"`
	AgentID   string                 `json:"agentId"`  // 涉及的 Agent ID
	NodeID    string                 `json:"nodeId"`   // 涉及的节点 ID
	TaskDesc  string                 `json:"taskDesc"` // 任务描述
	Message   string                 `json:"message"`  // 事件消息
	Metadata  map[string]interface{} `json:"metadata"` // 额外元数据
}

// TraceRun 一次运行记录的 Trace
type TraceRun struct {
	ID          string        `json:"id"`
	RunID       string        `json:"runId"`
	SchemeID    string        `json:"schemeId"`  // 方案 ID
	UserInput   string        `json:"userInput"` // 用户输入
	Status      string        `json:"status"`    // running/completed/failed
	StartTime   *time.Time    `json:"startTime"`
	EndTime     *time.Time    `json:"endTime"`
	TotalMs     int64         `json:"totalMs"`     // 总耗时（毫秒）
	Events      []*TraceEvent `json:"events"`      // 所有事件
	FinalOutput string        `json:"finalOutput"` // 最终输出
}

// TraceFilter Trace 查询过滤条件
type TraceFilter struct {
	RunID     string
	SchemeID  string
	AgentID   string
	EventType TraceEventType
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// NewTraceEvent 创建 Trace 事件
func NewTraceEvent(runID string, eventType TraceEventType, agentID, nodeID, taskDesc, message string) *TraceEvent {
	return &TraceEvent{
		ID:        generateUUID(),
		RunID:     runID,
		Timestamp: time.Now().UnixMilli(),
		Type:      eventType,
		AgentID:   agentID,
		NodeID:    nodeID,
		TaskDesc:  taskDesc,
		Message:   message,
		Metadata:  make(map[string]interface{}),
	}
}

// WithMetadata 添加元数据
func (e *TraceEvent) WithMetadata(key string, value interface{}) *TraceEvent {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// generateUUID 生成 UUID
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
