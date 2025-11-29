package biz

import _ "embed"

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
