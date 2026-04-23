package trace

import (
	"context"
	"ruleGoKratos/internal/biz/entity"
	playgrounddata "ruleGoKratos/internal/data/playground"
	"testing"
	"time"
)

func TestTraceEngine_StartAndEndRun(t *testing.T) {
	repo := playgrounddata.NewTraceRepo()
	engine := NewTraceEngine(repo)

	ctx := context.Background()

	// 开始运行
	run, err := engine.StartRun(ctx, "scheme-1", "测试输入")
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	if run == nil {
		t.Fatal("StartRun returned nil run")
	}

	if run.RunID == "" {
		t.Error("RunID should not be empty")
	}

	if run.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", run.Status)
	}

	// 结束运行
	err = engine.EndRun(ctx, run.RunID, "测试输出", "completed")
	if err != nil {
		t.Fatalf("EndRun failed: %v", err)
	}

	// 验证状态更新
	updatedRun, err := engine.GetRun(ctx, run.RunID)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}

	if updatedRun.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", updatedRun.Status)
	}

	if updatedRun.FinalOutput != "测试输出" {
		t.Errorf("expected output '测试输出', got '%s'", updatedRun.FinalOutput)
	}
}

func TestTraceEngine_EmitEvents(t *testing.T) {
	repo := playgrounddata.NewTraceRepo()
	engine := NewTraceEngine(repo)

	ctx := context.Background()

	run, _ := engine.StartRun(ctx, "scheme-1", "测试")

	// 触发各类事件
	engine.TaskAssigned(ctx, run.RunID, "designer", "设计界面")
	engine.AgentEnterWorker(ctx, run.RunID, "designer", "design_node")
	engine.Thinking(ctx, run.RunID, "designer", "思考中...")
	engine.ToolCall(ctx, run.RunID, "designer", "read_file", "{}")
	engine.ToolResult(ctx, run.RunID, "designer", "read_file", "file content", true)
	engine.AgentExitWorker(ctx, run.RunID, "designer", "design_node", "完成")

	_ = engine.EndRun(ctx, run.RunID, "", "completed")

	// 获取事件
	events, err := engine.GetEvents(ctx, &entity.TraceFilter{RunID: run.RunID, Limit: 100})
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}

	// 验证事件数量（包含 WORKFLOW_START, 各类事件, WORKFLOW_END）
	if len(events) < 6 {
		t.Errorf("expected at least 6 events, got %d", len(events))
	}

	// 验证事件类型
	eventTypes := make(map[entity.TraceEventType]bool)
	for _, e := range events {
		eventTypes[e.Type] = true
	}

	expectedTypes := []entity.TraceEventType{
		entity.TraceEventWorkflowStart,
		entity.TraceEventTaskAssigned,
		entity.TraceEventAgentEnterWorker,
		entity.TraceEventThinking,
		entity.TraceEventToolCall,
		entity.TraceEventToolResult,
		entity.TraceEventAgentExitWorker,
		entity.TraceEventWorkflowEnd,
	}

	for _, et := range expectedTypes {
		if !eventTypes[et] {
			t.Errorf("missing event type: %s", et)
		}
	}
}

func TestTraceEngine_ToolResultRunSubAgentAddsConcurrencyMetadata(t *testing.T) {
	repo := playgrounddata.NewTraceRepo()
	engine := NewTraceEngine(repo)
	ctx := context.Background()
	run, _ := engine.StartRun(ctx, "scheme-1", "测试")

	engine.ToolResult(ctx, run.RunID, "designer", "run_sub_agent", `{"task_count":3,"effective_concurrency":2,"concurrency_reason":"auto_estimated_by_task_count"}`, true)
	events, err := engine.GetEvents(ctx, &entity.TraceFilter{RunID: run.RunID, Limit: 100})
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}
	var got *entity.TraceEvent
	for _, ev := range events {
		if ev.Type == entity.TraceEventToolResult && ev.Metadata["toolName"] == "run_sub_agent" {
			got = ev
			break
		}
	}
	if got == nil {
		t.Fatal("expected run_sub_agent tool result event")
	}
	if got.Metadata["subAgentTaskCount"] != 3 {
		t.Fatalf("unexpected subAgentTaskCount: %#v", got.Metadata["subAgentTaskCount"])
	}
	if got.Metadata["subAgentEffectiveConcurrency"] != 2 {
		t.Fatalf("unexpected subAgentEffectiveConcurrency: %#v", got.Metadata["subAgentEffectiveConcurrency"])
	}
	if got.Metadata["subAgentConcurrencyReason"] != "auto_estimated_by_task_count" {
		t.Fatalf("unexpected subAgentConcurrencyReason: %#v", got.Metadata["subAgentConcurrencyReason"])
	}
}

