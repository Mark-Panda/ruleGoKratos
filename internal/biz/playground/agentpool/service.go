package agentpool

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AgentPoolRepo Agent 池仓储接口
type AgentPoolRepo interface {
	Create(ctx context.Context, pool *entity.AgentPool) error
	Update(ctx context.Context, pool *entity.AgentPool) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*entity.AgentPool, error)
	FindAll(ctx context.Context) ([]*entity.AgentPool, error)
}

// AgentPoolService Agent 池服务
type AgentPoolService struct {
	repo   AgentPoolRepo
	pools  map[string]*entity.AgentPool
	poolsMu sync.RWMutex
}

func NewAgentPoolService(repo AgentPoolRepo) *AgentPoolService {
	return &AgentPoolService{
		repo:  repo,
		pools: make(map[string]*entity.AgentPool),
	}
}

// CreatePool 创建 Agent 池
func (s *AgentPoolService) CreatePool(ctx context.Context, name, desc string, agents []*entity.AgentDefinition) (*entity.AgentPool, error) {
	pool := &entity.AgentPool{
		ID:          uuid.NewString(),
		Name:        name,
		Description: desc,
		Agents:      NormalizeAgentDefinitions(agents),
		CreatedAt:   nowPtr(),
		UpdatedAt:   nowPtr(),
	}

	if err := s.repo.Create(ctx, pool); err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	s.poolsMu.Lock()
	s.pools[pool.ID] = pool
	s.poolsMu.Unlock()

	return pool, nil
}

// AddAgent 添加 Agent 到池
func (s *AgentPoolService) AddAgent(ctx context.Context, poolID string, agent *entity.AgentDefinition) error {
	if _, err := s.GetPool(ctx, poolID); err != nil {
		return err
	}

	s.poolsMu.Lock()
	defer s.poolsMu.Unlock()

	pool := s.pools[poolID]
	if pool == nil {
		return fmt.Errorf("pool not found: %s", poolID)
	}

	agent.ID = uuid.NewString()
	pool.Agents = append(pool.Agents, agent)
	pool.UpdatedAt = nowPtr()

	if err := s.repo.Update(ctx, pool); err != nil {
		return fmt.Errorf("update pool: %w", err)
	}

	return nil
}

// RemoveAgent 从池中移除 Agent
func (s *AgentPoolService) RemoveAgent(ctx context.Context, poolID, agentID string) error {
	if _, err := s.GetPool(ctx, poolID); err != nil {
		return err
	}

	s.poolsMu.Lock()
	defer s.poolsMu.Unlock()

	pool := s.pools[poolID]
	if pool == nil {
		return fmt.Errorf("pool not found: %s", poolID)
	}

	newAgents := make([]*entity.AgentDefinition, 0, len(pool.Agents))
	for _, a := range pool.Agents {
		if a.ID != agentID {
			newAgents = append(newAgents, a)
		}
	}
	pool.Agents = newAgents
	pool.UpdatedAt = nowPtr()

	if err := s.repo.Update(ctx, pool); err != nil {
		return fmt.Errorf("update pool: %w", err)
	}

	return nil
}

// GetPool 获取 Agent 池（始终以仓储为准，避免缓存命中已过期的内存副本）
func (s *AgentPoolService) GetPool(ctx context.Context, id string) (*entity.AgentPool, error) {
	if id == "" {
		return nil, fmt.Errorf("pool id is empty")
	}
	pool, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.poolsMu.Lock()
		delete(s.pools, id)
		s.poolsMu.Unlock()
		return nil, fmt.Errorf("find pool: %w", err)
	}

	s.poolsMu.Lock()
	s.pools[pool.ID] = pool
	s.poolsMu.Unlock()

	return pool, nil
}

