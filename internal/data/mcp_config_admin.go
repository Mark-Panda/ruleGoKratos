package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/data/dao"
)

type mcpConfigAdmin struct{}

// NewMcpConfigAdmin 供 Harness 工具 save_mcp_server_config 写入 mcp_config 表。
func NewMcpConfigAdmin() biz.McpConfigAdmin {
	return &mcpConfigAdmin{}
}

func (a *mcpConfigAdmin) UpsertMcpConfig(ctx context.Context, args biz.McpConfigUpsertArgs) (int64, string, error) {
	args.Name = strings.TrimSpace(args.Name)
	args.Server = strings.TrimSpace(args.Server)
	args.Endpoint = strings.TrimSpace(args.Endpoint)
	args.Description = strings.TrimSpace(args.Description)
	args.Transport = strings.ToLower(strings.TrimSpace(args.Transport))
	if args.Transport == "" {
		args.Transport = "http"
	}
	args.StdioCommand = strings.TrimSpace(args.StdioCommand)
	args.StdioArgsJSON = strings.TrimSpace(args.StdioArgsJSON)
	args.StdioEnvJSON = strings.TrimSpace(args.StdioEnvJSON)
	if args.Transport == "stdio" {
		if args.StdioArgsJSON == "" {
			args.StdioArgsJSON = "[]"
		}
		if args.StdioEnvJSON == "" {
			args.StdioEnvJSON = "{}"
		}
	}
	h := strings.TrimSpace(args.HeadersJSON)
	if h == "" {
		h = "{}"
	}
	var hdrObj map[string]interface{}
	if err := json.Unmarshal([]byte(h), &hdrObj); err != nil {
		return 0, "", fmt.Errorf("headers_json 须为 JSON 对象: %w", err)
	}
	args.HeadersJSON = h

	if err := validateMcpUpsertFields(args); err != nil {
		return 0, "", err
	}

	if args.ID > 0 {
		rows, err := dao.NewMCPConfig().FindByIDs(ctx, []int64{args.ID})
		if err != nil {
			return 0, "", err
		}
		if len(rows) == 0 {
			return 0, "", errors.New("MCP 配置不存在")
		}
		err = dao.NewMCPConfig().Updates(ctx, map[string]interface{}{"id": args.ID}, map[string]interface{}{
			"name":            args.Name,
			"server":          args.Server,
			"endpoint":        args.Endpoint,
			"headers_json":    args.HeadersJSON,
			"transport":       args.Transport,
			"stdio_command":   args.StdioCommand,
			"stdio_args_json": args.StdioArgsJSON,
			"stdio_env_json":  args.StdioEnvJSON,
			"enabled":         args.Enabled,
			"description":     args.Description,
			"updated_at":      time.Now(),
		})
		if err != nil {
			return 0, "", err
		}
		return args.ID, "updated", nil
	}

	now := time.Now()
	row := &dao.MCPConfig{
		Name:          args.Name,
		Server:        args.Server,
		Endpoint:      args.Endpoint,
		HeadersJSON:   args.HeadersJSON,
		Transport:     args.Transport,
		StdioCommand:  args.StdioCommand,
		StdioArgsJSON: args.StdioArgsJSON,
		StdioEnvJSON:  args.StdioEnvJSON,
		Enabled:       args.Enabled,
		Description:   args.Description,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := row.Create(ctx); err != nil {
		return 0, "", err
	}
	return row.ID, "created", nil
}

func validateMcpUpsertFields(args biz.McpConfigUpsertArgs) error {
	if args.Name == "" || args.Server == "" {
		return errors.New("name、server 不能为空")
	}
	switch args.Transport {
	case "http":
		if args.Endpoint == "" {
			return errors.New("http 模式 endpoint 不能为空")
		}
	case "stdio":
		if args.StdioCommand == "" {
			return errors.New("stdio 模式 stdio_command 不能为空")
		}
		var arr []string
		if err := json.Unmarshal([]byte(args.StdioArgsJSON), &arr); err != nil {
			return fmt.Errorf("stdio_args_json 须为 JSON 字符串数组: %w", err)
		}
		var envObj map[string]interface{}
		if err := json.Unmarshal([]byte(args.StdioEnvJSON), &envObj); err != nil {
			return fmt.Errorf("stdio_env_json 须为 JSON 对象: %w", err)
		}
	default:
		return errors.New("transport 须为 http 或 stdio")
	}
	return nil
}
