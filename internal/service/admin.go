package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz/playground/agentpool"
	"ruleGoKratos/internal/conf"
	"ruleGoKratos/internal/data/dao"
	"ruleGoKratos/internal/mcpprobe"
	"sort"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"google.golang.org/protobuf/types/known/structpb"
)

type AdminService struct {
	v1.UnimplementedAdminServer
	log       *log.Helper
	config    *conf.Bootstrap
	skillRoot string
	// playground Agent 池服务：删除「Agent 配置」前校验是否被池内 Agent 引用
	poolSvc *agentpool.AgentPoolService
}

type mcpConfigPayload struct {
	Name          string
	Server        string
	Endpoint      string
	Enabled       bool
	Description   string
	Transport     string
	StdioCommand  string
	StdioArgsJSON string
	StdioEnvJSON  string
}

func NewAdminService(logger log.Logger, config *conf.Bootstrap, poolSvc *agentpool.AgentPoolService) *AdminService {
	helper := log.NewHelper(logger)
	root := "skills"
	if config != nil && config.Agent != nil && config.Agent.Skill != nil && strings.TrimSpace(config.Agent.Skill.Dir) != "" {
		root = strings.TrimSpace(config.Agent.Skill.Dir)
	}
	return &AdminService{
		log:       helper,
		config:    config,
		skillRoot: root,
		poolSvc:   poolSvc,
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
		Name:          req.GetName(),
		Server:        req.GetServer(),
		Endpoint:      req.GetEndpoint(),
		Enabled:       req.GetEnabled(),
		Description:   req.GetDescription(),
		Transport:     req.GetTransport(),
		StdioCommand:  req.GetStdioCommand(),
		StdioArgsJSON: req.GetStdioArgsJson(),
		StdioEnvJSON:  req.GetStdioEnvJson(),
	}
	normalizeMCPPayload(&payload)
	if err := validateMCPPayload(payload); err != nil {
		return nil, err
	}
	headersJSON := normalizeHeaders(req.GetHeaders())
	now := time.Now()
	row := dao.MCPConfig{
		Name:          strings.TrimSpace(payload.Name),
		Server:        strings.TrimSpace(payload.Server),
		Endpoint:      strings.TrimSpace(payload.Endpoint),
		HeadersJSON:   headersJSON,
		Transport:     payload.Transport,
		StdioCommand:  strings.TrimSpace(payload.StdioCommand),
		StdioArgsJSON: payload.StdioArgsJSON,
		StdioEnvJSON:  payload.StdioEnvJSON,
		Enabled:       payload.Enabled,
		Description:   strings.TrimSpace(payload.Description),
		CreatedAt:     &now,
		UpdatedAt:     &now,
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
		Name:          req.GetName(),
		Server:        req.GetServer(),
		Endpoint:      req.GetEndpoint(),
		Enabled:       req.GetEnabled(),
		Description:   req.GetDescription(),
		Transport:     req.GetTransport(),
		StdioCommand:  req.GetStdioCommand(),
		StdioArgsJSON: req.GetStdioArgsJson(),
		StdioEnvJSON:  req.GetStdioEnvJson(),
	}
	normalizeMCPPayload(&payload)
	if err := validateMCPPayload(payload); err != nil {
		return nil, err
	}
	headersJSON := normalizeHeaders(req.GetHeaders())
	err := dao.NewMCPConfig().Updates(ctx, map[string]interface{}{"id": req.GetId()}, map[string]interface{}{
		"name":            strings.TrimSpace(payload.Name),
		"server":          strings.TrimSpace(payload.Server),
		"endpoint":        strings.TrimSpace(payload.Endpoint),
		"headers_json":    headersJSON,
		"transport":       payload.Transport,
		"stdio_command":   strings.TrimSpace(payload.StdioCommand),
		"stdio_args_json": payload.StdioArgsJSON,
		"stdio_env_json":  payload.StdioEnvJSON,
		"enabled":         payload.Enabled,
		"description":     strings.TrimSpace(payload.Description),
		"updated_at":      time.Now(),
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

func (s *AdminService) TestMcpConfig(ctx context.Context, req *v1.TestMcpConfigRequest) (*v1.TestMcpConfigReply, error) {
	if req.GetId() <= 0 {
		return nil, errors.New("id不合法")
	}
	rows, err := dao.NewMCPConfig().FindByIDs(ctx, []int64{req.GetId()})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("MCP 配置不存在")
	}
	row := rows[0]
	if strings.EqualFold(strings.TrimSpace(row.Transport), "stdio") {
		cmd := strings.TrimSpace(row.StdioCommand)
		if cmd == "" {
			return &v1.TestMcpConfigReply{
				Ok:      false,
				Message: "stdio 模式须配置 stdio_command",
			}, nil
		}
		args, err := stdioArgsJSONToSlice(row.StdioArgsJSON)
		if err != nil {
			return &v1.TestMcpConfigReply{
				Ok:      false,
				Message: fmt.Sprintf("stdio_args_json 解析失败: %v", err),
			}, nil
		}
		env, err := stdioEnvJSONToStringMap(row.StdioEnvJSON)
		if err != nil {
			return &v1.TestMcpConfigReply{
				Ok:      false,
				Message: fmt.Sprintf("stdio_env_json 解析失败: %v", err),
			}, nil
		}
		// stdio 首次 npx/uv 拉依赖、冷启动常超过 60s；与 mcpbridge 长调用一致给足时间
		probeCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
		defer cancel()
		serverName, protoVer, tools, probeErr := mcpprobe.ProbeStdio(probeCtx, cmd, args, env)
		if probeErr != nil {
			return &v1.TestMcpConfigReply{
				Ok:      false,
				Message: probeErr.Error(),
			}, nil
		}
		return &v1.TestMcpConfigReply{
			Ok:      true,
			Message: fmt.Sprintf("stdio 子进程启动成功，共列出 %d 个工具", len(tools)),
			ToolNames:        tools,
			ServerName:       serverName,
			ProtocolVersion:  protoVer,
		}, nil
	}
	headers := headersJSONToStringMap(row.HeadersJSON)
	probeCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	serverName, protoVer, tools, probeErr := mcpprobe.Probe(probeCtx, row.Endpoint, headers)
	if probeErr != nil {
		return &v1.TestMcpConfigReply{
			Ok:      false,
			Message: probeErr.Error(),
		}, nil
	}
	return &v1.TestMcpConfigReply{
		Ok:               true,
		Message:          fmt.Sprintf("连接成功，共列出 %d 个工具", len(tools)),
		ToolNames:        tools,
		ServerName:       serverName,
		ProtocolVersion:  protoVer,
	}, nil
}

const (
	terminalMaxCommandLen = 8000
	terminalMaxOutputEach = 256 * 1024
	terminalExecTimeout   = 120 * time.Second
)

// RunTerminal 在服务端以 sh -c 执行命令；cwd 限制在 /app 与配置的 Agent 工作区内。
func (s *AdminService) RunTerminal(ctx context.Context, req *v1.RunTerminalRequest) (*v1.RunTerminalReply, error) {
	cmd := strings.TrimSpace(req.GetCommand())
	if cmd == "" {
		return nil, errors.New("command 不能为空")
	}
	if len(cmd) > terminalMaxCommandLen {
		return nil, fmt.Errorf("command 超过 %d 字符", terminalMaxCommandLen)
	}
	cwd, err := s.resolveTerminalCwd(req.GetCwd())
	if err != nil {
		return nil, err
	}
	execCtx, cancel := context.WithTimeout(ctx, terminalExecTimeout)
	defer cancel()
	c := exec.CommandContext(execCtx, "sh", "-c", cmd)
	c.Dir = cwd
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	runErr := c.Run()
	outStr := truncateTerminalOutput(stdout.String())
	errStr := truncateTerminalOutput(stderr.String())
	exit := int32(0)
	if runErr != nil {
		if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return &v1.RunTerminalReply{
				ExitCode:    -1,
				Stdout:      outStr,
				Stderr:      errStr,
				Diagnostic:  "命令执行超时（120s）",
			}, nil
		}
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			exit = int32(ee.ExitCode())
			return &v1.RunTerminalReply{ExitCode: exit, Stdout: outStr, Stderr: errStr}, nil
		}
		return &v1.RunTerminalReply{
			ExitCode:    -1,
			Stdout:      outStr,
			Stderr:      errStr,
			Diagnostic:  runErr.Error(),
		}, nil
	}
	return &v1.RunTerminalReply{ExitCode: exit, Stdout: outStr, Stderr: errStr}, nil
}

