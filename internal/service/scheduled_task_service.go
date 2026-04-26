package service

import (
	"context"
	"time"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ v1.ScheduledTaskServiceServer = (*ScheduledTaskService)(nil)

type scheduledTaskUsecase interface {
	CreateScheduledTask(ctx context.Context, name, description, ruleChainID, cronExpr, scheduleType, scheduleConfig string) (*entity.ScheduledTask, error)
	GetScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error)
	ListScheduledTasks(ctx context.Context, name, ruleChainID string, disabled *bool, page, pageSize int32) ([]*entity.ScheduledTask, int64, error)
	UpdateScheduledTask(ctx context.Context, id int64, name, description, ruleChainID, cronExpr, scheduleType, scheduleConfig string) (*entity.ScheduledTask, error)
	DeleteScheduledTask(ctx context.Context, id int64) error
	EnableScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error)
	DisableScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error)
	ListScheduledTaskRuns(ctx context.Context, taskID int64, page, pageSize int32) ([]*entity.ScheduledTaskRun, int64, error)
}

type ScheduledTaskService struct {
	v1.UnimplementedScheduledTaskServiceServer
	uc scheduledTaskUsecase
}

func NewScheduledTaskService(uc *biz.ScheduledTaskUsecase) *ScheduledTaskService {
	return &ScheduledTaskService{uc: uc}
}

func (s *ScheduledTaskService) CreateScheduledTask(ctx context.Context, req *v1.CreateScheduledTaskReq) (*v1.CreateScheduledTaskReply, error) {
	task, err := s.uc.CreateScheduledTask(ctx, req.GetName(), req.GetDescription(), req.GetRuleChainId(), req.GetCronExpr(), req.GetScheduleType(), req.GetScheduleConfig())
	if err != nil {
		return nil, err
	}
	return &v1.CreateScheduledTaskReply{Task: scheduledTaskToProto(task)}, nil
}

func (s *ScheduledTaskService) GetScheduledTask(ctx context.Context, req *v1.GetScheduledTaskReq) (*v1.GetScheduledTaskReply, error) {
	task, err := s.uc.GetScheduledTask(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.GetScheduledTaskReply{Task: scheduledTaskToProto(task)}, nil
}

func (s *ScheduledTaskService) ListScheduledTasks(ctx context.Context, req *v1.ListScheduledTasksReq) (*v1.ListScheduledTasksReply, error) {
	tasks, total, err := s.uc.ListScheduledTasks(ctx, req.GetName(), req.GetRuleChainId(), req.Disabled, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &v1.ListScheduledTasksReply{
		Tasks: scheduledTasksToProto(tasks),
		Total: total,
	}, nil
}

func (s *ScheduledTaskService) UpdateScheduledTask(ctx context.Context, req *v1.UpdateScheduledTaskReq) (*v1.UpdateScheduledTaskReply, error) {
	task, err := s.uc.UpdateScheduledTask(ctx, req.GetId(), req.GetName(), req.GetDescription(), req.GetRuleChainId(), req.GetCronExpr(), req.GetScheduleType(), req.GetScheduleConfig())
	if err != nil {
		return nil, err
	}
	return &v1.UpdateScheduledTaskReply{Task: scheduledTaskToProto(task)}, nil
}

func (s *ScheduledTaskService) DeleteScheduledTask(ctx context.Context, req *v1.DeleteScheduledTaskReq) (*v1.DeleteScheduledTaskReply, error) {
	if err := s.uc.DeleteScheduledTask(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteScheduledTaskReply{}, nil
}

func (s *ScheduledTaskService) EnableScheduledTask(ctx context.Context, req *v1.EnableScheduledTaskReq) (*v1.EnableScheduledTaskReply, error) {
	task, err := s.uc.EnableScheduledTask(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.EnableScheduledTaskReply{Task: scheduledTaskToProto(task)}, nil
}

func (s *ScheduledTaskService) DisableScheduledTask(ctx context.Context, req *v1.DisableScheduledTaskReq) (*v1.DisableScheduledTaskReply, error) {
	task, err := s.uc.DisableScheduledTask(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.DisableScheduledTaskReply{Task: scheduledTaskToProto(task)}, nil
}

func (s *ScheduledTaskService) ListScheduledTaskRuns(ctx context.Context, req *v1.ListScheduledTaskRunsReq) (*v1.ListScheduledTaskRunsReply, error) {
	runs, total, err := s.uc.ListScheduledTaskRuns(ctx, req.GetTaskId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &v1.ListScheduledTaskRunsReply{
		Runs:  scheduledTaskRunsToProto(runs),
		Total: total,
	}, nil
}

func scheduledTasksToProto(tasks []*entity.ScheduledTask) []*v1.ScheduledTask {
	pbTasks := make([]*v1.ScheduledTask, 0, len(tasks))
	for _, task := range tasks {
		pbTasks = append(pbTasks, scheduledTaskToProto(task))
	}
	return pbTasks
}

func scheduledTaskToProto(task *entity.ScheduledTask) *v1.ScheduledTask {
	if task == nil {
		return nil
	}
	pbTask := &v1.ScheduledTask{
		Id:             task.ID,
		Name:           task.Name,
		Description:    task.Description,
		RuleChainId:    task.RuleChainID,
		CronExpr:       task.CronExpr,
		ScheduleType:   task.ScheduleType,
		ScheduleConfig: task.ScheduleConfig,
		Disabled:       task.Disabled,
		LastStatus:     v1.ScheduledTaskRunStatus(task.LastStatus),
		LastError:      task.LastError,
		CreatedAt:      timestampFromTime(task.CreatedAt),
		UpdatedAt:      timestampFromTime(task.UpdatedAt),
	}
	if task.LastRunAt != nil {
		pbTask.LastRunAt = timestampFromTime(*task.LastRunAt)
	}
	if task.DeletedAt != nil {
		pbTask.DeletedAt = timestampFromTime(*task.DeletedAt)
	}
	return pbTask
}

func scheduledTaskRunsToProto(runs []*entity.ScheduledTaskRun) []*v1.ScheduledTaskRun {
	pbRuns := make([]*v1.ScheduledTaskRun, 0, len(runs))
	for _, run := range runs {
		pbRuns = append(pbRuns, scheduledTaskRunToProto(run))
	}
	return pbRuns
}

func scheduledTaskRunToProto(run *entity.ScheduledTaskRun) *v1.ScheduledTaskRun {
	if run == nil {
		return nil
	}
	return &v1.ScheduledTaskRun{
		Id:             run.ID,
		TaskId:         run.TaskID,
		RuleChainId:    run.RuleChainID,
		Status:         v1.ScheduledTaskRunStatus(run.Status),
		TriggerPayload: run.TriggerPayload,
		ErrorMessage:   run.ErrorMessage,
		StartedAt:      timestampFromTime(run.StartedAt),
		FinishedAt:     timestampFromTime(run.FinishedAt),
		CreatedAt:      timestampFromTime(run.CreatedAt),
	}
}

func timestampFromTime(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}
