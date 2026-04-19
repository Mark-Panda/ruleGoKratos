package data

import playgroundruntime "ruleGoKratos/internal/biz/playground/runtime"

// NewMemoryRuntimeRepo 创建内存版 runtime repo，由 provider 显式选择是否作为默认实现。
func NewMemoryRuntimeRepo() playgroundruntime.Repo {
	return playgroundruntime.NewMemoryRepo()
}
