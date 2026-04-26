package data

import (
	"context"
	"time"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/data/dao"

	"github.com/go-kratos/kratos/v2/log"
)

var _ biz.ScheduledTaskRepo = &scheduledTaskRepo{}

type scheduledTaskRepo struct {
	data *Data
	log  *log.Helper
}

func NewScheduledTaskRepo(data *Data, logger log.Logger) biz.ScheduledTaskRepo {
	return &scheduledTaskRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *scheduledTaskRepo) CreateScheduledTask(ctx context.Context, task *entity.ScheduledTask) error {
	info := scheduledTaskEntityToDAO(task)
	if err := info.Create(ctx); err != nil {
		return err
	}
	task.ID = info.ID
	return nil
}

func (r *scheduledTaskRepo) GetScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error) {
	info, err := dao.NewScheduledTask().GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return scheduledTaskDAOToEntity(info), nil
}

func (r *scheduledTaskRepo) ListScheduledTasks(ctx context.Context, name, ruleChainID string, disabled *bool, page, pageSize int32) ([]*entity.ScheduledTask, int64, error) {
	tasks, total, err := dao.NewScheduledTask().List(ctx, name, ruleChainID, disabled, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return scheduledTaskDAOListToEntity(tasks), total, nil
}

func (r *scheduledTaskRepo) ListEnabledScheduledTasks(ctx context.Context) ([]*entity.ScheduledTask, error) {
	tasks, err := dao.NewScheduledTask().ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	return scheduledTaskDAOListToEntity(tasks), nil
}

func (r *scheduledTaskRepo) UpdateScheduledTask(ctx context.Context, task *entity.ScheduledTask) error {
	data := map[string]interface{}{
		"name":            task.Name,
		"description":     task.Description,
		"rule_chain_id":   task.RuleChainID,
		"cron_expr":       task.CronExpr,
		"schedule_type":   task.ScheduleType,
		"schedule_config": task.ScheduleConfig,
		"disabled":        task.Disabled,
		"last_run_at":     task.LastRunAt,
		"last_status":     task.LastStatus,
		"last_error":      task.LastError,
		"updated_at":      task.UpdatedAt,
	}
	return dao.NewScheduledTask().Update(ctx, task.ID, data)
}

func (r *scheduledTaskRepo) DeleteScheduledTask(ctx context.Context, id int64) error {
	return dao.NewScheduledTask().Delete(ctx, id)
}

func (r *scheduledTaskRepo) CreateScheduledTaskRun(ctx context.Context, run *entity.ScheduledTaskRun) error {
	info := scheduledTaskRunEntityToDAO(run)
	if info.CreatedAt.IsZero() {
		info.CreatedAt = time.Now()
	}
	if err := info.Create(ctx); err != nil {
		return err
	}
	run.ID = info.ID
	return nil
}

func (r *scheduledTaskRepo) ListScheduledTaskRuns(ctx context.Context, taskID int64, page, pageSize int32) ([]*entity.ScheduledTaskRun, int64, error) {
	runs, total, err := dao.NewScheduledTaskRun().ListByTaskID(ctx, taskID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*entity.ScheduledTaskRun, 0, len(runs))
	for _, run := range runs {
		out = append(out, scheduledTaskRunDAOToEntity(run))
	}
	return out, total, nil
}

func scheduledTaskEntityToDAO(task *entity.ScheduledTask) *dao.ScheduledTask {
	if task == nil {
		return nil
	}
	return &dao.ScheduledTask{
		ID:             task.ID,
		Name:           task.Name,
		Description:    task.Description,
		RuleChainID:    task.RuleChainID,
		CronExpr:       task.CronExpr,
		ScheduleType:   task.ScheduleType,
		ScheduleConfig: task.ScheduleConfig,
		Disabled:       task.Disabled,
		LastRunAt:      task.LastRunAt,
		LastStatus:     task.LastStatus,
		LastError:      task.LastError,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
		DeletedAt:      task.DeletedAt,
	}
}

func scheduledTaskDAOToEntity(task *dao.ScheduledTask) *entity.ScheduledTask {
	if task == nil {
		return nil
	}
	return &entity.ScheduledTask{
		ID:             task.ID,
		Name:           task.Name,
		Description:    task.Description,
		RuleChainID:    task.RuleChainID,
		CronExpr:       task.CronExpr,
		ScheduleType:   task.ScheduleType,
		ScheduleConfig: task.ScheduleConfig,
		Disabled:       task.Disabled,
		LastRunAt:      task.LastRunAt,
		LastStatus:     task.LastStatus,
		LastError:      task.LastError,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
		DeletedAt:      task.DeletedAt,
	}
}

func scheduledTaskDAOListToEntity(tasks []*dao.ScheduledTask) []*entity.ScheduledTask {
	out := make([]*entity.ScheduledTask, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, scheduledTaskDAOToEntity(task))
	}
	return out
}

func scheduledTaskRunEntityToDAO(run *entity.ScheduledTaskRun) *dao.ScheduledTaskRun {
	if run == nil {
		return nil
	}
	return &dao.ScheduledTaskRun{
		ID:             run.ID,
		TaskID:         run.TaskID,
		RuleChainID:    run.RuleChainID,
		Status:         run.Status,
		TriggerPayload: run.TriggerPayload,
		ErrorMessage:   run.ErrorMessage,
		StartedAt:      run.StartedAt,
		FinishedAt:     run.FinishedAt,
		CreatedAt:      run.CreatedAt,
	}
}

func scheduledTaskRunDAOToEntity(run *dao.ScheduledTaskRun) *entity.ScheduledTaskRun {
	if run == nil {
		return nil
	}
	return &entity.ScheduledTaskRun{
		ID:             run.ID,
		TaskID:         run.TaskID,
		RuleChainID:    run.RuleChainID,
		Status:         run.Status,
		TriggerPayload: run.TriggerPayload,
		ErrorMessage:   run.ErrorMessage,
		StartedAt:      run.StartedAt,
		FinishedAt:     run.FinishedAt,
		CreatedAt:      run.CreatedAt,
	}
}
