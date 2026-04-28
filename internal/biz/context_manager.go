package biz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-kratos/kratos/v2/log"
)

// ContextConfig 上下文管理配置
type ContextConfig struct {
	MaxHistoryMessages int  // 滑动窗口：最大保留消息数
	SummaryThreshold   int  // 触发摘要的消息数阈值
	MemoryEnabled      bool // 是否启用记忆
}

// DefaultContextConfig 默认配置
func DefaultContextConfig() ContextConfig {
	return ContextConfig{
		MaxHistoryMessages: 20,
		SummaryThreshold:   15,
		MemoryEnabled:      true,
	}
}

// ContextManager 上下文管理器
// 负责滑动窗口、消息摘要、记忆注入
type ContextManager struct {
	config      ContextConfig
	memoryStore MemoryStore
	chatModelFunc func(ctx context.Context) (model.ToolCallingChatModel, error)
	chatModel   interface{} // 直接设置的模型实例，优先级高于 chatModelFunc
	log         *log.Helper
}

// NewContextManager 创建上下文管理器
func NewContextManager(config ContextConfig, store MemoryStore, modelFunc func(ctx context.Context) (model.ToolCallingChatModel, error), logger *log.Helper) *ContextManager {
	if config.MaxHistoryMessages <= 0 {
		config.MaxHistoryMessages = 20
	}
	if config.SummaryThreshold <= 0 {
		config.SummaryThreshold = 15
	}
	return &ContextManager{
		config:        config,
		memoryStore:   store,
		chatModelFunc: modelFunc,
		log:           logger,
	}
}

// SetChatModelFunc 设置用于摘要的 chatModelFunc
func (cm *ContextManager) SetChatModelFunc(fn func(ctx context.Context) (model.ToolCallingChatModel, error)) {
	cm.chatModelFunc = fn
}

// SetChatModel 直接设置用于摘要的模型实例
func (cm *ContextManager) SetChatModel(m interface{}) {
	cm.chatModel = m
}

// BuildMessages 构建最终的消息列表
// 应用滑动窗口、摘要、记忆注入策略
func (cm *ContextManager) BuildMessages(
	ctx context.Context,
	history []HistoryMessage,
	currentInput string,
	systemPrompt string,
	userID string,
	projectPath string,
	attachments []HarnessAttachment,
) ([]*schema.Message, error) {
	// 1. 构建基础消息列表（不含摘要）
	msgs := make([]*schema.Message, 0, len(history)+4)

	// 2. 系统提示（含记忆上下文）
	effectiveSystemPrompt := systemPrompt
	// 兜底：若 userID 为空，生成临时标识以启用会话内记忆
	if userID == "" {
		userID = generateTempUserID()
	}
	if cm.config.MemoryEnabled && userID != "" {
		if memCtx := cm.getUserContext(ctx, userID, projectPath); memCtx != "" {
			effectiveSystemPrompt = strings.TrimSpace(systemPrompt) + "\n\n" + memCtx
		}
	}
	msgs = append(msgs, schema.SystemMessage(effectiveSystemPrompt))

	// 3. 应用滑动窗口和摘要
	processedHistory, summary, err := cm.processHistory(ctx, history)
	if err != nil {
		cm.log.Warnf("process history failed: %v", err)
		// 失败时降级为直接使用历史（不做摘要）
		processedHistory = history
	}

	// 4. 如果有摘要，注入摘要消息
	if summary != "" {
		summaryMsg := fmt.Sprintf("[对话摘要] 以下是之前对话的简要总结，请结合此上下文理解当前问题：\n%s", summary)
		msgs = append(msgs, schema.UserMessage(summaryMsg))
	}

	// 5. 添加处理后的历史消息
	for _, item := range processedHistory {
		switch item.Role {
		case "assistant":
			msgs = append(msgs, schema.AssistantMessage(item.Content, nil))
		default:
			msgs = append(msgs, schema.UserMessage(item.Content))
		}
	}

	// 6. 添加当前用户消息（含附件）
	parts := buildHarnessInputParts(currentInput, attachments)
	msgs = append(msgs, lastUserMessageFromParts(parts))

	return msgs, nil
}

