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
	_ = rulego.Registry.Register(&ServiceManagementComponent{})
}

// ServiceManagementComponent 服务管理组件
type ServiceManagementComponent struct {
	Config ServiceManagementConfiguration

	actionTpl           el.Template
	nameTpl             el.Template
	statusTpl           el.Template
	volcLogServiceIDTpl el.Template
	gitRepoURLTpl       el.Template
	descriptionTpl      el.Template
	serviceIDTpl        el.Template
	hasVar              bool
}

type ServiceManagementConfiguration struct {
	// 操作类型: save, get, delete
	Action string `json:"action"`
	// 服务名称
	Name string `json:"name"`
	// 服务状态
	Status int `json:"status"`
	// 火山引擎日志服务ID
	VolcLogServiceID string `json:"volcLogServiceId"`
	// Git仓库URL
	GitRepoURL string `json:"gitRepoUrl"`
	// 服务描述
	Description string `json:"description"`
	// 服务ID（用于get/update/delete）
	ServiceID int64 `json:"serviceId"`
	// 替换数据
	ReplaceData bool `json:"replaceData"`
}

func (c *ServiceManagementComponent) New() types.Node {
	return &ServiceManagementComponent{Config: ServiceManagementConfiguration{
		ReplaceData: true,
	}}
}

func (c *ServiceManagementComponent) Type() string {
	return "x/serviceManagement"
}

func (c *ServiceManagementComponent) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "服务管理",
		Desc:  "创建和管理服务目录中的服务",
	}
}

func (c *ServiceManagementComponent) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}

	tpls := map[*el.Template]string{
		&c.actionTpl:           c.Config.Action,
		&c.nameTpl:             c.Config.Name,
		&c.statusTpl:           strconv.Itoa(c.Config.Status),
		&c.volcLogServiceIDTpl: c.Config.VolcLogServiceID,
		&c.gitRepoURLTpl:       c.Config.GitRepoURL,
		&c.descriptionTpl:      c.Config.Description,
		&c.serviceIDTpl:        strconv.FormatInt(c.Config.ServiceID, 10),
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

func (c *ServiceManagementComponent) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	evn := base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	if evn == nil {
		evn = map[string]interface{}{}
	}

	action := strings.TrimSpace(c.actionTpl.ExecuteAsString(evn))
	if action == "" {
		action = "save"
	}
	// 兼容历史流程中的 create/update 配置，统一降级为 save。
	if action == "create" || action == "update" {
		action = "save"
	}

	serviceUsecase := globalServiceUsecase
	if serviceUsecase == nil {
		ctx.TellFailure(msg, errors.New("serviceManagement: 无法获取服务管理服务"))
		return
	}

	var result interface{}
	var err error

	switch action {
	case "save":
		result, err = c.doSave(ctx, evn, serviceUsecase)
	case "get":
		result, err = c.doGet(ctx, evn, serviceUsecase)
	case "delete":
		result, err = c.doDelete(ctx, evn, serviceUsecase)
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

func (c *ServiceManagementComponent) doSave(ctx types.RuleContext, evn map[string]interface{}, usecase *biz.ServiceManagementUsecase) (interface{}, error) {
	name := strings.TrimSpace(c.nameTpl.ExecuteAsString(evn))
	if name == "" {
		return nil, errors.New("服务名称不能为空")
	}

	status := int32(c.Config.Status)
	statusStr := c.statusTpl.ExecuteAsString(evn)
	if s, err := strconv.Atoi(strings.TrimSpace(statusStr)); err == nil {
		status = int32(s)
	}

	volcLogServiceID := strings.TrimSpace(c.volcLogServiceIDTpl.ExecuteAsString(evn))
	gitRepoURL := strings.TrimSpace(c.gitRepoURLTpl.ExecuteAsString(evn))
	description := strings.TrimSpace(c.descriptionTpl.ExecuteAsString(evn))

	service, err := usecase.SaveServiceByName(context.Background(), name, status, volcLogServiceID, gitRepoURL, description)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"service": map[string]interface{}{
			"id":                  service.ID,
			"name":                service.Name,
			"status":              service.Status,
			"volc_log_service_id": service.VolcLogServiceID,
			"git_repo_url":        service.GitRepoURL,
			"description":         service.Description,
			"created_at":          service.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

func (c *ServiceManagementComponent) doGet(ctx types.RuleContext, evn map[string]interface{}, usecase *biz.ServiceManagementUsecase) (interface{}, error) {
	serviceIDStr := c.serviceIDTpl.ExecuteAsString(evn)
	serviceID, _ := strconv.ParseInt(strings.TrimSpace(serviceIDStr), 10, 64)
	if serviceID <= 0 {
		serviceID = c.Config.ServiceID
	}

	if serviceID <= 0 {
		return nil, errors.New("服务ID不能为空")
	}

	service, err := usecase.GetService(context.Background(), serviceID)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return map[string]interface{}{"success": false, "error": "服务不存在"}, nil
	}

	return map[string]interface{}{
		"success": true,
		"service": map[string]interface{}{
			"id":                  service.ID,
			"name":                service.Name,
			"status":              service.Status,
			"volc_log_service_id": service.VolcLogServiceID,
			"git_repo_url":        service.GitRepoURL,
			"description":         service.Description,
			"created_at":          service.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":          service.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

func (c *ServiceManagementComponent) doDelete(ctx types.RuleContext, evn map[string]interface{}, usecase *biz.ServiceManagementUsecase) (interface{}, error) {
	serviceIDStr := c.serviceIDTpl.ExecuteAsString(evn)
	serviceID, _ := strconv.ParseInt(strings.TrimSpace(serviceIDStr), 10, 64)
	if serviceID <= 0 {
		serviceID = c.Config.ServiceID
	}

	if serviceID <= 0 {
		return nil, errors.New("服务ID不能为空")
	}

	if err := usecase.DeleteService(context.Background(), serviceID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": "服务已删除",
	}, nil
}

var globalServiceUsecase *biz.ServiceManagementUsecase

// SetServiceUsecase 设置全局 serviceUsecase 实例
func SetServiceUsecase(u *biz.ServiceManagementUsecase) {
	globalServiceUsecase = u
}

func (c *ServiceManagementComponent) Destroy()     {}
func (c *ServiceManagementComponent) Close() error { return nil }
