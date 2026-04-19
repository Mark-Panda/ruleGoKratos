package planbuilder

import "ruleGoKratos/internal/biz/entity"

// Builder 将协作方案编译为可执行计划。
type Builder interface {
	Mode() entity.CollaborationMode
	Build(scheme *entity.CollaborationScheme, userInput string) (*entity.ExecutionPlan, error)
}

// Registry 提供 PlanBuilder 的注册与查询。
type Registry struct {
	builders map[entity.CollaborationMode]Builder
}

// NewRegistry 创建新的 Builder 注册中心。
func NewRegistry() *Registry {
	return &Registry{
		builders: make(map[entity.CollaborationMode]Builder),
	}
}

// Register 注册指定模式的 Builder。
func (r *Registry) Register(builder Builder) {
	if r == nil || builder == nil {
		return
	}
	r.builders[builder.Mode()] = builder
}

// Get 返回指定模式对应的 Builder。
func (r *Registry) Get(mode entity.CollaborationMode) (Builder, bool) {
	if r == nil {
		return nil, false
	}
	builder, ok := r.builders[mode]
	return builder, ok
}
