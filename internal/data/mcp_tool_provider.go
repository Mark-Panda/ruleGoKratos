package data

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/data/dao"
	"ruleGoKratos/internal/mcpbridge"
)

type databaseMcpToolProvider struct{}

// NewDatabaseMcpToolProvider 将已启用 MCP 配置中的 server tools 转换为 Eino/Harness 可直接调用的具体工具。
func NewDatabaseMcpToolProvider() biz.McpToolProvider {
	return &databaseMcpToolProvider{}
}

func (databaseMcpToolProvider) BuildMcpTools(ctx context.Context, allowlist []string) ([]*biz.HarnessTool, error) {
	rows, err := dao.NewMCPConfig().FindEnabled(ctx)
	if err != nil {
		return nil, err
	}
	allow := mcpAllowSetForProvider(allowlist)
	out := make([]*biz.HarnessTool, 0)
	seen := make(map[string]struct{})
	for _, row := range rows {
		server := strings.TrimSpace(row.Server)
		if server == "" {
			continue
		}
		baseTools, err := loadMcpBaseTools(ctx, row, nil)
		if err != nil {
			return nil, fmt.Errorf("加载 MCP server=%q tools 失败: %w", server, err)
		}
		for _, baseTool := range baseTools {
			info, err := baseTool.Info(ctx)
			if err != nil {
				return nil, fmt.Errorf("读取 MCP server=%q tool 信息失败: %w", server, err)
			}
			if info == nil || strings.TrimSpace(info.Name) == "" {
				continue
			}
			originalName := strings.TrimSpace(info.Name)
			if !mcpProviderAllowed(allow, server, originalName) {
				continue
			}
			registryName := MCPRegistryToolName(server, originalName)
			if registryName == "" {
				continue
			}
			if _, ok := seen[registryName]; ok {
				continue
			}
			seen[registryName] = struct{}{}
			toolInfo := *info
			toolInfo.Name = registryName
			if strings.TrimSpace(toolInfo.Desc) == "" {
				toolInfo.Desc = fmt.Sprintf("MCP tool %s from server %s", originalName, server)
			} else {
				toolInfo.Desc = fmt.Sprintf("[MCP server=%s tool=%s] %s", server, originalName, toolInfo.Desc)
			}
			rowCopy := row
			out = append(out, &biz.HarnessTool{
				Info: &toolInfo,
				Invoke: func(ctx context.Context, rawArgs string) (string, error) {
					return invokeOfficialMcpTool(ctx, rowCopy, originalName, rawArgs)
				},
			})
		}
	}
	return out, nil
}

func mcpAllowSetForProvider(list []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, item := range list {
		normalized, ok := normalizeProviderMcpAllowKey(item)
		if ok {
			out[normalized] = struct{}{}
		}
	}
	return out
}

func normalizeProviderMcpAllowKey(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if strings.Contains(raw, "\x00") {
		parts := strings.SplitN(raw, "\x00", 2)
		server := strings.TrimSpace(parts[0])
		toolName := strings.TrimSpace(parts[1])
		if server == "" || toolName == "" {
			return "", false
		}
		return server + "\x00" + toolName, true
	}
	parsed := biz.ParseMcpAllowlist(raw)
	if len(parsed) != 1 {
		return "", false
	}
	return parsed[0], true
}

func mcpProviderAllowed(allow map[string]struct{}, server, toolName string) bool {
	if len(allow) == 0 {
		return true
	}
	if _, ok := allow[server+"\x00"+toolName]; ok {
		return true
	}
	if _, ok := allow[server+"\x00*"]; ok {
		return true
	}
	return false
}

var nonMCPToolNameChar = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// MCPRegistryToolName 避免不同 MCP server 下同名工具冲突，同时保持模型可调用的 tool name 合法。
func MCPRegistryToolName(server, toolName string) string {
	server = sanitizeMCPToolNamePart(server)
	toolName = sanitizeMCPToolNamePart(toolName)
	if server == "" || toolName == "" {
		return ""
	}
	return "mcp_" + server + "_" + toolName
}

func sanitizeMCPToolNamePart(s string) string {
	s = strings.TrimSpace(s)
	s = nonMCPToolNameChar.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_-")
	return s
}

