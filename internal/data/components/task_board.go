package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
	"ruleGoKratos/internal/biz"
)

func init() {
	_ = rulego.Registry.Register(&TaskBoardComponent{})
}

// TaskBoardComponent 任务看板组件
type TaskBoardComponent struct {
	Config TaskBoardConfiguration

	actionTpl        el.Template
	nameTpl          el.Template
	handlerUserIDTpl el.Template
	descriptionTpl   el.Template
	priorityTpl      el.Template
	taskTypeTpl      el.Template
	taskIDTpl        el.Template
	statusTpl        el.Template
	ruleChainIDTpl   el.Template
	parentIDTpl      el.Template
	clearRuleChainIDTpl el.Template
	hasVar           bool
}

type TaskBoardConfiguration struct {
	// 操作类型: create, get, update, delete
	Action string `json:"action"`
	// 任务名称
	Name string `json:"name"`
	// 优先级
	Priority int `json:"priority"`
	// 任务类型
	TaskType int `json:"taskType"`
	// 处理人用户ID
	HandlerUserID string `json:"handlerUserId"`
	// 任务描述
	Description string `json:"description"`
	// 任务ID（用于get/update/delete）
	TaskID int64 `json:"taskId"`
	// 状态（用于update）
	Status int `json:"status"`
	// 关联的规则链ID
	RuleChainID string `json:"ruleChainId"`
	// 父任务ID
	ParentID int64 `json:"parentId"`
	// 清除规则链关联
	ClearRuleChainID bool `json:"clearRuleChainId"`
	// 替换数据
	ReplaceData bool `json:"replaceData"`
}

func (c *TaskBoardComponent) New() types.Node {
	return &TaskBoardComponent{Config: TaskBoardConfiguration{
		ReplaceData: true,
	}}
}

func (c *TaskBoardComponent) Type() string {
	return "x/taskBoard"
}

func (c *TaskBoardComponent) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "任务看板",
		Desc:  "创建和管理任务看板中的任务",
	}
}

func (c *TaskBoardComponent) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}

	tpls := map[*el.Template]string{
		&c.actionTpl:        c.Config.Action,
		&c.nameTpl:          c.Config.Name,
		&c.handlerUserIDTpl: c.Config.HandlerUserID,
		&c.descriptionTpl:   c.Config.Description,
		&c.priorityTpl:      strconv.Itoa(c.Config.Priority),
		&c.taskTypeTpl:      strconv.Itoa(c.Config.TaskType),
		&c.taskIDTpl:        strconv.FormatInt(c.Config.TaskID, 10),
		&c.statusTpl:        strconv.Itoa(c.Config.Status),
		&c.ruleChainIDTpl:   c.Config.RuleChainID,
		&c.parentIDTpl:      strconv.FormatInt(c.Config.ParentID, 10),
		&c.clearRuleChainIDTpl: strconv.FormatBool(c.Config.ClearRuleChainID),
	}

	for tpl, s := range tpls {
		t, err := el.NewTemplate(s)
		if err != nil {
			return err
		}
		*tpl = t
		if t.HasVar() {
			c.hasVar = true
		}
	}

	return nil
}

func (c *TaskBoardComponent) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	evn := base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	if evn == nil {
		evn = map[string]interface{}{}
	}

	action := strings.TrimSpace(c.actionTpl.ExecuteAsString(evn))
	if action == "" {
		action = "create"
	}

	taskUsecase := globalTaskUsecase
	if taskUsecase == nil {
		ctx.TellFailure(msg, errors.New("taskBoard: 无法获取任务看板服务"))
		return
	}

	var result interface{}
	var err error

	switch action {
	case "create":
		result, err = c.doCreate(ctx, evn, taskUsecase)
	case "get":
		result, err = c.doGet(ctx, evn, taskUsecase)
	case "update":
		result, err = c.doUpdate(ctx, evn, taskUsecase)
	case "delete":
		result, err = c.doDelete(ctx, evn, taskUsecase)
	default:
		err = fmt.Errorf("不支持的操作: %s", action)
	}

	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}

	resultBytes, _ := json.Marshal(result)
	msg.SetData(string(resultBytes))
	ctx.TellSuccess(msg)
}

func (c *TaskBoardComponent) doCreate(ctx types.RuleContext, evn map[string]interface{}, usecase *biz.TaskBoardUsecase) (interface{}, error) {
	name := strings.TrimSpace(c.nameTpl.ExecuteAsString(evn))
	if name == "" {
		return nil, errors.New("任务名称不能为空")
	}

	priorityStr := c.priorityTpl.ExecuteAsString(evn)
	priority := int32(c.Config.Priority)
	if p, err := strconv.Atoi(strings.TrimSpace(priorityStr)); err == nil {
		priority = int32(p)
	}

	taskTypeStr := c.taskTypeTpl.ExecuteAsString(evn)
	taskType := int32(c.Config.TaskType)
	if t, err := strconv.Atoi(strings.TrimSpace(taskTypeStr)); err == nil {
		taskType = int32(t)
	}

	handlerUserID := strings.TrimSpace(c.handlerUserIDTpl.ExecuteAsString(evn))
	description := strings.TrimSpace(c.descriptionTpl.ExecuteAsString(evn))
	ruleChainID := strings.TrimSpace(c.ruleChainIDTpl.ExecuteAsString(evn))

	task, err := usecase.CreateTask(context.Background(), name, priority, taskType, handlerUserID, description, ruleChainID)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"success": true,
		"task": map[string]interface{}{
			"id":            task.ID,
			"name":          task.Name,
			"priority":      task.Priority,
			"status":        task.Status,
			"type":          task.Type,
			"description":   task.Description,
			"rule_chain_id": task.RuleChainID,
			"last_run_id":   task.LastRunID,
			"created_at":    task.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}
	if task.ParentID != nil {
		result["task"].(map[string]interface{})["parent_id"] = *task.ParentID
	}
	return result, nil
}

