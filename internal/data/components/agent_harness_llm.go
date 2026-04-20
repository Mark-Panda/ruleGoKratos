package data

import (
	"encoding/json"
	"errors"
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

// AgentHarnessLLM 自定义 LLM 节点：走 Eino tool-calling，可按节点配置启用 Skill / MCP 等工具。
type AgentHarnessLLM struct {
	Config AgentHarnessLLMConfig

	modelTpl   str.Template
	systemTpl  str.Template
	userTpl    str.Template
	hasVar     bool
	skillAllow []string // 从 configuration 解析（支持 string 或 JSON 数组）
	mcpAllow   []string // 同上；元素为 ParseMcpAllowlist 可识别的 server:tool 或 server:*
}

// AgentHarnessLLMConfig 与 flowgram DSL 导出字段对齐（camelCase）；白名单见 skillAllow / mcpAllow。
type AgentHarnessLLMConfig struct {
	LlmConfigID      int64 `json:"llmConfigId"`
	LlmModelEntryID int64 `json:"llmModelEntryId"`
	ManagedAgentID       int64  `json:"managedAgentId"`
	Model                string `json:"model"`
	SystemPrompt         string `json:"systemPrompt"`
	UserPrompt           string `json:"userPrompt"`
	EnableSkillTool      bool   `json:"enableSkillTool"`
	EnableMcpTool        bool   `json:"enableMcpTool"`
	// EnableUUIDTool 已废弃：运行时固定启用 UUID 工具，DSL 若仍含该字段会被忽略。
	EnableUUIDTool bool `json:"enableUUIDTool"`
	EnableWorkspaceTools bool   `json:"enableWorkspaceTools"`
	MaxIterations        int    `json:"maxIterations"`
	MaxToolCalls         int    `json:"maxToolCalls"`
	ToolTimeoutSecs      int    `json:"toolTimeoutSecs"`
}

func (x *AgentHarnessLLM) Type() string {
	return "ai/agentHarness"
}

func (x *AgentHarnessLLM) New() types.Node {
	return &AgentHarnessLLM{Config: AgentHarnessLLMConfig{
		EnableSkillTool: true,
		EnableMcpTool:   true,
	}}
}

func (x *AgentHarnessLLM) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "agentHarness",
		Desc:  "Agent LLM（可配置 Skill / MCP 工具，与 Chat Harness 一致）",
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
	x.skillAllow = biz.NormalizeSkillAllowlistInput(raw["skillAllowlist"])
	x.mcpAllow = biz.NormalizeMcpAllowlistInput(raw["mcpAllowlist"])
	delete(raw, "skillAllowlist")
	delete(raw, "mcpAllowlist")
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

	toolOpts := &biz.HarnessToolOptions{
		EnableUUIDTool:       true, // 不在节点上暴露开关，固定启用 generate_uuid
		EnableSkillTool:      x.Config.EnableSkillTool,
		EnableMcpTool:        x.Config.EnableMcpTool,
		EnableWorkspaceTools: x.Config.EnableWorkspaceTools,
		SkillAllowlist:       x.skillAllow,
		McpAllowlist:         x.mcpAllow,
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
		SystemPrompt:    systemPrompt,
		ConfigOverride:  cfgOverride,
		ToolOptions:     toolOpts,
		ManagedAgentID:  x.Config.ManagedAgentID,
		LlmConfigID:     x.Config.LlmConfigID,
		LlmModelEntryID: x.Config.LlmModelEntryID,
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
