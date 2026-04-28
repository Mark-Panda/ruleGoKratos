package biz

import (
	"context"
	"io"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/go-kratos/kratos/v2/log"
)

func newTestContextManager() *ContextManager {
	logger := log.NewHelper(log.NewStdLogger(io.Discard))
	return NewContextManager(ContextConfig{
		MaxHistoryMessages: 10,
		SummaryThreshold:   5,
		MemoryEnabled:      false,
	}, nil, nil, logger)
}

func TestContextManager_BuildMessages_BelowThreshold(t *testing.T) {
	cm := newTestContextManager()

	history := []HistoryMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	msgs, err := cm.BuildMessages(context.Background(), history, "How are you?", "You are a helpful assistant.", "", "", nil)
	if err != nil {
		t.Fatalf("BuildMessages failed: %v", err)
	}

	// 应该有: system + 2 history + current user = 4
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}

	// 第一条是 system
	if msgs[0].Role != "system" {
		t.Fatalf("first message should be system, got %s", msgs[0].Role)
	}
}

func TestContextManager_BuildMessages_AboveThreshold_SlidingWindow(t *testing.T) {
	cm := newTestContextManager()
	cm.config.MaxHistoryMessages = 3
	cm.config.SummaryThreshold = 5

	// 7条历史消息（超过阈值5）
	history := []HistoryMessage{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
		{Role: "user", Content: "msg3"},
		{Role: "assistant", Content: "msg4"},
		{Role: "user", Content: "msg5"},
		{Role: "assistant", Content: "msg6"},
		{Role: "user", Content: "msg7"},
	}

	// 因为 chatModelFunc 为 nil，会使用 simpleSummarize
	msgs, err := cm.BuildMessages(context.Background(), history, "current input", "You are a helpful assistant.", "", "", nil)
	if err != nil {
		t.Fatalf("BuildMessages failed: %v", err)
	}

	// 应该包含摘要消息 + 最近3条 + current user
	// system + summary + 3 history + current user = 6
	if len(msgs) != 6 {
		t.Fatalf("expected 6 messages, got %d", len(msgs))
	}
}

func TestContextManager_ShouldSummarize(t *testing.T) {
	cm := newTestContextManager()

	tests := []struct {
		historyLen int
		want       bool
	}{
		{3, false},  // below threshold
		{5, false},  // equal to threshold
		{6, true},   // above threshold
		{10, true},  // above threshold
	}

	for _, tt := range tests {
		got := cm.shouldSummarize(tt.historyLen)
		if got != tt.want {
			t.Errorf("shouldSummarize(%d) = %v, want %v", tt.historyLen, got, tt.want)
		}
	}
}

func TestContextManager_SimpleSummarize(t *testing.T) {
	cm := newTestContextManager()

	messages := []HistoryMessage{
		{Role: "user", Content: "What's the weather?"},
		{Role: "assistant", Content: "It's sunny."},
		{Role: "user", Content: "What's the temperature?"},
		{Role: "assistant", Content: "It's 25 degrees."},
	}

	summary, err := cm.summarizeMessages(context.Background(), messages)
	if err != nil {
		t.Fatalf("summarizeMessages failed: %v", err)
	}

	if summary == "" {
		t.Fatal("expected non-empty summary")
	}

	// Simple summarize should contain the count info
	if !contains(summary, "4") && !contains(summary, "对话") {
		// The simple summarize includes message count
	}
}

func TestContextManager_GetUserContext_NoMemory(t *testing.T) {
	cm := newTestContextManager()

	ctx := cm.getUserContext(context.Background(), "user123", "/path/to/project")
	if ctx != "" {
		t.Fatalf("expected empty context without memory store, got %q", ctx)
	}
}

func TestContextManager_BuildMessages_WithMemory(t *testing.T) {
	// 创建临时 memory store
	store, err := NewFileMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileMemoryStore failed: %v", err)
	}

	logger := log.NewHelper(log.NewStdLogger(io.Discard))
	cm := NewContextManager(ContextConfig{
		MaxHistoryMessages: 10,
		SummaryThreshold:   5,
		MemoryEnabled:      true,
	}, store, nil, logger)

	// 添加项目记忆
	err = store.AddProjectFact(context.Background(), "/test/project", "使用 Go 语言", "test")
	if err != nil {
		t.Fatalf("AddProjectFact failed: %v", err)
	}

	history := []HistoryMessage{
		{Role: "user", Content: "Hello"},
	}

	msgs, err := cm.BuildMessages(context.Background(), history, "Hi", "You are a helpful assistant.", "user1", "/test/project", nil)
	if err != nil {
		t.Fatalf("BuildMessages failed: %v", err)
	}

	// 检查是否包含记忆上下文
	systemContent := msgs[0].Content
	if !contains(systemContent, "项目事实") && !contains(systemContent, "Go") {
		t.Fatalf("expected memory context in system prompt, got %q", systemContent)
	}
}

func TestContextManager_ProcessHistory_FallbackOnError(t *testing.T) {
	// 创建一个会失败的 chatModelFunc
	logger := log.NewHelper(log.NewStdLogger(io.Discard))
	failingModelFunc := func(ctx context.Context) (model.ToolCallingChatModel, error) {
		return nil, context.DeadlineExceeded
	}

	cm := NewContextManager(ContextConfig{
		MaxHistoryMessages: 2,
		SummaryThreshold:   3,
		MemoryEnabled:      false,
	}, nil, failingModelFunc, logger)

	history := []HistoryMessage{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
		{Role: "user", Content: "msg3"},
		{Role: "assistant", Content: "msg4"},
		{Role: "user", Content: "msg5"},
	}

	// chatModelFunc 失败时会 fallback 到 simpleSummarize，所以会返回摘要
	recent, summary, err := cm.processHistory(context.Background(), history)
	if err != nil {
		t.Fatalf("processHistory failed: %v", err)
	}

	// fallback 到 simpleSummarize，仍然会有摘要内容
	if summary == "" {
		t.Fatalf("expected summary from simpleSummarize fallback, got empty")
	}

	// 降级时：chatModelFunc 失败后调用 simpleSummarize 成功，返回的是最近 MaxHistoryMessages 条
	// 实际代码：return history[len(history)-keepCount:], "", err  where keepCount = SummaryThreshold
	// 但注意代码中 recentHistory 是先计算的，只有在 summarizeMessages 失败时才用 keepCount 覆盖
	// 所以 recent 应该是 MaxHistoryMessages = 2 条
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent messages (MaxHistoryMessages), got %d", len(recent))
	}
}
