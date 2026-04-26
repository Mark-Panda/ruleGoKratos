package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz/entity"

	klog "github.com/go-kratos/kratos/v2/log"
)

func TestScheduledTaskCreateDefaultsToDisabled(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	uc := &ScheduledTaskUsecase{repo: repo}

	task, err := uc.CreateScheduledTask(context.Background(), "daily", "desc", "chain-root", "0 8 * * *", "daily", `{"hour":8}`)
	if err != nil {
		t.Fatalf("CreateScheduledTask returned error: %v", err)
	}

	if task.ID == 0 {
		t.Fatalf("created task should get an ID")
	}
	if !task.Disabled {
		t.Fatalf("created task should default to disabled")
	}
	if task.CreatedAt.IsZero() || task.UpdatedAt.IsZero() {
		t.Fatalf("created task should have timestamps: %#v", task)
	}
	if repo.createdTask == nil || !repo.createdTask.Disabled {
		t.Fatalf("repo should receive disabled task: %#v", repo.createdTask)
	}
}

func TestScheduledTaskListDefaultsPagination(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	uc := &ScheduledTaskUsecase{repo: repo}

	_, _, err := uc.ListScheduledTasks(context.Background(), "", "", nil, 0, 0)
	if err != nil {
		t.Fatalf("ListScheduledTasks returned error: %v", err)
	}

	if repo.lastListPage != 1 || repo.lastListPageSize != 10 {
		t.Fatalf("default pagination mismatch page=%d pageSize=%d", repo.lastListPage, repo.lastListPageSize)
	}
}

func TestScheduledTaskEnableRequiresUsableRootRuleChain(t *testing.T) {
	testCases := []struct {
		name      string
		ruleRoot  bool
		disabled  bool
		loaded    bool
		wantError string
	}{
		{name: "not root", ruleRoot: false, disabled: false, loaded: true, wantError: "主规则链"},
		{name: "disabled", ruleRoot: true, disabled: true, loaded: true, wantError: "启用"},
		{name: "not loaded", ruleRoot: true, disabled: false, loaded: false, wantError: "未加载"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeScheduledTaskRepo()
			repo.tasks[1] = &entity.ScheduledTask{ID: 1, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: true}
			ruleChain := &fakeScheduledTaskRuleChain{
				chains: map[string]*entity.RuleChain{
					"chain-root": {RuleChainID: "chain-root", Root: tc.ruleRoot, Disabled: tc.disabled},
				},
				loaded: map[string]bool{"chain-root": tc.loaded},
			}
			uc := &ScheduledTaskUsecase{repo: repo, ruleChain: ruleChain, scheduler: &fakeScheduledTaskScheduler{}}

			_, err := uc.EnableScheduledTask(context.Background(), 1)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("EnableScheduledTask error=%v, want contains %q", err, tc.wantError)
			}
			if !repo.tasks[1].Disabled {
				t.Fatalf("task should remain disabled after validation failure")
			}
			if len(repo.updatedTasks) != 0 {
				t.Fatalf("repo should not update task on validation failure")
			}
		})
	}
}

func TestScheduledTaskEnableSchedulerAddFailureDoesNotEnable(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[1] = &entity.ScheduledTask{ID: 1, RuleChainID: "chain-root", CronExpr: "bad cron", Disabled: true}
	scheduler := &fakeScheduledTaskScheduler{addErr: errors.New("invalid cron")}
	uc := &ScheduledTaskUsecase{
		repo:      repo,
		ruleChain: newUsableFakeRuleChain("chain-root"),
		scheduler: scheduler,
	}

	_, err := uc.EnableScheduledTask(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "invalid cron") {
		t.Fatalf("EnableScheduledTask error=%v, want scheduler error", err)
	}
	if !repo.tasks[1].Disabled {
		t.Fatalf("task should remain disabled when scheduler add fails")
	}
	if len(repo.updatedTasks) != 0 {
		t.Fatalf("repo should not persist enable when scheduler add fails")
	}
}

