package biz

import (
	"context"
	"sort"
	"strings"
)

// ManagedAgentProfile 与「Agent 配置」表一致，供执行时解析；由 data 层从 DB 加载。
type ManagedAgentProfile struct {
	Enabled         bool
	SystemPrompt    string
	WorkspaceID     string
	WorkspacePrompt string
	SkillPackageIDs []string
	LLMConfigID     int64
	ModelScope      string // all | explicit
	ModelEntryIDs   []int64
}

// ManagedAgentLoader 从存储加载 Agent 配置并解析模型、全局启用 MCP（实现放在 data 包，避免 biz 依赖 dao）。
type ManagedAgentLoader interface {
	Load(ctx context.Context, id int64) (*ManagedAgentProfile, error)
	ResolveModelEntryForHarness(ctx context.Context, p *ManagedAgentProfile) (configID, entryID int64, err error)
	EnabledMcpAllowlistStrings(ctx context.Context) ([]string, error)
}

// PackageIDFromSkillName 与官方 Skill package name 规则一致：取技能名首段路径。
func PackageIDFromSkillName(skillName string) string {
	skillName = strings.Trim(strings.ReplaceAll(skillName, "\\", "/"), "/")
	if i := strings.Index(skillName, "/"); i >= 0 {
		return skillName[:i]
	}
	return skillName
}

// FilterSkillNamesByPackages 在已按全局 FileSkillExecutor 白名单筛过的技能名中，再按技能包 id 过滤。
func FilterSkillNamesByPackages(all []string, packages []string) []string {
	if len(packages) == 0 {
		return nil
	}
	allow := make(map[string]struct{}, len(packages))
	for _, p := range packages {
		p = strings.TrimSpace(p)
		if p != "" {
			allow[p] = struct{}{}
		}
	}
	out := make([]string, 0, len(all))
	for _, name := range all {
		if _, ok := allow[PackageIDFromSkillName(name)]; ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
