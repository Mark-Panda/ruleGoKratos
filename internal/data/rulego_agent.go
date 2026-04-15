package data

import (
	"ruleGoKratos/internal/biz"
	rulegodatacomp "ruleGoKratos/internal/data/components"
)

// WireRuleGoAgent 将 Agent 用例注入 RuleGo 自定义节点（由 wire.Invoke 调用）。
func WireRuleGoAgent(uc *biz.AgentUsecase) {
	rulegodatacomp.SetRuleGoAgentUsecase(uc)
}