// UpdatePool 更新 Agent 池（整体替换 Agent 列表）
func (s *AgentPoolService) UpdatePool(ctx context.Context, poolID, name, desc string, agents []*entity.AgentDefinition) (*entity.AgentPool, error) {
	pool, err := s.GetPool(ctx, poolID)
	if err != nil {
		return nil, fmt.Errorf("get pool: %w", err)
	}
	pool.Name = name
	pool.Description = desc
	pool.Agents = NormalizeAgentDefinitions(agents)
	pool.UpdatedAt = nowPtr()

	if err := s.repo.Update(ctx, pool); err != nil {
		return nil, fmt.Errorf("update pool: %w", err)
	}

	return pool, nil
}

// DeletePool 删除 Agent 池（默认池不可删）
func (s *AgentPoolService) DeletePool(ctx context.Context, id string) error {
	if id == "default" {
		return fmt.Errorf("cannot delete the default agent pool")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete pool: %w", err)
	}
	s.poolsMu.Lock()
	delete(s.pools, id)
	s.poolsMu.Unlock()
	return nil
}

// ListPools 列出所有 Agent 池（始终从仓储加载并同步内存缓存，避免「只读缓存」与 DB 不一致导致删除 404）
func (s *AgentPoolService) ListPools(ctx context.Context) ([]*entity.AgentPool, error) {
	pools, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("find all pools: %w", err)
	}

	s.poolsMu.Lock()
	s.pools = make(map[string]*entity.AgentPool, len(pools))
	for _, p := range pools {
		if p != nil {
			s.pools[p.ID] = p
		}
	}
	s.poolsMu.Unlock()

	sortPoolsByUpdatedDesc(pools)
	return pools, nil
}

// PoolRef Playground Agent 池引用（用于删除主站 Agent 配置前校验）
type PoolRef struct {
	ID   string
	Name string
}

// PoolsReferencingManagedAgent 列出池内某条 Agent 关联了给定主站「Agent 配置」id 的池（同一池只出现一次）
func (s *AgentPoolService) PoolsReferencingManagedAgent(ctx context.Context, managedAgentID int64) ([]PoolRef, error) {
	if managedAgentID <= 0 {
		return nil, nil
	}
	pools, err := s.ListPools(ctx)
	if err != nil {
		return nil, err
	}
	var refs []PoolRef
	seen := make(map[string]struct{})
	for _, p := range pools {
		if p == nil {
			continue
		}
		for _, a := range p.Agents {
			if a != nil && a.ManagedAgentID == managedAgentID {
				if _, ok := seen[p.ID]; !ok {
					seen[p.ID] = struct{}{}
					refs = append(refs, PoolRef{ID: p.ID, Name: p.Name})
				}
				break
			}
		}
	}
	return refs, nil
}

// GetAgentByID 根据 ID 获取 Agent 定义
func (s *AgentPoolService) GetAgentByID(poolID, agentID string) (*entity.AgentDefinition, error) {
	s.poolsMu.RLock()
	defer s.poolsMu.RUnlock()

	pool, ok := s.pools[poolID]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}

	for _, a := range pool.Agents {
		if a.ID == agentID {
			return a, nil
		}
	}

	return nil, fmt.Errorf("agent not found: %s", agentID)
}

// CreateAgentInstance 根据定义创建 Agent 实例
func (s *AgentPoolService) CreateAgentInstance(poolID, agentID string) (*entity.AgentInstance, error) {
	def, err := s.GetAgentByID(poolID, agentID)
	if err != nil {
		return nil, err
	}

	return &entity.AgentInstance{
		ID:         uuid.NewString(),
		Definition: def,
		State:      entity.AgentStateIdle,
		History:    make([]entity.Turn, 0),
		StartTime:  nowPtr(),
		Metadata:   make(map[string]interface{}),
	}, nil
}

