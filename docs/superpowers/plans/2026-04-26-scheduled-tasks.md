# Scheduled Tasks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增定时任务管理能力，支持菜单列表、cron 可视化配置、数据库持久化、启停、绑定已启用主规则链和执行历史。

**Architecture:** 后端沿用 Kratos 的 `proto -> service -> biz usecase -> data repo -> dao` 分层，新增单实例内置 cron scheduler，数据库作为任务配置事实源。前端沿用 `AdminPanel` hash 菜单和 Semi UI 列表表单模式，新增 cron 可视化工具函数和 `ScheduledTaskSection`。

**Tech Stack:** Go 1.24、Kratos、GORM、PostgreSQL、`github.com/robfig/cron/v3`、React 18、TypeScript、Semi UI、Vitest。

**Commit Policy:** 当前会话未收到提交请求；实现时不要执行 `git commit`，除非用户明确要求。

---

## File Structure

### Backend API

- Create: `api/rulego/v1/scheduled_task_service.proto`，定义 `ScheduledTaskService`、任务实体、执行历史实体、CRUD/启停/历史查询请求响应。
- Generate: `api/rulego/v1/scheduled_task_service.pb.go`、`api/rulego/v1/scheduled_task_service_http.pb.go`、`api/rulego/v1/scheduled_task_service_grpc.pb.go`，通过 `make api` 与 `make validate` 生成。

### Backend Domain

- Create: `internal/biz/entity/scheduled_task.go`，定义业务实体、状态常量和 payload 构造。
- Create: `internal/biz/scheduled_task.go`，定义 repo/scheduler 接口和 usecase。
- Create: `internal/biz/scheduled_task_test.go`，覆盖默认关闭、开启校验、触发历史、规则链不可用自动关闭。
- Modify: `internal/biz/biz.go`，注册 `NewScheduledTaskUsecase` 与 `NewScheduledTaskScheduler`。

### Backend Scheduler

- Create: `internal/biz/scheduled_task_scheduler.go`，封装 `robfig/cron/v3`，提供注册、移除、启动、停止。
- Create: `internal/biz/scheduled_task_scheduler_test.go`，覆盖替换 job、移除 job、非法 cron 报错。

### Backend Data

- Create: `internal/data/dao/scheduled_task.go`，定义 GORM 模型和 CRUD/历史查询方法。
- Create: `internal/data/scheduled_task.go`，实现 `biz.ScheduledTaskRepo`。
- Create: `internal/data/scheduled_task_test.go`，使用 sqlite 或现有数据测试模式验证 DAO 映射。
- Modify: `internal/data/dao/dao.go`，加入迁移。
- Modify: `internal/data/data.go`，注册 `NewScheduledTaskRepo`。
- Create: `sql/scheduled_task.sql`，提供建表 SQL。

### Backend Service And Wiring

- Create: `internal/service/scheduled_task_service.go`，实现 proto 生成的 server 接口。
- Create: `internal/service/scheduled_task_service_test.go`，覆盖 service 到 usecase 的字段映射。
- Modify: `internal/service/service.go`，注册 `NewScheduledTaskService`。
- Modify: `internal/server/http.go`，注册 `v1.RegisterScheduledTaskServiceHTTPServer`。
- Modify: `internal/server/grpc.go`，注册 `v1.RegisterScheduledTaskServiceServer`。
- Regenerate: `cmd/ruleGoKratos/wire_gen.go`，通过 `make wire` 生成。

### Frontend

- Create: `flowgram/src/services/api-scheduled-task.ts`，封装任务与执行历史 API。
- Create: `flowgram/src/management/sections/scheduled-task-cron.ts`，封装 cron 可视化配置转换。
- Create: `flowgram/src/management/sections/scheduled-task-cron.test.ts`，覆盖 cron 转换和回显。
- Create: `flowgram/src/management/sections/ScheduledTaskSection.tsx`，实现列表、表单、启停、历史抽屉。
- Modify: `flowgram/src/management/admin-panel.tsx`，新增“定时任务”菜单。

---

## Task 1: Backend API Contract

**Files:**
- Create: `api/rulego/v1/scheduled_task_service.proto`
- Generate: `api/rulego/v1/scheduled_task_service.pb.go`
- Generate: `api/rulego/v1/scheduled_task_service_http.pb.go`
- Generate: `api/rulego/v1/scheduled_task_service_grpc.pb.go`

- [ ] **Step 1: Write proto contract**

Create `api/rulego/v1/scheduled_task_service.proto` with this contract shape:

