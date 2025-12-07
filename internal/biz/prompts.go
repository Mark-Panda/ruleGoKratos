package biz

import (
	"bytes"
	_ "embed"
	"ruleGoKratos/internal/biz/entity"
	"text/template"
)

//go:embed prompt/planner.tpl
var RuleChainPlannerPrompt string // 任务规划

//go:embed prompt/summary.tpl
var RuleChainSummaryPrompt string // 工作流总结

//go:embed prompt/execute.tpl
var RuleChainExecutePrompt string // 执行

//go:embed prompt/node_tool.tpl
var RuleChainNodeToolPrompt string // 节点组件生成工具

//go:embed prompt/connect_tool.tpl
var RuleChainConnectToolPrompt string // 连接组件生成工具

//go:embed prompt/assembly.tpl
var RuleChainAssemblyPrompt string // 规则链组装工具

func getNodeToolPrompt(param entity.NodeToolTpl) (string, error) {
	tpl, err := template.New(`RuleChainNodeToolPrompt`).Parse(RuleChainNodeToolPrompt)
	if err != nil {
		return "", err
	}
	var headerTPL bytes.Buffer
	if err1 := tpl.Execute(&headerTPL, param); err1 != nil {
		return "", err1
	}
	return headerTPL.String(), nil
}

func getConnectToolPrompt(param entity.ConnectUseRuleTpl) (string, error) {
	tpl, err := template.New(`RuleChainConnectToolPrompt`).Parse(RuleChainConnectToolPrompt)
	if err != nil {
		return "", err
	}
	var headerTPL bytes.Buffer
	if err1 := tpl.Execute(&headerTPL, param); err1 != nil {
		return "", err1
	}
	return headerTPL.String(), nil
}

func getPlannerPrompt(param entity.PlannerTpl) (string, error) {
	tpl, err := template.New(`RuleChainPlannerPrompt`).Parse(RuleChainPlannerPrompt)
	if err != nil {
		return "", err
	}
	var headerTPL bytes.Buffer
	if err1 := tpl.Execute(&headerTPL, param); err1 != nil {
		return "", err1
	}
	return headerTPL.String(), nil
}

func getAssemblyPrompt(param entity.AssemblyTpl) (string, error) {
	tpl, err := template.New(`RuleChainAssemblyPrompt`).Parse(RuleChainAssemblyPrompt)
	if err != nil {
		return "", err
	}
	var headerTPL bytes.Buffer
	if err1 := tpl.Execute(&headerTPL, param); err1 != nil {
		return "", err1
	}
	return headerTPL.String(), nil
}
