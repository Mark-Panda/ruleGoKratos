package biz

import (
	"context"
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
	EnableSubAgentTool   bool
	SkillAllowlist       []string
	McpAllowlist         []string
}

func cloneHarnessToolOptions(in *HarnessToolOptions) *HarnessToolOptions {
	if in == nil {
		return nil
	}
	out := *in
	out.SkillAllowlist = append([]string(nil), in.SkillAllowlist...)
	out.McpAllowlist = append([]string(nil), in.McpAllowlist...)
	return &out
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

func mcpAllowSet(keys []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, k := range keys {
		normalized, ok := normalizeMcpAllowKey(k)
		if ok {
			m[normalized] = struct{}{}
		}
	}
	return m
}

func normalizeMcpAllowKey(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if strings.Contains(raw, "\x00") {
		parts := strings.SplitN(raw, "\x00", 2)
		server := strings.TrimSpace(parts[0])
		tool := strings.TrimSpace(parts[1])
		if server == "" || tool == "" {
			return "", false
		}
		return mcpPairKey(server, tool), true
	}
	parsed := ParseMcpAllowlist(raw)
	if len(parsed) != 1 {
		return "", false
	}
	return parsed[0], true
}

// NormalizeSkillAllowlistInput 解析 DSL / 配置中的 Skill 白名单：逗号分隔字符串或字符串数组。
func NormalizeSkillAllowlistInput(v interface{}) []string {
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		return ParseCommaSeparated(s)
	}
	if arr, ok := v.([]interface{}); ok {
		out := make([]string, 0, len(arr))
		for _, e := range arr {
			if t := strings.TrimSpace(fmt.Sprint(e)); t != "" {
				out = append(out, t)
			}
		}
		return out
	}
	if arr, ok := v.([]string); ok {
		out := make([]string, 0, len(arr))
		for _, e := range arr {
			if t := strings.TrimSpace(e); t != "" {
				out = append(out, t)
			}
		}
		return out
	}
	return nil
}

// NormalizeMcpAllowlistInput 解析 MCP 白名单：逗号分隔字符串或元素为 server:tool / server:* 的数组。
func NormalizeMcpAllowlistInput(v interface{}) []string {
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		return ParseMcpAllowlist(s)
	}
	if arr, ok := v.([]interface{}); ok {
		parts := make([]string, 0, len(arr))
		for _, e := range arr {
			if t := strings.TrimSpace(fmt.Sprint(e)); t != "" {
				parts = append(parts, t)
			}
		}
		return ParseMcpAllowlist(strings.Join(parts, ","))
	}
	if arr, ok := v.([]string); ok {
		parts := make([]string, 0, len(arr))
		for _, e := range arr {
			if t := strings.TrimSpace(e); t != "" {
				parts = append(parts, t)
			}
		}
		return ParseMcpAllowlist(strings.Join(parts, ","))
	}
	return nil
}

// BuildToolRegistryWithOptions 按选项装配工具；至少启用一项，否则返回错误。
func (uc *AgentUsecase) BuildToolRegistryWithOptions(opts *HarnessToolOptions) (map[string]*HarnessTool, []*schema.ToolInfo, error) {
	return uc.BuildToolRegistryWithOptionsForContext(context.Background(), opts)
}

func (uc *AgentUsecase) BuildToolRegistryWithOptionsForContext(ctx context.Context, opts *HarnessToolOptions) (map[string]*HarnessTool, []*schema.ToolInfo, error) {
	if opts == nil {
		return uc.BuildToolRegistryForContext(ctx)
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
		tools, _, err := uc.buildOfficialSkillTools(ctx, opts.SkillAllowlist)
		if err != nil {
			return nil, nil, err
		}
		for _, t := range tools {
			if t == nil || t.Info == nil || strings.TrimSpace(t.Info.Name) == "" {
				continue
			}
			registry[t.Info.Name] = t
			infos = append(infos, t.Info)
		}
	}
	if opts.EnableMcpTool {
		mcpTools, err := uc.buildMcpTools(ctx, opts.McpAllowlist)
		if err != nil {
			return nil, nil, err
		}
		for _, t := range mcpTools {
			if t == nil || t.Info == nil || strings.TrimSpace(t.Info.Name) == "" {
				continue
			}
			if _, exists := registry[t.Info.Name]; exists {
				continue
			}
			registry[t.Info.Name] = t
			infos = append(infos, t.Info)
		}
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
	if opts.EnableSubAgentTool {
		t, err := uc.BuildSubAgentTool()
		if err != nil {
			return nil, nil, err
		}
		registry[t.Info.Name] = t
		infos = append(infos, t.Info)
	}

	return registry, infos, nil
}

func (uc *AgentUsecase) buildMcpTools(ctx context.Context, allowlist []string) ([]*HarnessTool, error) {
	if uc.mcpToolProvider == nil {
		return nil, nil
	}
	return uc.mcpToolProvider.BuildMcpTools(ctx, allowlist)
}

func (uc *AgentUsecase) resolveToolRegistry(ctx context.Context, opts *HarnessToolOptions) (map[string]*HarnessTool, []*schema.ToolInfo, error) {
	if opts == nil {
		return uc.BuildToolRegistryForContext(ctx)
	}
	return uc.BuildToolRegistryWithOptionsForContext(ctx, opts)
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
	if override.StreamTimeoutSecs > 0 {
		cfg.StreamTimeoutSecs = override.StreamTimeoutSecs
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
