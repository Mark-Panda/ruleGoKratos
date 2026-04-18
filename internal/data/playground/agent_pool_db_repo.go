package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/data/dao"

	"gorm.io/gorm"
)

// GormAgentPoolRepo Agent 池 PostgreSQL 持久化实现。
type GormAgentPoolRepo struct {
	db *gorm.DB
}

// NewGormAgentPoolRepo 由 Wire 注入 *gorm.DB（来自 data.Data）。
func NewGormAgentPoolRepo(db *gorm.DB) *GormAgentPoolRepo {
	return &GormAgentPoolRepo{db: db}
}

func poolEntityToDAO(pool *entity.AgentPool) (*dao.PlaygroundAgentPool, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool is nil")
	}
	agents := pool.Agents
	if agents == nil {
		agents = []*entity.AgentDefinition{}
	}
	b, err := json.Marshal(agents)
	if err != nil {
		return nil, fmt.Errorf("marshal agents: %w", err)
	}
	return &dao.PlaygroundAgentPool{
		ID:          pool.ID,
		Name:        pool.Name,
		Description: pool.Description,
		AgentsJSON:  string(b),
		CreatedAt:   pool.CreatedAt,
		UpdatedAt:   pool.UpdatedAt,
	}, nil
}

func poolDAOToEntity(row *dao.PlaygroundAgentPool) (*entity.AgentPool, error) {
	if row == nil {
		return nil, fmt.Errorf("row is nil")
	}
	var agents []*entity.AgentDefinition
	if row.AgentsJSON != "" {
		if err := json.Unmarshal([]byte(row.AgentsJSON), &agents); err != nil {
			return nil, fmt.Errorf("decode agents_json: %w", err)
		}
	}
	if agents == nil {
		agents = []*entity.AgentDefinition{}
	}
	return &entity.AgentPool{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Agents:      agents,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

// Create 新建池（id 已存在则失败）。
func (r *GormAgentPoolRepo) Create(ctx context.Context, pool *entity.AgentPool) error {
	row, err := poolEntityToDAO(pool)
	if err != nil {
		return err
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&dao.PlaygroundAgentPool{}).Where("id = ?", pool.ID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("pool already exists: %s", pool.ID)
	}
	return r.db.WithContext(ctx).Create(row).Error
}

// Update 整体覆盖池记录（幂等写入）。
func (r *GormAgentPoolRepo) Update(ctx context.Context, pool *entity.AgentPool) error {
	if pool == nil {
		return fmt.Errorf("pool is nil")
	}
	if pool.ID == "" {
		return fmt.Errorf("pool id is empty")
	}
	row, err := poolEntityToDAO(pool)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(row).Error
}

// Delete 按 id 删除；无记录时返回 pool not found（与 DeletePool 期望一致）。
func (r *GormAgentPoolRepo) Delete(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&dao.PlaygroundAgentPool{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("pool not found: %s", id)
	}
	return nil
}

// FindByID 查询单池。
func (r *GormAgentPoolRepo) FindByID(ctx context.Context, id string) (*entity.AgentPool, error) {
	var row dao.PlaygroundAgentPool
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("pool not found: %s", id)
		}
		return nil, err
	}
	return poolDAOToEntity(&row)
}

// FindAll 列出全部池（按更新时间倒序）。
func (r *GormAgentPoolRepo) FindAll(ctx context.Context) ([]*entity.AgentPool, error) {
	var rows []dao.PlaygroundAgentPool
	if err := r.db.WithContext(ctx).Model(&dao.PlaygroundAgentPool{}).
		Order("updated_at DESC NULLS LAST, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*entity.AgentPool, 0, len(rows))
	for i := range rows {
		p, err := poolDAOToEntity(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}