```proto
syntax = "proto3";

package rulego.v1;

import "google/api/annotations.proto";
import "google/protobuf/timestamp.proto";
import "validate/validate.proto";
import "openapi/v3/annotations.proto";

option go_package = "ruleGoKratos/api/rulego/v1;v1";

enum ScheduledTaskRunStatus {
  SCHEDULED_TASK_RUN_STATUS_UNSPECIFIED = 0;
  SCHEDULED_TASK_RUN_STATUS_SUCCESS = 1;
  SCHEDULED_TASK_RUN_STATUS_FAILED = 2;
}

message ScheduledTask {
  int64 id = 1;
  string name = 2;
  string description = 3;
  string rule_chain_id = 4;
  string cron_expr = 5;
  string schedule_type = 6;
  string schedule_config = 7;
  bool disabled = 8;
  google.protobuf.Timestamp last_run_at = 9;
  ScheduledTaskRunStatus last_status = 10;
  string last_error = 11;
  google.protobuf.Timestamp created_at = 12;
  google.protobuf.Timestamp updated_at = 13;
  google.protobuf.Timestamp deleted_at = 14;
}

message ScheduledTaskRun {
  int64 id = 1;
  int64 task_id = 2;
  string rule_chain_id = 3;
  ScheduledTaskRunStatus status = 4;
  string trigger_payload = 5;
  string error_message = 6;
  google.protobuf.Timestamp started_at = 7;
  google.protobuf.Timestamp finished_at = 8;
  google.protobuf.Timestamp created_at = 9;
}

service ScheduledTaskService {
  rpc CreateScheduledTask(CreateScheduledTaskReq) returns (CreateScheduledTaskReply) {
    option (google.api.http) = { post: "/api/v1/scheduled-tasks" body: "*" };
    option (openapi.v3.operation) = { operation_id: "CreateScheduledTask" tags: "ScheduledTask" summary: "创建定时任务" };
  }
  rpc GetScheduledTask(GetScheduledTaskReq) returns (GetScheduledTaskReply) {
    option (google.api.http) = { get: "/api/v1/scheduled-tasks/{id}" };
  }
  rpc ListScheduledTasks(ListScheduledTasksReq) returns (ListScheduledTasksReply) {
    option (google.api.http) = { get: "/api/v1/scheduled-tasks" };
  }
  rpc UpdateScheduledTask(UpdateScheduledTaskReq) returns (UpdateScheduledTaskReply) {
    option (google.api.http) = { put: "/api/v1/scheduled-tasks/{id}" body: "*" };
  }
  rpc DeleteScheduledTask(DeleteScheduledTaskReq) returns (DeleteScheduledTaskReply) {
    option (google.api.http) = { delete: "/api/v1/scheduled-tasks/{id}" };
  }
  rpc EnableScheduledTask(EnableScheduledTaskReq) returns (EnableScheduledTaskReply) {
    option (google.api.http) = { post: "/api/v1/scheduled-tasks/{id}/enable" body: "*" };
  }
  rpc DisableScheduledTask(DisableScheduledTaskReq) returns (DisableScheduledTaskReply) {
    option (google.api.http) = { post: "/api/v1/scheduled-tasks/{id}/disable" body: "*" };
  }
  rpc ListScheduledTaskRuns(ListScheduledTaskRunsReq) returns (ListScheduledTaskRunsReply) {
    option (google.api.http) = { get: "/api/v1/scheduled-tasks/{task_id}/runs" };
  }
}

message CreateScheduledTaskReq {
  string name = 1 [(validate.rules).string.min_len = 1];
  string description = 2;
  string rule_chain_id = 3 [(validate.rules).string.min_len = 1];
  string cron_expr = 4 [(validate.rules).string.min_len = 1];
  string schedule_type = 5 [(validate.rules).string.min_len = 1];
  string schedule_config = 6;
}

message CreateScheduledTaskReply { ScheduledTask task = 1; }
message GetScheduledTaskReq { int64 id = 1 [(validate.rules).int64.gt = 0]; }
message GetScheduledTaskReply { ScheduledTask task = 1; }

message ListScheduledTasksReq {
  string name = 1;
  string rule_chain_id = 2;
  optional bool disabled = 3;
  int32 page = 4;
  int32 page_size = 5;
}
message ListScheduledTasksReply {
  repeated ScheduledTask tasks = 1;
  int64 total = 2;
}

message UpdateScheduledTaskReq {
  int64 id = 1 [(validate.rules).int64.gt = 0];
  string name = 2;
  string description = 3;
  string rule_chain_id = 4;
  string cron_expr = 5;
  string schedule_type = 6;
  string schedule_config = 7;
}
message UpdateScheduledTaskReply { ScheduledTask task = 1; }

message DeleteScheduledTaskReq { int64 id = 1 [(validate.rules).int64.gt = 0]; }
message DeleteScheduledTaskReply {}
message EnableScheduledTaskReq { int64 id = 1 [(validate.rules).int64.gt = 0]; }
message EnableScheduledTaskReply { ScheduledTask task = 1; }
message DisableScheduledTaskReq { int64 id = 1 [(validate.rules).int64.gt = 0]; }
message DisableScheduledTaskReply { ScheduledTask task = 1; }

message ListScheduledTaskRunsReq {
  int64 task_id = 1 [(validate.rules).int64.gt = 0];
  int32 page = 2;
  int32 page_size = 3;
}
message ListScheduledTaskRunsReply {
  repeated ScheduledTaskRun runs = 1;
  int64 total = 2;
}
```

- [ ] **Step 2: Generate API code**

Run:

```bash
make api && make validate
```

Expected: generated `scheduled_task_service*.pb.go` files appear under `api/rulego/v1/` and command exits with code 0.

---

## Task 2: Backend Entities, DAO, Repo

**Files:**
- Create: `internal/biz/entity/scheduled_task.go`
- Create: `internal/data/dao/scheduled_task.go`
- Create: `internal/data/scheduled_task.go`
- Create: `sql/scheduled_task.sql`
- Modify: `internal/data/dao/dao.go`
- Modify: `internal/data/data.go`
- Test: `internal/data/scheduled_task_test.go`

- [ ] **Step 1: Add entity types**

Create `internal/biz/entity/scheduled_task.go`:

