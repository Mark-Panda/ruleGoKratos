package biz

import (
	"context"
	"time"

	"ruleGoKratos/internal/biz/entity"
)

// TaskBoardRepo 任务看板仓库接口
type TaskBoardRepo interface {
	Create(ctx context.Context, task *entity.TaskBoard) error
	GetByID(ctx context.Context, id int64) (*entity.TaskBoard, error)
	List(ctx context.Context, status, typ int32, handlerUserID string, page, pageSize int32) ([]*entity.TaskBoard, int64, error)
	Update(ctx context.Context, task *entity.TaskBoard) error
	Delete(ctx context.Context, id int64) error
}

// TaskBoardUsecase 任务看板业务逻辑
type TaskBoardUsecase struct {
	repo TaskBoardRepo
}

// NewTaskBoardUsecase 创建任务看板业务逻辑实例
func NewTaskBoardUsecase(repo TaskBoardRepo) *TaskBoardUsecase {
	return &TaskBoardUsecase{repo: repo}
}

// CreateTask 创建任务
func (uc *TaskBoardUsecase) CreateTask(ctx context.Context, name string, priority int32, typ int32, handlerUserID, description string) (*entity.TaskBoard, error) {
	task := &entity.TaskBoard{
		Name:          name,
		Priority:      priority,
		Status:        entity.TaskStatusPending,
		Type:          typ,
		HandlerUserID: handlerUserID,
		Description:   description,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if priority == 0 {
		task.Priority = 99
	}
	if err := uc.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

// GetTask 获取任务详情
func (uc *TaskBoardUsecase) GetTask(ctx context.Context, id int64) (*entity.TaskBoard, error) {
	return uc.repo.GetByID(ctx, id)
}

// ListTasks 查询任务列表
func (uc *TaskBoardUsecase) ListTasks(ctx context.Context, status, typ int32, handlerUserID string, page, pageSize int32) ([]*entity.TaskBoard, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return uc.repo.List(ctx, status, typ, handlerUserID, page, pageSize)
}

// UpdateTask 更新任务
func (uc *TaskBoardUsecase) UpdateTask(ctx context.Context, id int64, name *string, priority *int32, status *int32, handlerUserID *string, description *string) (*entity.TaskBoard, error) {
	task, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		task.Name = *name
	}
	if priority != nil {
		task.Priority = *priority
	}
	if status != nil {
		task.Status = *status
	}
	if handlerUserID != nil {
		task.HandlerUserID = *handlerUserID
	}
	if description != nil {
		task.Description = *description
	}
	task.UpdatedAt = time.Now()
	if err := uc.repo.Update(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

// DeleteTask 删除任务
func (uc *TaskBoardUsecase) DeleteTask(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}
