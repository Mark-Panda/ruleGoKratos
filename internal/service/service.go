package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(NewRuleGoService, NewRunLogService, NewComponentService, NewMdWorkflowService, NewAdminService, NewChatService)