func TestScheduledTaskUpdateEnabledSchedulerAddFailureKeepsOldTaskAndJob(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[1] = &entity.ScheduledTask{
		ID:             1,
		Name:           "old name",
		Description:    "old desc",
		RuleChainID:    "chain-root",
		CronExpr:       "0 8 * * *",
		ScheduleType:   "daily",
		ScheduleConfig: `{"hour":8}`,
		Disabled:       false,
	}
	scheduler := newFakeScheduledTaskScheduler()
	scheduler.failCron = "bad cron"
	scheduler.jobs[1] = scheduledTaskAddCall{taskID: 1, cronExpr: "0 8 * * *"}
	uc := &ScheduledTaskUsecase{repo: repo, scheduler: scheduler}

	_, err := uc.UpdateScheduledTask(context.Background(), 1, "new name", "new desc", "chain-root", "bad cron", "custom", `{"bad":true}`)
	if err == nil || !strings.Contains(err.Error(), "invalid cron") {
		t.Fatalf("UpdateScheduledTask error=%v, want scheduler error", err)
	}

	task := repo.tasks[1]
	if task.Name != "old name" || task.Description != "old desc" || task.CronExpr != "0 8 * * *" ||
		task.ScheduleType != "daily" || task.ScheduleConfig != `{"hour":8}` || task.Disabled {
		t.Fatalf("task should keep old persisted config after scheduler add failure: %#v", task)
	}
	if len(scheduler.adds) != 2 || scheduler.adds[0].cronExpr != "bad cron" || scheduler.adds[1].cronExpr != "0 8 * * *" {
		t.Fatalf("scheduler should try new job then restore old job, got adds=%#v", scheduler.adds)
	}
	if job, ok := scheduler.jobs[1]; !ok || job.cronExpr != "0 8 * * *" {
		t.Fatalf("old scheduler job should be restored after new add fails, jobs=%#v", scheduler.jobs)
	}
}

func TestScheduledTaskUpdateEnabledDBFailureRestoresOldSchedulerJob(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[1] = &entity.ScheduledTask{
		ID:             1,
		Name:           "old name",
		Description:    "old desc",
		RuleChainID:    "chain-root",
		CronExpr:       "0 8 * * *",
		ScheduleType:   "daily",
		ScheduleConfig: `{"hour":8}`,
		Disabled:       false,
	}
	repo.updateErr = errors.New("update failed")
	scheduler := newFakeScheduledTaskScheduler()
	scheduler.jobs[1] = scheduledTaskAddCall{taskID: 1, cronExpr: "0 8 * * *"}
	uc := &ScheduledTaskUsecase{repo: repo, scheduler: scheduler}

	_, err := uc.UpdateScheduledTask(context.Background(), 1, "new name", "new desc", "chain-root", "0 9 * * *", "daily", `{"hour":9}`)
	if err == nil || !strings.Contains(err.Error(), "update failed") {
		t.Fatalf("UpdateScheduledTask error=%v, want repo update error", err)
	}

	task := repo.tasks[1]
	if task.Name != "old name" || task.Description != "old desc" || task.CronExpr != "0 8 * * *" ||
		task.ScheduleType != "daily" || task.ScheduleConfig != `{"hour":8}` || task.Disabled {
		t.Fatalf("task should keep old persisted config after DB update failure: %#v", task)
	}
	if len(scheduler.adds) != 2 || scheduler.adds[0].cronExpr != "0 9 * * *" || scheduler.adds[1].cronExpr != "0 8 * * *" {
		t.Fatalf("scheduler should add new job then restore old job, got adds=%#v", scheduler.adds)
	}
	if len(scheduler.removed) != 1 || scheduler.removed[0] != 1 {
		t.Fatalf("scheduler should remove new job before restoring old job, removed=%v", scheduler.removed)
	}
	if job, ok := scheduler.jobs[1]; !ok || job.cronExpr != "0 8 * * *" {
		t.Fatalf("old scheduler job should be restored after DB update failure, jobs=%#v", scheduler.jobs)
	}
}

func TestScheduledTaskDisableRemovesSchedulerJob(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[1] = &entity.ScheduledTask{ID: 1, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: false}
	scheduler := &fakeScheduledTaskScheduler{}
	uc := &ScheduledTaskUsecase{repo: repo, scheduler: scheduler}

	task, err := uc.DisableScheduledTask(context.Background(), 1)
	if err != nil {
		t.Fatalf("DisableScheduledTask returned error: %v", err)
	}

	if !task.Disabled {
		t.Fatalf("disabled task should have Disabled=true")
	}
	if len(scheduler.removed) != 1 || scheduler.removed[0] != 1 {
		t.Fatalf("scheduler remove mismatch: %v", scheduler.removed)
	}
}

