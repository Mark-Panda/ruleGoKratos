package collaboration

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
	"testing"
)

// MockTraceEmitter Trace 模拟实现
type MockTraceEmitter struct {
	events []string
}

func (m *MockTraceEmitter) TaskAssigned(runID, agentID, taskDesc string) {
	m.events = append(m.events, fmt.Sprintf("TaskAssigned:%s:%s:%s", runID, agentID, taskDesc))
}

func (m *MockTraceEmitter) AgentEnterWorker(runID, agentID, nodeID string) {
	m.events = append(m.events, fmt.Sprintf("AgentEnterWorker:%s:%s:%s", runID, agentID, nodeID))
}

func (m *MockTraceEmitter) AgentExitWorker(runID, agentID, nodeID, message string) {
	m.events = append(m.events, fmt.Sprintf("AgentExitWorker:%s:%s:%s:%s", runID, agentID, nodeID, message))
}

func (m *MockTraceEmitter) WorkerDelegated(runID, agentID, workerAgentID, reason string) {
	m.events = append(m.events, fmt.Sprintf("WorkerDelegated:%s:%s:%s:%s", runID, agentID, workerAgentID, reason))
}

func (m *MockTraceEmitter) Thinking(runID, agentID, message string) {
	m.events = append(m.events, fmt.Sprintf("Thinking:%s:%s:%s", runID, agentID, message))
}

func (m *MockTraceEmitter) ToolCall(runID, agentID, toolName, args string) {
	m.events = append(m.events, fmt.Sprintf("ToolCall:%s:%s:%s:%s", runID, agentID, toolName, args))
}

func (m *MockTraceEmitter) ToolResult(runID, agentID, toolName, result string, success bool) {
	m.events = append(m.events, fmt.Sprintf("ToolResult:%s:%s:%s:%s:%v", runID, agentID, toolName, result, success))
}

func (m *MockTraceEmitter) Handoff(runID, fromAgent, toAgent, reason string) {
	m.events = append(m.events, fmt.Sprintf("Handoff:%s:%s:%s:%s", runID, fromAgent, toAgent, reason))
}

func (m *MockTraceEmitter) Error(runID, agentID, message string) {
	m.events = append(m.events, fmt.Sprintf("Error:%s:%s:%s", runID, agentID, message))
}

func (m *MockTraceEmitter) EmitEvent(ctx context.Context, event *entity.TraceEvent) {
	m.events = append(m.events, fmt.Sprintf("EmitEvent:%s:%s:%s", event.RunID, event.Type, event.Message))
}

func newTestPool() *entity.AgentPool {
	return &entity.AgentPool{
		ID:   "test-pool",
		Name: "测试池",
		Agents: []*entity.AgentDefinition{
			{ID: "designer", Name: "设计师", Enabled: true, Priority: 1},
			{ID: "pm", Name: "产品经理", Enabled: true, Priority: 2},
			{ID: "engineer", Name: "工程师", Enabled: true, Priority: 3},
			{ID: "planner", Name: "规划师", Enabled: true, Priority: 0},
			{ID: "supervisor", Name: "监督者", Enabled: true, Priority: 0},
		},
	}
}

func newTestScheme(mode entity.CollaborationMode) *entity.CollaborationScheme {
	return &entity.CollaborationScheme{
		ID:   "test-scheme",
		Name: "测试方案",
		Mode: mode,
		BindAgents: []*entity.AgentBinding{
			{AgentID: "designer", Role: "设计师"},
			{AgentID: "pm", Role: "产品经理"},
			{AgentID: "engineer", Role: "工程师"},
			{AgentID: "planner", Role: "规划师"},
		},
		Config: entity.DefaultSchemeConfig,
	}
}

func TestRouterExpertHandler_SelectAgent(t *testing.T) {
	handler := NewRouterExpertHandler()
	pool := newTestPool()
	scheme := newTestScheme(entity.ModeRouterExpert)

	if err := handler.Init(context.Background(), scheme, pool, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	tests := []struct {
		input       string
		expectedID  string
	}{
		{"设计一个好看的界面", "designer"},
		{"分析需求功能点", "pm"},
		{"编写代码实现功能", "engineer"},
		{"拆解任务步骤", "planner"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			agent := handler.selectAgent(tt.input)
			if agent == nil {
				t.Fatalf("selectAgent returned nil for input: %s", tt.input)
			}
			if agent.Definition.ID != tt.expectedID {
				t.Errorf("expected %s, got %s", tt.expectedID, agent.Definition.ID)
			}
		})
	}
}