func (c *TaskBoardComponent) doGet(ctx types.RuleContext, evn map[string]interface{}, usecase *biz.TaskBoardUsecase) (interface{}, error) {
	taskIDStr := c.taskIDTpl.ExecuteAsString(evn)
	taskID, _ := strconv.ParseInt(strings.TrimSpace(taskIDStr), 10, 64)
	if taskID <= 0 {
		taskID = c.Config.TaskID
	}

	if taskID <= 0 {
		return nil, errors.New("任务ID不能为空")
	}

	task, err := usecase.GetTask(context.Background(), taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return map[string]interface{}{"success": false, "error": "任务不存在"}, nil
	}

	result := map[string]interface{}{
		"success": true,
		"task": map[string]interface{}{
			"id":            task.ID,
			"name":          task.Name,
			"priority":      task.Priority,
			"status":        task.Status,
			"type":          task.Type,
			"description":   task.Description,
			"rule_chain_id": task.RuleChainID,
			"last_run_id":   task.LastRunID,
			"created_at":    task.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":    task.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	}
	if task.ParentID != nil {
		result["task"].(map[string]interface{})["parent_id"] = *task.ParentID
	}
	return result, nil
}

func (c *TaskBoardComponent) doUpdate(ctx types.RuleContext, evn map[string]interface{}, usecase *biz.TaskBoardUsecase) (interface{}, error) {
	taskIDStr := c.taskIDTpl.ExecuteAsString(evn)
	taskID, _ := strconv.ParseInt(strings.TrimSpace(taskIDStr), 10, 64)
	if taskID <= 0 {
		taskID = c.Config.TaskID
	}

	if taskID <= 0 {
		return nil, errors.New("任务ID不能为空")
	}

	var (
		name          *string
		priority      *int32
		status       *int32
		handlerUserID *string
		description   *string
	)

	if n := strings.TrimSpace(c.nameTpl.ExecuteAsString(evn)); n != "" {
		name = &n
	}

	priorityStr := c.priorityTpl.ExecuteAsString(evn)
	if p, err := strconv.Atoi(strings.TrimSpace(priorityStr)); err == nil && p > 0 {
		pr := int32(p)
		priority = &pr
	}

	statusStr := c.statusTpl.ExecuteAsString(evn)
	if s, err := strconv.Atoi(strings.TrimSpace(statusStr)); err == nil && s > 0 {
		st := int32(s)
		status = &st
	}

	if uid := strings.TrimSpace(c.handlerUserIDTpl.ExecuteAsString(evn)); uid != "" {
		handlerUserID = &uid
	}
	if desc := strings.TrimSpace(c.descriptionTpl.ExecuteAsString(evn)); desc != "" {
		description = &desc
	}

	var ruleChainID *string
	rc := strings.TrimSpace(c.ruleChainIDTpl.ExecuteAsString(evn))
	if rc != "" {
		ruleChainID = &rc
	} else if strings.TrimSpace(c.clearRuleChainIDTpl.ExecuteAsString(evn)) == "true" {
		empty := ""
		ruleChainID = &empty
	}

	task, err := usecase.UpdateTask(context.Background(), taskID, name, priority, status, handlerUserID, description, ruleChainID, nil)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"task": map[string]interface{}{
			"id":            task.ID,
			"name":          task.Name,
			"priority":      task.Priority,
			"status":        task.Status,
			"type":          task.Type,
			"description":   task.Description,
			"rule_chain_id": task.RuleChainID,
			"last_run_id":   task.LastRunID,
			"updated_at":    task.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

func (c *TaskBoardComponent) doDelete(ctx types.RuleContext, evn map[string]interface{}, usecase *biz.TaskBoardUsecase) (interface{}, error) {
	taskIDStr := c.taskIDTpl.ExecuteAsString(evn)
	taskID, _ := strconv.ParseInt(strings.TrimSpace(taskIDStr), 10, 64)
	if taskID <= 0 {
		taskID = c.Config.TaskID
	}

	if taskID <= 0 {
		return nil, errors.New("任务ID不能为空")
	}

	if err := usecase.DeleteTask(context.Background(), taskID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": "任务已删除",
	}, nil
}

var globalTaskUsecase *biz.TaskBoardUsecase

// SetTaskUsecase 设置全局 taskUsecase 实例
func SetTaskUsecase(u *biz.TaskBoardUsecase) {
	globalTaskUsecase = u
}

func (c *TaskBoardComponent) Destroy() {}
func (c *TaskBoardComponent) Close() error { return nil }
