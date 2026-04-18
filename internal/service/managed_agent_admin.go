package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/data/dao"
	"strings"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

const (
	modelScopeAll      = "all"
	modelScopeExplicit = "explicit"
)

type managedAgentDTO struct {
	ID               int64    `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	SystemPrompt     string   `json:"systemPrompt"`
	SkillPackageIDs  []string `json:"skillPackageIds"`
	McpIDs           []int64  `json:"mcpIds"`
	LLMConfigID      int64    `json:"llmConfigId"`
	ModelScope       string   `json:"modelScope"`
	ModelEntryIDs    []int64  `json:"modelEntryIds"`
	Enabled          bool     `json:"enabled"`
	CreatedAt        string   `json:"createdAt,omitempty"`
	UpdatedAt        string   `json:"updatedAt,omitempty"`
}

type managedAgentWriteReq struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	SystemPrompt      string   `json:"systemPrompt"`
	SkillPackageIDs   []string `json:"skillPackageIds"`
	McpIDs            []int64  `json:"mcpIds"`
	LLMConfigID       int64    `json:"llmConfigId"`
	ModelScope        string   `json:"modelScope"`
	ModelEntryIDs     []int64  `json:"modelEntryIds"`
	Enabled           bool     `json:"enabled"`
}

// RegisterManagedAgentHTTPRoutes 注册可编排 Agent（Managed Agent）JSON API。
func RegisterManagedAgentHTTPRoutes(s *khttp.Server, admin *AdminService) {
	r := s.Route("/api/v1/admin/managed-agents")
	r.GET("", admin.listManagedAgents)
	r.GET("/{id}", admin.getManagedAgent)
	r.POST("", admin.createManagedAgent)
	r.PUT("/{id}", admin.updateManagedAgent)
	r.DELETE("/{id}", admin.deleteManagedAgent)

	p := s.Route("/api/v1/admin/skill-packages")
	p.GET("", admin.listSkillPackagesHTTP)
}

func (s *AdminService) listManagedAgents(ctx khttp.Context) error {
	c := ctx.Request().Context()
	list, err := dao.NewManagedAgent().FindAll(c)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	out := make([]managedAgentDTO, 0, len(list))
	for _, row := range list {
		dto, err := managedAgentToDTO(&row)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		out = append(out, *dto)
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{"items": out})
}

func (s *AdminService) getManagedAgent(ctx khttp.Context) error {
	var path struct {
		// BindVars 使用 form 解码器，默认读取 json struct tag（与 path tag 无关）
		ID int64 `json:"id"`
	}
	if err := ctx.BindVars(&path); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	row, err := dao.NewManagedAgent().FindByID(ctx.Request().Context(), path.ID)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	dto, err := managedAgentToDTO(row)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{"item": dto})
}

func (s *AdminService) createManagedAgent(ctx khttp.Context) error {
	var req managedAgentWriteReq
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := s.validateManagedAgentPayload(ctx.Request().Context(), &req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	now := time.Now()
	row := &dao.ManagedAgent{
		Name:         strings.TrimSpace(req.Name),
		Description:  strings.TrimSpace(req.Description),
		SystemPrompt: req.SystemPrompt,
		LLMConfigID:  req.LLMConfigID,
		ModelScope:   normalizeModelScope(req.ModelScope),
		Enabled:      req.Enabled,
		CreatedAt:    &now,
		UpdatedAt:    &now,
	}
	var err error
	row.SkillPathsJSON, err = marshalJSON(biz.NormalizeStoredSkillPackageIDs(req.SkillPackageIDs))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	row.McpIDsJSON, err = marshalJSON(req.McpIDs)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	row.ModelEntryIDsJSON, err = marshalJSON(filterEntryIDs(req.ModelScope, req.ModelEntryIDs))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := row.Create(ctx.Request().Context()); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	dto, err := managedAgentToDTO(row)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusCreated, map[string]interface{}{"item": dto})
}

func (s *AdminService) updateManagedAgent(ctx khttp.Context) error {
	var path struct {
		ID int64 `json:"id"`
	}
	if err := ctx.BindVars(&path); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if path.ID <= 0 {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "无效的 id"})
	}
	if _, err := dao.NewManagedAgent().FindByID(ctx.Request().Context(), path.ID); err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Agent 配置不存在（id=%d）。请先在「Agent 管理 → Agent 配置」新建并保存，或刷新列表后重试。", path.ID),
		})
	}
	var req managedAgentWriteReq
	if err := json.NewDecoder(ctx.Request().Body).Decode(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := s.validateManagedAgentPayload(ctx.Request().Context(), &req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	sp, err := marshalJSON(biz.NormalizeStoredSkillPackageIDs(req.SkillPackageIDs))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	mp, err := marshalJSON(req.McpIDs)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	me, err := marshalJSON(filterEntryIDs(req.ModelScope, req.ModelEntryIDs))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	data := map[string]interface{}{
		"name":              strings.TrimSpace(req.Name),
		"description":       strings.TrimSpace(req.Description),
		"system_prompt":     req.SystemPrompt,
		"skill_paths":       sp,
		"mcp_ids":           mp,
		"llm_config_id":     req.LLMConfigID,
		"model_scope":       normalizeModelScope(req.ModelScope),
		"model_entry_ids":   me,
		"enabled":           req.Enabled,
		"updated_at":        time.Now(),
	}
	if err := dao.NewManagedAgent().Updates(ctx.Request().Context(), map[string]interface{}{"id": path.ID}, data); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	row, err := dao.NewManagedAgent().FindByID(ctx.Request().Context(), path.ID)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	dto, err := managedAgentToDTO(row)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{"item": dto})
}

func (s *AdminService) deleteManagedAgent(ctx khttp.Context) error {
	var path struct {
		ID int64 `json:"id"`
	}
	if err := ctx.BindVars(&path); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if path.ID <= 0 {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "无效的 id"})
	}
	if s.poolSvc != nil {
		refs, err := s.poolSvc.PoolsReferencingManagedAgent(ctx.Request().Context(), path.ID)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		if len(refs) > 0 {
			parts := make([]string, len(refs))
			for i, r := range refs {
				parts[i] = fmt.Sprintf("%s（%s）", r.Name, r.ID)
			}
			return ctx.JSON(http.StatusConflict, map[string]string{
				"error": fmt.Sprintf(
					"无法删除：该 Agent 配置仍被 Agent Playground 中的 Agent 池引用：%s。请先删除或调整相关 Agent 池后再删除。",
					strings.Join(parts, "、"),
				),
			})
		}
	}
	if err := dao.NewManagedAgent().Delete(ctx.Request().Context(), map[string]interface{}{"id": path.ID}); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, map[string]string{"ok": "true"})
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

func (s *AdminService) listSkillPackagesHTTP(ctx khttp.Context) error {
	items, err := discoverSkillPackages(s.skillRoot)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"root":  s.skillRoot,
		"items": items,
	})
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
		LLMConfigID:   row.LLMConfigID,
		ModelScope:    row.ModelScope,
		ModelEntryIDs: entryIDs,
		Enabled:       row.Enabled,
	}
	if row.CreatedAt != nil {
		dto.CreatedAt = row.CreatedAt.Format(time.RFC3339)
	}
	if row.UpdatedAt != nil {
		dto.UpdatedAt = row.UpdatedAt.Format(time.RFC3339)
	}
	return dto, nil
}
