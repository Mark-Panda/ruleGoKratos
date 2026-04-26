package service

import (
	"context"
	"testing"
	"time"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz/entity"
)

func TestScheduledTaskServiceCreateForwardsFields(t *testing.T) {
	createdAt := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	fake := &fakeScheduledTaskUsecase{
		createTask: &entity.ScheduledTask{
			ID:             101,
			Name:           "nightly",
			Description:    "nightly run",
			RuleChainID:    "rc-1",
			CronExpr:       "0 0 * * *",
			ScheduleType:   "cron",
			ScheduleConfig: `{"tz":"UTC"}`,
			Disabled:       true,
			CreatedAt:      createdAt,
			UpdatedAt:      createdAt,
		},
	}
	svc := newScheduledTaskServiceForTest(fake)

	reply, err := svc.CreateScheduledTask(context.Background(), &v1.CreateScheduledTaskReq{
		Name:           "nightly",
		Description:    "nightly run",
		RuleChainId:    "rc-1",
		CronExpr:       "0 0 * * *",
		ScheduleType:   "cron",
		ScheduleConfig: `{"tz":"UTC"}`,
	})
	if err != nil {
		t.Fatalf("CreateScheduledTask() error = %v", err)
	}

	want := createScheduledTaskCall{
		name:           "nightly",
		description:    "nightly run",
		ruleChainID:    "rc-1",
		cronExpr:       "0 0 * * *",
		scheduleType:   "cron",
		scheduleConfig: `{"tz":"UTC"}`,
	}
	if fake.createCall != want {
		t.Fatalf("CreateScheduledTask forwarded %#v, want %#v", fake.createCall, want)
	}
	if got := reply.GetTask().GetId(); got != 101 {
		t.Fatalf("reply task id = %d, want 101", got)
	}
}

func TestScheduledTaskServiceListPreservesOptionalDisabled(t *testing.T) {
	fake := &fakeScheduledTaskUsecase{}
	svc := newScheduledTaskServiceForTest(fake)

	if _, err := svc.ListScheduledTasks(context.Background(), &v1.ListScheduledTasksReq{
		Name:        "night",
		RuleChainId: "rc-1",
		Page:        2,
		PageSize:    20,
	}); err != nil {
		t.Fatalf("ListScheduledTasks(nil disabled) error = %v", err)
	}

	disabled := false
	if _, err := svc.ListScheduledTasks(context.Background(), &v1.ListScheduledTasksReq{
		Disabled: &disabled,
	}); err != nil {
		t.Fatalf("ListScheduledTasks(false disabled) error = %v", err)
	}

	if len(fake.listCalls) != 2 {
		t.Fatalf("ListScheduledTasks calls = %d, want 2", len(fake.listCalls))
	}
	if fake.listCalls[0].disabled != nil {
		t.Fatalf("first disabled = %v, want nil", *fake.listCalls[0].disabled)
	}
	if fake.listCalls[0].name != "night" || fake.listCalls[0].ruleChainID != "rc-1" || fake.listCalls[0].page != 2 || fake.listCalls[0].pageSize != 20 {
		t.Fatalf("first list call = %#v", fake.listCalls[0])
	}
	if fake.listCalls[1].disabled == nil {
		t.Fatal("second disabled = nil, want pointer to false")
	}
	if *fake.listCalls[1].disabled {
		t.Fatal("second disabled = true, want false")
	}
}

func TestScheduledTaskServiceEnableDisableForwardToUsecase(t *testing.T) {
	fake := &fakeScheduledTaskUsecase{
		enableTask:  &entity.ScheduledTask{ID: 7},
		disableTask: &entity.ScheduledTask{ID: 8},
	}
	svc := newScheduledTaskServiceForTest(fake)

	enableReply, err := svc.EnableScheduledTask(context.Background(), &v1.EnableScheduledTaskReq{Id: 7})
	if err != nil {
		t.Fatalf("EnableScheduledTask() error = %v", err)
	}
	disableReply, err := svc.DisableScheduledTask(context.Background(), &v1.DisableScheduledTaskReq{Id: 8})
	if err != nil {
		t.Fatalf("DisableScheduledTask() error = %v", err)
	}

	if fake.enableID != 7 {
		t.Fatalf("EnableScheduledTask id = %d, want 7", fake.enableID)
	}
	if fake.disableID != 8 {
		t.Fatalf("DisableScheduledTask id = %d, want 8", fake.disableID)
	}
	if enableReply.GetTask().GetId() != 7 || disableReply.GetTask().GetId() != 8 {
		t.Fatalf("reply ids = %d/%d, want 7/8", enableReply.GetTask().GetId(), disableReply.GetTask().GetId())
	}
}

