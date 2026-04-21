// Package mcpprobe 使用 MCP over HTTP(SSE) 客户端探测远端 MCP 服务是否可用。
package mcpprobe

import (
	"context"
	"fmt"
	"strings"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Probe 连接 endpoint（须为 MCP SSE 入口），完成 initialize 并列出工具名称。
func Probe(ctx context.Context, endpoint string, headers map[string]string) (serverName string, protocolVersion string, toolNames []string, err error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", "", nil, fmt.Errorf("endpoint 为空")
	}
	cli, err := mcpclient.NewSSEMCPClient(endpoint, mcpclient.WithHeaders(headers))
	if err != nil {
		return "", "", nil, fmt.Errorf("创建 MCP 客户端失败: %w", err)
	}
	defer func() { _ = cli.Close() }()

	if err := cli.Start(ctx); err != nil {
		return "", "", nil, fmt.Errorf("SSE 连接失败: %w（请确认 endpoint 为支持 MCP 的 SSE 入口，例如 …/sse）", err)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "ruleGoKratos-admin", Version: "1.0"}

	initRes, err := cli.Initialize(ctx, initReq)
	if err != nil {
		return "", "", nil, fmt.Errorf("initialize 失败: %w", err)
	}
	serverName = initRes.ServerInfo.Name
	protocolVersion = initRes.ProtocolVersion

	listRes, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return "", "", nil, fmt.Errorf("tools/list 失败: %w", err)
	}
	for _, t := range listRes.Tools {
		toolNames = append(toolNames, t.Name)
	}
	return serverName, protocolVersion, toolNames, nil
}
