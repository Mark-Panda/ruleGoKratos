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

type ruleGoRepo struct {
	data *Data
	log  *log.Helper
}

// NewRuleGoRepo .
func NewRuleGoRepo(data *Data, logger log.Logger) biz.RuleGoRepo {
	return &ruleGoRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *ruleGoRepo) CreateRegulation(ctx context.Context, regulation *entity.Regulation) error {
	t := time.Now()
	info := dao.Regulation{
		UserName:    regulation.UserName,
		Root:        regulation.Root,
		Disabled:    regulation.Disabled,
		Name:        regulation.Name,
		RuleChainID: regulation.RuleChainID,
		RuleVersion: regulation.RuleVersion,
		RuleConfig:  regulation.RuleConfig,
		CreatedAt:   &t,
		UpdatedAt:   &t,
	}
	err := info.Create(ctx)
	return err
}

func (r *ruleGoRepo) UpdateRegulation(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error {
	info := dao.NewRegulation()
	return info.Updates(ctx, date, where)
}

func (r *ruleGoRepo) DeleteRegulation(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewRegulation()
	return info.Delete(ctx, where)
}

func (r *ruleGoRepo) FindOneRegulation(ctx context.Context, where map[string]interface{}) (*entity.Regulation, error) {
	info := dao.NewRegulation()
	regulation, err := info.FindOne(ctx, where)
	if err != nil {
		return nil, err
	}

	regulationInfo := entity.Regulation{}
	_ = copier.Copy(&regulationInfo, regulation)
	return &regulationInfo, err
}

func (r *ruleGoRepo) FindListRegulation(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.Regulation, int64, error) {
	info := dao.NewRegulation()
	regulations, count, err := info.FindList(ctx, where, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	regulationList := make([]entity.Regulation, len(regulations))
	for i, regulation := range regulations {
		_ = copier.Copy(&regulationList[i], &regulation)
	}
	return regulationList, count, err
}

func (r *ruleGoRepo) FindAllRegulation(ctx context.Context, where map[string]interface{}) ([]entity.Regulation, error) {
	info := dao.NewRegulation()
	regulations, err := info.FindAll(ctx, where)
	if err != nil {
		return nil, err
	}
	regulationList := make([]entity.Regulation, len(regulations))
	for i, regulation := range regulations {
		_ = copier.Copy(&regulationList[i], &regulation)
	}
	return regulationList, err
}

// RunLog Methods
func (r *ruleGoRepo) CreateRunLog(ctx context.Context, runLog *entity.RunLog) error {
	t := time.Now()
	info := dao.RunLog{}
	_ = copier.Copy(&info, runLog)
	info.CreatedAt = &t
	info.UpdatedAt = &t
	return info.Create(ctx)
}

func (r *ruleGoRepo) UpdateRunLog(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error {
	info := dao.NewRunLog()
	return info.Updates(ctx, date, where)
}

func (r *ruleGoRepo) DeleteRunLog(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewRunLog()
	return info.Delete(ctx, where)
}

func (r *ruleGoRepo) FindOneRunLog(ctx context.Context, where map[string]interface{}) (*entity.RunLog, error) {
	info := dao.NewRunLog()
	result, err := info.FindOne(ctx, where)
	if err != nil {
		return nil, err
	}
	res := entity.RunLog{}
	_ = copier.Copy(&res, result)
	return &res, err
}

func (r *ruleGoRepo) FindListRunLog(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.RunLog, int64, error) {
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

func (r *ruleGoRepo) FindAllRunLog(ctx context.Context, where map[string]interface{}) ([]entity.RunLog, error) {
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

// MdWorkflow Methods
func (r *ruleGoRepo) CreateMdWorkflow(ctx context.Context, mdWorkflow *entity.MdWorkflow) error {
	t := time.Now()
	info := dao.MdWorkflow{}
	_ = copier.Copy(&info, mdWorkflow)
	info.CreatedAt = &t
	info.UpdatedAt = &t
	return info.Create(ctx)
}

func (r *ruleGoRepo) UpdateMdWorkflow(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error {
	info := dao.NewMdWorkflow()
	return info.Updates(ctx, date, where)
}

func (r *ruleGoRepo) DeleteMdWorkflow(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewMdWorkflow()
	return info.Delete(ctx, where)
}

func (r *ruleGoRepo) FindOneMdWorkflow(ctx context.Context, where map[string]interface{}) (*entity.MdWorkflow, error) {
	info := dao.NewMdWorkflow()
	result, err := info.FindOne(ctx, where)
	if err != nil {
		return nil, err
	}
	res := entity.MdWorkflow{}
	_ = copier.Copy(&res, result)
	return &res, err
}

func (r *ruleGoRepo) FindListMdWorkflow(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.MdWorkflow, int64, error) {
	info := dao.NewMdWorkflow()
	results, count, err := info.FindList(ctx, where, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	resList := make([]entity.MdWorkflow, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, count, err
}

func (r *ruleGoRepo) FindAllMdWorkflow(ctx context.Context, where map[string]interface{}) ([]entity.MdWorkflow, error) {
	info := dao.NewMdWorkflow()
	results, err := info.FindAll(ctx, where)
	if err != nil {
		return nil, err
	}
	resList := make([]entity.MdWorkflow, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, err
}

// ComponentUseRule Methods
func (r *ruleGoRepo) CreateComponentUseRule(ctx context.Context, componentUseRule *entity.ComponentUseRule) error {
	t := time.Now()
	info := dao.ComponentUseRule{}
	_ = copier.Copy(&info, componentUseRule)
	info.CreatedAt = &t
	info.UpdatedAt = &t
	return info.Create(ctx)
}

func (r *ruleGoRepo) UpdateComponentUseRule(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error {
	info := dao.NewComponentUseRule()
	return info.Updates(ctx, date, where)
}

func (r *ruleGoRepo) DeleteComponentUseRule(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewComponentUseRule()
	return info.Delete(ctx, where)
}

func (r *ruleGoRepo) FindOneComponentUseRule(ctx context.Context, where map[string]interface{}) (*entity.ComponentUseRule, error) {
	info := dao.NewComponentUseRule()
	result, err := info.FindOne(ctx, where)
	if err != nil {
		return nil, err
	}
	res := entity.ComponentUseRule{}
	_ = copier.Copy(&res, result)
	return &res, err
}

func (r *ruleGoRepo) FindListComponentUseRule(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.ComponentUseRule, int64, error) {
	info := dao.NewComponentUseRule()
	results, count, err := info.FindList(ctx, where, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	resList := make([]entity.ComponentUseRule, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, count, err
}

func (r *ruleGoRepo) FindAllComponentUseRule(ctx context.Context, where map[string]interface{}) ([]entity.ComponentUseRule, error) {
	info := dao.NewComponentUseRule()
	results, err := info.FindAll(ctx, where)
	if err != nil {
		return nil, err
	}
	resList := make([]entity.ComponentUseRule, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, err
}

// ComponentRegulation Methods
func (r *ruleGoRepo) CreateComponentRegulation(ctx context.Context, componentRegulation *entity.ComponentRegulation) error {
	t := time.Now()
	info := dao.ComponentRegulation{}
	_ = copier.Copy(&info, componentRegulation)
	info.CreatedAt = &t
	info.UpdatedAt = &t
	return info.Create(ctx)
}

func (r *ruleGoRepo) UpdateComponentRegulation(ctx context.Context, where map[string]interface{}, date map[string]interface{}) error {
	info := dao.NewComponentRegulation()
	return info.Updates(ctx, date, where)
}

func (r *ruleGoRepo) DeleteComponentRegulation(ctx context.Context, where map[string]interface{}) error {
	info := dao.NewComponentRegulation()
	return info.Delete(ctx, where)
}

func (r *ruleGoRepo) FindOneComponentRegulation(ctx context.Context, where map[string]interface{}) (*entity.ComponentRegulation, error) {
	info := dao.NewComponentRegulation()
	result, err := info.FindOne(ctx, where)
	if err != nil {
		return nil, err
	}
	res := entity.ComponentRegulation{}
	_ = copier.Copy(&res, result)
	return &res, err
}

func (r *ruleGoRepo) FindListComponentRegulation(ctx context.Context, where map[string]interface{}, page int, pageSize int) ([]entity.ComponentRegulation, int64, error) {
	info := dao.NewComponentRegulation()
	results, count, err := info.FindList(ctx, where, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	resList := make([]entity.ComponentRegulation, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, count, err
}

func (r *ruleGoRepo) FindAllComponentRegulation(ctx context.Context, where map[string]interface{}) ([]entity.ComponentRegulation, error) {
	info := dao.NewComponentRegulation()
	results, err := info.FindAll(ctx, where)
	if err != nil {
		return nil, err
	}
	resList := make([]entity.ComponentRegulation, len(results))
	for i, v := range results {
		_ = copier.Copy(&resList[i], &v)
	}
	return resList, err
}