func TestScheduledTaskServiceMapsTaskTimestampsAndLastStatus(t *testing.T) {
	lastRunAt := time.Date(2026, 4, 26, 11, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 4, 26, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 4, 26, 9, 30, 0, 0, time.UTC)
	deletedAt := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	fake := &fakeScheduledTaskUsecase{
		getTask: &entity.ScheduledTask{
			ID:             11,
			Name:           "mapped",
			Description:    "desc",
			RuleChainID:    "rc-2",
			CronExpr:       "*/5 * * * *",
			ScheduleType:   "cron",
			ScheduleConfig: "{}",
			Disabled:       true,
			LastRunAt:      &lastRunAt,
			LastStatus:     entity.ScheduledTaskRunStatusFailed,
			LastError:      "boom",
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			DeletedAt:      &deletedAt,
		},
	}
	svc := newScheduledTaskServiceForTest(fake)

	reply, err := svc.GetScheduledTask(context.Background(), &v1.GetScheduledTaskReq{Id: 11})
	if err != nil {
		t.Fatalf("GetScheduledTask() error = %v", err)
	}

	got := reply.GetTask()
	if got.GetId() != 11 ||
		got.GetName() != "mapped" ||
		got.GetDescription() != "desc" ||
		got.GetRuleChainId() != "rc-2" ||
		got.GetCronExpr() != "*/5 * * * *" ||
		got.GetScheduleType() != "cron" ||
		got.GetScheduleConfig() != "{}" ||
		!got.GetDisabled() ||
		got.GetLastStatus() != v1.ScheduledTaskRunStatus_SCHEDULED_TASK_RUN_STATUS_FAILED ||
		got.GetLastError() != "boom" {
		t.Fatalf("mapped task scalar fields = %#v", got)
	}
	assertTimestampEqual(t, "last_run_at", got.GetLastRunAt(), lastRunAt)
	assertTimestampEqual(t, "created_at", got.GetCreatedAt(), createdAt)
	assertTimestampEqual(t, "updated_at", got.GetUpdatedAt(), updatedAt)
	assertTimestampEqual(t, "deleted_at", got.GetDeletedAt(), deletedAt)

	fake.getTask.LastRunAt = nil
	fake.getTask.DeletedAt = nil
	reply, err = svc.GetScheduledTask(context.Background(), &v1.GetScheduledTaskReq{Id: 11})
	if err != nil {
		t.Fatalf("GetScheduledTask(nil timestamps) error = %v", err)
	}
	if reply.GetTask().GetLastRunAt() != nil || reply.GetTask().GetDeletedAt() != nil {
		t.Fatalf("nil optional timestamps mapped to %v/%v, want nil/nil", reply.GetTask().GetLastRunAt(), reply.GetTask().GetDeletedAt())
	}
}

func TestScheduledTaskServiceListRunsMapsRunFields(t *testing.T) {
	startedAt := time.Date(2026, 4, 26, 13, 0, 0, 0, time.UTC)
	finishedAt := time.Date(2026, 4, 26, 13, 1, 0, 0, time.UTC)
	createdAt := time.Date(2026, 4, 26, 13, 1, 1, 0, time.UTC)
	fake := &fakeScheduledTaskUsecase{
		listRuns: []*entity.ScheduledTaskRun{{
			ID:             201,
			TaskID:         101,
			RuleChainID:    "rc-1",
			Status:         entity.ScheduledTaskRunStatusSuccess,
			TriggerPayload: `{"trigger":"schedule"}`,
			ErrorMessage:   "",
			StartedAt:      startedAt,
			FinishedAt:     finishedAt,
			CreatedAt:      createdAt,
		}},
		listRunsTotal: 1,
	}
	svc := newScheduledTaskServiceForTest(fake)

	reply, err := svc.ListScheduledTaskRuns(context.Background(), &v1.ListScheduledTaskRunsReq{
		TaskId:   101,
		Page:     3,
		PageSize: 30,
	})
	if err != nil {
		t.Fatalf("ListScheduledTaskRuns() error = %v", err)
	}

	wantCall := listRunsCall{taskID: 101, page: 3, pageSize: 30}
	if fake.listRunsCall != wantCall {
		t.Fatalf("ListScheduledTaskRuns forwarded %#v, want %#v", fake.listRunsCall, wantCall)
	}
	if reply.GetTotal() != 1 || len(reply.GetRuns()) != 1 {
		t.Fatalf("runs total/len = %d/%d, want 1/1", reply.GetTotal(), len(reply.GetRuns()))
	}
	got := reply.GetRuns()[0]
	if got.GetId() != 201 ||
		got.GetTaskId() != 101 ||
		got.GetRuleChainId() != "rc-1" ||
		got.GetStatus() != v1.ScheduledTaskRunStatus_SCHEDULED_TASK_RUN_STATUS_SUCCESS ||
		got.GetTriggerPayload() != `{"trigger":"schedule"}` ||
		got.GetErrorMessage() != "" {
		t.Fatalf("mapped run scalar fields = %#v", got)
	}
	assertTimestampEqual(t, "started_at", got.GetStartedAt(), startedAt)
	assertTimestampEqual(t, "finished_at", got.GetFinishedAt(), finishedAt)
	assertTimestampEqual(t, "created_at", got.GetCreatedAt(), createdAt)
}