func TestScheduledTaskDisableDBFailureDoesNotRemoveSchedulerJob(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[1] = &entity.ScheduledTask{ID: 1, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: false}
	repo.updateErr = errors.New("update failed")
	scheduler := &fakeScheduledTaskScheduler{}
	uc := &ScheduledTaskUsecase{repo: repo, scheduler: scheduler}

	_, err := uc.DisableScheduledTask(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "update failed") {
		t.Fatalf("DisableScheduledTask error=%v, want repo update error", err)
	}

	if len(scheduler.removed) != 0 {
		t.Fatalf("scheduler should keep old job when DB update fails, removed=%v", scheduler.removed)
	}
	if repo.tasks[1].Disabled {
		t.Fatalf("persisted task should remain enabled when DB update fails")
	}
}

func TestScheduledTaskDeleteEnabledRemovesSchedulerJobAndDeletes(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[1] = &entity.ScheduledTask{ID: 1, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: false}
	scheduler := &fakeScheduledTaskScheduler{}
	uc := &ScheduledTaskUsecase{repo: repo, scheduler: scheduler}

	if err := uc.DeleteScheduledTask(context.Background(), 1); err != nil {
		t.Fatalf("DeleteScheduledTask returned error: %v", err)
	}

	if len(scheduler.removed) != 1 || scheduler.removed[0] != 1 {
		t.Fatalf("scheduler remove mismatch: %v", scheduler.removed)
	}
	if len(repo.deletedIDs) != 1 || repo.deletedIDs[0] != 1 {
		t.Fatalf("repo delete mismatch: %v", repo.deletedIDs)
	}
}

func TestScheduledTaskDeleteDBFailureDoesNotRemoveSchedulerJob(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[1] = &entity.ScheduledTask{ID: 1, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: false}
	repo.deleteErr = errors.New("delete failed")
	scheduler := &fakeScheduledTaskScheduler{}
	uc := &ScheduledTaskUsecase{repo: repo, scheduler: scheduler}

	err := uc.DeleteScheduledTask(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "delete failed") {
		t.Fatalf("DeleteScheduledTask error=%v, want repo delete error", err)
	}

	if len(scheduler.removed) != 0 {
		t.Fatalf("scheduler should keep old job when DB delete fails, removed=%v", scheduler.removed)
	}
	if repo.tasks[1] == nil {
		t.Fatalf("persisted task should remain when DB delete fails")
	}
}

func TestScheduledTaskStartEnabledTasksAddsAllAndStarts(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.enabledTasks = []*entity.ScheduledTask{
		{ID: 1, RuleChainID: "chain-a", CronExpr: "0 8 * * *", Disabled: false},
		{ID: 2, RuleChainID: "chain-b", CronExpr: "0 9 * * *", Disabled: false},
	}
	scheduler := &fakeScheduledTaskScheduler{}
	uc := &ScheduledTaskUsecase{repo: repo, scheduler: scheduler}

	if err := uc.StartEnabledTasks(context.Background()); err != nil {
		t.Fatalf("StartEnabledTasks returned error: %v", err)
	}

	if len(scheduler.adds) != 2 || scheduler.adds[0].taskID != 1 || scheduler.adds[1].taskID != 2 {
		t.Fatalf("scheduler add mismatch: %#v", scheduler.adds)
	}
	if !scheduler.started {
		t.Fatalf("scheduler should be started")
	}
}

func TestScheduledTaskStartEnabledTasksReturnsTaskIDWhenAddFails(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.enabledTasks = []*entity.ScheduledTask{
		{ID: 42, RuleChainID: "chain-root", CronExpr: "invalid cron", Disabled: false},
	}
	scheduler := &fakeScheduledTaskScheduler{addErr: errors.New("invalid cron")}
	uc := &ScheduledTaskUsecase{repo: repo, scheduler: scheduler}

	err := uc.StartEnabledTasks(context.Background())
	if err == nil {
		t.Fatal("expected StartEnabledTasks to return error")
	}
	if !strings.Contains(err.Error(), "42") {
		t.Fatalf("expected error to include taskID, got %q", err.Error())
	}
	if scheduler.started {
		t.Fatal("scheduler should not start after add failure")
	}
}