```go
package entity

import "time"

const (
	ScheduledTaskRunStatusSuccess int32 = 1
	ScheduledTaskRunStatusFailed  int32 = 2
)

type ScheduledTask struct {
	ID             int64
	Name           string
	Description    string
	RuleChainID    string
	CronExpr       string
	ScheduleType   string
	ScheduleConfig string
	Disabled       bool
	LastRunAt      *time.Time
	LastStatus     int32
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type ScheduledTaskRun struct {
	ID             int64
	TaskID         int64
	RuleChainID    string
	Status         int32
	TriggerPayload string
	ErrorMessage   string
	StartedAt      time.Time
	FinishedAt     time.Time
	CreatedAt      time.Time
}

func NewScheduledTriggerPayload(taskID int64) string {
	return `{"trigger":"schedule","taskId":"` + strconv.FormatInt(taskID, 10) + `"}`
}
```

After adding this file, include `strconv` in the import block:

```go
import (
	"strconv"
	"time"
)
```

- [ ] **Step 2: Write DAO test first**

Create `internal/data/scheduled_task_test.go` with sqlite setup and assertions:

```go
package data

import (
	"context"
	"testing"
	"time"

	"ruleGoKratos/internal/data/dao"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScheduledTaskDAOCreateListAndRuns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	dao.Init(db)

	taskDAO := dao.NewScheduledTask()
	task := &dao.ScheduledTask{
		Name:           "每 5 分钟同步",
		RuleChainID:    "chain-root",
		CronExpr:       "*/5 * * * *",
		ScheduleType:   "every_minutes",
		ScheduleConfig: `{"minutes":5}`,
		Disabled:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := task.Create(context.Background()); err != nil {
		t.Fatalf("create task: %v", err)
	}

	tasks, total, err := taskDAO.List(context.Background(), "", "", nil, 1, 10)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if total != 1 || len(tasks) != 1 || !tasks[0].Disabled {
		t.Fatalf("unexpected list result total=%d len=%d disabled=%v", total, len(tasks), tasks[0].Disabled)
	}

	run := &dao.ScheduledTaskRun{
		TaskID:         task.ID,
		RuleChainID:    "chain-root",
		Status:         1,
		TriggerPayload: `{"trigger":"schedule","taskId":"1"}`,
		StartedAt:      time.Now(),
		FinishedAt:     time.Now(),
		CreatedAt:      time.Now(),
	}
	if err := run.Create(context.Background()); err != nil {
		t.Fatalf("create run: %v", err)
	}

	runs, runTotal, err := dao.NewScheduledTaskRun().ListByTaskID(context.Background(), task.ID, 1, 10)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if runTotal != 1 || len(runs) != 1 {
		t.Fatalf("unexpected run list total=%d len=%d", runTotal, len(runs))
	}
}
```

- [ ] **Step 3: Run test and verify failure**

Run:

```bash
go test ./internal/data -run TestScheduledTaskDAOCreateListAndRuns -count=1
```

Expected: FAIL because `dao.NewScheduledTask`, `dao.ScheduledTask`, and `dao.ScheduledTaskRun` are not defined.

- [ ] **Step 4: Implement DAO**

Create `internal/data/dao/scheduled_task.go`:

```go
package dao

import (
	"context"
	"time"
)

type ScheduledTask struct {
	ID             int64      `gorm:"primaryKey;column:id;comment:定时任务ID"`
	Name           string     `gorm:"column:name;size:255;not null;comment:任务名称"`
	Description    string     `gorm:"column:description;type:text;comment:任务描述"`
	RuleChainID    string     `gorm:"column:rule_chain_id;size:255;not null;index;comment:绑定规则链ID"`
	CronExpr       string     `gorm:"column:cron_expr;size:255;not null;comment:cron表达式"`
	ScheduleType   string     `gorm:"column:schedule_type;size:64;not null;comment:可视化配置类型"`
	ScheduleConfig string     `gorm:"column:schedule_config;type:text;comment:可视化配置JSON"`
	Disabled       bool       `gorm:"column:disabled;default:true;index;comment:是否关闭"`
	LastRunAt      *time.Time `gorm:"column:last_run_at;comment:最近运行时间"`
	LastStatus     int32      `gorm:"column:last_status;comment:最近运行结果 1:成功 2:失败"`
	LastError      string     `gorm:"column:last_error;type:text;comment:最近失败原因"`
	CreatedAt      time.Time  `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;comment:更新时间"`
	DeletedAt      *time.Time `gorm:"column:deleted_at;index;comment:删除时间"`
}

func (ScheduledTask) TableName() string { return "scheduled_tasks" }
func NewScheduledTask() *ScheduledTask  { return &ScheduledTask{} }

func (t *ScheduledTask) Create(ctx context.Context) error {
	return db.WithContext(ctx).Create(t).Error
}

func (t *ScheduledTask) GetByID(ctx context.Context, id int64) (*ScheduledTask, error) {
	var task ScheduledTask
	err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&task).Error
	return &task, err
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
	err := query.Offset(int(offset)).Limit(int(pageSize)).Order("created_at desc").Find(&tasks).Error
	return tasks, count, err
}

func (t *ScheduledTask) ListEnabled(ctx context.Context) ([]*ScheduledTask, error) {
	var tasks []*ScheduledTask
	err := db.WithContext(ctx).Where("deleted_at IS NULL AND disabled = ?", false).Find(&tasks).Error
	return tasks, err
}

func (t *ScheduledTask) Update(ctx context.Context, id int64, data map[string]interface{}) error {
	return db.WithContext(ctx).Model(t).Where("id = ? AND deleted_at IS NULL", id).Updates(data).Error
}

func (t *ScheduledTask) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	return db.WithContext(ctx).Model(t).Where("id = ?", id).Update("deleted_at", &now).Error
}

