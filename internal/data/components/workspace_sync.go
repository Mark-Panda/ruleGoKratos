package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
	"ruleGoKratos/internal/service"
)

func init() {
	_ = rulego.Registry.Register(&WorkspaceSyncComponent{})
}

const workspaceSyncTimeout = 11 * time.Minute

// WorkspaceSyncComponent 同步「工作区管理」中已配置工作区的磁盘仓库（与 Admin 同步仓库一致）。
type WorkspaceSyncComponent struct {
	Config WorkspaceSyncConfiguration

	workspaceIDTpl el.Template
	hasVar         bool
}

// WorkspaceSyncConfiguration 节点配置（与画布 DSL configuration 字段对齐）。
type WorkspaceSyncConfiguration struct {
	WorkspaceID string `json:"workspaceId"`
	ReplaceData bool   `json:"replaceData"`
}

func (c *WorkspaceSyncComponent) New() types.Node {
	return &WorkspaceSyncComponent{Config: WorkspaceSyncConfiguration{
		ReplaceData: true,
	}}
}

func (c *WorkspaceSyncComponent) Type() string {
	return "x/workspaceSync"
}

func (c *WorkspaceSyncComponent) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "工作区刷新",
		Desc:  "同步指定工作区下的全部 Git 仓库（git pull --ff-only / clone），并更新工作区元数据。",
	}
}

func (c *WorkspaceSyncComponent) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}
	tpl, err := el.NewTemplate(c.Config.WorkspaceID)
	if err != nil {
		return err
	}
	c.workspaceIDTpl = tpl
	if tpl.HasVar() {
		c.hasVar = true
	}
	return nil
}

func (c *WorkspaceSyncComponent) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	var evn map[string]interface{}
	if c.hasVar {
		evn = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	}
	if evn == nil {
		evn = map[string]interface{}{}
	}
	id := strings.TrimSpace(c.workspaceIDTpl.ExecuteAsString(evn))
	if id == "" {
		ctx.TellFailure(msg, errors.New("workspaceSync: 请选择或填写工作区 ID"))
		return
	}
	runCtx, cancel := context.WithTimeout(context.Background(), workspaceSyncTimeout)
	defer cancel()
	if err := service.RunWorkspaceRepoSync(runCtx, id); err != nil {
		ctx.TellFailure(msg, fmt.Errorf("workspaceSync: %w", err))
		return
	}
	if c.Config.ReplaceData {
		payload := map[string]interface{}{
			"ok":          true,
			"workspaceId": id,
			"message":     "仓库同步完成",
		}
		if raw, err := json.Marshal(payload); err == nil {
			msg.SetData(string(raw))
		}
	}
	ctx.TellSuccess(msg)
}

func (c *WorkspaceSyncComponent) Destroy() {}