func TestTraceEngine_Handoff(t *testing.T) {
	repo := playgrounddata.NewTraceRepo()
	engine := NewTraceEngine(repo)

	ctx := context.Background()
	run, _ := engine.StartRun(ctx, "scheme-1", "测试")

	engine.Handoff(ctx, run.RunID, "designer", "engineer", "交接任务")

	events, err := engine.GetEvents(ctx, &entity.TraceFilter{RunID: run.RunID})
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}

	// 查找 Handoff 事件
	found := false
	for _, e := range events {
		if e.Type == entity.TraceEventHandoff {
			found = true
			if e.Metadata["toAgent"] != "engineer" {
				t.Errorf("expected toAgent 'engineer', got '%v'", e.Metadata["toAgent"])
			}
			break
		}
	}

	if !found {
		t.Error("Handoff event not found")
	}
}

func TestTraceEngine_Error(t *testing.T) {
	repo := playgrounddata.NewTraceRepo()
	engine := NewTraceEngine(repo)

	ctx := context.Background()
	run, _ := engine.StartRun(ctx, "scheme-1", "测试")

	engine.Error(ctx, run.RunID, "designer", "执行失败：文件不存在")

	events, err := engine.GetEvents(ctx, &entity.TraceFilter{RunID: run.RunID})
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}

	found := false
	for _, e := range events {
		if e.Type == entity.TraceEventError {
			found = true
			if e.Message != "执行失败：文件不存在" {
				t.Errorf("unexpected message: %s", e.Message)
			}
			break
		}
	}

	if !found {
		t.Error("Error event not found")
	}
}

func TestTraceEvent_WithMetadata(t *testing.T) {
	event := entity.NewTraceEvent("run-1", entity.TraceEventTaskAssigned, "agent-1", "node-1", "test task", "test message")

	event.WithMetadata("key1", "value1")
	event.WithMetadata("key2", 123)

	if event.Metadata["key1"] != "value1" {
		t.Errorf("expected key1='value1', got '%v'", event.Metadata["key1"])
	}

	if event.Metadata["key2"] != 123 {
		t.Errorf("expected key2=123, got '%v'", event.Metadata["key2"])
	}
}

func TestTraceRepo_InMemory(t *testing.T) {
	repo := playgrounddata.NewTraceRepo()
	ctx := context.Background()

	run := &entity.TraceRun{
		ID:        "run-1",
		RunID:     "run-id-1",
		SchemeID:  "scheme-1",
		UserInput: "test input",
		Status:    "running",
		Events:    make([]*entity.TraceEvent, 0),
	}

	// 保存
	err := repo.Save(ctx, run)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 查询
	found, err := repo.FindByRunID(ctx, "run-id-1")
	if err != nil {
		t.Fatalf("FindByRunID failed: %v", err)
	}

	if found.RunID != "run-id-1" {
		t.Errorf("expected RunID 'run-id-1', got '%s'", found.RunID)
	}

	// 追加事件
	event := entity.NewTraceEvent("run-id-1", entity.TraceEventThinking, "agent-1", "node-1", "", "thinking")
	err = repo.AppendEvent(ctx, event)
	if err != nil {
		t.Fatalf("AppendEvent failed: %v", err)
	}

	events, err := repo.FindEvents(ctx, &entity.TraceFilter{RunID: "run-id-1"})
	if err != nil {
		t.Fatalf("FindEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestTraceEngine_SubscribeRun_Fanout(t *testing.T) {
	repo := playgrounddata.NewTraceRepo()
	e := NewTraceEngine(repo)
	ctx := context.Background()
	run, err := e.StartRun(ctx, "s1", "x")
	if err != nil {
		t.Fatal(err)
	}
	ch, unsub := e.SubscribeRun(run.RunID)
	defer unsub()
	e.Thinking(ctx, run.RunID, "a1", "hi")
	select {
	case ev := <-ch:
		if ev == nil || ev.Type != entity.TraceEventThinking {
			t.Fatalf("unexpected event: %#v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for fan-out event")
	}
}