type ScheduledTaskRun struct {
	ID             int64     `gorm:"primaryKey;column:id;comment:执行历史ID"`
	TaskID         int64     `gorm:"column:task_id;not null;index;comment:定时任务ID"`
	RuleChainID    string    `gorm:"column:rule_chain_id;size:255;not null;comment:触发规则链ID"`
	Status         int32     `gorm:"column:status;not null;comment:执行结果 1:成功 2:失败"`
	TriggerPayload string    `gorm:"column:trigger_payload;type:text;comment:触发payload"`
	ErrorMessage   string    `gorm:"column:error_message;type:text;comment:失败原因"`
	StartedAt      time.Time `gorm:"column:started_at;comment:开始时间"`
	FinishedAt     time.Time `gorm:"column:finished_at;comment:结束时间"`
	CreatedAt      time.Time `gorm:"column:created_at;comment:创建时间"`
}

func (ScheduledTaskRun) TableName() string { return "scheduled_task_runs" }
func NewScheduledTaskRun() *ScheduledTaskRun { return &ScheduledTaskRun{} }

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
	err := query.Offset(int(offset)).Limit(int(pageSize)).Order("created_at desc").Find(&runs).Error
	return runs, count, err
}
```

- [ ] **Step 5: Register migration**

Modify `internal/data/dao/dao.go` inside `Init` to include:

```go
if err := db.AutoMigrate(&ScheduledTask{}, &ScheduledTaskRun{}); err != nil {
	panic(err)
}
```

- [ ] **Step 6: Add SQL script**

Create `sql/scheduled_task.sql` with tables matching the DAO:

```sql
CREATE TABLE IF NOT EXISTS scheduled_tasks (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  rule_chain_id VARCHAR(255) NOT NULL,
  cron_expr VARCHAR(255) NOT NULL,
  schedule_type VARCHAR(64) NOT NULL,
  schedule_config TEXT,
  disabled BOOLEAN NOT NULL DEFAULT TRUE,
  last_run_at TIMESTAMP,
  last_status INTEGER,
  last_error TEXT,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_rule_chain_id ON scheduled_tasks(rule_chain_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_disabled ON scheduled_tasks(disabled);
CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_deleted_at ON scheduled_tasks(deleted_at);

CREATE TABLE IF NOT EXISTS scheduled_task_runs (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL,
  rule_chain_id VARCHAR(255) NOT NULL,
  status INTEGER NOT NULL,
  trigger_payload TEXT,
  error_message TEXT,
  started_at TIMESTAMP NOT NULL,
  finished_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_scheduled_task_runs_task_id ON scheduled_task_runs(task_id);
```

- [ ] **Step 7: Implement data repo**

Create `internal/data/scheduled_task.go` implementing `biz.ScheduledTaskRepo` after Task 3 defines it. If Task 2 is implemented before Task 3, create this file in Task 3.

- [ ] **Step 8: Verify DAO test passes**

Run:

```bash
go test ./internal/data -run TestScheduledTaskDAOCreateListAndRuns -count=1
```

Expected: PASS.

---

## Task 3: Usecase And Scheduler Interfaces

**Files:**
- Create: `internal/biz/scheduled_task.go`
- Create: `internal/biz/scheduled_task_test.go`
- Complete: `internal/data/scheduled_task.go`
- Modify: `internal/data/data.go`
- Modify: `internal/biz/biz.go`

- [ ] **Step 1: Write usecase test with fakes**

Create `internal/biz/scheduled_task_test.go` with fake repo, fake scheduler, and fake rule chain validator/executor. Use these behaviors:

```go
func TestCreateScheduledTaskDefaultsDisabled(t *testing.T) {
	uc := newScheduledTaskUsecaseForTest()
	task, err := uc.CreateScheduledTask(context.Background(), "每日检查", "", "root-chain", "0 9 * * *", "daily", `{"hour":9,"minute":0}`)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if !task.Disabled {
		t.Fatalf("expected created task disabled by default")
	}
}

func TestEnableScheduledTaskRequiresEnabledRootRuleChain(t *testing.T) {
	uc := newScheduledTaskUsecaseForTest()
	uc.rules["root-chain"] = fakeRuleChainState{root: true, disabled: true, loaded: false}
	task := uc.repo.mustCreate("每日检查", "root-chain", "0 9 * * *")

	_, err := uc.EnableScheduledTask(context.Background(), task.ID)
	if err == nil || !strings.Contains(err.Error(), "已启用主规则链") {
		t.Fatalf("expected enabled root rule chain error, got %v", err)
	}
}

func TestRunScheduledTaskDisablesTaskWhenRuleChainUnavailable(t *testing.T) {
	uc := newScheduledTaskUsecaseForTest()
	uc.rules["root-chain"] = fakeRuleChainState{root: true, disabled: true, loaded: false}
	task := uc.repo.mustCreate("每日检查", "root-chain", "0 9 * * *")
	task.Disabled = false

	uc.runScheduledTask(context.Background(), task.ID)

	updated, _ := uc.repo.GetScheduledTask(context.Background(), task.ID)
	if !updated.Disabled {
		t.Fatalf("expected task disabled after unavailable rule chain")
	}
	runs, total, _ := uc.repo.ListScheduledTaskRuns(context.Background(), task.ID, 1, 10)
	if total != 1 || runs[0].Status != entity.ScheduledTaskRunStatusFailed {
		t.Fatalf("expected failed run history, total=%d runs=%v", total, runs)
	}
}
```

- [ ] **Step 2: Run test and verify failure**

Run:

```bash
go test ./internal/biz -run 'Test(CreateScheduledTaskDefaultsDisabled|EnableScheduledTaskRequiresEnabledRootRuleChain|RunScheduledTaskDisablesTaskWhenRuleChainUnavailable)' -count=1
```

Expected: FAIL because usecase and helper fakes are not implemented.

- [ ] **Step 3: Implement usecase interfaces and core methods**

Create `internal/biz/scheduled_task.go` with these interfaces:

```go
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
```

`ScheduledTaskUsecase` must provide:

```go
func NewScheduledTaskUsecase(repo ScheduledTaskRepo, ruleChain *RuleChainUsecase, scheduler ScheduledTaskScheduler) *ScheduledTaskUsecase
func (uc *ScheduledTaskUsecase) CreateScheduledTask(ctx context.Context, name, description, ruleChainID, cronExpr, scheduleType, scheduleConfig string) (*entity.ScheduledTask, error)
func (uc *ScheduledTaskUsecase) GetScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error)
func (uc *ScheduledTaskUsecase) ListScheduledTasks(ctx context.Context, name, ruleChainID string, disabled *bool, page, pageSize int32) ([]*entity.ScheduledTask, int64, error)
func (uc *ScheduledTaskUsecase) UpdateScheduledTask(ctx context.Context, id int64, name, description, ruleChainID, cronExpr, scheduleType, scheduleConfig string) (*entity.ScheduledTask, error)
func (uc *ScheduledTaskUsecase) DeleteScheduledTask(ctx context.Context, id int64) error
func (uc *ScheduledTaskUsecase) EnableScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error)
func (uc *ScheduledTaskUsecase) DisableScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error)
func (uc *ScheduledTaskUsecase) ListScheduledTaskRuns(ctx context.Context, taskID int64, page, pageSize int32) ([]*entity.ScheduledTaskRun, int64, error)
func (uc *ScheduledTaskUsecase) StartEnabledTasks(ctx context.Context) error
```

Validation rules:

```go
func (uc *ScheduledTaskUsecase) validateEnabledRootRuleChain(ctx context.Context, ruleChainID string) error {
	ruleChain, err := uc.ruleChain.GetRuleChain(ctx, ruleChainID)
	if err != nil {
		return fmt.Errorf("绑定规则链不存在或不可用: %w", err)
	}
	if !ruleChain.Root || ruleChain.Disabled {
		return fmt.Errorf("定时任务只能绑定已启用主规则链")
	}
	if _, ok := uc.ruleChain.ruleEngine.Get(ruleChainID); !ok {
		return fmt.Errorf("定时任务只能绑定已启用主规则链")
	}
	return nil
}
```

If direct access to `ruleEngine` is not suitable in tests, add a small method on `RuleChainUsecase`:

```go
func (s *RuleChainUsecase) IsRuleChainLoaded(id string) bool {
	_, ok := s.ruleEngine.Get(id)
	return ok
}
```

- [ ] **Step 4: Implement data repo mapping**

Create `internal/data/scheduled_task.go`:

```go
package data

