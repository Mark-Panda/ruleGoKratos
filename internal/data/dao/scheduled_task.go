package dao

import (
	"context"
	"time"
)

type ScheduledTask struct {
	ID             int64  `gorm:"primaryKey;column:id;comment:定时任务ID"`
	Name           string `gorm:"column:name;size:255;not null;comment:任务名称"`
	Description    string `gorm:"column:description;type:text;comment:任务描述"`
	RuleChainID    string `gorm:"column:rule_chain_id;size:255;not null;index;comment:绑定规则链ID"`
	CronExpr       string `gorm:"column:cron_expr;size:255;not null;comment:cron表达式"`
	ScheduleType   string `gorm:"column:schedule_type;size:64;not null;comment:可视化配置类型"`
	ScheduleConfig string `gorm:"column:schedule_config;type:text;comment:可视化配置JSON"`
	// 创建默认关闭；启用任务应通过 Update 写入 disabled=false，避免 bool 零值与 default tag 产生歧义。
	// disabled 复合过滤索引由 sql/scheduled_task.sql 维护，避免 AutoMigrate 创建重复单列短索引。
	Disabled   bool       `gorm:"column:disabled;not null;default:true;comment:是否关闭"`
	LastRunAt  *time.Time `gorm:"column:last_run_at;comment:最近运行时间"`
	LastStatus int32      `gorm:"column:last_status;comment:最近运行结果 1:成功 2:失败"`
	LastError  string     `gorm:"column:last_error;type:text;comment:最近失败原因"`
	CreatedAt  time.Time  `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;comment:更新时间"`
	DeletedAt       *time.Time `gorm:"column:deleted_at;index;comment:删除时间"`
	PayloadTemplate string     `gorm:"column:payload_template;type:text;comment:用户自定义触发payload模板"`
}

func (ScheduledTask) TableName() string {
	return "scheduled_tasks"
}

func NewScheduledTask() *ScheduledTask {
	return &ScheduledTask{}
}

func (t *ScheduledTask) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(t).Error
}

func (t *ScheduledTask) GetByID(ctx context.Context, id int64) (*ScheduledTask, error) {
	var task ScheduledTask
	err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (t *ScheduledTask) List(ctx context.Context, name, ruleChainID string, disabled *bool, page, pageSize int32) ([]*ScheduledTask, int64, error) {
	var tasks []*ScheduledTask
	var count int64
	query := db.WithContext(ctx).Model(t).Where("deleted_at IS NULL")
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if ruleChainID != "" {
		query = query.Where("rule_chain_id = ?", ruleChainID)
	}
	if disabled != nil {
		query = query.Where("disabled = ?", *disabled)
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Order("created_at desc, id desc").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, count, nil
}

func (t *ScheduledTask) ListEnabled(ctx context.Context) ([]*ScheduledTask, error) {
	var tasks []*ScheduledTask
	err := db.WithContext(ctx).Where("deleted_at IS NULL AND disabled = ?", false).Order("created_at desc, id desc").Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (t *ScheduledTask) Update(ctx context.Context, id int64, data map[string]interface{}) error {
	return db.WithContext(ctx).Model(t).Where("id = ? AND deleted_at IS NULL", id).Updates(data).Error
}

func (t *ScheduledTask) Delete(ctx context.Context, id int64) error {
	return db.WithContext(ctx).Model(t).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", time.Now()).Error
}

type ScheduledTaskRun struct {
	ID int64 `gorm:"primaryKey;column:id;comment:执行历史ID"`
	// 执行历史按 task_id + created_at/id 排序查询，复合排序索引由 sql/scheduled_task.sql 维护。
	TaskID         int64     `gorm:"column:task_id;not null;comment:定时任务ID"`
	RuleChainID    string    `gorm:"column:rule_chain_id;size:255;not null;comment:触发规则链ID"`
	Status         int32     `gorm:"column:status;not null;comment:执行结果 1:成功 2:失败"`
	TriggerPayload string    `gorm:"column:trigger_payload;type:text;comment:触发payload"`
	ErrorMessage   string    `gorm:"column:error_message;type:text;comment:失败原因"`
	StartedAt      time.Time `gorm:"column:started_at;comment:开始时间"`
	FinishedAt     time.Time `gorm:"column:finished_at;comment:结束时间"`
	CreatedAt      time.Time `gorm:"column:created_at;comment:创建时间"`
}

func (ScheduledTaskRun) TableName() string {
	return "scheduled_task_runs"
}

func NewScheduledTaskRun() *ScheduledTaskRun {
	return &ScheduledTaskRun{}
}

func (r *ScheduledTaskRun) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(r).Error
}

func (r *ScheduledTaskRun) ListByTaskID(ctx context.Context, taskID int64, page, pageSize int32) ([]*ScheduledTaskRun, int64, error) {
	var runs []*ScheduledTaskRun
	var count int64
	query := db.WithContext(ctx).Model(r).Where("task_id = ?", taskID)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Order("created_at desc, id desc").Find(&runs).Error; err != nil {
		return nil, 0, err
	}
	return runs, count, nil
}
