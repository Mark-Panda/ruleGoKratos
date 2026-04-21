package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/data/dao"
	"ruleGoKratos/internal/mcpbridge"
)

// NewDatabaseMcpExecutor 按「MCP 配置」表将 call_mcp_tool 路由到 HTTP(SSE) 或本地 stdio 进程。
func NewDatabaseMcpExecutor() biz.McpExecutor {
	return &databaseMcpExecutor{}
}

type databaseMcpExecutor struct{}

func (databaseMcpExecutor) Call(ctx context.Context, server, tool, arguments string) (string, error) {
	row, err := dao.NewMCPConfig().FindByServer(ctx, server)
	if err != nil {
		return "", fmt.Errorf("未找到已启用的 MCP 配置 server=%q: %w", server, err)
	}
	t := strings.ToLower(strings.TrimSpace(row.Transport))
	if t == "" {
		t = "http"
	}
	switch t {
	case "stdio":
		args, err := parseJSONStringArray(row.StdioArgsJSON)
		if err != nil {
			return "", fmt.Errorf("stdio_args_json: %w", err)
		}
		env, err := parseJSONStringMap(row.StdioEnvJSON)
		if err != nil {
			return "", fmt.Errorf("stdio_env_json: %w", err)
		}
		cmd := strings.TrimSpace(row.StdioCommand)
		if cmd == "" {
			return "", fmt.Errorf("stdio 模式需配置 stdio_command")
		}
		return mcpbridge.CallStdioMCPTool(ctx, cmd, args, env, tool, arguments)
	case "http":
		ep := strings.TrimSpace(row.Endpoint)
		if ep == "" {
			return "", fmt.Errorf("http 模式需配置 endpoint")
		}
		headers := headersMapFromJSON(row.HeadersJSON)
		return mcpbridge.CallHTTPMCPTool(ctx, ep, headers, tool, arguments)
	default:
		return "", fmt.Errorf("未知 transport=%q（支持 http、stdio）", row.Transport)
	}
}

func parseJSONStringArray(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseJSONStringMap(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = fmt.Sprint(v)
	}
	return out, nil
}

func headersMapFromJSON(raw string) map[string]string {
	m, err := parseJSONStringMap(raw)
	if err != nil {
		return nil
	}
	return m
}
