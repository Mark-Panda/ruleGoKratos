package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz/entity"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/structpb"
)

type ScheduledTaskRepo interface {
	CreateScheduledTask(ctx context.Context, task *entity.ScheduledTask) error
	GetScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error)
	ListScheduledTasks(ctx context.Context, name, ruleChainID string, disabled *bool, page, pageSize int32) ([]*entity.ScheduledTask, int64, error)
	ListEnabledScheduledTasks(ctx context.Context) ([]*entity.ScheduledTask, error)
	UpdateScheduledTask(ctx context.Context, task *entity.ScheduledTask) error
	DeleteScheduledTask(ctx context.Context, id int64) error
	CreateScheduledTaskRun(ctx context.Context, run *entity.ScheduledTaskRun) error
	ListScheduledTaskRuns(ctx context.Context, taskID int64, page, pageSize int32) ([]*entity.ScheduledTaskRun, int64, error)
}

type ScheduledTaskScheduler interface {
	Add(taskID int64, cronExpr string, fn func()) error
	Remove(taskID int64)
	Start()
	Stop()
}

type ScheduledTaskRuleChain interface {
	GetScheduledTaskRuleChain(ctx context.Context, id string) (*entity.RuleChain, error)
	ExecuteRuleChain(ctx context.Context, in *v1.ExecuteRuleChainReq) (*v1.ExecuteRuleChainReply, error)
	IsRuleChainLoaded(id string) bool
}

type ScheduledTaskUsecase struct {
	repo      ScheduledTaskRepo
	ruleChain ScheduledTaskRuleChain
	scheduler ScheduledTaskScheduler
	log       *log.Helper
}

func NewScheduledTaskUsecase(repo ScheduledTaskRepo, ruleChain ScheduledTaskRuleChain, scheduler ScheduledTaskScheduler, logger log.Logger) *ScheduledTaskUsecase {
	var helper *log.Helper
	if logger != nil {
		helper = log.NewHelper(logger)
	}
	return &ScheduledTaskUsecase{
		repo:      repo,
		ruleChain: ruleChain,
		scheduler: scheduler,
		log:       helper,
	}
}

func NewNilScheduledTaskScheduler() ScheduledTaskScheduler {
	return nil
}

