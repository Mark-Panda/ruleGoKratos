package entity

import (
	"testing"
	"time"
)

func TestAgentState(t *testing.T) {
	states := []AgentState{
		AgentStateIdle,
		AgentStateWorking,
		AgentStateDone,
		AgentStateFailed,
		AgentStateWaiting,
	}

	expected := []string{"idle", "working", "done", "failed", "waiting"}

	for i, state := range states {
		if string(state) != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], state)
		}
	}
}

func TestCollaborationMode(t *testing.T) {
	modes := []CollaborationMode{
		ModeRouterExpert,
		ModePlanExec,
		ModeSupervision,
		ModePeerHandoff,
	}

	expected := []string{"router_expert", "plan_exec", "supervision", "peer_handoff"}

	for i, mode := range modes {
		if string(mode) != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], mode)
		}
	}
}

func TestAgentDefinition(t *testing.T) {
	agent := &AgentDefinition{
		ID:       "test-agent",
		Name:     "测试Agent",
		Role:     "你是一个测试Agent",
		Desc:     "用于测试",
		Model:    "gpt-4o",
		Tools:    []string{"read_file", "write_file"},
		Enabled:  true,
		Priority: 1,
	}

	if agent.ID != "test-agent" {
		t.Errorf("expected ID 'test-agent', got '%s'", agent.ID)
	}
	if agent.Name != "测试Agent" {
		t.Errorf("expected Name '测试Agent', got '%s'", agent.Name)
	}
	if len(agent.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(agent.Tools))
	}
	if !agent.Enabled {
		t.Error("expected Enabled to be true")
	}
}

func TestAgentInstance(t *testing.T) {
	def := &AgentDefinition{
		ID:      "test-agent",
		Name:    "测试Agent",
		Enabled: true,
	}

	instance := &AgentInstance{
		ID:          "instance-1",
		Definition:  def,
		State:       AgentStateIdle,
		CurrentTask: "",
		Output:      "",
		History:     make([]Turn, 0),
		StartTime:   nil,
		Metadata:    make(map[string]interface{}),
	}

	if instance.ID != "instance-1" {
		t.Errorf("expected ID 'instance-1', got '%s'", instance.ID)
	}
	if instance.State != AgentStateIdle {
		t.Errorf("expected state 'idle', got '%s'", instance.State)
	}
	if instance.Definition.ID != "test-agent" {
		t.Errorf("expected Definition.ID 'test-agent', got '%s'", instance.Definition.ID)
	}
}

func TestTurn(t *testing.T) {
	turn := Turn{
		Timestamp: time.Now().UnixMilli(),
		Role:      "user",
		Content:   "测试输入",
	}

	if turn.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", turn.Role)
	}
	if turn.Content != "测试输入" {
		t.Errorf("expected content '测试输入', got '%s'", turn.Content)
	}
}

func TestToolCall(t *testing.T) {
	toolCall := ToolCall{
		Name:      "read_file",
		Args:      `{"path": "/test.txt"}`,
		Result:    "file content",
		Success:   true,
		Timestamp: time.Now().UnixMilli(),
	}

	if toolCall.Name != "read_file" {
		t.Errorf("expected name 'read_file', got '%s'", toolCall.Name)
	}
	if !toolCall.Success {
		t.Error("expected Success to be true")
	}
}

func TestAgentPool(t *testing.T) {
	now := time.Now()
	pool := &AgentPool{
		ID:          "pool-1",
		Name:        "测试池",
		Description: "用于测试",
		Agents:      make([]*AgentDefinition, 0),
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	if pool.ID != "pool-1" {
		t.Errorf("expected ID 'pool-1', got '%s'", pool.ID)
	}
	if len(pool.Agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(pool.Agents))
	}

	// 添加 Agent
	pool.Agents = append(pool.Agents, &AgentDefinition{
		ID:      "agent-1",
		Name:    "Agent1",
		Enabled: true,
	})

	if len(pool.Agents) != 1 {
		t.Errorf("expected 1 agent after add, got %d", len(pool.Agents))
	}
}

func TestSchemeTemplates(t *testing.T) {
	for mode, bindings := range SchemeTemplates {
		if len(bindings) == 0 {
			t.Errorf("mode %s has no bindings", mode)
		}

		for _, binding := range bindings {
			if binding.AgentID == "" {
				t.Error("binding has empty AgentID")
			}
			if binding.Role == "" {
				t.Error("binding has empty Role")
			}
		}
	}

	// 验证每种模式都有绑定
	expectedModes := []CollaborationMode{
		ModeRouterExpert,
		ModePlanExec,
		ModeSupervision,
		ModePeerHandoff,
	}

	for _, mode := range expectedModes {
		if _, ok := SchemeTemplates[mode]; !ok {
			t.Errorf("mode %s not found in templates", mode)
		}
	}
}

func TestDefaultSchemeConfig(t *testing.T) {
	cfg := DefaultSchemeConfig

	if cfg.MaxIterations != 32 {
		t.Errorf("expected MaxIterations 32, got %d", cfg.MaxIterations)
	}
	if cfg.MaxToolCalls != 64 {
		t.Errorf("expected MaxToolCalls 64, got %d", cfg.MaxToolCalls)
	}
	if cfg.TimeoutSeconds != 300 {
		t.Errorf("expected TimeoutSeconds 300, got %d", cfg.TimeoutSeconds)
	}
}
