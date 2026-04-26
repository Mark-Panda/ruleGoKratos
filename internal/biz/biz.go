package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewRuleChainUsecase,
	NewComponentRegulationUsecase,
	NewComponentUseRuleUsecase,
	NewMdWorkflowUsecase,
	NewRunLogUsecase,
	NewAgentUsecase,
	NewScheduledTaskScheduler,
	NewScheduledTaskUsecase,
	NewTaskBoardUsecase,
	NewServiceManagementUsecase,
	wire.Bind(new(RuleChainSkillAgentRunner), new(*AgentUsecase)),
	wire.Bind(new(ScheduledTaskRuleChain), new(*RuleChainUsecase)),
)
