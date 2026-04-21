// Package mcpbridge 封装 MCP 协议上的 tools/call（HTTP/SSE 与 stdio 两种传输）。
package mcpbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const defaultMCPCallTimeout = 120 * time.Second

// uvSyncMaxDuration stdio 下 command 为 uv 时，先于 MCP 执行「uv sync」；与 MCP 调用超时分离，避免首次解析依赖占满 120s。
const uvSyncMaxDuration = 15 * time.Minute

func mcpInitialize(ctx context.Context, cli mcpclient.MCPClient) error {
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "ruleGoKratos", Version: "1.0"}
	_, err := cli.Initialize(ctx, initReq)
	return err
}

// CallHTTPMCPTool 通过 SSE 连接 endpoint，执行 tools/call。
func CallHTTPMCPTool(ctx context.Context, endpoint string, headers map[string]string, toolName, argumentsJSON string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("endpoint 为空")
	}
	if headers == nil {
		headers = map[string]string{}
	}
	c, cancel := context.WithTimeout(ctx, defaultMCPCallTimeout)
	defer cancel()

	cli, err := mcpclient.NewSSEMCPClient(endpoint, mcpclient.WithHeaders(headers))
	if err != nil {
		return "", err
	}
	defer func() { _ = cli.Close() }()

	if err := cli.Start(c); err != nil {
		return "", fmt.Errorf("SSE 连接失败: %w", err)
	}
	if err := mcpInitialize(c, cli); err != nil {
		return "", fmt.Errorf("initialize 失败: %w", err)
	}
	return callTool(c, cli, toolName, argumentsJSON)
}

// CallStdioMCPTool 启动子进程并通过 stdio 执行 tools/call。
func CallStdioMCPTool(ctx context.Context, command string, args []string, env map[string]string, toolName, argumentsJSON string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("stdio_command 为空")
	}
	if err := EnsureUvSyncBeforeStdio(command, args, env); err != nil {
		return "", err
	}
	envSlice := envMapToSlice(env)
	c, cancel := context.WithTimeout(ctx, defaultMCPCallTimeout)
	defer cancel()

	cli, err := mcpclient.NewStdioMCPClient(command, envSlice, args...)
	if err != nil {
		return "", fmt.Errorf("启动 MCP 进程失败: %w", err)
	}
	defer func() { _ = cli.Close() }()

	if err := mcpInitialize(c, cli); err != nil {
		return "", fmt.Errorf("initialize 失败: %w", err)
	}
	return callTool(c, cli, toolName, argumentsJSON)
}

// EnsureUvSyncBeforeStdio 当 stdio_command 为 uv 且参数中含 --directory/-C 时，先在项目目录执行 uv sync，再供后续 MCP 子进程使用。
// 使用独立超时，不计入 defaultMCPCallTimeout。
func EnsureUvSyncBeforeStdio(command string, args []string, env map[string]string) error {
	base := filepath.Base(strings.TrimSpace(command))
	base = strings.TrimSuffix(base, ".exe")
	if !strings.EqualFold(base, "uv") {
		return nil
	}
	dir := parseUvProjectDirFromArgs(args)
	if dir == "" {
		return nil
	}
	dir = filepath.Clean(dir)
	st, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("uv sync: 目录不可用 %q: %w", dir, err)
	}
	if !st.IsDir() {
		return fmt.Errorf("uv sync: 不是目录 %q", dir)
	}
	syncCtx, cancel := context.WithTimeout(context.Background(), uvSyncMaxDuration)
	defer cancel()
	cmd := exec.CommandContext(syncCtx, command, "sync")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), envMapToSlice(env)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		tail := out.String()
		if len(tail) > 4000 {
			tail = tail[len(tail)-4000:]
		}
		if tail != "" {
			return fmt.Errorf("uv sync: %w\n%s", err, tail)
		}
		return fmt.Errorf("uv sync: %w", err)
	}
	return nil
}

func parseUvProjectDirFromArgs(args []string) string {
	for i := 0; i < len(args); i++ {
		a := strings.TrimSpace(args[i])
		switch a {
		case "--directory", "-C":
			if i+1 < len(args) {
				return strings.TrimSpace(args[i+1])
			}
		}
	}
	return ""
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

func callTool(ctx context.Context, cli mcpclient.MCPClient, toolName, argumentsJSON string) (string, error) {
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		return "", fmt.Errorf("tool 为空")
	}
	var req mcp.CallToolRequest
	req.Params.Name = toolName
	if strings.TrimSpace(argumentsJSON) != "" {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(argumentsJSON), &obj); err != nil {
			return "", fmt.Errorf("arguments 须为 JSON 对象: %w", err)
		}
		req.Params.Arguments = obj
	}
	res, err := cli.CallTool(ctx, req)
	if err != nil {
		return "", err
	}
	return formatCallToolResult(res), nil
}

func formatCallToolResult(res *mcp.CallToolResult) string {
	if res == nil {
		return ""
	}
	var parts []string
	for _, c := range res.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			parts = append(parts, tc.Text)
			continue
		}
		b, err := json.Marshal(c)
		if err != nil {
			continue
		}
		parts = append(parts, string(b))
	}
	out := strings.Join(parts, "\n")
	if res.IsError {
		out = "isError=true\n" + out
	}
	if out != "" {
		return out
	}
	b, _ := json.Marshal(res)
	return string(b)
}
