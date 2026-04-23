package service

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/data/dao"
	"strings"
	"time"
)

const (
	modelScopeAll      = "all"
	modelScopeExplicit = "explicit"
)

type managedAgentDTO struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	SystemPrompt    string   `json:"systemPrompt"`
	SkillPackageIDs []string `json:"skillPackageIds"`
	McpIDs          []int64  `json:"mcpIds"`
	LLMConfigID     int64    `json:"llmConfigId"`
	ModelScope      string   `json:"modelScope"`
	ModelEntryIDs   []int64  `json:"modelEntryIds"`
	Enabled         bool     `json:"enabled"`
	CreatedAt       string   `json:"createdAt,omitempty"`
	UpdatedAt       string   `json:"updatedAt,omitempty"`
}

type managedAgentWriteReq struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	SystemPrompt    string   `json:"systemPrompt"`
	SkillPackageIDs []string `json:"skillPackageIds"`
	McpIDs          []int64  `json:"mcpIds"`
	LLMConfigID     int64    `json:"llmConfigId"`
	ModelScope      string   `json:"modelScope"`
	ModelEntryIDs   []int64  `json:"modelEntryIds"`
	Enabled         bool     `json:"enabled"`
}

func (s *AdminService) ListManagedAgents(ctx context.Context, _ *v1.ListManagedAgentsRequest) (*v1.ListManagedAgentsReply, error) {
	list, err := dao.NewManagedAgent().FindAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.ManagedAgentItem, 0, len(list))
	for _, row := range list {
		dto, err := managedAgentToDTO(&row)
		if err != nil {
			return nil, err
		}
		out = append(out, managedAgentDTOToProto(dto))
	}
	return &v1.ListManagedAgentsReply{Items: out}, nil
}

func (s *AdminService) GetManagedAgent(ctx context.Context, req *v1.GetManagedAgentRequest) (*v1.GetManagedAgentReply, error) {
	row, err := dao.NewManagedAgent().FindByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	dto, err := managedAgentToDTO(row)
	if err != nil {
		return nil, err
	}
	return &v1.GetManagedAgentReply{Item: managedAgentDTOToProto(dto)}, nil
}

func (s *AdminService) CreateManagedAgent(ctx context.Context, req *v1.CreateManagedAgentRequest) (*v1.CreateManagedAgentReply, error) {
	writeReq := managedAgentWriteReqFromCreate(req)
	if err := s.validateManagedAgentPayload(ctx, &writeReq); err != nil {
		return nil, err
	}
	now := time.Now()
	row := &dao.ManagedAgent{
		Name:         strings.TrimSpace(writeReq.Name),
		Description:  strings.TrimSpace(writeReq.Description),
		SystemPrompt: writeReq.SystemPrompt,
		LLMConfigID:  writeReq.LLMConfigID,
		ModelScope:   normalizeModelScope(writeReq.ModelScope),
		Enabled:      writeReq.Enabled,
		CreatedAt:    &now,
		UpdatedAt:    &now,
	}
	var err error
	row.SkillPathsJSON, err = marshalJSON(biz.NormalizeStoredSkillPackageIDs(writeReq.SkillPackageIDs))
	if err != nil {
		return nil, err
	}
	row.McpIDsJSON, err = marshalJSON(writeReq.McpIDs)
	if err != nil {
		return nil, err
	}
	row.ModelEntryIDsJSON, err = marshalJSON(filterEntryIDs(writeReq.ModelScope, writeReq.ModelEntryIDs))
	if err != nil {
		return nil, err
	}
	if err := row.Create(ctx); err != nil {
		return nil, err
	}
	dto, err := managedAgentToDTO(row)
	if err != nil {
		return nil, err
	}
	return &v1.CreateManagedAgentReply{Item: managedAgentDTOToProto(dto)}, nil
}

