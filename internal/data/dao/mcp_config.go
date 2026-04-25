package dao

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

type MCPConfig struct {
	ID          int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string     `gorm:"column:name;size:128;not null;uniqueIndex:mcp_config_name_unique_idx" json:"name"`
	Server      string     `gorm:"column:server;size:128;not null" json:"server"`
	Endpoint    string     `gorm:"column:endpoint;type:text;not null;default:''" json:"endpoint"`
	HeadersJSON string     `gorm:"column:headers_json;type:text;not null;default:'{}'" json:"headersJson"`
	// Transport: http（默认，走 SSE endpoint）或 stdio（本地子进程）。
	Transport     string `gorm:"column:transport;size:16;not null;default:http" json:"transport"`
	StdioCommand  string `gorm:"column:stdio_command;type:text;not null;default:''" json:"stdioCommand"`
	StdioArgsJSON string `gorm:"column:stdio_args_json;type:text;not null;default:'[]'" json:"stdioArgsJson"`
	StdioEnvJSON  string `gorm:"column:stdio_env_json;type:text;not null;default:'{}'" json:"stdioEnvJson"`
	Enabled       bool   `gorm:"column:enabled;not null;default:true" json:"enabled"`
	Description   string `gorm:"column:description;type:text;not null;default:''" json:"description"`
	CreatedAt     *time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt     *time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (MCPConfig) TableName() string {
	return "mcp_config"
}

func NewMCPConfig() *MCPConfig {
	return &MCPConfig{}
}

func (m *MCPConfig) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(m).Error
}

func (m *MCPConfig) FindAll(ctx context.Context) ([]MCPConfig, error) {
	var list []MCPConfig
	err := db.WithContext(ctx).Model(m).Order("id desc").Find(&list).Error
	return list, err
}

func (m *MCPConfig) FindEnabled(ctx context.Context) ([]MCPConfig, error) {
	var list []MCPConfig
	err := db.WithContext(ctx).Model(m).Where("enabled = ?", true).Order("id desc").Find(&list).Error
	return list, err
}

func (m *MCPConfig) FindByIDs(ctx context.Context, ids []int64) ([]MCPConfig, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []MCPConfig
	err := db.WithContext(ctx).Model(m).Where("id IN ?", ids).Find(&list).Error
	return list, err
}

// FindByServer 按逻辑 server 名查找已启用的 MCP 配置（call_mcp_tool 路由用）。
func (m *MCPConfig) FindByServer(ctx context.Context, server string) (*MCPConfig, error) {
	s := strings.TrimSpace(server)
	if s == "" {
		return nil, errors.New("server 为空")
	}
	var row MCPConfig
	err := db.WithContext(ctx).Model(m).Where("server = ? AND enabled = ?", s, true).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (m *MCPConfig) Updates(ctx context.Context, where map[string]interface{}, data map[string]interface{}) error {
	return db.WithContext(ctx).Model(m).Where(where).Updates(data).Error
}

func (m *MCPConfig) Delete(ctx context.Context, where map[string]interface{}) error {
	return db.WithContext(ctx).Where(where).Delete(m).Error
}

// migrateMcpConfigTable 表不存在时自动建表；已存在时补齐缺失列（与 sql/mcp_config.sql 对齐）。
func migrateMcpConfigTable(g *gorm.DB) error {
	return g.AutoMigrate(&MCPConfig{})
}
