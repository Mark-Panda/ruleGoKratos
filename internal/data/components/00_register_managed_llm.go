package data

import "github.com/rulego/rulego"

func init() {
	// 必须在 github.com/rulego/rulego-components-ai/ai/action 之前注册 ai/llm（见 internal/data/data.go import 顺序）
	_ = rulego.Registry.Register(&ManagedTextGenerateNode{})
}
