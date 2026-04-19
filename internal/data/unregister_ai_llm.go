package data

import "github.com/rulego/rulego"

func init() {
	// 画布已下线 ai/llm；rulego-components-ai/ai/action 会在 init 中注册 TextGenerateNode（类型 ai/llm），此处统一摘除。
	_ = rulego.Registry.Unregister("ai/llm")
}
