package main

import (
	"context"
	"testing"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
)

func TestScheduledTaskBeforeStartRestoresEnabledTasks(t *testing.T) {
	repo := &mainScheduledTaskRepo{
		enabledTasks: []*entity.ScheduledTask{
			{ID: 1, CronExpr: "@every 1h", Disabled: false},
		},
	}
	scheduler := &mainScheduledTaskScheduler{}
	uc := biz.NewScheduledTaskUsecase(repo, nil, scheduler, nil)

	if err := scheduledTaskBeforeStart(uc)(context.Background()); err != nil {
		t.Fatalf("scheduledTaskBeforeStart returned error: %v", err)
	}

	if !scheduler.started {
		t.Fatal("expected lifecycle start hook to start scheduled tasks")
	}
	if len(scheduler.added) != 1 || scheduler.added[0] != 1 {
		t.Fatalf("expected enabled task to be added, added=%v", scheduler.added)
	}
}

func TestScheduledTaskAfterStopStopsScheduler(t *testing.T) {
	scheduler := &mainScheduledTaskScheduler{}
	uc := biz.NewScheduledTaskUsecase(&mainScheduledTaskRepo{}, nil, scheduler, nil)

	if err := scheduledTaskAfterStop(uc)(context.Background()); err != nil {
		t.Fatalf("scheduledTaskAfterStop returned error: %v", err)
	}

	if !scheduler.stopped {
		t.Fatal("expected lifecycle stop hook to stop scheduled tasks")
	}
}

type mainScheduledTaskRepo struct {
	enabledTasks []*entity.ScheduledTask
}

func (r *mainScheduledTaskRepo) CreateScheduledTask(context.Context, *entity.ScheduledTask) error {
	return nil
}

func (r *mainScheduledTaskRepo) GetScheduledTask(context.Context, int64) (*entity.ScheduledTask, error) {
	return nil, nil
}

func (r *mainScheduledTaskRepo) ListScheduledTasks(context.Context, string, string, *bool, int32, int32) ([]*entity.ScheduledTask, int64, error) {
	return nil, 0, nil
}

func (r *mainScheduledTaskRepo) ListEnabledScheduledTasks(context.Context) ([]*entity.ScheduledTask, error) {
	return r.enabledTasks, nil
}

func (r *mainScheduledTaskRepo) UpdateScheduledTask(context.Context, *entity.ScheduledTask) error {
	return nil
}

func (r *mainScheduledTaskRepo) DeleteScheduledTask(context.Context, int64) error {
	return nil
}

func (r *mainScheduledTaskRepo) CreateScheduledTaskRun(context.Context, *entity.ScheduledTaskRun) error {
	return nil
}

func (r *mainScheduledTaskRepo) ListScheduledTaskRuns(context.Context, int64, int32, int32) ([]*entity.ScheduledTaskRun, int64, error) {
	return nil, 0, nil
}

type mainScheduledTaskScheduler struct {
	added   []int64
	started bool
	stopped bool
}

func (s *mainScheduledTaskScheduler) Add(taskID int64, _ string, _ func()) error {
	s.added = append(s.added, taskID)
	return nil
}

func (s *mainScheduledTaskScheduler) Remove(int64) {}

func (s *mainScheduledTaskScheduler) Start() {
	s.started = true
}

func (s *mainScheduledTaskScheduler) Stop() {
	s.stopped = true
}
