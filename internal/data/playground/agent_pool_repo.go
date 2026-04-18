package data

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
	"sync"
)

// AgentPoolRepo 内存实现
type AgentPoolRepo struct {
	pools map[string]*entity.AgentPool
	mu    sync.RWMutex
}

func NewAgentPoolRepo() *AgentPoolRepo {
	return &AgentPoolRepo{
		pools: make(map[string]*entity.AgentPool),
	}
}

func (r *AgentPoolRepo) Create(ctx context.Context, pool *entity.AgentPool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pools[pool.ID]; exists {
		return fmt.Errorf("pool already exists: %s", pool.ID)
	}

	// 深拷贝
	p := *pool
	agents := make([]*entity.AgentDefinition, len(pool.Agents))
	copy(agents, pool.Agents)
	p.Agents = agents

	r.pools[pool.ID] = &p
	return nil
}

func (r *AgentPoolRepo) Update(ctx context.Context, pool *entity.AgentPool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 幂等写入：与业务层内存缓存对齐，避免「服务层已有池、仓储 map 未命中」时的误报；
	// Create 仍负责「已存在则报错」的唯一性约束。
	if pool.ID == "" {
		return fmt.Errorf("pool id is empty")
	}

	// 深拷贝
	p := *pool
	agents := make([]*entity.AgentDefinition, len(pool.Agents))
	copy(agents, pool.Agents)
	p.Agents = agents

	r.pools[pool.ID] = &p
	return nil
}

func (r *AgentPoolRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.pools[id]; !ok {
		return fmt.Errorf("pool not found: %s", id)
	}
	delete(r.pools, id)
	return nil
}

func (r *AgentPoolRepo) FindByID(ctx context.Context, id string) (*entity.AgentPool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if pool, ok := r.pools[id]; ok {
		// 深拷贝
		p := *pool
		agents := make([]*entity.AgentDefinition, len(pool.Agents))
		copy(agents, pool.Agents)
		p.Agents = agents
		return &p, nil
	}

	return nil, fmt.Errorf("pool not found: %s", id)
}

func (r *AgentPoolRepo) FindAll(ctx context.Context) ([]*entity.AgentPool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*entity.AgentPool, 0, len(r.pools))
	for _, pool := range r.pools {
		p := *pool
		agents := make([]*entity.AgentDefinition, len(pool.Agents))
		copy(agents, pool.Agents)
		p.Agents = agents
		result = append(result, &p)
	}

	return result, nil
}