func (s *AdminService) UpdateManagedAgent(ctx context.Context, req *v1.UpdateManagedAgentRequest) (*v1.UpdateManagedAgentReply, error) {
	if req.GetId() <= 0 {
		return nil, fmt.Errorf("无效的 id")
	}
	if _, err := dao.NewManagedAgent().FindByID(ctx, req.GetId()); err != nil {
		return nil, fmt.Errorf("Agent 配置不存在（id=%d）。请先在「Agent 管理 → Agent 配置」新建并保存，或刷新列表后重试。", req.GetId())
	}
	writeReq := managedAgentWriteReqFromUpdate(req)
	if err := s.validateManagedAgentPayload(ctx, &writeReq); err != nil {
		return nil, err
	}
	sp, err := marshalJSON(biz.NormalizeStoredSkillPackageIDs(writeReq.SkillPackageIDs))
	if err != nil {
		return nil, err
	}
	mp, err := marshalJSON(writeReq.McpIDs)
	if err != nil {
		return nil, err
	}
	me, err := marshalJSON(filterEntryIDs(writeReq.ModelScope, writeReq.ModelEntryIDs))
	if err != nil {
		return nil, err
	}
	data := map[string]interface{}{
		"name":            strings.TrimSpace(writeReq.Name),
		"description":     strings.TrimSpace(writeReq.Description),
		"system_prompt":   writeReq.SystemPrompt,
		"skill_paths":     sp,
		"mcp_ids":         mp,
		"llm_config_id":   writeReq.LLMConfigID,
		"model_scope":     normalizeModelScope(writeReq.ModelScope),
		"model_entry_ids": me,
		"enabled":         writeReq.Enabled,
		"updated_at":      time.Now(),
	}
	if err := dao.NewManagedAgent().Updates(ctx, map[string]interface{}{"id": req.GetId()}, data); err != nil {
		return nil, err
	}
	row, err := dao.NewManagedAgent().FindByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	dto, err := managedAgentToDTO(row)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateManagedAgentReply{Item: managedAgentDTOToProto(dto)}, nil
}

func (s *AdminService) DeleteManagedAgent(ctx context.Context, req *v1.DeleteManagedAgentRequest) (*v1.DeleteManagedAgentReply, error) {
	if req.GetId() <= 0 {
		return nil, fmt.Errorf("无效的 id")
	}
	if s.poolSvc != nil {
		refs, err := s.poolSvc.PoolsReferencingManagedAgent(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		if len(refs) > 0 {
			parts := make([]string, len(refs))
			for i, r := range refs {
				parts[i] = fmt.Sprintf("%s（%s）", r.Name, r.ID)
			}
			return nil, fmt.Errorf("无法删除：该 Agent 配置仍被 Agent Playground 中的 Agent 池引用：%s。请先删除或调整相关 Agent 池后再删除。", strings.Join(parts, "、"))
		}
	}
	if err := dao.NewManagedAgent().Delete(ctx, map[string]interface{}{"id": req.GetId()}); err != nil {
		return nil, err
	}
	return &v1.DeleteManagedAgentReply{Ok: true}, nil
}

func (s *AdminService) validateManagedAgentPayload(ctx context.Context, req *managedAgentWriteReq) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name 不能为空")
	}
	if req.LLMConfigID <= 0 {
		return fmt.Errorf("请选择 LLM 配置（站点）")
	}
	if _, err := dao.NewLLMConfig().FindByID(ctx, req.LLMConfigID); err != nil {
		return fmt.Errorf("LLM 配置不存在")
	}
	scope := normalizeModelScope(req.ModelScope)
	if scope == modelScopeExplicit {
		if len(req.ModelEntryIDs) == 0 {
			return fmt.Errorf("指定模型时至少选择一个模型条目")
		}
		entries, err := dao.NewLLMModelEntry().FindByConfigIDs(ctx, []int64{req.LLMConfigID})
		if err != nil {
			return err
		}
		allowed := make(map[int64]struct{}, len(entries))
		for _, e := range entries {
			allowed[e.ID] = struct{}{}
		}
		for _, id := range req.ModelEntryIDs {
			if _, ok := allowed[id]; !ok {
				return fmt.Errorf("模型条目 %d 不属于所选 LLM 配置", id)
			}
		}
	}
	if err := validateMcpIDsExist(ctx, req.McpIDs); err != nil {
		return err
	}
	return s.validateSkillPackageIDs(req.SkillPackageIDs)
}

func (s *AdminService) validateSkillPackageIDs(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	valid, err := discoverSkillPackageSet(s.skillRoot)
	if err != nil {
		return fmt.Errorf("扫描技能目录失败: %w", err)
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := valid[id]; !ok {
			return fmt.Errorf("未知技能包: %s", id)
		}
	}
	return nil
}

func (s *AdminService) ListSkillPackages(_ context.Context, _ *v1.ListSkillPackagesRequest) (*v1.ListSkillPackagesReply, error) {
	items, err := discoverSkillPackages(s.skillRoot)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.SkillPackageItem, 0, len(items))
	for _, it := range items {
		out = append(out, &v1.SkillPackageItem{
			Id:             it.ID,
			SkillFileCount: int64(it.SkillFileCount),
		})
	}
	return &v1.ListSkillPackagesReply{
		Root:  s.skillRoot,
		Items: out,
	}, nil
}