func (uc *ScheduledTaskUsecase) CreateScheduledTask(ctx context.Context, name, description, ruleChainID, cronExpr, scheduleType, scheduleConfig string) (*entity.ScheduledTask, error) {
	now := time.Now()
	task := &entity.ScheduledTask{
		Name:           name,
		Description:    description,
		RuleChainID:    ruleChainID,
		CronExpr:       cronExpr,
		ScheduleType:   scheduleType,
		ScheduleConfig: scheduleConfig,
		Disabled:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := uc.repo.CreateScheduledTask(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (uc *ScheduledTaskUsecase) GetScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error) {
	return uc.repo.GetScheduledTask(ctx, id)
}

func (uc *ScheduledTaskUsecase) ListScheduledTasks(ctx context.Context, name, ruleChainID string, disabled *bool, page, pageSize int32) ([]*entity.ScheduledTask, int64, error) {
	page, pageSize = normalizeScheduledTaskPage(page, pageSize)
	return uc.repo.ListScheduledTasks(ctx, name, ruleChainID, disabled, page, pageSize)
}

func (uc *ScheduledTaskUsecase) UpdateScheduledTask(ctx context.Context, id int64, name, description, ruleChainID, cronExpr, scheduleType, scheduleConfig string) (*entity.ScheduledTask, error) {
	task, err := uc.repo.GetScheduledTask(ctx, id)
	if err != nil {
		return nil, err
	}
	oldTask := cloneScheduledTaskEntity(task)
	task.Name = name
	task.Description = description
	task.RuleChainID = ruleChainID
	task.CronExpr = cronExpr
	task.ScheduleType = scheduleType
	task.ScheduleConfig = scheduleConfig
	task.UpdatedAt = time.Now()
	if !task.Disabled && uc.scheduler != nil {
		if err := uc.scheduler.Add(task.ID, task.CronExpr, uc.scheduledTaskCallback(task.ID)); err != nil {
			if rollbackErr := uc.scheduler.Add(oldTask.ID, oldTask.CronExpr, uc.scheduledTaskCallback(oldTask.ID)); rollbackErr != nil {
				return nil, fmt.Errorf("add scheduled task job: %w; restore scheduler job: %v", err, rollbackErr)
			}
			return nil, err
		}
	}
	if err := uc.repo.UpdateScheduledTask(ctx, task); err != nil {
		if !task.Disabled && uc.scheduler != nil {
			uc.scheduler.Remove(task.ID)
			if rollbackErr := uc.scheduler.Add(oldTask.ID, oldTask.CronExpr, uc.scheduledTaskCallback(oldTask.ID)); rollbackErr != nil {
				return nil, fmt.Errorf("update scheduled task: %w; restore scheduler job: %v", err, rollbackErr)
			}
		}
		return nil, err
	}
	return task, nil
}

func (uc *ScheduledTaskUsecase) DeleteScheduledTask(ctx context.Context, id int64) error {
	task, err := uc.repo.GetScheduledTask(ctx, id)
	if err != nil {
		return err
	}
	if err := uc.repo.DeleteScheduledTask(ctx, id); err != nil {
		return err
	}
	if !task.Disabled && uc.scheduler != nil {
		uc.scheduler.Remove(task.ID)
	}
	return nil
}

func (uc *ScheduledTaskUsecase) EnableScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error) {
	task, err := uc.repo.GetScheduledTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := uc.validateScheduledTaskRuleChain(ctx, task.RuleChainID); err != nil {
		return nil, err
	}
	if uc.scheduler != nil {
		if err := uc.scheduler.Add(task.ID, task.CronExpr, uc.scheduledTaskCallback(task.ID)); err != nil {
			return nil, err
		}
	}
	task.Disabled = false
	task.UpdatedAt = time.Now()
	if err := uc.repo.UpdateScheduledTask(ctx, task); err != nil {
		if uc.scheduler != nil {
			uc.scheduler.Remove(task.ID)
		}
		return nil, err
	}
	return task, nil
}

func (uc *ScheduledTaskUsecase) DisableScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error) {
	task, err := uc.repo.GetScheduledTask(ctx, id)
	if err != nil {
		return nil, err
	}
	task.Disabled = true
	task.UpdatedAt = time.Now()
	if err := uc.repo.UpdateScheduledTask(ctx, task); err != nil {
		return nil, err
	}
	if uc.scheduler != nil {
		uc.scheduler.Remove(task.ID)
	}
	return task, nil
}

func (uc *ScheduledTaskUsecase) ListScheduledTaskRuns(ctx context.Context, taskID int64, page, pageSize int32) ([]*entity.ScheduledTaskRun, int64, error) {
	page, pageSize = normalizeScheduledTaskPage(page, pageSize)
	return uc.repo.ListScheduledTaskRuns(ctx, taskID, page, pageSize)
}

func (uc *ScheduledTaskUsecase) StartEnabledTasks(ctx context.Context) error {
	if uc.scheduler == nil {
		return nil
	}
	tasks, err := uc.repo.ListEnabledScheduledTasks(ctx)
	if err != nil {
		return err
	}
	addedTaskIDs := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		taskID := task.ID
		if err := uc.scheduler.Add(taskID, task.CronExpr, uc.scheduledTaskCallback(taskID)); err != nil {
			for _, addedTaskID := range addedTaskIDs {
				uc.scheduler.Remove(addedTaskID)
			}
			return fmt.Errorf("add scheduled task job %d: %w", taskID, err)
		}
		addedTaskIDs = append(addedTaskIDs, taskID)
	}
	uc.scheduler.Start()
	return nil
}

func (uc *ScheduledTaskUsecase) StopScheduledTasks() {
	if uc == nil || uc.scheduler == nil {
		return
	}
	uc.scheduler.Stop()
}

func (uc *ScheduledTaskUsecase) scheduledTaskCallback(taskID int64) func() {
	return func() {
		if err := uc.runScheduledTask(context.Background(), taskID); err != nil {
			uc.logScheduledTaskRunError(taskID, err)
		}
	}
}

