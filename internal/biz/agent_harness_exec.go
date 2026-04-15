package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// HarnessToolOptions 控制向模型暴露的工具子集；用于 RuleGo Agent 节点等场景。
// 与 BuildToolRegistry() 的全量工具（含 workspace）不同，此处按需装配。
type HarnessToolOptions struct {
	EnableUUIDTool       bool
	EnableSkillTool      bool
	EnableMcpTool        bool
	EnableWorkspaceTools bool
	SkillAllowlist       []string
	McpAllowlist         []string
}

// ParseCommaSeparated 解析逗号分隔列表，去空、去首尾空格。
func ParseCommaSeparated(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ParseMcpAllowlist 解析 "server:tool,server2:tool2" 为规范化键列表（用于白名单校验）。
func ParseMcpAllowlist(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		idx := strings.Index(p, ":")
		if idx <= 0 || idx == len(p)-1 {
			continue
		}
		srv := strings.TrimSpace(p[:idx])
		tn := strings.TrimSpace(p[idx+1:])
		if srv == "" || tn == "" {
			continue
		}
		out = append(out, mcpPairKey(srv, tn))
	}
	return out
}

func mcpPairKey(server, tool string) string {
	return server + "\x00" + tool
}

func skillAllowSet(list []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s != "" {
			m[s] = struct{}{}
		}
	}
	return m
}

func mcpAllowSet(keys []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			m[k] = struct{}{}
		}
	}
	return m
}

func (uc *AgentUsecase) wrapSkillWithAllowlist(base *HarnessTool, allow map[string]struct{}) *HarnessTool {
	if len(allow) == 0 {
		return base
	}
	return &HarnessTool{
		Info: base.Info,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			var args struct {
				SkillName string `json:"skill_name"`
			}
			if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
				return "", err
			}
			name := strings.TrimSpace(args.SkillName)
			if _, ok := allow[name]; !ok {
				return "", fmt.Errorf("skill 不在白名单: %s", name)
			}
			return base.Invoke(ctx, rawArgs)
		},
	}
}

func (uc *AgentUsecase) wrapMcpWithAllowlist(base *HarnessTool, allow map[string]struct{}) *HarnessTool {
	if len(allow) == 0 {
		return base
	}
	return &HarnessTool{
		Info: base.Info,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			var args struct {
				Server string `json:"server"`
				Tool   string `json:"tool"`
			}
			if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
				return "", err
			}
			k := mcpPairKey(args.Server, args.Tool)
			if _, ok := allow[k]; !ok {
				return "", fmt.Errorf("MCP server:tool 不在白名单: %s:%s", args.Server, args.Tool)
			}
			return base.Invoke(ctx, rawArgs)
		},
	}
}

// BuildToolRegistryWithOptions 按选项装配工具；至少启用一项，否则返回错误。
func (uc *AgentUsecase) BuildToolRegistryWithOptions(opts *HarnessToolOptions) (map[string]*HarnessTool, []*schema.ToolInfo, error) {
	if opts == nil {
		return uc.BuildToolRegistry()
	}
	registry := map[string]*HarnessTool{}
	var infos []*schema.ToolInfo

	if opts.EnableUUIDTool {
		t, err := uc.BuildUUIDTool()
		if err != nil {
			return nil, nil, err
		}
		registry[t.Info.Name] = t
		infos = append(infos, t.Info)
	}
	if opts.EnableSkillTool {
		t, err := uc.BuildSkillTool()
		if err != nil {
			return nil, nil, err
		}
		allow := skillAllowSet(opts.SkillAllowlist)
		t = uc.wrapSkillWithAllowlist(t, allow)
		registry[t.Info.Name] = t
		infos = append(infos, t.Info)
	}
	if opts.EnableMcpTool {
		t, err := uc.BuildMCPTool()
		if err != nil {
			return nil, nil, err
		}
		allow := mcpAllowSet(opts.McpAllowlist)
		t = uc.wrapMcpWithAllowlist(t, allow)
		registry[t.Info.Name] = t
		infos = append(infos, t.Info)
	}
	if opts.EnableWorkspaceTools {
		readFileTool, err := uc.BuildReadWorkspaceFileTool()
		if err != nil {
			return nil, nil, err
		}
		writeFileTool, err := uc.BuildWriteWorkspaceFileTool()
		if err != nil {
			return nil, nil, err
		}
		shellTool, err := uc.BuildRunWorkspaceShellTool()
		if err != nil {
			return nil, nil, err
		}
		registry[readFileTool.Info.Name] = readFileTool
		registry[writeFileTool.Info.Name] = writeFileTool
		registry[shellTool.Info.Name] = shellTool
		infos = append(infos, readFileTool.Info, writeFileTool.Info, shellTool.Info)
	}

	return registry, infos, nil
}

func (uc *AgentUsecase) resolveToolRegistry(opts *HarnessToolOptions) (map[string]*HarnessTool, []*schema.ToolInfo, error) {
	if opts == nil {
		return uc.BuildToolRegistry()
	}
	return uc.BuildToolRegistryWithOptions(opts)
}

func (uc *AgentUsecase) effectiveHarnessConfig(override *HarnessConfig) HarnessConfig {
	cfg := uc.sanitizeConfig()
	if override == nil {
		return cfg
	}
	if override.MaxIterations > 0 {
		cfg.MaxIterations = override.MaxIterations
	}
	if override.MaxToolCalls > 0 {
		cfg.MaxToolCalls = override.MaxToolCalls
	}
	if override.ToolTimeoutSecs > 0 {
		cfg.ToolTimeoutSecs = override.ToolTimeoutSecs
	}
	if override.ChunkSize > 0 {
		cfg.ChunkSize = override.ChunkSize
	}
	return cfg
}

// ExecuteHarnessSync 非流式执行，将助手文本拼成单个字符串返回。
func (uc *AgentUsecase) ExecuteHarnessSync(ctx context.Context, req HarnessRequest) (string, error) {
	var b strings.Builder
	var runErr error
	uc.executeHarness(req, ctx)(func(sm *StreamMessage, err error) bool {
		if err != nil {
			runErr = err
			return false
		}
		if sm != nil && sm.Content != "" {
			b.WriteString(sm.Content)
		}
		return true
	})
	return b.String(), runErr
}