func (s *AdminService) terminalAllowedRoots() []string {
	seen := make(map[string]struct{})
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return
		}
		seen[filepath.Clean(abs)] = struct{}{}
	}
	add("/app")
	if s.config != nil && s.config.Agent != nil {
		add(s.config.Agent.WorkspaceRoot)
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

func (s *AdminService) resolveTerminalCwd(raw string) (string, error) {
	cwd := strings.TrimSpace(raw)
	if cwd == "" {
		cwd = "/app"
	}
	if !filepath.IsAbs(cwd) {
		return "", errors.New("cwd 须为绝对路径")
	}
	abs := filepath.Clean(cwd)
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("cwd 不可用: %w", err)
	}
	if !st.IsDir() {
		return "", errors.New("cwd 不是目录")
	}
	for _, root := range s.terminalAllowedRoots() {
		rc := filepath.Clean(root)
		if abs == rc || strings.HasPrefix(abs, rc+string(filepath.Separator)) {
			return abs, nil
		}
	}
	return "", errors.New("cwd 不在允许目录内（一般为 /app 或配置的 Agent 工作区）")
}

// ValidateTerminalCwd 校验终端工作目录（与 RunTerminal 白名单一致），供 WebSocket 交互式终端等复用。
func (s *AdminService) ValidateTerminalCwd(raw string) (string, error) {
	return s.resolveTerminalCwd(raw)
}

