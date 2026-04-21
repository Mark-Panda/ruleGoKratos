package biz

import (
	"context"
	"errors"
	"fmt"
)

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

// enrichHarnessWithManagedAgent 当 ManagedAgentID>0 时加载配置：系统提示、托管模型、技能/MCP 白名单与 SKILL 目录段落。
func (uc *AgentUsecase) enrichHarnessWithManagedAgent(ctx context.Context, req HarnessRequest) (HarnessRequest, error) {
	if req.ManagedAgentID <= 0 {
		return req, nil
	}
	if uc.managedAgentLoader == nil {
		return req, errors.New("Managed Agent 加载器未注入")
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
	filtered := FilterSkillNamesByPackages(all, p.SkillPackageIDs)
	if len(p.SkillPackageIDs) > 0 {
		cp := filtered
		out.SkillCatalogFilter = &cp
	} else {
		empty := []string{}
		out.SkillCatalogFilter = &empty
	}

	mcpAllow, err := uc.managedAgentLoader.McpAllowlistStrings(ctx, p.McpIDs)
	if err != nil {
		return req, err
	}
	out.ToolOptions = &HarnessToolOptions{
		EnableUUIDTool:       true,
		EnableSkillTool:      len(filtered) > 0,
		EnableMcpTool:        len(mcpAllow) > 0,
		EnableWorkspaceTools: true,
		SkillAllowlist:       filtered,
		McpAllowlist:         mcpAllow,
	}
	return out, nil
}
