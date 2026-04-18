package biz

import "context"

// ManagedLLMResolver 从模型管理（PostgreSQL）解析启用配置下的模型名与凭证。
type ManagedLLMResolver interface {
	ResolveManagedLLM(ctx context.Context, configID int64, entryID int64) (modelName string, apiKey string, baseURL string, err error)
}
