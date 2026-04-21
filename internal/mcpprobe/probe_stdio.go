package mcpprobe

import (
	"context"
	"fmt"
	"strings"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"ruleGoKratos/internal/mcpbridge"
)

// ProbeStdio 启动本地子进程（MCP stdio），完成 initialize 并列出工具名称。
func ProbeStdio(ctx context.Context, command string, args []string, env map[string]string) (serverName string, protocolVersion string, toolNames []string, err error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", "", nil, fmt.Errorf("stdio_command 为空")
	}
	if err := mcpbridge.EnsureUvSyncBeforeStdio(command, args, env); err != nil {
		return "", "", nil, err
	}
	envSlice := envMapToEnvSlice(env)

	cli, err := mcpclient.NewStdioMCPClient(command, envSlice, args...)
	if err != nil {
		return "", "", nil, fmt.Errorf("启动 MCP 子进程失败: %w", err)
	}
	defer func() { _ = cli.Close() }()

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

func envMapToEnvSlice(m map[string]string) []string {
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
