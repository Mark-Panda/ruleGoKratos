package collaboration

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
)

// RunAgentHarness 使用主站 Harness（工具链 / 托管 Agent 配置）执行单轮对话并收集完整文本。
// 池内 Agent 必须设置 ManagedAgentID，否则无法解析模型与凭证（与 Chat 一致）。
func RunAgentHarness(
	ctx context.Context,
	rt *CollaborationRuntime,
	runID string,
	def *entity.AgentDefinition,
	userInput string,
	history []biz.HistoryMessage,
	trace TraceEmitter,
	schemeCfg *entity.SchemeConfig,
) (string, error) {
	if rt == nil || rt.AgentUC == nil {
		return mockAgentOutput(def, userInput), nil
	}
	if def == nil {
		return "", fmt.Errorf("agent definition is nil")
	}
	if def.ManagedAgentID <= 0 {
		return "", fmt.Errorf(
			"Agent「%s」未关联主站「Agent 配置」(managedAgentId)。请在 Agent Playground 中为该成员绑定托管配置后再运行协作",
			def.Name,
		)
	}

	msg := buildCollaborationUserMessage(def, userInput)
	req := biz.HarnessRequest{
		ManagedAgentID:      def.ManagedAgentID,
		Input:               msg,
		History:             history,
		WorkspaceSessionDir: filepath.Join("playground", "run_"+sanitizePlaygroundRunIDForPath(runID)),
		PlaygroundRunID:     runID,
		PlaygroundAgentID:   def.ID,
		TraceSink:           traceForwarder{t: trace},
	}
	if schemeCfg != nil {
		req.ConfigOverride = harnessConfigFromScheme(schemeCfg)
	}

	raw, err := rt.AgentUC.ExecuteHarnessSync(ctx, req)
	if err != nil {
		if trace != nil {
			trace.Error(runID, def.ID, err.Error())
		}
		return "", err
	}
	out := strings.TrimSpace(raw)
	if out == "" {
		return "", fmt.Errorf("模型未返回有效内容（可能超时或被中断）")
	}
	return out, nil
}

// traceForwarder 将 biz 层工具回调转给 collaboration TraceEmitter（非 nil 时生效）。
type traceForwarder struct{ t TraceEmitter }

func (f traceForwarder) EmitToolCall(runID, agentID, toolName, args string) {
	if f.t == nil {
		return
	}
	f.t.ToolCall(runID, agentID, toolName, args)
}

func (f traceForwarder) EmitToolResult(runID, agentID, toolName, result string, success bool) {
	if f.t == nil {
		return
	}
	f.t.ToolResult(runID, agentID, toolName, result, success)
}

// buildCollaborationUserMessage 构造送入 Harness 的「用户」消息正文。
// 已绑定托管 Agent 时：系统提示词仅来自主站 Agent 配置（见 enrichHarnessWithManagedAgent），
// 此处不再拼接池内 Role/Desc，避免覆盖或与托管 prompt 冲突。
func buildCollaborationUserMessage(def *entity.AgentDefinition, userInput string) string {
	in := strings.TrimSpace(userInput)
	if in == "" {
		return ""
	}
	if def != nil && def.ManagedAgentID > 0 {
		return "【Playground 协作】用户任务：\n\n" + in
	}
	name := strings.TrimSpace(def.Name)
	if name == "" && def != nil {
		name = def.ID
	}
	role := ""
	if def != nil {
		role = strings.TrimSpace(def.Role)
	}
	header := fmt.Sprintf("【Playground 多 Agent 协作】角色：%s", name)
	if role != "" {
		header += fmt.Sprintf("\n说明：%s", role)
	}
	return header + "\n\n用户任务：\n" + in
}

func harnessConfigFromScheme(cfg *entity.SchemeConfig) *biz.HarnessConfig {
	if cfg == nil {
		return nil
	}
	out := &biz.HarnessConfig{}
	if cfg.MaxIterations > 0 {
		out.MaxIterations = cfg.MaxIterations
	}
	if cfg.MaxToolCalls > 0 {
		out.MaxToolCalls = cfg.MaxToolCalls
	}
	// 方案 TimeoutSeconds：用于 LLM 流式 HTTP 整体读超时（规划、长上下文）；勿再映射到 ToolTimeoutSecs。
	if cfg.TimeoutSeconds > 0 {
		out.StreamTimeoutSecs = cfg.TimeoutSeconds
	}
	if out.MaxIterations == 0 && out.MaxToolCalls == 0 && out.StreamTimeoutSecs == 0 {
		return nil
	}
	return out
}

// sanitizePlaygroundRunIDForPath 将 runId 转为目录名安全片段（仅保留字母数字、-、_）。
func sanitizePlaygroundRunIDForPath(runID string) string {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return "unknown"
	}
	const allowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	var b strings.Builder
	b.Grow(len(runID))
	for _, r := range runID {
		if strings.ContainsRune(allowed, r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "unknown"
	}
	return out
}

func mockAgentOutput(def *entity.AgentDefinition, userInput string) string {
	name := "Agent"
	if def != nil && strings.TrimSpace(def.Name) != "" {
		name = def.Name
	}
	return fmt.Sprintf("[%s] 已完成处理: %s", name, userInput)
}
