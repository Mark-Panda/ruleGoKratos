package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"
)


//lint:ignore U1000 "kept for future use"
func mergeAllowlist(base, extra []string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	appendIfNeeded := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	for _, v := range base {
		appendIfNeeded(v)
	}
	for _, v := range extra {
		appendIfNeeded(v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (uc *AgentUsecase) SetManagedAgentLoader(l ManagedAgentLoader) {
	if l != nil {
		uc.managedAgentLoader = l
	}
}

func (uc *AgentUsecase) SetMcpConfigAdmin(a McpConfigAdmin) {
	if a != nil {
		uc.mcpConfigAdmin = a
	}
}

// enrichHarnessWithManagedAgent 当 ManagedAgentID>0 时加载配置：系统提示、托管模型、技能/MCP 白名单。
func (uc *AgentUsecase) enrichHarnessWithManagedAgent(ctx context.Context, req HarnessRequest) (HarnessRequest, error) {
	if req.ManagedAgentID <= 0 {
		return req, nil
	}
	if uc.managedAgentLoader == nil {
		return req, errors.New("Managed Agent 加载器未注入") //lint:ignore ST1005 "Chinese error message"
	}
	p, err := uc.managedAgentLoader.Load(ctx, req.ManagedAgentID)
	if err != nil {
		return req, err
	}
	if !p.Enabled {
		return req, fmt.Errorf("该 Agent 配置已停用")
	}
	out := req
	out.SystemPrompt = p.SystemPrompt
	if strings.TrimSpace(p.WorkspacePrompt) != "" {
		if strings.TrimSpace(out.SystemPrompt) == "" {
			out.SystemPrompt = p.WorkspacePrompt
		} else {
			out.SystemPrompt = out.SystemPrompt + "\n\n" + p.WorkspacePrompt
		}
	}
	cfgID, entryID, err := uc.managedAgentLoader.ResolveModelEntryForHarness(ctx, p)
	if err != nil {
		return req, err
	}
	out.LlmConfigID = cfgID
	out.LlmModelEntryID = entryID
	out.Model = ""

	fe, ok := uc.skillExecutor.(*FileSkillExecutor)
	if !ok {
		return req, errors.New("技能执行器未就绪")
	}
	all := fe.ListAvailableSkillNames()
	out.SkillCatalogFilter = nil

	mcpAllow, err := uc.managedAgentLoader.EnabledMcpAllowlistStrings(ctx)
	if err != nil {
		return req, err
	}
	managedEnableSkill := len(all) > 0
	managedEnableMcp := len(mcpAllow) > 0

	// ManagedAgentID>0 时，优先保障托管 Agent 的工具能力生效，同时兼容请求侧已有配置：
	// - 开关按“或”合并，避免请求侧误关导致托管能力失效；
	// - Skill 不再使用 Agent 级白名单，MCP 白名单仍按并集合并。
	if out.ToolOptions == nil {
		out.ToolOptions = &HarnessToolOptions{
			EnableUUIDTool:       true,
			EnableSkillTool:      managedEnableSkill,
			EnableMcpTool:        managedEnableMcp,
			EnableWorkspaceTools: true,
			EnableSubAgentTool:   true,
			McpAllowlist:         mcpAllow,
		}
	} else {
		merged := cloneHarnessToolOptions(out.ToolOptions)
		if managedEnableSkill {
			merged.EnableSkillTool = true
		}
		merged.EnableMcpTool = managedEnableMcp
		merged.SkillAllowlist = nil
		merged.McpAllowlist = append([]string(nil), mcpAllow...)
		out.ToolOptions = merged
	}
	return out, nil
}
