package biz

import "context"

// McpConfigUpsertArgs 写入「MCP 配置」表（server 会作为具体 MCP tool 的命名前缀）。
type McpConfigUpsertArgs struct {
	// ID 非零表示更新已有记录；为零表示新建。
	ID int64
	// Name 展示名（唯一性由存储层/业务约束，与 server 不同）。
	Name string
	// Server 逻辑服务名，用于区分不同 MCP server 下的同名工具。
	Server string
	// Transport: http（默认）或 stdio。
	Transport string
	// Endpoint 须为可访问的 MCP over HTTP(SSE) 入口 URL；stdio 时可为空。
	Endpoint string
	// HeadersJSON 请求头 JSON 对象字符串，如 {} 或 {"Authorization":"Bearer ..."}。
	HeadersJSON string
	// StdioCommand / StdioArgsJSON / StdioEnvJSON 在 transport=stdio 时使用。
	StdioCommand  string
	StdioArgsJSON string
	StdioEnvJSON  string
	Enabled       bool
	Description   string
}

// McpConfigAdmin 将 MCP 连接登记到本机配置表，供管理端与 Harness 使用。
type McpConfigAdmin interface {
	UpsertMcpConfig(ctx context.Context, args McpConfigUpsertArgs) (id int64, action string, err error)
}
