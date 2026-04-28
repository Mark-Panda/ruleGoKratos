package biz

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

func TestPersistSessionMemoryWritesSummary(t *testing.T) {
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
	uc := &AgentUsecase{
		log:            logger,
		contextManager: cm,
	}

	req := HarnessRequest{
		UserID:      "user-1",
		ProjectPath: "/workspace/demo",
		Input:       "请总结本次执行",
	}
	uc.persistSessionMemory(context.Background(), req, "本次执行成功")

	mem, err := store.GetProjectMemory(context.Background(), "/workspace/demo")
	if err != nil {
		t.Fatalf("GetProjectMemory failed: %v", err)
	}
	if len(mem.Summaries.Entries) != 1 {
		t.Fatalf("expected one summary entry, got %d", len(mem.Summaries.Entries))
	}
	content := mem.Summaries.Entries[0].Content
	if !strings.Contains(content, "用户问题：请总结本次执行") || !strings.Contains(content, "助手结论：本次执行成功") {
		t.Fatalf("unexpected summary content: %q", content)
	}
	userMem, err := store.GetUserMemory(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetUserMemory failed: %v", err)
	}
	if len(userMem.Feedback.Entries) != 1 {
		t.Fatalf("expected one user feedback entry, got %d", len(userMem.Feedback.Entries))
	}
}

func TestPersistSessionMemorySkipsSubAgent(t *testing.T) {
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
	uc := &AgentUsecase{
		log:            logger,
		contextManager: cm,
	}

	req := HarnessRequest{
		ProjectPath:   "/workspace/demo",
		Input:         "子任务",
		SubAgentDepth: 1,
	}
	uc.persistSessionMemory(context.Background(), req, "子任务输出")

	mem, err := store.GetProjectMemory(context.Background(), "/workspace/demo")
	if err != nil {
		t.Fatalf("GetProjectMemory failed: %v", err)
	}
	if len(mem.Summaries.Entries) != 0 {
		t.Fatalf("expected sub-agent memory write skipped, got %d entries", len(mem.Summaries.Entries))
	}
}
