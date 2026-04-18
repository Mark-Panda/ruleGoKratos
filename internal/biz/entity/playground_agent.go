package entity

import (
	"time"
)

// AgentState Agent 状态
type AgentState string

const (
	AgentStateIdle    AgentState = "idle"
	AgentStateWorking AgentState = "working"
	AgentStateDone    AgentState = "done"
	AgentStateFailed  AgentState = "failed"
	AgentStateWaiting AgentState = "waiting" // 等待其他Agent完成
)

// CollaborationMode 协作模式
type CollaborationMode string

const (
	ModeRouterExpert CollaborationMode = "router_expert" // 路由专家
	ModePlanExec     CollaborationMode = "plan_exec"     // 规划执行
	ModeSupervision  CollaborationMode = "supervision"   // 动态监督
	ModePeerHandoff  CollaborationMode = "peer_handoff"  // 同伴交接
)

// AgentDefinition Agent 定义（配置级别）
type AgentDefinition struct {
	ID             string   `json:"id"`                       // 唯一标识 uuid
	Name           string   `json:"name"`                     // 显示名称：设计师、产品经理、工程师
	Role           string   `json:"role"`                     // 系统提示词角色
	Desc           string   `json:"desc"`                     // 描述
	Model          string   `json:"model"`                    // 使用的模型名称（未关联托管配置时使用）
	Tools          []string `json:"tools"`                    // 允许的工具列表（未关联托管配置时使用）
	Enabled        bool     `json:"enabled"`                  // 是否启用
	Priority       int      `json:"priority"`                 // 优先级（数字越小优先级越高）
	ManagedAgentID int64    `json:"managedAgentId,omitempty"` // 非零时关联「Agent 配置」管理中的托管 Agent（模型/SKILL/MCP 以托管为准）
}

// AgentInstance Agent 实例（运行时）
type AgentInstance struct {
	ID          string                 `json:"id"`
	Definition  *AgentDefinition       `json:"definition"`
	State       AgentState             `json:"state"`
	CurrentTask string                 `json:"currentTask"` // 当前任务
	Output      string                 `json:"output"`      // 输出结果
	History     []Turn                 `json:"history"`     // 对话历史
	StartTime   *time.Time             `json:"startTime"`
	EndTime     *time.Time             `json:"endTime"`
	Metadata    map[string]interface{} `json:"metadata"` // 运行时元数据
}

// Turn 单轮对话
type Turn struct {
	Timestamp int64      `json:"timestamp"`
	Role      string     `json:"role"` // user/assistant/system
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
}

// ToolCall 工具调用记录
type ToolCall struct {
	Name      string `json:"name"`
	Args      string `json:"args"`
	Result    string `json:"result"`
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
}

// AgentPool Agent 池
type AgentPool struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Agents      []*AgentDefinition `json:"agents"`
	CreatedAt   *time.Time         `json:"createdAt"`
	UpdatedAt   *time.Time         `json:"updatedAt"`
}
