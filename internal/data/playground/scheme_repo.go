package data

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
	"sync"
)

// SchemeRepo 内存实现
type SchemeRepo struct {
	schemes map[string]*entity.CollaborationScheme
	mu      sync.RWMutex
}

func NewSchemeRepo() *SchemeRepo {
	return &SchemeRepo{
		schemes: make(map[string]*entity.CollaborationScheme),
	}
}

func (r *SchemeRepo) SaveScheme(ctx context.Context, scheme *entity.CollaborationScheme) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.schemes[scheme.ID]; exists {
		return fmt.Errorf("scheme already exists: %s", scheme.ID)
	}

	// 深拷贝
	s := *scheme
	if scheme.BindAgents != nil {
		bindings := make([]*entity.AgentBinding, len(scheme.BindAgents))
		for i, b := range scheme.BindAgents {
			bCopy := *b
			bindings[i] = &bCopy
		}
		s.BindAgents = bindings
	}

	r.schemes[scheme.ID] = &s
	return nil
}

func (r *SchemeRepo) UpdateScheme(ctx context.Context, scheme *entity.CollaborationScheme) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.schemes[scheme.ID]; !exists {
		return fmt.Errorf("scheme not found: %s", scheme.ID)
	}

	// 深拷贝
	s := *scheme
	if scheme.BindAgents != nil {
		bindings := make([]*entity.AgentBinding, len(scheme.BindAgents))
		for i, b := range scheme.BindAgents {
			bCopy := *b
			bindings[i] = &bCopy
		}
		s.BindAgents = bindings
	}

	r.schemes[scheme.ID] = &s
	return nil
}

func (r *SchemeRepo) DeleteScheme(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.schemes, id)
	return nil
}

func (r *SchemeRepo) FindSchemeByID(ctx context.Context, id string) (*entity.CollaborationScheme, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if scheme, ok := r.schemes[id]; ok {
		s := *scheme
		return &s, nil
	}

	return nil, fmt.Errorf("scheme not found: %s", id)
}

func (r *SchemeRepo) FindAllSchemes(ctx context.Context) ([]*entity.CollaborationScheme, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*entity.CollaborationScheme, 0, len(r.schemes))
	for _, scheme := range r.schemes {
		s := *scheme
		result = append(result, &s)
	}

	return result, nil
}