import (
	"context"
	"time"

	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/data/dao"
)

type scheduledTaskRepo struct{}

var _ biz.ScheduledTaskRepo = (*scheduledTaskRepo)(nil)

func NewScheduledTaskRepo() biz.ScheduledTaskRepo { return &scheduledTaskRepo{} }
```

Add conversion functions `scheduledTaskDAOToEntity`, `scheduledTaskEntityToDAO`, `scheduledTaskRunDAOToEntity`, and `scheduledTaskRunEntityToDAO`. Preserve `LastRunAt`, `DeletedAt`, and all string fields exactly.

- [ ] **Step 5: Register providers**

Modify `internal/data/data.go` provider set:

```go
NewScheduledTaskRepo,
```

Modify `internal/biz/biz.go` provider set:

```go
NewScheduledTaskUsecase,
NewScheduledTaskScheduler,
```

- [ ] **Step 6: Verify tests pass**

Run:

```bash
go test ./internal/biz ./internal/data -run 'ScheduledTask' -count=1
```

Expected: PASS.

---

## Task 4: Cron Scheduler And Startup Restore

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `internal/biz/scheduled_task_scheduler.go`
- Create: `internal/biz/scheduled_task_scheduler_test.go`
- Modify: `internal/biz/scheduled_task.go`

- [ ] **Step 1: Add cron dependency**

Run:

```bash
go get github.com/robfig/cron/v3
```

Expected: `go.mod` and `go.sum` include `github.com/robfig/cron/v3`.

- [ ] **Step 2: Write scheduler tests**

Create tests for invalid cron and replace/remove behavior:

```go
func TestScheduledTaskSchedulerRejectsInvalidCron(t *testing.T) {
	s := NewScheduledTaskScheduler()
	if err := s.Add(1, "invalid cron", func() {}); err == nil {
		t.Fatalf("expected invalid cron error")
	}
}

func TestScheduledTaskSchedulerAddReplaceRemove(t *testing.T) {
	s := NewScheduledTaskScheduler()
	if err := s.Add(1, "*/5 * * * *", func() {}); err != nil {
		t.Fatalf("add cron: %v", err)
	}
	if err := s.Add(1, "*/10 * * * *", func() {}); err != nil {
		t.Fatalf("replace cron: %v", err)
	}
	s.Remove(1)
	s.Stop()
}
```

- [ ] **Step 3: Implement scheduler**

Create `internal/biz/scheduled_task_scheduler.go`:

```go
package biz

