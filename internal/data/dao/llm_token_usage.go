package dao

import (
	"context"
	"time"
)

type LLMTokenUsage struct {
	ID           int64     `gorm:"column:id;primaryKey" json:"id"`
	ConfigID     int64     `gorm:"column:config_id" json:"configId"`
	ModelEntryID int64     `gorm:"column:model_entry_id" json:"modelEntryId"`
	SessionID    string    `gorm:"column:session_id" json:"sessionId"`
	RequestID    string    `gorm:"column:request_id" json:"requestId"`
	PromptTokens int64     `gorm:"column:prompt_tokens" json:"promptTokens"`
	CompletionTokens int64  `gorm:"column:completion_tokens" json:"completionTokens"`
	TotalTokens  int64     `gorm:"column:total_tokens" json:"totalTokens"`
	ModelName   string    `gorm:"column:model_name" json:"modelName"`
	ActionType  string    `gorm:"column:action_type" json:"actionType"`
	UserID      string    `gorm:"column:user_id" json:"userId"`
	ProjectPath string    `gorm:"column:project_path" json:"projectPath"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (LLMTokenUsage) TableName() string {
	return "llm_token_usage"
}

func NewLLMTokenUsage() *LLMTokenUsage {
	return &LLMTokenUsage{}
}

func (m *LLMTokenUsage) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(m).Error
}

func (m *LLMTokenUsage) CreateBatch(ctx context.Context, records []*LLMTokenUsage) error {
	if len(records) == 0 {
		return nil
	}
	return db.WithContext(ctx).CreateInBatches(records, 100).Error
}

type TokenStatItem struct {
	Period         string `json:"period"`
	PromptTokens   int64  `json:"promptTokens"`
	CompletionTokens int64 `json:"completionTokens"`
	TotalTokens    int64  `json:"totalTokens"`
	RequestCount   int64  `json:"requestCount"`
}

func (m *LLMTokenUsage) GetDailyStats(ctx context.Context, configID, modelEntryID int64, startDate, endDate string) ([]TokenStatItem, error) {
	sql := `SELECT
		DATE(created_at) as period,
		COALESCE(SUM(prompt_tokens), 0) as promptTokens,
		COALESCE(SUM(completion_tokens), 0) as completionTokens,
		COALESCE(SUM(total_tokens), 0) as totalTokens,
		COUNT(*) as requestCount
	FROM llm_token_usage WHERE 1=1`

	args := []interface{}{}
	if startDate != "" && endDate != "" {
		sql += " AND created_at >= ? AND created_at <= ?"
		args = append(args, startDate, endDate)
	}
	if configID > 0 {
		sql += " AND config_id = ?"
		args = append(args, configID)
	}
	if modelEntryID > 0 {
		sql += " AND model_entry_id = ?"
		args = append(args, modelEntryID)
	}
	sql += " GROUP BY DATE(created_at) ORDER BY period asc"

	rows, err := db.WithContext(ctx).Raw(sql, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TokenStatItem
	for rows.Next() {
		var item TokenStatItem
		if err := rows.Scan(&item.Period, &item.PromptTokens, &item.CompletionTokens, &item.TotalTokens, &item.RequestCount); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, nil
}

func (m *LLMTokenUsage) GetStatsByModel(ctx context.Context, configID int64, startDate, endDate string) ([]TokenStatItem, error) {
	sql := `SELECT
		model_name as period,
		COALESCE(SUM(prompt_tokens), 0) as promptTokens,
		COALESCE(SUM(completion_tokens), 0) as completionTokens,
		COALESCE(SUM(total_tokens), 0) as totalTokens,
		COUNT(*) as requestCount
	FROM llm_token_usage WHERE 1=1`

	args := []interface{}{}
	if startDate != "" && endDate != "" {
		sql += " AND created_at >= ? AND created_at <= ?"
		args = append(args, startDate, endDate)
	}
	if configID > 0 {
		sql += " AND config_id = ?"
		args = append(args, configID)
	}
	sql += " GROUP BY model_name ORDER BY totalTokens desc"

	rows, err := db.WithContext(ctx).Raw(sql, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TokenStatItem
	for rows.Next() {
		var item TokenStatItem
		if err := rows.Scan(&item.Period, &item.PromptTokens, &item.CompletionTokens, &item.TotalTokens, &item.RequestCount); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, nil
}

func (m *LLMTokenUsage) ListUsage(ctx context.Context, configID, modelEntryID int64, startDate, endDate string, limit, offset int) ([]LLMTokenUsage, int64, error) {
	var list []LLMTokenUsage
	var total int64

	query := db.WithContext(ctx).Model(m)
	if startDate != "" && endDate != "" {
		query = query.Where("created_at >= ? AND created_at <= ?", startDate, endDate)
	}
	if configID > 0 {
		query = query.Where("config_id = ?", configID)
	}
	if modelEntryID > 0 {
		query = query.Where("model_entry_id = ?", modelEntryID)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at desc").Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}