func newScheduledTaskServiceForTest(uc *fakeScheduledTaskUsecase) *ScheduledTaskService {
	return &ScheduledTaskService{uc: uc}
}

func assertTimestampEqual(t *testing.T, name string, got interface{ AsTime() time.Time }, want time.Time) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %v", name, want)
	}
	if !got.AsTime().Equal(want) {
		t.Fatalf("%s = %v, want %v", name, got.AsTime(), want)
	}
}

type fakeScheduledTaskUsecase struct {
	createCall createScheduledTaskCall
	createTask *entity.ScheduledTask

	getTask *entity.ScheduledTask

	listCalls []listScheduledTasksCall
	listTasks []*entity.ScheduledTask
	listTotal int64

	updateCall updateScheduledTaskCall
	updateTask *entity.ScheduledTask

	deleteID int64

	enableID   int64
	enableTask *entity.ScheduledTask

	disableID   int64
	disableTask *entity.ScheduledTask

	listRunsCall  listRunsCall
	listRuns      []*entity.ScheduledTaskRun
	listRunsTotal int64
}

func (f *fakeScheduledTaskUsecase) CreateScheduledTask(ctx context.Context, name, description, ruleChainID, cronExpr, scheduleType, scheduleConfig string) (*entity.ScheduledTask, error) {
	f.createCall = createScheduledTaskCall{name: name, description: description, ruleChainID: ruleChainID, cronExpr: cronExpr, scheduleType: scheduleType, scheduleConfig: scheduleConfig}
	return f.createTask, nil
}

func (f *fakeScheduledTaskUsecase) GetScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error) {
	return f.getTask, nil
}

func (f *fakeScheduledTaskUsecase) ListScheduledTasks(ctx context.Context, name, ruleChainID string, disabled *bool, page, pageSize int32) ([]*entity.ScheduledTask, int64, error) {
	f.listCalls = append(f.listCalls, listScheduledTasksCall{name: name, ruleChainID: ruleChainID, disabled: disabled, page: page, pageSize: pageSize})
	return f.listTasks, f.listTotal, nil
}

func (f *fakeScheduledTaskUsecase) UpdateScheduledTask(ctx context.Context, id int64, name, description, ruleChainID, cronExpr, scheduleType, scheduleConfig string) (*entity.ScheduledTask, error) {
	f.updateCall = updateScheduledTaskCall{id: id, name: name, description: description, ruleChainID: ruleChainID, cronExpr: cronExpr, scheduleType: scheduleType, scheduleConfig: scheduleConfig}
	return f.updateTask, nil
}

func (f *fakeScheduledTaskUsecase) DeleteScheduledTask(ctx context.Context, id int64) error {
	f.deleteID = id
	return nil
}

func (f *fakeScheduledTaskUsecase) EnableScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error) {
	f.enableID = id
	return f.enableTask, nil
}

func (f *fakeScheduledTaskUsecase) DisableScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error) {
	f.disableID = id
	return f.disableTask, nil
}

func (f *fakeScheduledTaskUsecase) ListScheduledTaskRuns(ctx context.Context, taskID int64, page, pageSize int32) ([]*entity.ScheduledTaskRun, int64, error) {
	f.listRunsCall = listRunsCall{taskID: taskID, page: page, pageSize: pageSize}
	return f.listRuns, f.listRunsTotal, nil
}

type createScheduledTaskCall struct {
	name           string
	description    string
	ruleChainID    string
	cronExpr       string
	scheduleType   string
	scheduleConfig string
}

type listScheduledTasksCall struct {
	name        string
	ruleChainID string
	disabled    *bool
	page        int32
	pageSize    int32
}

type updateScheduledTaskCall struct {
	id             int64
	name           string
	description    string
	ruleChainID    string
	cronExpr       string
	scheduleType   string
	scheduleConfig string
}

type listRunsCall struct {
	taskID   int64
	page     int32
	pageSize int32
}