func TestScheduledTaskStartEnabledTasksRollsBackAddedJobsWhenLaterAddFails(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.enabledTasks = []*entity.ScheduledTask{
		{ID: 1, RuleChainID: "chain-a", CronExpr: "0 8 * * *", Disabled: false},
		{ID: 2, RuleChainID: "chain-b", CronExpr: "bad cron", Disabled: false},
	}
	scheduler := newFakeScheduledTaskScheduler()
	scheduler.failCron = "bad cron"
	uc := &ScheduledTaskUsecase{repo: repo, scheduler: scheduler}

	err := uc.StartEnabledTasks(context.Background())
	if err == nil {
		t.Fatal("expected StartEnabledTasks to return error")
	}
	if !strings.Contains(err.Error(), "2") {
		t.Fatalf("expected error to include failed taskID, got %q", err.Error())
	}
	if scheduler.started {
		t.Fatal("scheduler should not start after add failure")
	}
	if len(scheduler.removed) != 1 || scheduler.removed[0] != 1 {
		t.Fatalf("expected first added job to be removed after later failure, removed=%v", scheduler.removed)
	}
	if _, ok := scheduler.jobs[1]; ok {
		t.Fatalf("expected first job to be removed after rollback, jobs=%#v", scheduler.jobs)
	}
}

func TestScheduledTaskStopScheduledTasksStopsScheduler(t *testing.T) {
	scheduler := &fakeScheduledTaskScheduler{}
	uc := &ScheduledTaskUsecase{scheduler: scheduler}

	uc.StopScheduledTasks()

	if !scheduler.stopped {
		t.Fatal("expected StopScheduledTasks to stop scheduler")
	}
}

func TestScheduledTaskStopScheduledTasksAllowsNilScheduler(t *testing.T) {
	uc := &ScheduledTaskUsecase{}

	uc.StopScheduledTasks()
}

func TestScheduledTaskSchedulerCallbackLogsRunError(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.enabledTasks = []*entity.ScheduledTask{
		{ID: 12, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: false},
	}
	repo.tasks[12] = cloneScheduledTask(repo.enabledTasks[0])
	repo.createRunErr = errors.New("insert run failed")
	scheduler := newFakeScheduledTaskScheduler()
	logger := &fakeScheduledTaskLogger{}
	uc := NewScheduledTaskUsecase(repo, newUsableFakeRuleChain("chain-root"), scheduler, logger)

	if err := uc.StartEnabledTasks(context.Background()); err != nil {
		t.Fatalf("StartEnabledTasks returned error: %v", err)
	}
	job, ok := scheduler.jobs[12]
	if !ok || job.fn == nil {
		t.Fatalf("expected scheduler job callback for task 12, jobs=%#v", scheduler.jobs)
	}

	job.fn()

	if !logger.contains("taskID", int64(12)) || !logger.contains("err", "create scheduled task run") {
		t.Fatalf("scheduler callback should log taskID and error, logs=%#v", logger.entries)
	}
}

func TestScheduledTaskRunDisablesUnavailableRuleChainAndWritesFailedRun(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[1] = &entity.ScheduledTask{ID: 1, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: false}
	scheduler := &fakeScheduledTaskScheduler{}
	ruleChain := newUsableFakeRuleChain("chain-root")
	ruleChain.loaded["chain-root"] = false
	uc := &ScheduledTaskUsecase{repo: repo, ruleChain: ruleChain, scheduler: scheduler}

	if err := uc.runScheduledTask(context.Background(), 1); err != nil {
		t.Fatalf("runScheduledTask returned error: %v", err)
	}

	if !repo.tasks[1].Disabled {
		t.Fatalf("task should be disabled when rule chain is unavailable")
	}
	if len(scheduler.removed) != 1 || scheduler.removed[0] != 1 {
		t.Fatalf("scheduler remove mismatch: %v", scheduler.removed)
	}
	if len(repo.runs) != 1 {
		t.Fatalf("expected one run history, got %d", len(repo.runs))
	}
	run := repo.runs[0]
	if run.Status != entity.ScheduledTaskRunStatusFailed || run.ErrorMessage == "" {
		t.Fatalf("failed run mismatch: %#v", run)
	}
	if repo.tasks[1].LastStatus != entity.ScheduledTaskRunStatusFailed || repo.tasks[1].LastError == "" || repo.tasks[1].LastRunAt == nil {
		t.Fatalf("task last run fields mismatch: %#v", repo.tasks[1])
	}
}

