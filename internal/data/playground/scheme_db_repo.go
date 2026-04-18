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

// GormSchemeRepo 协作编排方案 PostgreSQL 持久化实现。
type GormSchemeRepo struct {
	db *gorm.DB
}

// NewGormSchemeRepo 由 Wire 注入 *gorm.DB。
func NewGormSchemeRepo(db *gorm.DB) *GormSchemeRepo {
	return &GormSchemeRepo{db: db}
}

func schemeEntityToDAO(s *entity.CollaborationScheme) (*dao.PlaygroundCollaborationScheme, error) {
	if s == nil {
		return nil, fmt.Errorf("scheme is nil")
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal scheme: %w", err)
	}
	return &dao.PlaygroundCollaborationScheme{
		ID:         s.ID,
		SchemeJSON: string(b),
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}, nil
}

func schemeDAOToEntity(row *dao.PlaygroundCollaborationScheme) (*entity.CollaborationScheme, error) {
	if row == nil || row.SchemeJSON == "" {
		return nil, fmt.Errorf("scheme row invalid")
	}
	var s entity.CollaborationScheme
	if err := json.Unmarshal([]byte(row.SchemeJSON), &s); err != nil {
		return nil, fmt.Errorf("decode scheme_json: %w", err)
	}
	if row.ID != "" {
		s.ID = row.ID
	}
	return &s, nil
}

// SaveScheme 新建方案（id 已存在则失败）。
func (r *GormSchemeRepo) SaveScheme(ctx context.Context, scheme *entity.CollaborationScheme) error {
	row, err := schemeEntityToDAO(scheme)
	if err != nil {
		return err
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&dao.PlaygroundCollaborationScheme{}).Where("id = ?", scheme.ID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("scheme already exists: %s", scheme.ID)
	}
	return r.db.WithContext(ctx).Create(row).Error
}

// UpdateScheme 整体覆盖。
func (r *GormSchemeRepo) UpdateScheme(ctx context.Context, scheme *entity.CollaborationScheme) error {
	if scheme == nil || scheme.ID == "" {
		return fmt.Errorf("scheme id is empty")
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&dao.PlaygroundCollaborationScheme{}).Where("id = ?", scheme.ID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("scheme not found: %s", scheme.ID)
	}
	row, err := schemeEntityToDAO(scheme)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(row).Error
}

// DeleteScheme 按 id 删除。
func (r *GormSchemeRepo) DeleteScheme(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&dao.PlaygroundCollaborationScheme{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("scheme not found: %s", id)
	}
	return nil
}

// FindSchemeByID 查询单条。
func (r *GormSchemeRepo) FindSchemeByID(ctx context.Context, id string) (*entity.CollaborationScheme, error) {
	var row dao.PlaygroundCollaborationScheme
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("scheme not found: %s", id)
		}
		return nil, err
	}
	return schemeDAOToEntity(&row)
}

// FindAllSchemes 全部方案（按更新时间倒序）。
func (r *GormSchemeRepo) FindAllSchemes(ctx context.Context) ([]*entity.CollaborationScheme, error) {
	var rows []dao.PlaygroundCollaborationScheme
	if err := r.db.WithContext(ctx).Model(&dao.PlaygroundCollaborationScheme{}).
		Order("updated_at DESC NULLS LAST, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*entity.CollaborationScheme, 0, len(rows))
	for i := range rows {
		s, err := schemeDAOToEntity(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}
