package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ruleGoKratos/internal/biz"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/str"
)

// ruleGoAgentUsecase 由应用启动时 wire.Invoke 注入，供规则链节点调用 Agent Harness。
var ruleGoAgentUsecase *biz.AgentUsecase

// SetRuleGoAgentUsecase 注册 Agent 业务实例（由 internal/data.WireRuleGoAgent 调用）。
func SetRuleGoAgentUsecase(uc *biz.AgentUsecase) {
	ruleGoAgentUsecase = uc
}

func init() {
	_ = rulego.Registry.Register(&AgentHarnessLLM{})
}

// AgentHarnessLLM 自定义 LLM 节点：走 Eino tool-calling，可按节点配置启用 Skill 等工具。
type AgentHarnessLLM struct {
	Config AgentHarnessLLMConfig

	modelTpl  str.Template
	systemTpl str.Template
	userTpl   str.Template
	hasVar    bool
}

// AgentHarnessLLMConfig 与 flowgram DSL 导出字段对齐（camelCase）。
type AgentHarnessLLMConfig struct {
	LlmConfigID        int64  `json:"llmConfigId"`
	LlmModelEntryID    int64  `json:"llmModelEntryId"`
	ManagedAgentID     int64  `json:"managedAgentId"`
	WorkspaceID        string `json:"workspaceId"`
	Model              string `json:"model"`
	SystemPrompt       string `json:"systemPrompt"`
	UserPrompt         string `json:"userPrompt"`
	EnableSkillTool    bool   `json:"enableSkillTool"`
	EnableSubAgentTool bool   `json:"enableSubAgentTool"`
	// EnableUUIDTool 已废弃：运行时固定启用 UUID 工具，DSL 若仍含该字段会被忽略。
	EnableUUIDTool       bool `json:"enableUUIDTool"`
	EnableWorkspaceTools bool `json:"enableWorkspaceTools"`
	MaxIterations        int  `json:"maxIterations"`
	MaxToolCalls         int  `json:"maxToolCalls"`
	ToolTimeoutSecs      int  `json:"toolTimeoutSecs"`
	// GitWorktreeMode 启用后，运行时会在系统提示词中注入强制约束：
	// 模型对 git 仓库的所有修改性操作必须通过 git worktree 创建隔离工作树，禁止直接在主分支上操作。
	GitWorktreeMode bool `json:"gitWorktreeMode"`
}

func (x *AgentHarnessLLM) Type() string {
	return "ai/agentHarness"
}

func (x *AgentHarnessLLM) New() types.Node {
	return &AgentHarnessLLM{Config: AgentHarnessLLMConfig{
		EnableSkillTool:    true,
		EnableSubAgentTool: true,
	}}
}

func (x *AgentHarnessLLM) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "agentHarness",
		Desc:  "Agent LLM（可配置 Skill 工具；MCP 默认加载所有已启用配置；支持从 msg.data.attachments / metadata.attachments 读取多模态附件）",
	}
}

func (x *AgentHarnessLLM) Init(_ types.Config, configuration types.Configuration) error {
	b, err := json.Marshal(configuration)
	if err != nil {
		return err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	delete(raw, "skillAllowlist")
	if _, ok := raw["enableSubAgentTool"]; !ok {
		raw["enableSubAgentTool"] = true
	}
	b2, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b2, &x.Config); err != nil {
		return err
	}

	x.hasVar = false
	track := func(t str.Template) {
		if !t.IsNotVar() {
			x.hasVar = true
		}
	}
	x.modelTpl = str.NewTemplate(x.Config.Model)
	track(x.modelTpl)
	x.systemTpl = str.NewTemplate(x.Config.SystemPrompt)
	track(x.systemTpl)
	x.userTpl = str.NewTemplate(x.Config.UserPrompt)
	track(x.userTpl)
	return nil
}

