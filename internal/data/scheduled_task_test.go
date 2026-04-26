package data

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"ruleGoKratos/internal/biz/entity"
	"ruleGoKratos/internal/data/dao"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScheduledTaskDAOCreateListUpdateDeleteAndRuns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	dao.Init(db)
	assertSingleColumnIndex(t, db, "scheduled_tasks", "rule_chain_id", true)
	assertSingleColumnIndex(t, db, "scheduled_tasks", "deleted_at", true)
	assertSingleColumnIndex(t, db, "scheduled_tasks", "disabled", false)
	assertSingleColumnIndex(t, db, "scheduled_task_runs", "task_id", false)
	assertSingleColumnIndex(t, db, "scheduled_task_runs", "rule_chain_id", false)
	assertSingleColumnIndex(t, db, "scheduled_task_runs", "status", false)

	ctx := context.Background()

	disabledTask := &dao.ScheduledTask{
		Name:           "每 5 分钟同步",
		Description:    "同步数据",
		RuleChainID:    "chain-root",
		CronExpr:       "*/5 * * * *",
		ScheduleType:   "every_minutes",
		ScheduleConfig: `{"minutes":5}`,
		Disabled:       true,
	}
	if err := disabledTask.Create(ctx); err != nil {
		t.Fatalf("create disabled task: %v", err)
	}

	defaultDisabledTask := &dao.ScheduledTask{
		Name:           "默认关闭任务",
		RuleChainID:    "chain-default",
		CronExpr:       "0 9 * * *",
		ScheduleType:   "daily",
		ScheduleConfig: `{"hour":9,"minute":0}`,
	}
	if err := defaultDisabledTask.Create(ctx); err != nil {
		t.Fatalf("create default disabled task: %v", err)
	}
	defaultDisabledTask, err = dao.NewScheduledTask().GetByID(ctx, defaultDisabledTask.ID)
	if err != nil {
		t.Fatalf("get default disabled task: %v", err)
	}
	if !defaultDisabledTask.Disabled {
		t.Fatalf("task created without disabled should default to disabled")
	}

	tasks, total, err := dao.NewScheduledTask().List(ctx, "", "", nil, 1, 10)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if total != 2 || len(tasks) != 2 {
		t.Fatalf("unexpected list result total=%d len=%d", total, len(tasks))
	}
	for _, task := range tasks {
		if !task.Disabled {
			t.Fatalf("created task should be disabled: %#v", task)
		}
	}

	sameCreatedAt := time.Now().UTC().Truncate(time.Second)
	firstSortedTask := &dao.ScheduledTask{
		Name:         "排序任务 1",
		RuleChainID:  "chain-sort",
		CronExpr:     "0 10 * * *",
		ScheduleType: "daily",
		CreatedAt:    sameCreatedAt,
		UpdatedAt:    sameCreatedAt,
	}
	if err := firstSortedTask.Create(ctx); err != nil {
		t.Fatalf("create first sorted task: %v", err)
	}
	secondSortedTask := &dao.ScheduledTask{
		Name:         "排序任务 2",
		RuleChainID:  "chain-sort",
		CronExpr:     "0 10 * * *",
		ScheduleType: "daily",
		CreatedAt:    sameCreatedAt,
		UpdatedAt:    sameCreatedAt,
	}
	if err := secondSortedTask.Create(ctx); err != nil {
		t.Fatalf("create second sorted task: %v", err)
	}
	sortedTasks, sortedTotal, err := dao.NewScheduledTask().List(ctx, "", "chain-sort", nil, 1, 10)
	if err != nil {
		t.Fatalf("list sorted tasks: %v", err)
	}
	if sortedTotal != 2 || len(sortedTasks) != 2 || sortedTasks[0].ID != secondSortedTask.ID || sortedTasks[1].ID != firstSortedTask.ID {
		t.Fatalf("tasks should be ordered by created_at desc, id desc: total=%d tasks=%#v", sortedTotal, sortedTasks)
	}

	enabledTask := &dao.ScheduledTask{
		Name:           "每日巡检",
		RuleChainID:    "chain-enabled",
		CronExpr:       "0 8 * * *",
		ScheduleType:   "daily",
		ScheduleConfig: `{"hour":8,"minute":0}`,
	}
	if err := enabledTask.Create(ctx); err != nil {
		t.Fatalf("create enabled task: %v", err)
	}
	if err := enabledTask.Update(ctx, enabledTask.ID, map[string]interface{}{"disabled": false}); err != nil {
		t.Fatalf("enable task before delete: %v", err)
	}
	if err := enabledTask.Delete(ctx, enabledTask.ID); err != nil {
		t.Fatalf("delete enabled task: %v", err)
	}

	remainingEnabled := &dao.ScheduledTask{
		Name:           "每小时检查",
		RuleChainID:    "chain-enabled",
		CronExpr:       "0 * * * *",
		ScheduleType:   "every_hours",
		ScheduleConfig: `{"hours":1}`,
	}
	if err := remainingEnabled.Create(ctx); err != nil {
		t.Fatalf("create remaining enabled task: %v", err)
	}
	if err := remainingEnabled.Update(ctx, remainingEnabled.ID, map[string]interface{}{"disabled": false}); err != nil {
		t.Fatalf("enable remaining task: %v", err)
	}

	enabledTasks, err := dao.NewScheduledTask().ListEnabled(ctx)
	if err != nil {
		t.Fatalf("list enabled tasks: %v", err)
	}
	if len(enabledTasks) != 1 || enabledTasks[0].ID != remainingEnabled.ID {
		t.Fatalf("expected only non-deleted enabled task, got len=%d", len(enabledTasks))
	}

	enabledCreatedAt := time.Now().Add(time.Hour).Truncate(time.Second)
	firstEnabledSortedTask := &dao.ScheduledTask{
		Name:         "启用排序任务 1",
		RuleChainID:  "chain-enabled-sort",
		CronExpr:     "0 11 * * *",
		ScheduleType: "daily",
		CreatedAt:    enabledCreatedAt,
		UpdatedAt:    enabledCreatedAt,
	}
	if err := firstEnabledSortedTask.Create(ctx); err != nil {
		t.Fatalf("create first enabled sorted task: %v", err)
	}
	if err := firstEnabledSortedTask.Update(ctx, firstEnabledSortedTask.ID, map[string]interface{}{"disabled": false}); err != nil {
		t.Fatalf("enable first sorted task: %v", err)
	}
	secondEnabledSortedTask := &dao.ScheduledTask{
		Name:         "启用排序任务 2",
		RuleChainID:  "chain-enabled-sort",
		CronExpr:     "0 11 * * *",
		ScheduleType: "daily",
		CreatedAt:    enabledCreatedAt,
		UpdatedAt:    enabledCreatedAt,
	}
	if err := secondEnabledSortedTask.Create(ctx); err != nil {
		t.Fatalf("create second enabled sorted task: %v", err)
	}
	if err := secondEnabledSortedTask.Update(ctx, secondEnabledSortedTask.ID, map[string]interface{}{"disabled": false}); err != nil {
		t.Fatalf("enable second sorted task: %v", err)
	}
	enabledTasks, err = dao.NewScheduledTask().ListEnabled(ctx)
	if err != nil {
		t.Fatalf("list enabled sorted tasks: %v", err)
	}
	if len(enabledTasks) < 2 || enabledTasks[0].ID != secondEnabledSortedTask.ID || enabledTasks[1].ID != firstEnabledSortedTask.ID {
		t.Fatalf("enabled tasks should be ordered by created_at desc, id desc: got=%s want first IDs=%d,%d", formatTaskOrder(enabledTasks), secondEnabledSortedTask.ID, firstEnabledSortedTask.ID)
	}

	lastRunAt := time.Now().UTC().Truncate(time.Second)
	updateData := map[string]interface{}{
		"disabled":    false,
		"last_run_at": lastRunAt,
		"last_status": entity.ScheduledTaskRunStatusFailed,
		"last_error":  "rule chain unavailable",
	}
	if err := disabledTask.Update(ctx, disabledTask.ID, updateData); err != nil {
		t.Fatalf("update task: %v", err)
	}
	updated, err := dao.NewScheduledTask().GetByID(ctx, disabledTask.ID)
	if err != nil {
		t.Fatalf("get updated task: %v", err)
	}
	if updated.Disabled || updated.LastRunAt == nil || !updated.LastRunAt.Equal(lastRunAt) ||
		updated.LastStatus != entity.ScheduledTaskRunStatusFailed || updated.LastError != "rule chain unavailable" {
		t.Fatalf("task was not updated as expected: %#v", updated)
	}

	run := &dao.ScheduledTaskRun{
		TaskID:         disabledTask.ID,
		RuleChainID:    disabledTask.RuleChainID,
		Status:         entity.ScheduledTaskRunStatusSuccess,
		TriggerPayload: entity.NewScheduledTriggerPayload(disabledTask.ID, ""),
		StartedAt:      time.Now().Add(-time.Second),
		FinishedAt:     time.Now(),
	}
	if err := run.Create(ctx); err != nil {
		t.Fatalf("create run: %v", err)
	}
	runs, runTotal, err := dao.NewScheduledTaskRun().ListByTaskID(ctx, disabledTask.ID, 1, 10)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if runTotal != 1 || len(runs) != 1 || runs[0].TriggerPayload != entity.NewScheduledTriggerPayload(disabledTask.ID, "") {
		t.Fatalf("unexpected run list total=%d len=%d", runTotal, len(runs))
	}

	firstSortedRun := &dao.ScheduledTaskRun{
		TaskID:         remainingEnabled.ID,
		RuleChainID:    remainingEnabled.RuleChainID,
		Status:         entity.ScheduledTaskRunStatusSuccess,
		TriggerPayload: entity.NewScheduledTriggerPayload(remainingEnabled.ID, ""),
		StartedAt:      sameCreatedAt,
		FinishedAt:     sameCreatedAt,
		CreatedAt:      sameCreatedAt,
	}
	if err := firstSortedRun.Create(ctx); err != nil {
		t.Fatalf("create first sorted run: %v", err)
	}
	secondSortedRun := &dao.ScheduledTaskRun{
		TaskID:         remainingEnabled.ID,
		RuleChainID:    remainingEnabled.RuleChainID,
		Status:         entity.ScheduledTaskRunStatusFailed,
		TriggerPayload: entity.NewScheduledTriggerPayload(remainingEnabled.ID, ""),
		ErrorMessage:   "failed",
		StartedAt:      sameCreatedAt,
		FinishedAt:     sameCreatedAt,
		CreatedAt:      sameCreatedAt,
	}
	if err := secondSortedRun.Create(ctx); err != nil {
		t.Fatalf("create second sorted run: %v", err)
	}
	sortedRuns, sortedRunTotal, err := dao.NewScheduledTaskRun().ListByTaskID(ctx, remainingEnabled.ID, 1, 10)
	if err != nil {
		t.Fatalf("list sorted runs: %v", err)
	}
	if sortedRunTotal != 2 || len(sortedRuns) != 2 || sortedRuns[0].ID != secondSortedRun.ID || sortedRuns[1].ID != firstSortedRun.ID {
		t.Fatalf("runs should be ordered by created_at desc, id desc: total=%d runs=%#v", sortedRunTotal, sortedRuns)
	}

	if err := disabledTask.Delete(ctx, disabledTask.ID); err != nil {
		t.Fatalf("delete task: %v", err)
	}
	if err := disabledTask.Update(ctx, disabledTask.ID, map[string]interface{}{
		"disabled":   false,
		"last_error": "should not update deleted task",
	}); err != nil {
		t.Fatalf("update deleted task: %v", err)
	}
	if _, err := dao.NewScheduledTask().GetByID(ctx, disabledTask.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected deleted task not found, got %v", err)
	}
	var deletedRecord dao.ScheduledTask
	if err := db.Where("id = ?", disabledTask.ID).First(&deletedRecord).Error; err != nil {
		t.Fatalf("get deleted task directly: %v", err)
	}
	if deletedRecord.DeletedAt == nil {
		t.Fatalf("deleted task should keep deleted_at after update")
	}
	if deletedRecord.LastError == "should not update deleted task" {
		t.Fatalf("deleted task should not be updated by business update")
	}
	filterEnabled := false
	tasks, total, err = dao.NewScheduledTask().List(ctx, "", "", &filterEnabled, 1, 10)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if total != 3 || len(tasks) != 3 {
		t.Fatalf("deleted or disabled tasks should not be listed, total=%d len=%d", total, len(tasks))
	}
	for _, task := range tasks {
		if task.ID == disabledTask.ID {
			t.Fatalf("deleted task should not be listed: %#v", task)
		}
	}
	runs, runTotal, err = dao.NewScheduledTaskRun().ListByTaskID(ctx, disabledTask.ID, 1, 10)
	if err != nil {
		t.Fatalf("list runs after delete: %v", err)
	}
	if runTotal != 1 || len(runs) != 1 {
		t.Fatalf("run history should remain after delete, total=%d len=%d", runTotal, len(runs))
	}
}

