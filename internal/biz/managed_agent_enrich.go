package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

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
	var managedSkillAllow []string
	if len(p.SkillPackageIDs) > 0 {
		managedSkillAllow = filtered
	} else {
		// 未勾选技能包时：run_skill 不做额外白名单限制。
		managedSkillAllow = nil
	}
	managedEnableSkill := len(all) > 0 && (len(p.SkillPackageIDs) == 0 || len(filtered) > 0)
	managedEnableMcp := len(mcpAllow) > 0

	// ManagedAgentID>0 时，优先保障托管 Agent 的工具能力生效，同时兼容请求侧已有配置：
	// - 开关按“或”合并，避免请求侧误关导致托管能力失效；
	// - 白名单按并集合并，保留请求侧附加项并补齐托管侧必需项。
	if out.ToolOptions == nil {
		out.ToolOptions = &HarnessToolOptions{
			EnableUUIDTool:       true,
			EnableSkillTool:      managedEnableSkill,
			EnableMcpTool:        managedEnableMcp,
			EnableWorkspaceTools: true,
			EnableSubAgentTool:   true,
			SkillAllowlist:       managedSkillAllow,
			McpAllowlist:         mcpAllow,
		}
	} else {
		merged := cloneHarnessToolOptions(out.ToolOptions)
		if managedEnableSkill {
			merged.EnableSkillTool = true
		}
		if managedEnableMcp {
			merged.EnableMcpTool = true
		}
		merged.SkillAllowlist = mergeAllowlist(merged.SkillAllowlist, managedSkillAllow)
		merged.McpAllowlist = mergeAllowlist(merged.McpAllowlist, mcpAllow)
		out.ToolOptions = merged
	}
	return out, nil
}