// DefaultAgentPool 创建默认 Agent 池
func (s *AgentPoolService) CreateDefaultAgentPool(ctx context.Context) (*entity.AgentPool, error) {
	defaultPool := &entity.AgentPool{
		ID:          "default",
		Name:        "默认Agent池",
		Description: "系统预置的默认 Agent 池",
		Agents:      defaultAgents(),
		CreatedAt:   nowPtr(),
		UpdatedAt:   nowPtr(),
	}

	// 尝试创建，如果已存在则跳过
	if err := s.repo.Create(ctx, defaultPool); err != nil {
		// 可能已存在，尝试获取
		existing, findErr := s.repo.FindByID(ctx, "default")
		if findErr == nil {
			s.poolsMu.Lock()
			s.pools["default"] = existing
			s.poolsMu.Unlock()
			return existing, nil
		}
		return nil, fmt.Errorf("create default pool: %w", err)
	}

	s.poolsMu.Lock()
	s.pools[defaultPool.ID] = defaultPool
	s.poolsMu.Unlock()

	return defaultPool, nil
}

// defaultAgents 返回默认的 Agent 定义
func defaultAgents() []*entity.AgentDefinition {
	return []*entity.AgentDefinition{
		{
			ID:       "designer",
			Name:     "设计师",
			Role:     "你是一位专业的 UI/UX 设计师，擅长设计美观、用户体验友好的界面。",
			Desc:     "负责界面设计和视觉风格",
			Model:    "gpt-4o",
			Tools:    []string{"read_file", "write_file"},
			Enabled:  true,
			Priority: 1,
		},
		{
			ID:       "pm",
			Name:     "产品经理",
			Role:     "你是一位专业的产品经理，擅长需求分析、产品规划和项目管理。",
			Desc:     "负责需求分析和产品规划",
			Model:    "gpt-4o",
			Tools:    []string{"read_file"},
			Enabled:  true,
			Priority: 2,
		},
		{
			ID:       "engineer",
			Name:     "工程师",
			Role:     "你是一位专业的软件工程师，擅长编写高质量、生产级的代码。",
			Desc:     "负责代码开发和实现",
			Model:    "gpt-4o",
			Tools:    []string{"read_file", "write_file", "shell"},
			Enabled:  true,
			Priority: 3,
		},
		{
			ID:       "planner",
			Name:     "规划师",
			Role:     "你是一位专业的任务规划师，擅长将复杂任务拆解为可执行的步骤。",
			Desc:     "负责任务规划和拆分",
			Model:    "gpt-4o",
			Tools:    []string{},
			Enabled:  true,
			Priority: 0,
		},
		{
			ID:       "supervisor",
			Name:     "监督者",
			Role:     "你是一位专业的任务监督者，擅长监控进度、发现问题并及时干预。",
			Desc:     "负责监控和干预",
			Model:    "gpt-4o",
			Tools:    []string{"read_file"},
			Enabled:  true,
			Priority: 0,
		},
	}
}

// NormalizeAgentDefinitions 深拷贝 Agent 定义，空 id 时生成 UUID，避免多处共享同一指针。
func NormalizeAgentDefinitions(agents []*entity.AgentDefinition) []*entity.AgentDefinition {
	if len(agents) == 0 {
		return nil
	}
	out := make([]*entity.AgentDefinition, 0, len(agents))
	for _, a := range agents {
		if a == nil {
			continue
		}
		cp := *a
		if cp.ID == "" {
			cp.ID = uuid.NewString()
		}
		if cp.Tools == nil {
			cp.Tools = []string{}
		}
		out = append(out, &cp)
	}
	return out
}

func sortPoolsByUpdatedDesc(list []*entity.AgentPool) {
	sort.SliceStable(list, func(i, j int) bool {
		ui, uj := list[i].UpdatedAt, list[j].UpdatedAt
		if ui == nil && uj == nil {
			return list[i].Name < list[j].Name
		}
		if ui == nil {
			return false
		}
		if uj == nil {
			return true
		}
		if ui.Equal(*uj) {
			return list[i].Name < list[j].Name
		}
		return ui.After(*uj)
	})
}

// nowPtr 返回当前时间指针
func nowPtr() *time.Time {
	now := time.Now()
	return &now
}