func (x *AgentHarnessLLM) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	if ruleGoAgentUsecase == nil {
		ctx.TellFailure(msg, errors.New("Agent harness 未注入，请检查服务启动是否调用 WireRuleGoAgent"))
		return
	}
	var env map[string]interface{}
	if x.hasVar {
		env = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	} else {
		env = map[string]interface{}{}
	}

	modelName := x.modelTpl.Execute(env)
	systemPrompt := x.systemTpl.Execute(env)
	userPrompt := x.userTpl.Execute(env)
	workspaceID := strings.TrimSpace(x.Config.WorkspaceID)
	if wp := buildWorkspacePromptForComponent(workspaceID); wp != "" {
		if strings.TrimSpace(systemPrompt) == "" {
			systemPrompt = wp
		} else {
			systemPrompt = systemPrompt + "\n\n" + wp
		}
	}
	workspaceRoot := resolveWorkspaceRootForComponent(workspaceID)
	if workspaceID != "" && workspaceRoot == "" {
		ctx.TellFailure(msg, fmt.Errorf("agentHarness: 工作区 %s 不可用，请先执行“工作区刷新”并确认仓库已同步", workspaceID))
		return
	}

	toolOpts := &biz.HarnessToolOptions{
		EnableUUIDTool:       true, // 不在节点上暴露开关，固定启用 generate_uuid
		EnableSkillTool:      x.Config.EnableSkillTool,
		EnableMcpTool:        true,
		EnableWorkspaceTools: true, // Agent-LLM 节点固定开启，不允许关闭
		EnableSubAgentTool:   x.Config.EnableSubAgentTool,
	}

	var cfgOverride *biz.HarnessConfig
	if x.Config.MaxIterations > 0 || x.Config.MaxToolCalls > 0 || x.Config.ToolTimeoutSecs > 0 {
		cfgOverride = &biz.HarnessConfig{
			MaxIterations:   x.Config.MaxIterations,
			MaxToolCalls:    x.Config.MaxToolCalls,
			ToolTimeoutSecs: x.Config.ToolTimeoutSecs,
		}
	}

	req := biz.HarnessRequest{
		Model:           strings.TrimSpace(modelName),
		History:         nil,
		Input:           userPrompt,
		Attachments:     extractHarnessAttachments(msg.GetData(), env),
		SystemPrompt:    systemPrompt,
		ConfigOverride:  cfgOverride,
		ToolOptions:     toolOpts,
		ManagedAgentID:  x.Config.ManagedAgentID,
		LlmConfigID:     x.Config.LlmConfigID,
		LlmModelEntryID: x.Config.LlmModelEntryID,
		UserID:          extractUserIDFromMsg(msg),
		ProjectPath:     extractProjectPathFromMsg(msg),
		WorkspaceRoot:   workspaceRoot,
		GitWorktreeMode: x.Config.GitWorktreeMode,
	}

	out, err := ruleGoAgentUsecase.ExecuteHarnessSync(ctx.GetContext(), req)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	msg.DataType = types.TEXT
	msg.SetData(out)
	ctx.TellSuccess(msg)
}

func (x *AgentHarnessLLM) Destroy() {}

func extractUserIDFromMsg(msg types.RuleMsg) string {
	if msg.Metadata == nil {
		return ""
	}
	return msg.Metadata.GetValue("x-user-id")
}

func extractProjectPathFromMsg(msg types.RuleMsg) string {
	if msg.Metadata == nil {
		return ""
	}
	return msg.Metadata.GetValue("x-project-path")
}

type harnessAttachmentPayload struct {
	Filename      string `json:"filename"`
	MimeType      string `json:"mimeType"`
	Text          string `json:"text"`
	ContentBase64 string `json:"contentBase64"`
}

func extractHarnessAttachments(msgData string, env map[string]interface{}) []biz.HarnessAttachment {
	candidates := make([]interface{}, 0, 4)
	if s := strings.TrimSpace(msgData); s != "" {
		var parsed interface{}
		if err := json.Unmarshal([]byte(s), &parsed); err == nil {
			candidates = append(candidates, parsed)
		}
	}
	if env != nil {
		candidates = append(candidates, env)
		if md, ok := env["metadata"]; ok {
			candidates = append(candidates, md)
		}
		if msgVal, ok := env["msg"]; ok {
			candidates = append(candidates, msgVal)
		}
	}
	for _, candidate := range candidates {
		if atts := extractHarnessAttachmentsFromValue(candidate); len(atts) > 0 {
			return atts
		}
	}
	return nil
}

func extractHarnessAttachmentsFromValue(v interface{}) []biz.HarnessAttachment {
	switch typed := v.(type) {
	case map[string]interface{}:
		if raw, ok := typed["attachments"]; ok {
			return decodeHarnessAttachments(raw)
		}
	case []interface{}:
		return decodeHarnessAttachments(typed)
	case string:
		return decodeHarnessAttachments(typed)
	}
	return nil
}

