package entity

import (
	"time"
)

// CollaborationScheme 协作方案（一个方案包含多个 Agent 和一种协作模式）
type CollaborationScheme struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Mode            CollaborationMode `json:"mode"`
	BindAgents      []*AgentBinding   `json:"bindAgents"` // 绑定的 Agent 列表
	Config          *SchemeConfig     `json:"config"`     // 方案配置
	Enabled         bool              `json:"enabled"`
	EnableFinalizer bool              `json:"enableFinalizer"` // 启用最终整理
	CreatedAt       *time.Time        `json:"createdAt"`
	UpdatedAt       *time.Time        `json:"updatedAt"`
}

// AgentBinding Agent 绑定（关联 Agent 定义与角色）
type AgentBinding struct {
	AgentID string   `json:"agentId"` // Agent 定义 ID
	Role    string   `json:"role"`    // 角色名称
	Model   string   `json:"model"`   // 可选：覆盖默认模型
	Tools   []string `json:"tools"`   // 可选：覆盖默认工具
}

// SchemeConfig 方案配置
type SchemeConfig struct {
	MaxIterations   int    `json:"maxIterations"`   // 最大迭代次数
	MaxToolCalls    int    `json:"maxToolCalls"`    // 最大工具调用次数
	TimeoutSeconds  int    `json:"timeoutSeconds"`  // 超时时间（秒）
	FinalizerPrompt string `json:"finalizerPrompt"` // 最终整理提示词
	// 协作模式配置（不同模式的特定配置）
	ModeConfig *ModeConfig `json:"modeConfig,omitempty"`
}

// ModeConfig 协作模式配置（不同模式的特定配置）
type ModeConfig struct {
	// 路由专家模式
	RouterConfig *RouterConfig `json:"routerConfig,omitempty"`
	// 规划执行模式
	PlanExecConfig *PlanExecConfig `json:"planExecConfig,omitempty"`
	// 动态监督模式
	SupervisionConfig *SupervisionConfig `json:"supervisionConfig,omitempty"`
	// 同伴交接模式
	PeerHandoffConfig *PeerHandoffConfig `json:"peerHandoffConfig,omitempty"`
}

// RouterConfig 路由专家模式配置
type RouterConfig struct {
	RoutingPrompt string `json:"routingPrompt"` // 路由决策提示词
	FallbackAgent string `json:"fallbackAgent"` // 兜底 Agent ID
}

// PlanExecConfig 规划执行模式配置
type PlanExecConfig struct {
	PlannerAgent   string   `json:"plannerAgent"`   // 规划 Agent ID
	ExecutionOrder []string `json:"executionOrder"` // 执行顺序（Agent ID 列表）
}

// SupervisionConfig 动态监督模式配置
type SupervisionConfig struct {
	SupervisorAgent string   `json:"supervisorAgent"` // 监督 Agent ID
	WorkerAgents    []string `json:"workerAgents"`    // 工作 Agent 列表
	CheckInterval   int      `json:"checkInterval"`   // 检查间隔（秒）
}

// PeerHandoffConfig 同伴交接模式配置
type PeerHandoffConfig struct {
	EntryAgent   string   `json:"entryAgent"`   // 入口 Agent
	MeshAgents   []string `json:"meshAgents"`   // 参与协作的 Agent 列表
	HandoffRules string   `json:"handoffRules"` // 交接规则描述
}

// SchemeTemplate 方案模板（预置模板）
var SchemeTemplates = map[CollaborationMode][]*AgentBinding{
	ModeRouterExpert: {
		{AgentID: "designer", Role: "设计师"},
		{AgentID: "pm", Role: "产品经理"},
		{AgentID: "engineer", Role: "工程师"},
	},
	ModePlanExec: {
		{AgentID: "planner", Role: "规划师"},
		{AgentID: "designer", Role: "设计师"},
		{AgentID: "pm", Role: "产品经理"},
		{AgentID: "engineer", Role: "工程师"},
	},
	ModeSupervision: {
		{AgentID: "supervisor", Role: "监督者"},
		{AgentID: "designer", Role: "设计师"},
		{AgentID: "engineer", Role: "工程师"},
	},
	ModePeerHandoff: {
		{AgentID: "designer", Role: "设计师"},
		{AgentID: "pm", Role: "产品经理"},
		{AgentID: "engineer", Role: "工程师"},
	},
}

// DefaultSchemeConfig 默认方案配置
// MaxIterations：Harness 每产生一轮「模型→工具→再模型」计一次；复杂编码/多文件任务易超过个位数。
var DefaultSchemeConfig = &SchemeConfig{
	MaxIterations:  32,
	MaxToolCalls:   64,
	TimeoutSeconds: 300,
}
