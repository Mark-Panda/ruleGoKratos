package data

import (
	"errors"
	"strings"

	"ruleGoKratos/internal/biz"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/maps"
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

	modelTpl        str.Template
	systemTpl       str.Template
	userTpl         str.Template
	skillAllowTpl   str.Template
	mcpAllowTpl     str.Template
	hasVar          bool
}

// AgentHarnessLLMConfig 与 flowgram DSL 导出字段对齐（camelCase）。
type AgentHarnessLLMConfig struct {
	Model                string `json:"model"`
	SystemPrompt         string `json:"systemPrompt"`
	UserPrompt           string `json:"userPrompt"`
	EnableSkillTool      bool   `json:"enableSkillTool"`
	EnableMcpTool        bool   `json:"enableMcpTool"`
	EnableUUIDTool       bool   `json:"enableUUIDTool"`
	EnableWorkspaceTools bool   `json:"enableWorkspaceTools"`
	SkillAllowlist       string `json:"skillAllowlist"`
	McpAllowlist         string `json:"mcpAllowlist"`
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
		EnableUUIDTool:  true,
	}}
}

func (x *AgentHarnessLLM) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "agentHarness",
		Desc:  "Agent LLM（可配置 Skill / MCP 工具，与 Chat Harness 一致）",
	}
}

func (x *AgentHarnessLLM) Init(ruleConfig types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &x.Config); err != nil {
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
	x.skillAllowTpl = str.NewTemplate(x.Config.SkillAllowlist)
	track(x.skillAllowTpl)
	x.mcpAllowTpl = str.NewTemplate(x.Config.McpAllowlist)
	track(x.mcpAllowTpl)
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
	skillAllowRaw := x.skillAllowTpl.Execute(env)
	mcpAllowRaw := x.mcpAllowTpl.Execute(env)

	toolOpts := &biz.HarnessToolOptions{
		EnableUUIDTool:       x.Config.EnableUUIDTool,
		EnableSkillTool:      x.Config.EnableSkillTool,
		EnableMcpTool:        x.Config.EnableMcpTool,
		EnableWorkspaceTools: x.Config.EnableWorkspaceTools,
		SkillAllowlist:       biz.ParseCommaSeparated(skillAllowRaw),
		McpAllowlist:         biz.ParseMcpAllowlist(mcpAllowRaw),
	}

	var cfgOverride *biz.HarnessConfig
	if x.Config.MaxIterations > 0 || x.Config.MaxToolCalls > 0 || x.Config.ToolTimeoutSecs > 0 {
		cfgOverride = &biz.HarnessConfig{
			MaxIterations:   x.Config.MaxIterations,
			MaxToolCalls:      x.Config.MaxToolCalls,
			ToolTimeoutSecs: x.Config.ToolTimeoutSecs,
		}
	}

	req := biz.HarnessRequest{
		Model:          strings.TrimSpace(modelName),
		History:        nil,
		Input:          userPrompt,
		SystemPrompt:   systemPrompt,
		ToolOptions:    toolOpts,
		ConfigOverride: cfgOverride,
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
