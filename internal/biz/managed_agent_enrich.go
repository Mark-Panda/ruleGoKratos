package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	filtered := FilterSkillNamesByPackages(all, p.SkillPackageIDs)
	// 子 Agent 默认继承父 Agent 的技能目录过滤；仅在未显式携带过滤条件时按托管 Agent 配置注入。
	if out.SkillCatalogFilter == nil {
		if len(p.SkillPackageIDs) > 0 {
			cp := filtered
			out.SkillCatalogFilter = &cp
		} else {
			// 未勾选技能包时：系统提示附全量 SKILL 目录（SkillCatalogFilter=nil）。
			out.SkillCatalogFilter = nil
		}
	}

	mcpAllow, err := uc.managedAgentLoader.McpAllowlistStrings(ctx, p.McpIDs)
	if err != nil {
		return req, err
	}
	// 子 Agent 默认继承父 Agent 的工具配置（Skill/MCP 白名单与开关）。
	// 仅在请求侧未提供 ToolOptions 时，才由托管 Agent 配置注入默认工具集。
	if out.ToolOptions == nil {
		var skillAllow []string
		if len(p.SkillPackageIDs) > 0 {
			skillAllow = filtered
		} else {
			// 未勾选技能包时：run_skill 不做额外白名单限制。
			skillAllow = nil
		}
		enableSkill := len(all) > 0 && (len(p.SkillPackageIDs) == 0 || len(filtered) > 0)
		out.ToolOptions = &HarnessToolOptions{
			EnableUUIDTool:       true,
			EnableSkillTool:      enableSkill,
			EnableMcpTool:        len(mcpAllow) > 0,
			EnableWorkspaceTools: true,
			EnableSubAgentTool:   true,
			SkillAllowlist:       skillAllow,
			McpAllowlist:         mcpAllow,
		}
	}
	return out, nil
}
