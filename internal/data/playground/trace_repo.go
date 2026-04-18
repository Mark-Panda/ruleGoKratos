package data

import (
	"context"
	"fmt"
	"ruleGoKratos/internal/biz/entity"
	"sync"
)

// TraceRepo 内存实现
type TraceRepo struct {
	runs   map[string]*entity.TraceRun
	events map[string][]*entity.TraceEvent // runID -> events
	mu     sync.RWMutex
}

func NewTraceRepo() *TraceRepo {
	return &TraceRepo{
		runs:   make(map[string]*entity.TraceRun),
		events: make(map[string][]*entity.TraceEvent),
	}
}

func (r *TraceRepo) Save(ctx context.Context, run *entity.TraceRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.runs[run.RunID]; exists {
		return fmt.Errorf("run already exists: %s", run.RunID)
	}

	// 深拷贝
	runCopy := *run
	if run.Events != nil {
		events := make([]*entity.TraceEvent, len(run.Events))
		copy(events, run.Events)
		runCopy.Events = events
	}

	r.runs[run.RunID] = &runCopy
	r.events[run.RunID] = make([]*entity.TraceEvent, 0)

	return nil
}

func (r *TraceRepo) Update(ctx context.Context, run *entity.TraceRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.runs[run.RunID]; !exists {
		return fmt.Errorf("run not found: %s", run.RunID)
	}

	// 深拷贝
	runCopy := *run
	if run.Events != nil {
		events := make([]*entity.TraceEvent, len(run.Events))
		copy(events, run.Events)
		runCopy.Events = events
	}

	r.runs[run.RunID] = &runCopy
	return nil
}

func (r *TraceRepo) FindByID(ctx context.Context, id string) (*entity.TraceRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 遍历查找（因为 ID 可能对应不同字段）
	for _, run := range r.runs {
		if run.ID == id || run.RunID == id {
			runCopy := *run
			return &runCopy, nil
		}
	}

	return nil, fmt.Errorf("run not found: %s", id)
}

func (r *TraceRepo) FindByRunID(ctx context.Context, runID string) (*entity.TraceRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if run, ok := r.runs[runID]; ok {
		runCopy := *run
		events := make([]*entity.TraceEvent, len(run.Events))
		copy(events, run.Events)
		runCopy.Events = events
		return &runCopy, nil
	}

	return nil, fmt.Errorf("run not found: %s", runID)
}

func (r *TraceRepo) FindEvents(ctx context.Context, filter *entity.TraceFilter) ([]*entity.TraceEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var events []*entity.TraceEvent
	if filter.RunID != "" {
		events = r.events[filter.RunID]
	} else {
		// 合并所有事件
		for _, evts := range r.events {
			events = append(events, evts...)
		}
	}

	// 应用过滤
	result := make([]*entity.TraceEvent, 0, len(events))
	for _, e := range events {
		if filter.RunID != "" && e.RunID != filter.RunID {
			continue
		}
		if filter.AgentID != "" && e.AgentID != filter.AgentID {
			continue
		}
		if filter.EventType != "" && e.Type != filter.EventType {
			continue
		}
		result = append(result, e)
	}

	// 限制返回数量
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	// 偏移量
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	}

	return result, nil
}

func (r *TraceRepo) AppendEvent(ctx context.Context, event *entity.TraceEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 深拷贝
	e := *event
	if event.Metadata != nil {
		metadata := make(map[string]interface{}, len(event.Metadata))
		for k, v := range event.Metadata {
			metadata[k] = v
		}
		e.Metadata = metadata
	}

	r.events[event.RunID] = append(r.events[event.RunID], &e)

	// 同时更新 run 中的 events
	if run, ok := r.runs[event.RunID]; ok {
		run.Events = append(run.Events, &e)
	}

	return nil
}