import (
	"sync"

	"github.com/robfig/cron/v3"
)

type cronScheduledTaskScheduler struct {
	mu      sync.Mutex
	cron    *cron.Cron
	entries map[int64]cron.EntryID
}

func NewScheduledTaskScheduler() ScheduledTaskScheduler {
	return &cronScheduledTaskScheduler{
		cron:    cron.New(),
		entries: make(map[int64]cron.EntryID),
	}
}

func (s *cronScheduledTaskScheduler) Add(taskID int64, cronExpr string, fn func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entryID, ok := s.entries[taskID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, taskID)
	}
	entryID, err := s.cron.AddFunc(cronExpr, fn)
	if err != nil {
		return err
	}
	s.entries[taskID] = entryID
	return nil
}

func (s *cronScheduledTaskScheduler) Remove(taskID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entryID, ok := s.entries[taskID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, taskID)
	}
}

func (s *cronScheduledTaskScheduler) Start() { s.cron.Start() }
func (s *cronScheduledTaskScheduler) Stop()  { s.cron.Stop() }
```

- [ ] **Step 4: Restore enabled tasks**

In `ScheduledTaskUsecase.StartEnabledTasks`, implement:

```go
tasks, err := uc.repo.ListEnabledScheduledTasks(ctx)
if err != nil {
	return err
}
for _, task := range tasks {
	taskID := task.ID
	if err := uc.scheduler.Add(taskID, task.CronExpr, func() {
		uc.runScheduledTask(context.Background(), taskID)
	}); err != nil {
		return fmt.Errorf("注册定时任务 %d 失败: %w", taskID, err)
	}
}
uc.scheduler.Start()
return nil
```

- [ ] **Step 5: Verify scheduler tests**

Run:

```bash
go test ./internal/biz -run 'TestScheduledTaskScheduler|ScheduledTask' -count=1
```

Expected: PASS.

---

## Task 5: Service Layer, Server Registration, Wire

**Files:**
- Create: `internal/service/scheduled_task_service.go`
- Create: `internal/service/scheduled_task_service_test.go`
- Modify: `internal/service/service.go`
- Modify: `internal/server/http.go`
- Modify: `internal/server/grpc.go`
- Regenerate: `cmd/ruleGoKratos/wire_gen.go`

- [ ] **Step 1: Implement service mapping**

Create `internal/service/scheduled_task_service.go`:

```go
package service

import (
	"context"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/biz/entity"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type ScheduledTaskService struct {
	v1.UnimplementedScheduledTaskServiceServer
	uc *biz.ScheduledTaskUsecase
}

func NewScheduledTaskService(uc *biz.ScheduledTaskUsecase) *ScheduledTaskService {
	return &ScheduledTaskService{uc: uc}
}

func scheduledTaskToProto(task *entity.ScheduledTask) *v1.ScheduledTask {
	if task == nil {
		return nil
	}
	reply := &v1.ScheduledTask{
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
		CreatedAt:      timestamppb.New(task.CreatedAt),
		UpdatedAt:      timestamppb.New(task.UpdatedAt),
	}
	if task.LastRunAt != nil {
		reply.LastRunAt = timestamppb.New(*task.LastRunAt)
	}
	if task.DeletedAt != nil {
		reply.DeletedAt = timestamppb.New(*task.DeletedAt)
	}
	return reply
}
```

Implement each RPC by forwarding to the corresponding usecase method and mapping entities to proto messages.

- [ ] **Step 2: Register provider**

Modify `internal/service/service.go`:

```go
var ProviderSet = wire.NewSet(
	NewPlaygroundService,
	NewRuleGoService,
	NewRunLogService,
	NewComponentService,
	NewAdminService,
	NewChatService,
	NewTaskBoardService,
	NewScheduledTaskService,
)
```

- [ ] **Step 3: Register HTTP and gRPC servers**

Modify `internal/server/http.go` function signature to include `scheduledTaskService *service.ScheduledTaskService`, then register:

```go
v1.RegisterScheduledTaskServiceHTTPServer(srv, scheduledTaskService)
```

Modify `internal/server/grpc.go` function signature to include `scheduledTaskService *service.ScheduledTaskService`, then register:

```go
v1.RegisterScheduledTaskServiceServer(srv, scheduledTaskService)
```

- [ ] **Step 4: Regenerate wire**

Run:

```bash
make wire
```

Expected: `cmd/ruleGoKratos/wire_gen.go` updates and command exits with code 0.

- [ ] **Step 5: Verify backend package tests**

Run:

```bash
go test ./internal/biz ./internal/data ./internal/service ./internal/server -run 'ScheduledTask|TestNew' -count=1
```

Expected: PASS or no tests for packages without matching tests.

---

## Task 6: Frontend API And Cron Utilities

**Files:**
- Create: `flowgram/src/services/api-scheduled-task.ts`
- Create: `flowgram/src/management/sections/scheduled-task-cron.ts`
- Create: `flowgram/src/management/sections/scheduled-task-cron.test.ts`

- [ ] **Step 1: Write cron utility tests**

Create `flowgram/src/management/sections/scheduled-task-cron.test.ts`:

```ts
import { describe, expect, it } from 'vitest';

import { buildCronExpr, describeSchedule } from './scheduled-task-cron';

