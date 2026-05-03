package data

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/data/dao"
)

type managedAgentHarnessLoader struct{}

const defaultWorkspaceRootDir = "/app/code_workspace"

type workspaceMetaForAgent struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	RootDir      string `json:"rootDir"`
	Repositories []struct {
		URL string `json:"url"`
		Dir string `json:"dir"`
	} `json:"repositories"`
}

type workspaceFileForAgent struct {
	Meta workspaceMetaForAgent `json:"ruleGoWorkspace"`
}

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
	var entryIDs []int64
	if row.ModelEntryIDsJSON != "" {
		_ = json.Unmarshal([]byte(row.ModelEntryIDsJSON), &entryIDs)
	}
	return &biz.ManagedAgentProfile{
		Enabled:         row.Enabled,
		SystemPrompt:    row.SystemPrompt,
		WorkspaceID:     strings.TrimSpace(row.WorkspaceID),
		WorkspacePrompt: buildWorkspacePrompt(strings.TrimSpace(row.WorkspaceID)),
		SkillPackageIDs: pkgs,
		LLMConfigID:     row.LLMConfigID,
		ModelScope:      strings.TrimSpace(row.ModelScope),
		ModelEntryIDs:   entryIDs,
	}, nil
}

func buildWorkspacePrompt(workspaceID string) string {
	if workspaceID == "" {
		return ""
	}
	cfgPath := resolveWorkspaceConfigPath(workspaceID)
	if cfgPath == "" {
		return ""
	}
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return ""
	}
	var cfg workspaceFileForAgent
	if err := json.Unmarshal(b, &cfg); err != nil {
		return ""
	}
	name := strings.TrimSpace(cfg.Meta.Name)
	if name == "" {
		name = workspaceID
	}
	rootDir := strings.TrimSpace(cfg.Meta.RootDir)
	if rootDir == "" {
		rootDir = filepath.Join(defaultWorkspaceRootDir, workspaceID)
	}
	repoLines := make([]string, 0, len(cfg.Meta.Repositories))
	for _, repo := range cfg.Meta.Repositories {
		line := fmt.Sprintf("- %s", strings.TrimSpace(repo.URL))
		if d := strings.TrimSpace(repo.Dir); d != "" {
			line = fmt.Sprintf("- %s（目录: %s）", strings.TrimSpace(repo.URL), d)
		}
		repoLines = append(repoLines, line)
	}
	sort.Strings(repoLines)
	reposText := "（未配置仓库）"
	if len(repoLines) > 0 {
		reposText = strings.Join(repoLines, "\n")
	}
	return fmt.Sprintf(
		"【工作区使用模式（自动注入）】\n你当前绑定的工作区为「%s」（id=%s）。\n请遵循以下强制约束：\n1. 仅允许在该工作区目录及其子目录内进行文件读写与命令执行：%s\n2. 仅允许在以下仓库范围内完成任务：\n%s\n3. 严禁修改工作区外的任何路径或未列出的仓库。",
		name,
		workspaceID,
		rootDir,
		reposText,
	)
}

func resolveWorkspaceConfigPath(workspaceID string) string {
	if strings.TrimSpace(workspaceID) == "" {
		return ""
	}
	filename := workspaceID + ".code-workspace"
	candidates := []string{
		filepath.Join(defaultWorkspaceRootDir, filename),
		filepath.Join("code_workspace", filename),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func (managedAgentHarnessLoader) ResolveModelEntryForHarness(ctx context.Context, p *biz.ManagedAgentProfile) (configID int64, entryID int64, err error) {
	if p.LLMConfigID <= 0 {
		return 0, 0, fmt.Errorf("Agent 配置缺少 LLM 站点") //lint:ignore ST1005 "Chinese error message"
	}
	switch strings.TrimSpace(strings.ToLower(p.ModelScope)) {
	case "explicit":
		if len(p.ModelEntryIDs) == 0 {
			return 0, 0, fmt.Errorf("Agent 指定模型但未选择模型条目") //lint:ignore ST1005 "Chinese error message"
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

func (managedAgentHarnessLoader) EnabledMcpAllowlistStrings(ctx context.Context) ([]string, error) {
	rows, err := dao.NewMCPConfig().FindEnabled(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		srv := strings.TrimSpace(r.Server)
		if srv == "" {
			continue
		}
		out = append(out, biz.ParseMcpAllowlist(srv+":*")...)
	}
	return out, nil
}