func TestScheduledTaskTriggerPayload(t *testing.T) {
	got := entity.NewScheduledTriggerPayload(12, "")
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(got), &data); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}
	if data["trigger"] != "schedule" || data["taskId"] != "12" {
		t.Fatalf("payload mismatch got=%v, want trigger=schedule taskId=12", data)
	}
}

func formatTaskOrder(tasks []*dao.ScheduledTask) string {
	result := ""
	for i, task := range tasks {
		if i > 0 {
			result += "; "
		}
		result += task.Name + ":" + task.CreatedAt.Format(time.RFC3339Nano)
	}
	return result
}

func assertSingleColumnIndex(t *testing.T, db *gorm.DB, table, column string, want bool) {
	t.Helper()
	indexes, err := singleColumnIndexes(db, table)
	if err != nil {
		t.Fatalf("inspect %s indexes: %v", table, err)
	}
	got := false
	for _, indexedColumn := range indexes {
		if indexedColumn == column {
			got = true
			break
		}
	}
	if got != want {
		t.Fatalf("single-column index presence mismatch table=%s column=%s got=%v want=%v indexes=%v", table, column, got, want, indexes)
	}
}

func singleColumnIndexes(db *gorm.DB, table string) ([]string, error) {
	var indexRows []struct {
		Name string `gorm:"column:name"`
	}
	if err := db.Raw("PRAGMA index_list(" + table + ")").Scan(&indexRows).Error; err != nil {
		return nil, err
	}

	var result []string
	for _, indexRow := range indexRows {
		var columnRows []struct {
			Name string `gorm:"column:name"`
		}
		indexName := strings.ReplaceAll(indexRow.Name, "'", "''")
		if err := db.Raw("PRAGMA index_info('" + indexName + "')").Scan(&columnRows).Error; err != nil {
			return nil, err
		}
		if len(columnRows) == 1 {
			result = append(result, columnRows[0].Name)
		}
	}
	return result, nil
}