func (uc *ScheduledTaskUsecase) logScheduledTaskRunError(taskID int64, err error) {
	if uc == nil || uc.log == nil || err == nil {
		return
	}
	uc.log.Errorw("msg", "run scheduled task failed", "taskID", taskID, "err", err)
}

func (uc *ScheduledTaskUsecase) runScheduledTask(ctx context.Context, taskID int64) error {
	task, err := uc.repo.GetScheduledTask(ctx, taskID)
	if err != nil || task == nil || task.Disabled {
		return err
	}
	startedAt := time.Now()
	payload := entity.NewScheduledTriggerPayload(task.ID)
	status := entity.ScheduledTaskRunStatusSuccess
	errorMessage := ""
	shouldRemoveScheduler := false

	if err := uc.validateScheduledTaskRuleChain(ctx, task.RuleChainID); err != nil {
		status = entity.ScheduledTaskRunStatusFailed
		errorMessage = err.Error()
		task.Disabled = true
		shouldRemoveScheduler = true
	} else {
		var payloadData map[string]interface{}
		err := json.Unmarshal([]byte(payload), &payloadData)
		if err == nil && payloadData == nil {
			err = errors.New("定时任务触发 payload 为空")
		}
		data, structErr := structpb.NewStruct(payloadData)
		if err == nil {
			err = structErr
		}
		if err != nil {
			status = entity.ScheduledTaskRunStatusFailed
			errorMessage = err.Error()
		} else if _, err := uc.ruleChain.ExecuteRuleChain(ctx, &v1.ExecuteRuleChainReq{
			Id:      task.RuleChainID,
			MsgType: "SCHEDULED_TASK",
			Data:    data,
		}); err != nil {
			status = entity.ScheduledTaskRunStatusFailed
			errorMessage = err.Error()
		}
	}

	finishedAt := time.Now()
	if err := uc.repo.CreateScheduledTaskRun(ctx, &entity.ScheduledTaskRun{
		TaskID:         task.ID,
		RuleChainID:    task.RuleChainID,
		Status:         status,
		TriggerPayload: payload,
		ErrorMessage:   errorMessage,
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
		CreatedAt:      finishedAt,
	}); err != nil {
		return fmt.Errorf("create scheduled task run: %w", err)
	}
	task.LastRunAt = &finishedAt
	task.LastStatus = status
	task.LastError = errorMessage
	task.UpdatedAt = finishedAt
	if err := uc.repo.UpdateScheduledTask(ctx, task); err != nil {
		return fmt.Errorf("update scheduled task status: %w", err)
	}
	if shouldRemoveScheduler && uc.scheduler != nil {
		uc.scheduler.Remove(task.ID)
	}
	return nil
}

func (uc *ScheduledTaskUsecase) validateScheduledTaskRuleChain(ctx context.Context, ruleChainID string) error {
	if uc.ruleChain == nil {
		return errors.New("规则链依赖未配置")
	}
	ruleChain, err := uc.ruleChain.GetScheduledTaskRuleChain(ctx, ruleChainID)
	if err != nil {
		return err
	}
	if ruleChain == nil {
		return fmt.Errorf("规则链 %s 不存在", ruleChainID)
	}
	if !ruleChain.Root {
		return errors.New("定时任务只能绑定主规则链")
	}
	if ruleChain.Disabled {
		return errors.New("绑定规则链必须处于启用状态")
	}
	if !uc.ruleChain.IsRuleChainLoaded(ruleChainID) {
		return errors.New("规则链未加载")
	}
	return nil
}

func normalizeScheduledTaskPage(page, pageSize int32) (int32, int32) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return page, pageSize
}

func cloneScheduledTaskEntity(task *entity.ScheduledTask) *entity.ScheduledTask {
	if task == nil {
		return nil
	}
	cp := *task
	if task.LastRunAt != nil {
		t := *task.LastRunAt
		cp.LastRunAt = &t
	}
	if task.DeletedAt != nil {
		t := *task.DeletedAt
		cp.DeletedAt = &t
	}
	return &cp
}
