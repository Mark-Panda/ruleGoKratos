package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/conf"
	"ruleGoKratos/internal/data/dao"
	"sort"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/structpb"
)

type AdminService struct {
	v1.UnimplementedAdminServer
	log       *log.Helper
	config    *conf.Bootstrap
	skillRoot string
}

type mcpConfigPayload struct {
	Name        string
	Server      string
	Endpoint    string
	Enabled     bool
	Description string
}

func NewAdminService(logger log.Logger, config *conf.Bootstrap) *AdminService {
	helper := log.NewHelper(logger)
	root := "skills"
	if config != nil && config.Agent != nil && config.Agent.Skill != nil && strings.TrimSpace(config.Agent.Skill.Dir) != "" {
		root = strings.TrimSpace(config.Agent.Skill.Dir)
	}
	return &AdminService{
		log:       helper,
		config:    config,
		skillRoot: root,
	}
}

func (s *AdminService) ListSkills(ctx context.Context, _ *v1.ListSkillsRequest) (*v1.ListSkillsReply, error) {
	root := s.skillRoot
	items := make([]*v1.SkillItem, 0, 64)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".txt" && ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return nil
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = filepath.Base(path)
		}
		items = append(items, &v1.SkillItem{
			Name:      filepath.Base(path),
			Path:      filepath.ToSlash(rel),
			Size:      info.Size(),
			UpdatedAt: info.ModTime().Format(time.RFC3339),
		})
		return nil
	})
	sort.Slice(items, func(i, j int) bool { return items[i].GetPath() < items[j].GetPath() })
	return &v1.ListSkillsReply{Root: root, Items: items}, nil
}

func (s *AdminService) UploadSkill(ctx context.Context, req *v1.UploadSkillRequest) (*v1.UploadSkillReply, error) {
	if err := os.MkdirAll(s.skillRoot, 0o755); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.GetPath())
	if name == "" {
		return nil, errors.New("path不能为空")
	}
	safeName, err := sanitizeSkillFileName(name)
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(req.GetContentBase64())
	if raw == "" {
		return nil, errors.New("contentBase64不能为空")
	}
	content, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, errors.New("contentBase64格式错误")
	}
	dstPath := filepath.Join(s.skillRoot, safeName)
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(dstPath, content, 0o644); err != nil {
		return nil, err
	}
	return &v1.UploadSkillReply{Path: filepath.ToSlash(safeName)}, nil
}

func (s *AdminService) ListMcpConfigs(ctx context.Context, _ *v1.ListMcpConfigsRequest) (*v1.ListMcpConfigsReply, error) {
	list, err := dao.NewMCPConfig().FindAll(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*v1.McpConfigItem, 0, len(list))
	for _, it := range list {
		items = append(items, toMCPProto(it))
	}
	return &v1.ListMcpConfigsReply{Items: items}, nil
}

func (s *AdminService) CreateMcpConfig(ctx context.Context, req *v1.CreateMcpConfigRequest) (*v1.McpConfigItem, error) {
	payload := mcpConfigPayload{
		Name:        req.GetName(),
		Server:      req.GetServer(),
		Endpoint:    req.GetEndpoint(),
		Enabled:     req.GetEnabled(),
		Description: req.GetDescription(),
	}
	if err := validateMCPPayload(payload); err != nil {
		return nil, err
	}
	headersJSON := normalizeHeaders(req.GetHeaders())
	now := time.Now()
	row := dao.MCPConfig{
		Name:        strings.TrimSpace(payload.Name),
		Server:      strings.TrimSpace(payload.Server),
		Endpoint:    strings.TrimSpace(payload.Endpoint),
		HeadersJSON: headersJSON,
		Enabled:     payload.Enabled,
		Description: strings.TrimSpace(payload.Description),
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	if err := row.Create(ctx); err != nil {
		return nil, err
	}
	return toMCPProto(row), nil
}

func (s *AdminService) UpdateMcpConfig(ctx context.Context, req *v1.UpdateMcpConfigRequest) (*v1.UpdateMcpConfigReply, error) {
	if req.GetId() <= 0 {
		return nil, errors.New("id不合法")
	}
	payload := mcpConfigPayload{
		Name:        req.GetName(),
		Server:      req.GetServer(),
		Endpoint:    req.GetEndpoint(),
		Enabled:     req.GetEnabled(),
		Description: req.GetDescription(),
	}
	if err := validateMCPPayload(payload); err != nil {
		return nil, err
	}
	headersJSON := normalizeHeaders(req.GetHeaders())
	err := dao.NewMCPConfig().Updates(ctx, map[string]interface{}{"id": req.GetId()}, map[string]interface{}{
		"name":         strings.TrimSpace(payload.Name),
		"server":       strings.TrimSpace(payload.Server),
		"endpoint":     strings.TrimSpace(payload.Endpoint),
		"headers_json": headersJSON,
		"enabled":      payload.Enabled,
		"description":  strings.TrimSpace(payload.Description),
		"updated_at":   time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateMcpConfigReply{}, nil
}

func (s *AdminService) DeleteMcpConfig(ctx context.Context, req *v1.DeleteMcpConfigRequest) (*v1.DeleteMcpConfigReply, error) {
	if req.GetId() <= 0 {
		return nil, errors.New("id不合法")
	}
	if err := dao.NewMCPConfig().Delete(ctx, map[string]interface{}{"id": req.GetId()}); err != nil {
		return nil, err
	}
	return &v1.DeleteMcpConfigReply{}, nil
}

func validateMCPPayload(req mcpConfigPayload) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name不能为空")
	}
	if strings.TrimSpace(req.Server) == "" {
		return errors.New("server不能为空")
	}
	if strings.TrimSpace(req.Endpoint) == "" {
		return errors.New("endpoint不能为空")
	}
	return nil
}

func sanitizeSkillFileName(name string) (string, error) {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		return "", errors.New("文件名不能为空")
	}
	cleaned := filepath.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", errors.New("文件名不合法")
	}
	return cleaned, nil
}

func toMCPProto(it dao.MCPConfig) *v1.McpConfigItem {
	createdAt := ""
	updatedAt := ""
	if it.CreatedAt != nil {
		createdAt = it.CreatedAt.Format(time.RFC3339Nano)
	}
	if it.UpdatedAt != nil {
		updatedAt = it.UpdatedAt.Format(time.RFC3339Nano)
	}
	return &v1.McpConfigItem{
		Id:          it.ID,
		Name:        it.Name,
		Server:      it.Server,
		Endpoint:    it.Endpoint,
		Headers:     parseHeadersToStruct(it.HeadersJSON),
		Enabled:     it.Enabled,
		Description: it.Description,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func normalizeHeaders(headers *structpb.Struct) string {
	if headers == nil {
		return "{}"
	}
	obj := headers.AsMap()
	data, err := json.Marshal(obj)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func parseHeadersToStruct(raw string) *structpb.Struct {
	text := strings.TrimSpace(raw)
	if text == "" {
		st, _ := structpb.NewStruct(map[string]interface{}{})
		return st
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		st, _ := structpb.NewStruct(map[string]interface{}{})
		return st
	}
	st, err := structpb.NewStruct(obj)
	if err != nil {
		st, _ = structpb.NewStruct(map[string]interface{}{})
		return st
	}
	return st
}
