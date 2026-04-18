package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/data/dao"
)

type managedAgentHarnessLoader struct{}

// NewManagedAgentHarnessLoader 供 biz.AgentUsecase 注入，执行时解析 Agent 配置。
func NewManagedAgentHarnessLoader() biz.ManagedAgentLoader {
	return &managedAgentHarnessLoader{}
}

func (managedAgentHarnessLoader) Load(ctx context.Context, id int64) (*biz.ManagedAgentProfile, error) {
	row, err := dao.NewManagedAgent().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	var rawPkgs []string
	if row.SkillPathsJSON != "" {
		_ = json.Unmarshal([]byte(row.SkillPathsJSON), &rawPkgs)
	}
	pkgs := biz.NormalizeStoredSkillPackageIDs(rawPkgs)
	var mcpIDs []int64
	if row.McpIDsJSON != "" {
		_ = json.Unmarshal([]byte(row.McpIDsJSON), &mcpIDs)
	}
	var entryIDs []int64
	if row.ModelEntryIDsJSON != "" {
		_ = json.Unmarshal([]byte(row.ModelEntryIDsJSON), &entryIDs)
	}
	return &biz.ManagedAgentProfile{
		Enabled:         row.Enabled,
		SystemPrompt:    row.SystemPrompt,
		SkillPackageIDs: pkgs,
		McpIDs:          mcpIDs,
		LLMConfigID:     row.LLMConfigID,
		ModelScope:      strings.TrimSpace(row.ModelScope),
		ModelEntryIDs:   entryIDs,
	}, nil
}

func (managedAgentHarnessLoader) ResolveModelEntryForHarness(ctx context.Context, p *biz.ManagedAgentProfile) (configID int64, entryID int64, err error) {
	if p.LLMConfigID <= 0 {
		return 0, 0, fmt.Errorf("Agent 配置缺少 LLM 站点")
	}
	switch strings.TrimSpace(strings.ToLower(p.ModelScope)) {
	case "explicit":
		if len(p.ModelEntryIDs) == 0 {
			return 0, 0, fmt.Errorf("Agent 指定模型但未选择模型条目")
		}
		return p.LLMConfigID, p.ModelEntryIDs[0], nil
	default:
		entries, qerr := dao.NewLLMModelEntry().FindByConfigIDs(ctx, []int64{p.LLMConfigID})
		if qerr != nil {
			return 0, 0, qerr
		}
		for _, e := range entries {
			if e.Enabled {
				return p.LLMConfigID, e.ID, nil
			}
		}
		return 0, 0, fmt.Errorf("LLM 站点 id=%d 下没有启用的模型条目", p.LLMConfigID)
	}
}

func (managedAgentHarnessLoader) McpAllowlistStrings(ctx context.Context, ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := dao.NewMCPConfig().FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		srv := strings.TrimSpace(r.Server)
		if srv == "" || !r.Enabled {
			continue
		}
		out = append(out, srv+":*")
	}
	return out, nil
}
