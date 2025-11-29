package biz

import _ "embed"

//go:embed prompt/planner.tpl
var RuleChainPlannerPrompt string // 任务规划

//go:embed prompt/summary.tpl
var RuleChainSummaryPrompt string // 工作流总结

//go:embed prompt/execute.tpl
var RuleChainExecutePrompt string // 执行

//go:embed prompt/execute_node.tpl
var RuleChainExecuteNodePrompt string // 节点组件生成工具

//go:embed prompt/execute_connect.tpl
var RuleChainExecuteConnectPrompt string // 连接组件生成工具