func TestScheduledTaskRunUnavailableRuleChainCreateRunFailureDoesNotRemoveSchedulerJob(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[1] = &entity.ScheduledTask{ID: 1, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: false}
	repo.createRunErr = errors.New("insert run failed")
	scheduler := &fakeScheduledTaskScheduler{}
	ruleChain := newUsableFakeRuleChain("chain-root")
	ruleChain.loaded["chain-root"] = false
	uc := &ScheduledTaskUsecase{repo: repo, ruleChain: ruleChain, scheduler: scheduler}

	err := uc.runScheduledTask(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "create scheduled task run") {
		t.Fatalf("runScheduledTask error=%v, want create run error", err)
	}

	if len(scheduler.removed) != 0 {
		t.Fatalf("scheduler should keep old job when failed run history cannot be persisted, removed=%v", scheduler.removed)
	}
	if repo.tasks[1].Disabled {
		t.Fatalf("persisted task should remain enabled when failed run history cannot be persisted")
	}
}

func TestScheduledTaskRunUnavailableRuleChainUpdateTaskFailureDoesNotRemoveSchedulerJob(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[1] = &entity.ScheduledTask{ID: 1, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: false}
	repo.updateErr = errors.New("update task failed")
	scheduler := &fakeScheduledTaskScheduler{}
	ruleChain := newUsableFakeRuleChain("chain-root")
	ruleChain.loaded["chain-root"] = false
	uc := &ScheduledTaskUsecase{repo: repo, ruleChain: ruleChain, scheduler: scheduler}

	err := uc.runScheduledTask(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "update scheduled task status") {
		t.Fatalf("runScheduledTask error=%v, want update task error", err)
	}

	if len(scheduler.removed) != 0 {
		t.Fatalf("scheduler should keep old job when disabled status cannot be persisted, removed=%v", scheduler.removed)
	}
	if repo.tasks[1].Disabled {
		t.Fatalf("persisted task should remain enabled when disabled status cannot be persisted")
	}
}

func TestScheduledTaskRunExecutesRuleChainWithFixedPayloadAndWritesSuccessRun(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[12] = &entity.ScheduledTask{ID: 12, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: false}
	ruleChain := newUsableFakeRuleChain("chain-root")
	uc := &ScheduledTaskUsecase{repo: repo, ruleChain: ruleChain, scheduler: &fakeScheduledTaskScheduler{}}

	if err := uc.runScheduledTask(context.Background(), 12); err != nil {
		t.Fatalf("runScheduledTask returned error: %v", err)
	}

	if len(ruleChain.executed) != 1 {
		t.Fatalf("expected one ExecuteRuleChain call, got %d", len(ruleChain.executed))
	}
	req := ruleChain.executed[0]
	if req.GetId() != "chain-root" {
		t.Fatalf("execute rule chain id mismatch: %s", req.GetId())
	}
	data := req.GetData().AsMap()
	if _, ok := data["payload"]; ok {
		t.Fatalf("execute data should not wrap scheduled payload in payload field: %#v", data)
	}
	if got := data["trigger"]; got != "schedule" {
		t.Fatalf("trigger mismatch got=%v want=schedule", got)
	}
	if got := data["taskId"]; got != "12" {
		t.Fatalf("taskId mismatch got=%v want=12", got)
	}
	if len(repo.runs) != 1 {
		t.Fatalf("expected one run history, got %d", len(repo.runs))
	}
	run := repo.runs[0]
	if run.Status != entity.ScheduledTaskRunStatusSuccess || run.TriggerPayload != entity.NewScheduledTriggerPayload(12) || run.ErrorMessage != "" {
		t.Fatalf("success run mismatch: %#v", run)
	}
	if repo.tasks[12].LastStatus != entity.ScheduledTaskRunStatusSuccess || repo.tasks[12].LastError != "" || repo.tasks[12].LastRunAt == nil {
		t.Fatalf("task last run fields mismatch: %#v", repo.tasks[12])
	}
}

func TestScheduledTaskRunReturnsCreateRunError(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[12] = &entity.ScheduledTask{ID: 12, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: false}
	repo.createRunErr = errors.New("insert run failed")
	uc := &ScheduledTaskUsecase{repo: repo, ruleChain: newUsableFakeRuleChain("chain-root"), scheduler: &fakeScheduledTaskScheduler{}}

	err := uc.runScheduledTask(context.Background(), 12)
	if err == nil || !strings.Contains(err.Error(), "create scheduled task run") || !strings.Contains(err.Error(), "insert run failed") {
		t.Fatalf("runScheduledTask error=%v, want create run context", err)
	}
}