// processHistory 处理历史消息，应用滑动窗口策略
// 返回：处理后的消息、摘要（如果有）
func (cm *ContextManager) processHistory(ctx context.Context, history []HistoryMessage) ([]HistoryMessage, string, error) {
	if len(history) == 0 {
		return nil, "", nil
	}

	// 如果历史消息数量未超过阈值，直接返回
	if !cm.shouldSummarize(len(history)) {
		return history, "", nil
	}

	// 需要摘要：保留最近 N 条，早期消息进行摘要
	recentCount := cm.config.MaxHistoryMessages
	if recentCount > len(history) {
		recentCount = len(history)
	}

	recentHistory := history[len(history)-recentCount:]
	oldHistory := history[:len(history)-recentCount]

	// 调用 LLM 进行摘要
	summary, err := cm.summarizeMessages(ctx, oldHistory)
	if err != nil {
		cm.log.Warnf("summarize messages failed: %v", err)
		// 摘要失败时降级：保留最近的摘要阈值数量的消息
		keepCount := cm.config.SummaryThreshold
		if keepCount > len(history) {
			keepCount = len(history)
		}
		return history[len(history)-keepCount:], "", err
	}

	return recentHistory, summary, nil
}

// shouldSummarize 判断是否需要摘要
func (cm *ContextManager) shouldSummarize(historyLen int) bool {
	return historyLen > cm.config.SummaryThreshold
}

// summarizeMessages 调用 LLM 摘要一组消息
func (cm *ContextManager) summarizeMessages(ctx context.Context, messages []HistoryMessage) (string, error) {
	// 优先使用直接设置的模型实例
	if cm.chatModel != nil {
		return cm.doSummarize(ctx, cm.chatModel, messages)
	} else if cm.chatModelFunc != nil {
		m, err := cm.chatModelFunc(ctx)
		if err != nil {
			return cm.simpleSummarize(messages)
		}
		return cm.doSummarize(ctx, m, messages)
	}
	return cm.simpleSummarize(messages)
}

// doSummarize 执行实际的摘要调用
func (cm *ContextManager) doSummarize(ctx context.Context, model interface{}, messages []HistoryMessage) (string, error) {
	// 构建摘要 prompt
	var promptBuilder strings.Builder
	promptBuilder.WriteString("请简要总结以下对话的关键信息和结论，保留对后续决策重要的内容。\n\n格式要求：\n- summary: 简要概括对话主题和主要结论（1-2句话）\n- key_points: 列出关键发现或决策点（用1句话描述每个）\n- decisions: 列出做出的决定（如有）\n\n对话内容：\n")

	for i, msg := range messages {
		role := "用户"
		if msg.Role == "assistant" {
			role = "助手"
		}
		// 截断过长消息避免 token 浪费
		content := msg.Content
		if len(content) > 500 {
			content = content[:500] + "...(内容截断)"
		}
		promptBuilder.WriteString(fmt.Sprintf("[%d] %s: %s\n", i+1, role, content))
	}

	summaryPrompt := promptBuilder.String()

	// 使用反射调用模型的 Generate 方法
	resp, err := callModelGenerate(ctx, model, summaryPrompt)
	if err != nil {
		return "", fmt.Errorf("llm summarize failed: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}

// callModelGenerate 反射调用模型的 Generate 方法
func callModelGenerate(ctx context.Context, model interface{}, prompt string) (*schema.Message, error) {
	msgs := []*schema.Message{schema.UserMessage(prompt)}

	// 尝试 type assertion 处理不同接口
	switch m := model.(type) {
	case interface{ Generate(context.Context, []*schema.Message, ...interface{}) (*schema.Message, error) }:
		return m.Generate(ctx, msgs)
	case interface{ Generate(context.Context, []*schema.Message) (*schema.Message, error) }:
		return m.Generate(ctx, msgs)
	}

	// 使用反射调用
	generateVal := reflect.ValueOf(model).MethodByName("Generate")
	if !generateVal.IsValid() {
		return nil, fmt.Errorf("model does not have Generate method")
	}

	// BaseChatModel.Generate 签名: Generate(ctx, msgs, opts ...Option)
	// 反射调用需要正确处理 variadic 参数
	results := generateVal.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(msgs),
	})
	if len(results) < 2 {
		return nil, fmt.Errorf("unexpected Generate return count")
	}
	if !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}
	return results[0].Interface().(*schema.Message), nil
}