func truncateTerminalOutput(s string) string {
	if len(s) <= terminalMaxOutputEach {
		return s
	}
	return s[:terminalMaxOutputEach] + "\n...(truncated)"
}

func (s *AdminService) ListLlmConfigs(ctx context.Context, _ *v1.ListLlmConfigsRequest) (*v1.ListLlmConfigsReply, error) {
	configs, err := dao.NewLLMConfig().FindAll(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(configs))
	for _, c := range configs {
		ids = append(ids, c.ID)
	}
	entries, err := dao.NewLLMModelEntry().FindByConfigIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byCfg := make(map[int64][]dao.LLMModelEntry)
	for _, e := range entries {
		byCfg[e.ConfigID] = append(byCfg[e.ConfigID], e)
	}
	items := make([]*v1.LlmConfigItem, 0, len(configs))
	for _, c := range configs {
		items = append(items, toLlmConfigProto(c, byCfg[c.ID]))
	}
	return &v1.ListLlmConfigsReply{Items: items}, nil
}

func (s *AdminService) CreateLlmConfig(ctx context.Context, req *v1.CreateLlmConfigRequest) (*v1.LlmConfigItem, error) {
	if err := validateLlmConfigName(req.GetName()); err != nil {
		return nil, err
	}
	prov := strings.TrimSpace(req.GetProvider())
	if prov == "" {
		prov = "openai"
	}
	now := time.Now()
	var created dao.LLMConfig
	err := dao.Transaction(ctx, func(tx *gorm.DB) error {
		row := dao.LLMConfig{
			Name:        strings.TrimSpace(req.GetName()),
			Provider:    prov,
			BaseURL:     strings.TrimSpace(req.GetBaseUrl()),
			APIKey:      req.GetApiKey(),
			Enabled:     req.GetEnabled(),
			Description: strings.TrimSpace(req.GetDescription()),
			CreatedAt:   &now,
			UpdatedAt:   &now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		created = row
		seen := make(map[string]struct{})
		for _, d := range req.GetModels() {
			name := strings.TrimSpace(d.GetModelName())
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				return errors.New("模型列表中存在重复的 modelName: " + name)
			}
			seen[name] = struct{}{}
			ent := dao.LLMModelEntry{
				ConfigID:    row.ID,
				ModelName:   name,
				Description: strings.TrimSpace(d.GetDescription()),
				Enabled:     d.GetEnabled(),
				CreatedAt:   &now,
				UpdatedAt:   &now,
			}
			if err := tx.Create(&ent).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	list, err := dao.NewLLMModelEntry().FindByConfigIDs(ctx, []int64{created.ID})
	if err != nil {
		return nil, err
	}
	return toLlmConfigProto(created, list), nil
}

func (s *AdminService) UpdateLlmConfig(ctx context.Context, req *v1.UpdateLlmConfigRequest) (*v1.UpdateLlmConfigReply, error) {
	if req.GetId() <= 0 {
		return nil, errors.New("id不合法")
	}
	if err := validateLlmConfigName(req.GetName()); err != nil {
		return nil, err
	}
	prov := strings.TrimSpace(req.GetProvider())
	if prov == "" {
		prov = "openai"
	}
	data := map[string]interface{}{
		"name":        strings.TrimSpace(req.GetName()),
		"provider":    prov,
		"base_url":    strings.TrimSpace(req.GetBaseUrl()),
		"enabled":     req.GetEnabled(),
		"description": strings.TrimSpace(req.GetDescription()),
		"updated_at":  time.Now(),
	}
	if strings.TrimSpace(req.GetApiKey()) != "" {
		data["api_key"] = req.GetApiKey()
	}
	err := dao.NewLLMConfig().Updates(ctx, map[string]interface{}{"id": req.GetId()}, data)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateLlmConfigReply{}, nil
}

func (s *AdminService) DeleteLlmConfig(ctx context.Context, req *v1.DeleteLlmConfigRequest) (*v1.DeleteLlmConfigReply, error) {
	if req.GetId() <= 0 {
		return nil, errors.New("id不合法")
	}
	if err := dao.NewLLMConfig().Delete(ctx, map[string]interface{}{"id": req.GetId()}); err != nil {
		return nil, err
	}
	return &v1.DeleteLlmConfigReply{}, nil
}

func (s *AdminService) CreateLlmModelEntry(ctx context.Context, req *v1.CreateLlmModelEntryRequest) (*v1.LlmModelEntryItem, error) {
	if req.GetConfigId() <= 0 {
		return nil, errors.New("configId不合法")
	}
	if strings.TrimSpace(req.GetModelName()) == "" {
		return nil, errors.New("modelName不能为空")
	}
	if _, err := dao.NewLLMConfig().FindByID(ctx, req.GetConfigId()); err != nil {
		return nil, errors.New("配置不存在")
	}
	dup, err := dao.NewLLMModelEntry().CountByConfigAndModelName(ctx, req.GetConfigId(), strings.TrimSpace(req.GetModelName()), 0)
	if err != nil {
		return nil, err
	}
	if dup > 0 {
		return nil, errors.New("该配置下已存在同名模型")
	}
	now := time.Now()
	enabled := req.GetEnabled()
	row := dao.LLMModelEntry{
		ConfigID:    req.GetConfigId(),
		ModelName:   strings.TrimSpace(req.GetModelName()),
		Description: strings.TrimSpace(req.GetDescription()),
		Enabled:     enabled,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	if err := row.Create(ctx); err != nil {
		return nil, err
	}
	return toLlmModelEntryProto(row), nil
}

func (s *AdminService) UpdateLlmModelEntry(ctx context.Context, req *v1.UpdateLlmModelEntryRequest) (*v1.UpdateLlmModelEntryReply, error) {
	if req.GetId() <= 0 {
		return nil, errors.New("id不合法")
	}
	if strings.TrimSpace(req.GetModelName()) == "" {
		return nil, errors.New("modelName不能为空")
	}
	prev, err := dao.NewLLMModelEntry().FindByID(ctx, req.GetId())
	if err != nil {
		return nil, errors.New("模型不存在")
	}
	dup, err := dao.NewLLMModelEntry().CountByConfigAndModelName(ctx, prev.ConfigID, strings.TrimSpace(req.GetModelName()), req.GetId())
	if err != nil {
		return nil, err
	}
	if dup > 0 {
		return nil, errors.New("该配置下已存在同名模型")
	}
	err = dao.NewLLMModelEntry().Updates(ctx, map[string]interface{}{"id": req.GetId()}, map[string]interface{}{
		"model_name":  strings.TrimSpace(req.GetModelName()),
		"description": strings.TrimSpace(req.GetDescription()),
		"enabled":     req.GetEnabled(),
		"updated_at":  time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateLlmModelEntryReply{}, nil
}

func (s *AdminService) DeleteLlmModelEntry(ctx context.Context, req *v1.DeleteLlmModelEntryRequest) (*v1.DeleteLlmModelEntryReply, error) {
	if req.GetId() <= 0 {
		return nil, errors.New("id不合法")
	}
	if err := dao.NewLLMModelEntry().Delete(ctx, map[string]interface{}{"id": req.GetId()}); err != nil {
		return nil, err
	}
	return &v1.DeleteLlmModelEntryReply{}, nil
}

func validateLlmConfigName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name不能为空")
	}
	return nil
}

func toLlmModelEntryProto(it dao.LLMModelEntry) *v1.LlmModelEntryItem {
	createdAt := ""
	updatedAt := ""
	if it.CreatedAt != nil {
		createdAt = it.CreatedAt.Format(time.RFC3339Nano)
	}
	if it.UpdatedAt != nil {
		updatedAt = it.UpdatedAt.Format(time.RFC3339Nano)
	}
	return &v1.LlmModelEntryItem{
		Id:          it.ID,
		ConfigId:    it.ConfigID,
		ModelName:   it.ModelName,
		Description: it.Description,
		Enabled:     it.Enabled,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func toLlmConfigProto(c dao.LLMConfig, entries []dao.LLMModelEntry) *v1.LlmConfigItem {
	createdAt := ""
	updatedAt := ""
	if c.CreatedAt != nil {
		createdAt = c.CreatedAt.Format(time.RFC3339Nano)
	}
	if c.UpdatedAt != nil {
		updatedAt = c.UpdatedAt.Format(time.RFC3339Nano)
	}
	models := make([]*v1.LlmModelEntryItem, 0, len(entries))
	for _, e := range entries {
		models = append(models, toLlmModelEntryProto(e))
	}
	return &v1.LlmConfigItem{
		Id:          c.ID,
		Name:        c.Name,
		Provider:    c.Provider,
		BaseUrl:     c.BaseURL,
		Enabled:     c.Enabled,
		ApiKey:      c.APIKey,
		Description: c.Description,
		Models:      models,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func normalizeMCPPayload(p *mcpConfigPayload) {
	if strings.TrimSpace(p.Transport) == "" {
		p.Transport = "http"
	}
	p.Transport = strings.ToLower(strings.TrimSpace(p.Transport))
	if p.Transport == "stdio" {
		if strings.TrimSpace(p.StdioArgsJSON) == "" {
			p.StdioArgsJSON = "[]"
		}
		if strings.TrimSpace(p.StdioEnvJSON) == "" {
			p.StdioEnvJSON = "{}"
		}
	}
}

func validateMCPPayload(req mcpConfigPayload) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name不能为空")
	}
	if strings.TrimSpace(req.Server) == "" {
		return errors.New("server不能为空")
	}
	switch req.Transport {
	case "http":
		if strings.TrimSpace(req.Endpoint) == "" {
			return errors.New("http 模式 endpoint 不能为空")
		}
	case "stdio":
		if strings.TrimSpace(req.StdioCommand) == "" {
			return errors.New("stdio 模式 stdio_command 不能为空")
		}
		var args []string
		if err := json.Unmarshal([]byte(req.StdioArgsJSON), &args); err != nil {
			return fmt.Errorf("stdio_args_json 须为 JSON 字符串数组: %w", err)
		}
		var envObj map[string]interface{}
		if err := json.Unmarshal([]byte(req.StdioEnvJSON), &envObj); err != nil {
			return fmt.Errorf("stdio_env_json 须为 JSON 对象: %w", err)
		}
	default:
		return errors.New("transport 须为 http 或 stdio")
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
		Id:            it.ID,
		Name:          it.Name,
		Server:        it.Server,
		Endpoint:      it.Endpoint,
		Headers:       parseHeadersToStruct(it.HeadersJSON),
		Enabled:       it.Enabled,
		Description:   it.Description,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		Transport:     it.Transport,
		StdioCommand:  it.StdioCommand,
		StdioArgsJson: it.StdioArgsJSON,
		StdioEnvJson:  it.StdioEnvJSON,
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

func stdioArgsJSONToSlice(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func stdioEnvJSONToStringMap(raw string) (map[string]string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(obj))
	for k, v := range obj {
		if s, ok := v.(string); ok {
			out[k] = s
		} else {
			out[k] = fmt.Sprint(v)
		}
	}
	return out, nil
}

func headersJSONToStringMap(raw string) map[string]string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return nil
	}
	out := make(map[string]string, len(obj))
	for k, v := range obj {
		if s, ok := v.(string); ok {
			out[k] = s
		} else {
			out[k] = fmt.Sprint(v)
		}
	}
	return out
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
