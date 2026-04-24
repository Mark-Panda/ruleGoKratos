package data

import (
	"context"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/data/dao"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

var _ biz.TaskBoardRepo = &taskBoardRepo{}

type taskBoardRepo struct {
	data *Data
	log  *log.Helper
}

// NewTaskBoardRepo 创建任务看板仓库实例
func NewTaskBoardRepo(data *Data, logger log.Logger) biz.TaskBoardRepo {
	return &taskBoardRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// Create 创建任务
func (r *taskBoardRepo) Create(ctx context.Context, task *entity.TaskBoard) error {
	t := dao.TaskBoard{}
	_ = copier.Copy(&t, task)
	return t.Create(ctx)
}

// GetByID 根据ID获取任务
func (r *taskBoardRepo) GetByID(ctx context.Context, id int64) (*entity.TaskBoard, error) {
	t := dao.NewTaskBoard()
	task, err := t.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	res := entity.TaskBoard{}
	_ = copier.Copy(&res, task)
	return &res, nil
}

// List 查询任务列表
func (r *taskBoardRepo) List(ctx context.Context, status, typ int32, handlerUserID string, page, pageSize int32) ([]*entity.TaskBoard, int64, error) {
	t := dao.NewTaskBoard()
	tasks, count, err := t.List(ctx, status, typ, handlerUserID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	res := make([]*entity.TaskBoard, len(tasks))
	for i, item := range tasks {
		res[i] = &entity.TaskBoard{}
		_ = copier.Copy(res[i], item)
	}
	return res, count, nil
}

// Update 更新任务
func (r *taskBoardRepo) Update(ctx context.Context, task *entity.TaskBoard) error {
	data := map[string]interface{}{
		"name":            task.Name,
		"priority":        task.Priority,
		"status":          task.Status,
		"handler_user_id": task.HandlerUserID,
		"description":     task.Description,
		"updated_at":      task.UpdatedAt,
	}
	t := dao.NewTaskBoard()
	return t.Update(ctx, task.ID, data)
}

// Delete 删除任务
func (r *taskBoardRepo) Delete(ctx context.Context, id int64) error {
	t := dao.NewTaskBoard()
	return t.Delete(ctx, id)
}
