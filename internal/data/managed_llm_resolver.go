package data

import (
	"context"
	"errors"
	"strings"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/data/dao"
)

type managedLLMResolver struct{}

// NewManagedLLMResolver 从 llm_config / llm_model_entry 表解析凭证（供 Agent 与 ai/llm 节点使用）。
func NewManagedLLMResolver() biz.ManagedLLMResolver {
	return &managedLLMResolver{}
}

func (managedLLMResolver) ResolveManagedLLM(ctx context.Context, configID int64, entryID int64) (modelName string, apiKey string, baseURL string, err error) {
	if configID <= 0 || entryID <= 0 {
		return "", "", "", errors.New("非法的 LLM 配置或模型 ID")
	}
	cfg, err := dao.NewLLMConfig().FindByID(ctx, configID)
	if err != nil {
		return "", "", "", errors.New("LLM 配置不存在")
	}
	if !cfg.Enabled {
		return "", "", "", errors.New("LLM 配置已禁用")
	}
	ent, err := dao.NewLLMModelEntry().FindByID(ctx, entryID)
	if err != nil {
		return "", "", "", errors.New("模型条目不存在")
	}
	if ent.ConfigID != configID {
		return "", "", "", errors.New("模型不属于所选配置")
	}
	if !ent.Enabled {
		return "", "", "", errors.New("该模型已禁用")
	}
	return strings.TrimSpace(ent.ModelName), strings.TrimSpace(cfg.APIKey), strings.TrimSpace(cfg.BaseURL), nil
}