describe('scheduled-task-cron', () => {
  it('builds every minutes cron', () => {
    expect(buildCronExpr({ type: 'every_minutes', minutes: 5 })).toBe('*/5 * * * *');
  });

  it('builds every hours cron', () => {
    expect(buildCronExpr({ type: 'every_hours', hours: 2 })).toBe('0 */2 * * *');
  });

  it('builds daily cron', () => {
    expect(buildCronExpr({ type: 'daily', hour: 9, minute: 30 })).toBe('30 9 * * *');
  });

  it('describes advanced cron', () => {
    expect(describeSchedule('advanced', '{}', '0 9 * * *')).toBe('0 9 * * *');
  });
});
```

- [ ] **Step 2: Run test and verify failure**

Run:

```bash
npm --prefix flowgram run test:unit -- scheduled-task-cron
```

Expected: FAIL because `scheduled-task-cron.ts` does not exist.

- [ ] **Step 3: Implement cron utility**

Create `flowgram/src/management/sections/scheduled-task-cron.ts`:

```ts
export type ScheduleType =
  | 'every_minutes'
  | 'every_hours'
  | 'daily'
  | 'weekly'
  | 'monthly'
  | 'advanced';

export type ScheduleConfig =
  | { type: 'every_minutes'; minutes: number }
  | { type: 'every_hours'; hours: number }
  | { type: 'daily'; hour: number; minute: number }
  | { type: 'weekly'; dayOfWeek: number; hour: number; minute: number }
  | { type: 'monthly'; dayOfMonth: number; hour: number; minute: number }
  | { type: 'advanced'; cronExpr: string };

export function buildCronExpr(config: ScheduleConfig): string {
  switch (config.type) {
    case 'every_minutes':
      return `*/${config.minutes} * * * *`;
    case 'every_hours':
      return `0 */${config.hours} * * *`;
    case 'daily':
      return `${config.minute} ${config.hour} * * *`;
    case 'weekly':
      return `${config.minute} ${config.hour} * * ${config.dayOfWeek}`;
    case 'monthly':
      return `${config.minute} ${config.hour} ${config.dayOfMonth} * *`;
    case 'advanced':
      return config.cronExpr.trim();
  }
}

export function parseScheduleConfig(scheduleConfig: string): Record<string, unknown> {
  try {
    return scheduleConfig ? JSON.parse(scheduleConfig) : {};
  } catch {
    return {};
  }
}

