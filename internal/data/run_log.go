package data

import (
	"context"
	"time"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/data/dao"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jinzhu/copier"
)

var _ biz.RunLogRepo = &runLogRepo{}

type runLogRepo struct {
	data *Data
	log  *log.Helper
}

// NewRunLogRepo .
func NewRunLogRepo(data *Data, logger log.Logger) biz.RunLogRepo {
	return &runLogRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// RunLog Methods
func (r *runLogRepo) CreateRunLog(ctx context.Context, runLog *entity.RunLog) error {
	t := time.Now()
	info := dao.RunLog{}
	_ = copier.Copy(&info, runLog)
	info.CreatedAt = &t
	info.UpdatedAt = &t
	return info.Create(ctx)
}

func (r *runLogRepo) UpdateRunLog(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error {
	info := dao.NewRunLog()
	return info.Updates(ctx, date, where)
}

func (r *runLogRepo) DeleteRunLog(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewRunLog()
	return info.Delete(ctx, where)
}

func (r *runLogRepo) FindOneRunLog(ctx context.Context, where map[string]interface{}) (*entity.RunLog, error) {
	info := dao.NewRunLog()
	result, err := info.FindOne(ctx, where)
	if err != nil {
		return nil, err
	}
	res := entity.RunLog{}
	_ = copier.Copy(&res, result)
	return &res, err
}

func (r *runLogRepo) FindListRunLog(ctx context.Context, where string, page int, pageSize int) ([]entity.RunLog, int64, error) {
	info := dao.NewRunLog()
	results, count, err := info.FindList(ctx, where, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	resList := make([]entity.RunLog, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, count, err
}

func (r *runLogRepo) FindAllRunLog(ctx context.Context, where map[string]interface{}) ([]entity.RunLog, error) {
	info := dao.NewRunLog()
	results, err := info.FindAll(ctx, where)
	if err != nil {
		return nil, err
	}
	resList := make([]entity.RunLog, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, err
}