func decodeHarnessAttachments(raw interface{}) []biz.HarnessAttachment {
	switch typed := raw.(type) {
	case nil:
		return nil
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		var parsed interface{}
		if err := json.Unmarshal([]byte(typed), &parsed); err != nil {
			return nil
		}
		return decodeHarnessAttachments(parsed)
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var payloads []harnessAttachmentPayload
	if err := json.Unmarshal(b, &payloads); err != nil {
		return nil
	}
	out := make([]biz.HarnessAttachment, 0, len(payloads))
	for _, item := range payloads {
		if strings.TrimSpace(item.Filename) == "" &&
			strings.TrimSpace(item.MimeType) == "" &&
			strings.TrimSpace(item.Text) == "" &&
			strings.TrimSpace(item.ContentBase64) == "" {
			continue
		}
		out = append(out, biz.HarnessAttachment{
			Filename:      item.Filename,
			MimeType:      item.MimeType,
			Text:          item.Text,
			ContentBase64: item.ContentBase64,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type componentWorkspaceRepoItem struct {
	URL string `json:"url"`
	Dir string `json:"dir"`
}

type componentWorkspaceMeta struct {
	Name         string                       `json:"name"`
	RootDir      string                       `json:"rootDir"`
	Repositories []componentWorkspaceRepoItem `json:"repositories"`
}

type componentWorkspaceFile struct {
	Meta componentWorkspaceMeta `json:"ruleGoWorkspace"`
}

func resolveWorkspaceConfigPathForComponent(workspaceID string) string {
	if workspaceID == "" {
		return ""
	}
	filename := workspaceID + ".code-workspace"
	candidates := []string{
		filepath.Join("/app/code_workspace", filename),
		filepath.Join("code_workspace", filename),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func readWorkspaceMetaForComponent(workspaceID string) (*componentWorkspaceMeta, error) {
	cfgPath := resolveWorkspaceConfigPathForComponent(workspaceID)
	if cfgPath == "" {
		return nil, os.ErrNotExist
	}
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	var parsed componentWorkspaceFile
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, err
	}
	return &parsed.Meta, nil
}

func resolveWorkspaceRootForComponent(workspaceID string) string {
	meta, err := readWorkspaceMetaForComponent(workspaceID)
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(meta.RootDir)
	if root == "" {
		root = filepath.Join("/app/code_workspace", workspaceID)
	}
	if !filepath.IsAbs(root) {
		if abs, absErr := filepath.Abs(root); absErr == nil {
			root = abs
		}
	}
	if st, statErr := os.Stat(root); statErr != nil || !st.IsDir() {
		return ""
	}
	return root
}

func buildWorkspacePromptForComponent(workspaceID string) string {
	if workspaceID == "" {
		return ""
	}
	meta, err := readWorkspaceMetaForComponent(workspaceID)
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(meta.Name)
	if name == "" {
		name = workspaceID
	}
	root := strings.TrimSpace(meta.RootDir)
	if root == "" {
		root = filepath.Join("/app/code_workspace", workspaceID)
	}
	lines := make([]string, 0, len(meta.Repositories))
	for _, repo := range meta.Repositories {
		url := strings.TrimSpace(repo.URL)
		if url == "" {
			continue
		}
		if d := strings.TrimSpace(repo.Dir); d != "" {
			lines = append(lines, fmt.Sprintf("- %s（目录: %s）", url, d))
		} else {
			lines = append(lines, fmt.Sprintf("- %s", url))
		}
	}
	sort.Strings(lines)
	repos := "（未配置仓库）"
	if len(lines) > 0 {
		repos = strings.Join(lines, "\n")
	}
	return fmt.Sprintf(
		"【工作区使用模式（自动注入）】\n你当前绑定的工作区为「%s」（id=%s）。\n请遵循以下强制约束：\n1. 仅允许在该工作区目录及其子目录内进行文件读写与命令执行：%s\n2. 仅允许在以下仓库范围内完成任务：\n%s\n3. 严禁访问、读取、修改工作区外的任何路径或未列出的仓库。",
		name,
		workspaceID,
		root,
		repos,
	)
}