func TestScheduledTaskRunReturnsUpdateTaskError(t *testing.T) {
	repo := newFakeScheduledTaskRepo()
	repo.tasks[12] = &entity.ScheduledTask{ID: 12, RuleChainID: "chain-root", CronExpr: "0 8 * * *", Disabled: false}
	repo.updateErr = errors.New("update task failed")
	uc := &ScheduledTaskUsecase{repo: repo, ruleChain: newUsableFakeRuleChain("chain-root"), scheduler: &fakeScheduledTaskScheduler{}}

	err := uc.runScheduledTask(context.Background(), 12)
	if err == nil || !strings.Contains(err.Error(), "update scheduled task status") || !strings.Contains(err.Error(), "update task failed") {
		t.Fatalf("runScheduledTask error=%v, want update task context", err)
	}
}

type fakeScheduledTaskRepo struct {
	nextID           int64
	tasks            map[int64]*entity.ScheduledTask
	enabledTasks     []*entity.ScheduledTask
	runs             []*entity.ScheduledTaskRun
	createdTask      *entity.ScheduledTask
	updatedTasks     []*entity.ScheduledTask
	deletedIDs       []int64
	lastListPage     int32
	lastListPageSize int32
	lastRunPage      int32
	lastRunPageSize  int32
	createErr        error
	getErr           error
	listErr          error
	listEnabledErr   error
	updateErr        error
	deleteErr        error
	createRunErr     error
	listRunsErr      error
}

func newFakeScheduledTaskRepo() *fakeScheduledTaskRepo {
	return &fakeScheduledTaskRepo{
		nextID: 1,
		tasks:  map[int64]*entity.ScheduledTask{},
	}
}

func (r *fakeScheduledTaskRepo) CreateScheduledTask(ctx context.Context, task *entity.ScheduledTask) error {
	if r.createErr != nil {
		return r.createErr
	}
	if task.ID == 0 {
		task.ID = r.nextID
		r.nextID++
	}
	cp := cloneScheduledTask(task)
	r.createdTask = cp
	r.tasks[cp.ID] = cp
	return nil
}

func (r *fakeScheduledTaskRepo) GetScheduledTask(ctx context.Context, id int64) (*entity.ScheduledTask, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	task := r.tasks[id]
	if task == nil {
		return nil, errors.New("task not found")
	}
	return cloneScheduledTask(task), nil
}

func (r *fakeScheduledTaskRepo) ListScheduledTasks(ctx context.Context, name, ruleChainID string, disabled *bool, page, pageSize int32) ([]*entity.ScheduledTask, int64, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	r.lastListPage = page
	r.lastListPageSize = pageSize
	tasks := make([]*entity.ScheduledTask, 0, len(r.tasks))
	for _, task := range r.tasks {
		tasks = append(tasks, cloneScheduledTask(task))
	}
	return tasks, int64(len(tasks)), nil
}

func (r *fakeScheduledTaskRepo) ListEnabledScheduledTasks(ctx context.Context) ([]*entity.ScheduledTask, error) {
	if r.listEnabledErr != nil {
		return nil, r.listEnabledErr
	}
	tasks := r.enabledTasks
	if tasks == nil {
		for _, task := range r.tasks {
			if !task.Disabled {
				tasks = append(tasks, task)
			}
		}
	}
	out := make([]*entity.ScheduledTask, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, cloneScheduledTask(task))
	}
	return out, nil
}

func (r *fakeScheduledTaskRepo) UpdateScheduledTask(ctx context.Context, task *entity.ScheduledTask) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	cp := cloneScheduledTask(task)
	r.updatedTasks = append(r.updatedTasks, cp)
	r.tasks[cp.ID] = cp
	return nil
}

func (r *fakeScheduledTaskRepo) DeleteScheduledTask(ctx context.Context, id int64) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deletedIDs = append(r.deletedIDs, id)
	delete(r.tasks, id)
	return nil
}

func (r *fakeScheduledTaskRepo) CreateScheduledTaskRun(ctx context.Context, run *entity.ScheduledTaskRun) error {
	if r.createRunErr != nil {
		return r.createRunErr
	}
	cp := *run
	if cp.ID == 0 {
		cp.ID = int64(len(r.runs) + 1)
	}
	r.runs = append(r.runs, &cp)
	return nil
}