func loadMcpBaseTools(ctx context.Context, row dao.MCPConfig, toolNames []string) ([]tool.BaseTool, error) {
	cli, headers, closeFn, err := newMcpClientFromConfig(ctx, row)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	return mcpp.GetTools(ctx, &mcpp.Config{Cli: cli, ToolNameList: toolNames, CustomHeaders: headers})
}

func invokeOfficialMcpTool(ctx context.Context, row dao.MCPConfig, toolName string, rawArgs string) (string, error) {
	cli, headers, closeFn, err := newMcpClientFromConfig(ctx, row)
	if err != nil {
		return "", err
	}
	defer closeFn()
	tools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: cli, ToolNameList: []string{toolName}, CustomHeaders: headers})
	if err != nil {
		return "", err
	}
	for _, baseTool := range tools {
		info, err := baseTool.Info(ctx)
		if err != nil {
			return "", err
		}
		if info == nil || info.Name != toolName {
			continue
		}
		invokable, ok := baseTool.(tool.InvokableTool)
		if !ok {
			return "", fmt.Errorf("MCP tool %q 不支持同步调用", toolName)
		}
		return invokable.InvokableRun(ctx, rawArgs)
	}
	return "", fmt.Errorf("MCP tool 不存在: %s", toolName)
}

func newMcpClientFromConfig(ctx context.Context, row dao.MCPConfig) (mcpclient.MCPClient, map[string]string, func(), error) {
	t := strings.ToLower(strings.TrimSpace(row.Transport))
	if t == "" {
		t = "http"
	}
	switch t {
	case "stdio":
		args, err := parseJSONStringArray(row.StdioArgsJSON)
		if err != nil {
			return nil, nil, func() {}, fmt.Errorf("stdio_args_json: %w", err)
		}
		env, err := parseJSONStringMap(row.StdioEnvJSON)
		if err != nil {
			return nil, nil, func() {}, fmt.Errorf("stdio_env_json: %w", err)
		}
		cmd := strings.TrimSpace(row.StdioCommand)
		if cmd == "" {
			return nil, nil, func() {}, fmt.Errorf("stdio 模式需配置 stdio_command")
		}
		if err := mcpbridge.EnsureUvSyncBeforeStdio(cmd, args, env); err != nil {
			return nil, nil, func() {}, err
		}
		cli, err := mcpclient.NewStdioMCPClient(cmd, envMapToSlice(env), args...)
		if err != nil {
			return nil, nil, func() {}, fmt.Errorf("启动 MCP 进程失败: %w", err)
		}
		if err := initializeMcpClient(ctx, cli); err != nil {
			_ = cli.Close()
			return nil, nil, func() {}, fmt.Errorf("initialize 失败: %w", err)
		}
		return cli, nil, func() { _ = cli.Close() }, nil
	case "http":
		endpoint := strings.TrimSpace(row.Endpoint)
		if endpoint == "" {
			return nil, nil, func() {}, fmt.Errorf("http 模式需配置 endpoint")
		}
		headers := headersMapFromJSON(row.HeadersJSON)
		cli, err := mcpclient.NewSSEMCPClient(endpoint, mcpclient.WithHeaders(headers))
		if err != nil {
			return nil, nil, func() {}, err
		}
		if err := cli.Start(ctx); err != nil {
			_ = cli.Close()
			return nil, nil, func() {}, fmt.Errorf("SSE 连接失败: %w", err)
		}
		if err := initializeMcpClient(ctx, cli); err != nil {
			_ = cli.Close()
			return nil, nil, func() {}, fmt.Errorf("initialize 失败: %w", err)
		}
		return cli, headers, func() { _ = cli.Close() }, nil
	default:
		return nil, nil, func() {}, fmt.Errorf("未知 transport=%q（支持 http、stdio）", row.Transport)
	}
}

func initializeMcpClient(ctx context.Context, cli mcpclient.MCPClient) error {
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "ruleGoKratos", Version: "1.0"}
	_, err := cli.Initialize(ctx, initReq)
	return err
}

func envMapToSlice(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out = append(out, k+"="+v)
	}
	return out
}