// simpleSummarize 简单的非 LLM 摘要（降级方案）
func (cm *ContextManager) simpleSummarize(messages []HistoryMessage) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	var summaryBuilder strings.Builder
	userMsgs := 0
	assistantMsgs := 0

	for _, msg := range messages {
		if msg.Role == "user" {
			userMsgs++
		} else {
			assistantMsgs++
		}
	}

	summaryBuilder.WriteString(fmt.Sprintf("对话摘要：共 %d 轮对话（用户 %d 次，助手 %d 次）。",
		len(messages), userMsgs, assistantMsgs))

	// 保留最后一条用户消息的主题
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			content := messages[i].Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			summaryBuilder.WriteString(fmt.Sprintf("\n最后用户问题：%s", content))
			break
		}
	}

	return summaryBuilder.String(), nil
}

// getUserContext 获取用户和项目的记忆上下文
func (cm *ContextManager) getUserContext(ctx context.Context, userID, projectPath string) string {
	if cm.memoryStore == nil {
		return ""
	}

	var parts []string

	// 获取用户记忆
	if userID != "" {
		userMem, err := cm.memoryStore.GetUserMemory(ctx, userID)
		if err == nil && userMem != nil {
			if ctxStr := userMem.BuildContext(); ctxStr != "" {
				parts = append(parts, ctxStr)
			}
		}
	}

	// 获取项目记忆
	if projectPath != "" {
		projMem, err := cm.memoryStore.GetProjectMemory(ctx, projectPath)
		if err == nil && projMem != nil {
			if ctxStr := projMem.BuildContext(); ctxStr != "" {
				parts = append(parts, ctxStr)
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return "\n\n[记忆上下文]\n" + strings.Join(parts, "\n")
}

// SaveSessionSummary 保存会话摘要到记忆
func (cm *ContextManager) SaveSessionSummary(ctx context.Context, userID, projectPath string, summary string) error {
	if cm.memoryStore == nil || !cm.config.MemoryEnabled {
		return nil
	}

	store, ok := cm.memoryStore.(*FileMemoryStore)
	if !ok {
		return nil
	}

	if projectPath != "" {
		if err := store.AddSessionSummary(ctx, projectPath, summary); err != nil {
			cm.log.Warnf("save session summary failed: %v", err)
			return err
		}
	}

	return nil
}

// RecordUserPreference 记录用户偏好
func (cm *ContextManager) RecordUserPreference(ctx context.Context, userID, preference, source string) error {
	if cm.memoryStore == nil || !cm.config.MemoryEnabled {
		return nil
	}

	store, ok := cm.memoryStore.(*FileMemoryStore)
	if !ok {
		return nil
	}

	return store.AddUserPreference(ctx, userID, preference, source)
}

// RecordProjectFact 记录项目事实
func (cm *ContextManager) RecordProjectFact(ctx context.Context, projectPath, fact, source string) error {
	if cm.memoryStore == nil || !cm.config.MemoryEnabled {
		return nil
	}

	store, ok := cm.memoryStore.(*FileMemoryStore)
	if !ok {
		return nil
	}

	return store.AddProjectFact(ctx, projectPath, fact, source)
}

// RecordDecision 记录项目决策
func (cm *ContextManager) RecordDecision(ctx context.Context, projectPath, decision string) error {
	if cm.memoryStore == nil || !cm.config.MemoryEnabled {
		return nil
	}

	store, ok := cm.memoryStore.(*FileMemoryStore)
	if !ok {
		return nil
	}

	return store.AddDecision(ctx, projectPath, decision)
}

func generateTempUserID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "temp_" + hex.EncodeToString(b)
}