func (r *fakeScheduledTaskRepo) ListScheduledTaskRuns(ctx context.Context, taskID int64, page, pageSize int32) ([]*entity.ScheduledTaskRun, int64, error) {
	if r.listRunsErr != nil {
		return nil, 0, r.listRunsErr
	}
	r.lastRunPage = page
	r.lastRunPageSize = pageSize
	var out []*entity.ScheduledTaskRun
	for _, run := range r.runs {
		if run.TaskID == taskID {
			cp := *run
			out = append(out, &cp)
		}
	}
	return out, int64(len(out)), nil
}

type fakeScheduledTaskScheduler struct {
	adds     []scheduledTaskAddCall
	removed  []int64
	jobs     map[int64]scheduledTaskAddCall
	started  bool
	stopped  bool
	addErr   error
	failCron string
}

type scheduledTaskAddCall struct {
	taskID   int64
	cronExpr string
	fn       func()
}

func newFakeScheduledTaskScheduler() *fakeScheduledTaskScheduler {
	return &fakeScheduledTaskScheduler{
		jobs: map[int64]scheduledTaskAddCall{},
	}
}

func (s *fakeScheduledTaskScheduler) Add(taskID int64, cronExpr string, fn func()) error {
	if s.jobs == nil {
		s.jobs = map[int64]scheduledTaskAddCall{}
	}
	call := scheduledTaskAddCall{taskID: taskID, cronExpr: cronExpr, fn: fn}
	s.adds = append(s.adds, call)
	// 模拟 Task 4 真实 scheduler 的替换语义：同一 taskID Add 会先移除旧 job，再尝试添加新 job。
	delete(s.jobs, taskID)
	if s.addErr != nil {
		return s.addErr
	}
	if s.failCron != "" && cronExpr == s.failCron {
		return errors.New("invalid cron")
	}
	s.jobs[taskID] = call
	return nil
}

func (s *fakeScheduledTaskScheduler) Remove(taskID int64) {
	if s.jobs != nil {
		delete(s.jobs, taskID)
	}
	s.removed = append(s.removed, taskID)
}

func (s *fakeScheduledTaskScheduler) Start() {
	s.started = true
}

func (s *fakeScheduledTaskScheduler) Stop() {
	s.stopped = true
}

type fakeScheduledTaskLogger struct {
	entries [][]interface{}
}

func (l *fakeScheduledTaskLogger) Log(level klog.Level, keyvals ...interface{}) error {
	entry := append([]interface{}{"level", level}, keyvals...)
	l.entries = append(l.entries, entry)
	return nil
}

func (l *fakeScheduledTaskLogger) contains(key string, want interface{}) bool {
	for _, entry := range l.entries {
		for i := 0; i+1 < len(entry); i += 2 {
			if entry[i] != key {
				continue
			}
			value := entry[i+1]
			if wantString, ok := want.(string); ok {
				if strings.Contains(toTestString(value), wantString) {
					return true
				}
				continue
			}
			if value == want {
				return true
			}
		}
	}
	return false
}

func toTestString(value interface{}) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

type fakeScheduledTaskRuleChain struct {
	chains     map[string]*entity.RuleChain
	loaded     map[string]bool
	executed   []*v1.ExecuteRuleChainReq
	executeErr error
	getErr     error
}

func newUsableFakeRuleChain(id string) *fakeScheduledTaskRuleChain {
	return &fakeScheduledTaskRuleChain{
		chains: map[string]*entity.RuleChain{
			id: {RuleChainID: id, Root: true, Disabled: false},
		},
		loaded: map[string]bool{id: true},
	}
}

func (f *fakeScheduledTaskRuleChain) GetScheduledTaskRuleChain(ctx context.Context, id string) (*entity.RuleChain, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	chain := f.chains[id]
	if chain == nil {
		return nil, nil
	}
	cp := *chain
	return &cp, nil
}

func (f *fakeScheduledTaskRuleChain) ExecuteRuleChain(ctx context.Context, in *v1.ExecuteRuleChainReq) (*v1.ExecuteRuleChainReply, error) {
	if f.executeErr != nil {
		return nil, f.executeErr
	}
	f.executed = append(f.executed, in)
	return &v1.ExecuteRuleChainReply{}, nil
}

func (f *fakeScheduledTaskRuleChain) IsRuleChainLoaded(id string) bool {
	return f.loaded[id]
}

func cloneScheduledTask(task *entity.ScheduledTask) *entity.ScheduledTask {
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