func TestRouterExpertHandler_Execute(t *testing.T) {
	handler := NewRouterExpertHandler()
	pool := newTestPool()
	scheme := newTestScheme(entity.ModeRouterExpert)

	if err := handler.Init(context.Background(), scheme, pool, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	trace := &MockTraceEmitter{}
	result, err := handler.Execute(context.Background(), "run-1", "设计一个按钮", trace)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Execute returned nil result")
	}

	if len(trace.events) == 0 {
		t.Error("no trace events recorded")
	}
}

func TestPlanExecHandler_Execute(t *testing.T) {
	handler := NewPlanExecHandler()
	pool := newTestPool()
	scheme := newTestScheme(entity.ModePlanExec)

	if err := handler.Init(context.Background(), scheme, pool, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	trace := &MockTraceEmitter{}
	result, err := handler.Execute(context.Background(), "run-1", "开发一个计算器", trace)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Execute returned nil result")
	}

	// 验证有任务分配事件
	hasTaskAssigned := false
	for _, e := range trace.events {
		if len(e) > 15 && e[:15] == "TaskAssigned:ru" {
			hasTaskAssigned = true
			break
		}
	}
	if !hasTaskAssigned {
		t.Error("no TaskAssigned events recorded")
	}
}

func TestSupervisionHandler_Execute(t *testing.T) {
	handler := NewSupervisionHandler()
	pool := newTestPool()
	scheme := newTestScheme(entity.ModeSupervision)

	if err := handler.Init(context.Background(), scheme, pool, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	trace := &MockTraceEmitter{}
	result, err := handler.Execute(context.Background(), "run-1", "完成开发任务", trace)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Execute returned nil result")
	}

	// 验证有 Worker 委派事件
	hasDelegated := false
	for _, e := range trace.events {
		if len(e) > 16 && e[:16] == "WorkerDelegated:" {
			hasDelegated = true
			break
		}
	}
	if !hasDelegated {
		t.Error("no WorkerDelegated events recorded")
	}
}

func TestPeerHandoffHandler_Execute(t *testing.T) {
	handler := NewPeerHandoffHandler()
	pool := newTestPool()
	scheme := newTestScheme(entity.ModePeerHandoff)

	if err := handler.Init(context.Background(), scheme, pool, nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	trace := &MockTraceEmitter{}
	result, err := handler.Execute(context.Background(), "run-1", "开始协作任务", trace)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Execute returned nil result")
	}

	// 验证有 Handoff 事件
	hasHandoff := false
	for _, e := range trace.events {
		if len(e) > 8 && e[:8] == "Handoff:" {
			hasHandoff = true
			break
		}
	}
	if !hasHandoff {
		t.Error("no Handoff events recorded")
	}
}

func TestFactory_RegisterAndGet(t *testing.T) {
	factory := NewFactory()

	// 注册处理器
	routerHandler := NewRouterExpertHandler()
	factory.Register(entity.ModeRouterExpert, routerHandler)

	// 获取处理器
	handler, err := factory.Get(entity.ModeRouterExpert)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if handler.Name() != "router_expert" {
		t.Errorf("expected name 'router_expert', got '%s'", handler.Name())
	}

	// 获取不存在的模式
	_, err = factory.Get("unknown_mode")
	if err == nil {
		t.Error("expected error for unknown mode")
	}
}

func TestFactory_ListModes(t *testing.T) {
	factory := NewFactory()

	factory.Register(entity.ModeRouterExpert, NewRouterExpertHandler())
	factory.Register(entity.ModePlanExec, NewPlanExecHandler())

	modes := factory.ListModes()
	if len(modes) != 2 {
		t.Errorf("expected 2 modes, got %d", len(modes))
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s      string
		substr []string
		expected bool
	}{
		{"设计界面", []string{"设计", "界面"}, true},
		{"开发代码", []string{"设计", "界面"}, false},
		{"hello world", []string{"world", "hello"}, true},
	}

	for _, tt := range tests {
		result := containsAny(tt.s, tt.substr)
		if result != tt.expected {
			t.Errorf("containsAny(%s, %v) = %v, want %v", tt.s, tt.substr, result, tt.expected)
		}
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"ABC", "abc"},
		{"123", "123"},
	}

	for _, tt := range tests {
		result := toLower(tt.input)
		if result != tt.expected {
			t.Errorf("toLower(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}
