package biz

import (
	_ "embed"
)

//go:embed prompt/default_prompt.tpl
var DefaultSystemPrompt string // Agent 默认系统提示词