func validateMcpIDsExist(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	all, err := dao.NewMCPConfig().FindAll(ctx)
	if err != nil {
		return err
	}
	ok := make(map[int64]struct{}, len(all))
	for _, m := range all {
		ok[m.ID] = struct{}{}
	}
	for _, id := range ids {
		if _, exists := ok[id]; !exists {
			return fmt.Errorf("MCP 配置 id=%d 不存在", id)
		}
	}
	return nil
}

func normalizeModelScope(s string) string {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case modelScopeExplicit:
		return modelScopeExplicit
	default:
		return modelScopeAll
	}
}

func filterEntryIDs(scope string, ids []int64) []int64 {
	if normalizeModelScope(scope) != modelScopeExplicit {
		return nil
	}
	return ids
}

func marshalJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func managedAgentToDTO(row *dao.ManagedAgent) (*managedAgentDTO, error) {
	var raw []string
	if row.SkillPathsJSON != "" {
		_ = json.Unmarshal([]byte(row.SkillPathsJSON), &raw)
	}
	pkgs := biz.NormalizeStoredSkillPackageIDs(raw)
	var mcpIDs []int64
	if row.McpIDsJSON != "" {
		_ = json.Unmarshal([]byte(row.McpIDsJSON), &mcpIDs)
	}
	var entryIDs []int64
	if row.ModelEntryIDsJSON != "" {
		_ = json.Unmarshal([]byte(row.ModelEntryIDsJSON), &entryIDs)
	}
	dto := &managedAgentDTO{
		ID:              row.ID,
		Name:            row.Name,
		Description:     row.Description,
		SystemPrompt:    row.SystemPrompt,
		SkillPackageIDs: pkgs,
		McpIDs:          mcpIDs,
		LLMConfigID:     row.LLMConfigID,
		ModelScope:      row.ModelScope,
		ModelEntryIDs:   entryIDs,
		Enabled:         row.Enabled,
	}
	if row.CreatedAt != nil {
		dto.CreatedAt = row.CreatedAt.Format(time.RFC3339)
	}
	if row.UpdatedAt != nil {
		dto.UpdatedAt = row.UpdatedAt.Format(time.RFC3339)
	}
	return dto, nil
}

func managedAgentDTOToProto(in *managedAgentDTO) *v1.ManagedAgentItem {
	if in == nil {
		return nil
	}
	return &v1.ManagedAgentItem{
		Id:              in.ID,
		Name:            in.Name,
		Description:     in.Description,
		SystemPrompt:    in.SystemPrompt,
		SkillPackageIds: in.SkillPackageIDs,
		McpIds:          in.McpIDs,
		LlmConfigId:     in.LLMConfigID,
		ModelScope:      in.ModelScope,
		ModelEntryIds:   in.ModelEntryIDs,
		Enabled:         in.Enabled,
		CreatedAt:       in.CreatedAt,
		UpdatedAt:       in.UpdatedAt,
	}
}

func managedAgentWriteReqFromCreate(req *v1.CreateManagedAgentRequest) managedAgentWriteReq {
	return managedAgentWriteReq{
		Name:            req.GetName(),
		Description:     req.GetDescription(),
		SystemPrompt:    req.GetSystemPrompt(),
		SkillPackageIDs: req.GetSkillPackageIds(),
		McpIDs:          req.GetMcpIds(),
		LLMConfigID:     req.GetLlmConfigId(),
		ModelScope:      req.GetModelScope(),
		ModelEntryIDs:   req.GetModelEntryIds(),
		Enabled:         req.GetEnabled(),
	}
}

func managedAgentWriteReqFromUpdate(req *v1.UpdateManagedAgentRequest) managedAgentWriteReq {
	return managedAgentWriteReq{
		Name:            req.GetName(),
		Description:     req.GetDescription(),
		SystemPrompt:    req.GetSystemPrompt(),
		SkillPackageIDs: req.GetSkillPackageIds(),
		McpIDs:          req.GetMcpIds(),
		LLMConfigID:     req.GetLlmConfigId(),
		ModelScope:      req.GetModelScope(),
		ModelEntryIDs:   req.GetModelEntryIds(),
		Enabled:         req.GetEnabled(),
	}
}