export function describeSchedule(scheduleType: string, scheduleConfig: string, cronExpr: string): string {
  const config = parseScheduleConfig(scheduleConfig);
  if (scheduleType === 'every_minutes') return `每 ${config.minutes ?? '-'} 分钟`;
  if (scheduleType === 'every_hours') return `每 ${config.hours ?? '-'} 小时`;
  if (scheduleType === 'daily') return `每天 ${config.hour ?? '-'}:${String(config.minute ?? 0).padStart(2, '0')}`;
  if (scheduleType === 'weekly') return `每周 ${config.dayOfWeek ?? '-'} ${config.hour ?? '-'}:${String(config.minute ?? 0).padStart(2, '0')}`;
  if (scheduleType === 'monthly') return `每月 ${config.dayOfMonth ?? '-'} 日 ${config.hour ?? '-'}:${String(config.minute ?? 0).padStart(2, '0')}`;
  return cronExpr;
}
```

- [ ] **Step 4: Implement frontend API service**

Create `flowgram/src/services/api-scheduled-task.ts` using `requestJSON` from `flowgram/src/services/http.ts`. Export:

```ts
export interface ScheduledTask {
  id: number;
  name: string;
  description?: string;
  ruleChainId: string;
  cronExpr: string;
  scheduleType: string;
  scheduleConfig: string;
  disabled: boolean;
  lastRunAt?: string;
  lastStatus?: 'SCHEDULED_TASK_RUN_STATUS_SUCCESS' | 'SCHEDULED_TASK_RUN_STATUS_FAILED' | number;
  lastError?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface ScheduledTaskRun {
  id: number;
  taskId: number;
  ruleChainId: string;
  status: 'SCHEDULED_TASK_RUN_STATUS_SUCCESS' | 'SCHEDULED_TASK_RUN_STATUS_FAILED' | number;
  triggerPayload: string;
  errorMessage?: string;
  startedAt?: string;
  finishedAt?: string;
  createdAt?: string;
}
```

Functions:

```ts
export function listScheduledTasks(params: Record<string, unknown>) {
  return requestJSON<{ tasks: ScheduledTask[]; total: number }>('/api/v1/scheduled-tasks', { query: params });
}
export function createScheduledTask(payload: Partial<ScheduledTask>) {
  return requestJSON<{ task: ScheduledTask }>('/api/v1/scheduled-tasks', { method: 'POST', body: payload });
}
export function updateScheduledTask(id: number, payload: Partial<ScheduledTask>) {
  return requestJSON<{ task: ScheduledTask }>(`/api/v1/scheduled-tasks/${id}`, { method: 'PUT', body: payload });
}
export function deleteScheduledTask(id: number) {
  return requestJSON<{}>(`/api/v1/scheduled-tasks/${id}`, { method: 'DELETE' });
}
export function enableScheduledTask(id: number) {
  return requestJSON<{ task: ScheduledTask }>(`/api/v1/scheduled-tasks/${id}/enable`, { method: 'POST', body: {} });
}
export function disableScheduledTask(id: number) {
  return requestJSON<{ task: ScheduledTask }>(`/api/v1/scheduled-tasks/${id}/disable`, { method: 'POST', body: {} });
}
export function listScheduledTaskRuns(taskId: number, params: Record<string, unknown>) {
  return requestJSON<{ runs: ScheduledTaskRun[]; total: number }>(`/api/v1/scheduled-tasks/${taskId}/runs`, { query: params });
}
```

- [ ] **Step 5: Verify frontend utility tests**

Run:

```bash
npm --prefix flowgram run test:unit -- scheduled-task-cron
```

Expected: PASS.

---

## Task 7: Frontend Scheduled Task Page And Menu

**Files:**
- Create: `flowgram/src/management/sections/ScheduledTaskSection.tsx`
- Modify: `flowgram/src/management/admin-panel.tsx`

- [ ] **Step 1: Implement section skeleton**

Create `ScheduledTaskSection.tsx` with state for list, pagination, modal, history drawer, and form. Use Semi UI components already present in `TaskBoardSection`:

```tsx
export const ScheduledTaskSection: React.FC = () => {
  const [tasks, setTasks] = useState<ScheduledTask[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);

  const loadTasks = useCallback(async () => {
    setLoading(true);
    try {
      const reply = await listScheduledTasks({ page, pageSize });
      setTasks(reply.tasks ?? []);
      setTotal(reply.total ?? 0);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize]);

  useEffect(() => {
    void loadTasks();
  }, [loadTasks]);

  return (
    <div>
      <Button theme="solid" onClick={() => openCreateModal()}>
        新建定时任务
      </Button>
      <Table dataSource={tasks} loading={loading} pagination={{ currentPage: page, pageSize, total }} />
    </div>
  );
};
```

- [ ] **Step 2: Add table columns and actions**

Columns must include:

```tsx
<Table.Column title="任务名称" dataIndex="name" />
<Table.Column title="绑定主规则链" dataIndex="ruleChainId" />
<Table.Column title="执行周期" render={(_, record) => describeSchedule(record.scheduleType, record.scheduleConfig, record.cronExpr)} />
<Table.Column title="状态" render={(_, record) => (record.disabled ? <Tag>关闭</Tag> : <Tag color="green">开启</Tag>)} />
<Table.Column title="最近运行时间" dataIndex="lastRunAt" />
<Table.Column title="最近结果" dataIndex="lastStatus" />
<Table.Column title="最近错误" dataIndex="lastError" />
```

Actions must call `enableScheduledTask`, `disableScheduledTask`, `deleteScheduledTask`, and open history drawer.

- [ ] **Step 3: Add create/edit form**

The modal form fields:

- `name`
- `description`
- `ruleChainId`
- `scheduleType`
- schedule-specific fields
- advanced `cronExpr`

On submit:

```ts
const cronExpr = buildCronExpr(scheduleConfig);
const payload = {
  name: values.name,
  description: values.description,
  ruleChainId: values.ruleChainId,
  cronExpr,
  scheduleType: scheduleConfig.type,
  scheduleConfig: JSON.stringify(scheduleConfig),
};
```

- [ ] **Step 4: Add history drawer**

Use `listScheduledTaskRuns(task.id, { page, pageSize })` and render execution time, status, error message, trigger payload.

- [ ] **Step 5: Wire menu**

Modify `flowgram/src/management/admin-panel.tsx`:

```ts
import { ScheduledTaskSection } from './sections/ScheduledTaskSection';
```

Add menu key:

```ts
| 'scheduled-tasks'
```

Add hash:

```ts
if (h.startsWith('#/scheduled-tasks')) return 'scheduled-tasks';
else if (key === 'scheduled-tasks') window.location.hash = '#/scheduled-tasks';
```

Add render/title/parent:

```tsx
if (key === 'scheduled-tasks') return <ScheduledTaskSection />;
case 'scheduled-tasks':
  return '定时任务';
if (activeMenu === 'scheduled-tasks') return '运维';
```

Add Nav item under 运维:

```tsx
{ itemKey: 'scheduled-tasks', text: '定时任务', icon: <IconList /> }
```

- [ ] **Step 6: Verify TypeScript**

Run:

```bash
npm --prefix flowgram run ts-check
```

Expected: PASS.

---

## Task 8: Full Verification

**Files:**
- All files touched by previous tasks.

- [ ] **Step 1: Generate and wire check**

Run:

```bash
make api && make validate && make wire
```

Expected: all commands exit with code 0.

- [ ] **Step 2: Backend tests**

Run:

```bash
go test ./internal/biz ./internal/data ./internal/service ./internal/server -count=1
```

Expected: PASS.

- [ ] **Step 3: Frontend tests and typecheck**

Run:

```bash
npm --prefix flowgram run test:unit -- scheduled-task-cron && npm --prefix flowgram run ts-check
```

Expected: PASS.

- [ ] **Step 4: Whole repository sanity check**

Run:

```bash
go test ./...
```

Expected: PASS. If unrelated existing tests fail, capture the failing package and error, then verify all `ScheduledTask`-related tests pass.

---

## Self-Review

- Spec coverage: plan includes menu, CRUD, cron visual config, DB persistence, default disabled state, enable/disable, binding enabled root rule chain, fixed trigger payload, execution history, startup restore, and auto-disable on unavailable rule chain.
- Placeholder scan: 未发现未完成标记或延后补充标记。
- Type consistency: backend uses `ScheduledTask`, `ScheduledTaskRun`, `ScheduledTaskUsecase`, `ScheduledTaskRepo`, and `ScheduledTaskScheduler` consistently; frontend uses `ScheduledTask`, `ScheduledTaskRun`, `ScheduleType`, and `ScheduleConfig` consistently.
