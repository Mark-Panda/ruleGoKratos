package biz

import (
	"sync"
	"testing"
)

func TestScheduledTaskSchedulerRejectsInvalidCron(t *testing.T) {
	scheduler := NewScheduledTaskScheduler()
	defer scheduler.Stop()

	if err := scheduler.Add(1, "not a cron", func() {}); err == nil {
		t.Fatal("expected invalid cron expression to return error")
	}
}

func TestScheduledTaskSchedulerReplaceAndRemoveAreSafe(t *testing.T) {
	scheduler := NewScheduledTaskScheduler()
	cronScheduler := scheduler.(*cronScheduledTaskScheduler)
	defer scheduler.Stop()

	if err := scheduler.Add(1, "@every 1h", func() {}); err != nil {
		t.Fatalf("add scheduled task: %v", err)
	}
	if err := scheduler.Add(1, "@every 2h", func() {}); err != nil {
		t.Fatalf("replace scheduled task: %v", err)
	}
	if got := len(cronScheduler.entries); got != 1 {
		t.Fatalf("expected one job after replacing same taskID, got %d", got)
	}
	if got := len(cronScheduler.cron.Entries()); got != 1 {
		t.Fatalf("expected one cron entry after replacing same taskID, got %d", got)
	}

	scheduler.Remove(1)
	scheduler.Remove(1)
	if got := len(cronScheduler.entries); got != 0 {
		t.Fatalf("expected no jobs after remove, got %d", got)
	}
	if got := len(cronScheduler.cron.Entries()); got != 0 {
		t.Fatalf("expected no cron entries after remove, got %d", got)
	}
}

func TestScheduledTaskSchedulerInvalidReplaceKeepsOldJob(t *testing.T) {
	scheduler := NewScheduledTaskScheduler()
	cronScheduler := scheduler.(*cronScheduledTaskScheduler)
	defer scheduler.Stop()

	if err := scheduler.Add(1, "@every 1h", func() {}); err != nil {
		t.Fatalf("add scheduled task: %v", err)
	}
	oldEntryID := cronScheduler.entries[1]

	if err := scheduler.Add(1, "invalid cron", func() {}); err == nil {
		t.Fatal("expected invalid cron expression to return error")
	}
	if got := len(cronScheduler.entries); got != 1 {
		t.Fatalf("expected old job to remain after invalid replace, got %d jobs", got)
	}
	if got := cronScheduler.entries[1]; got != oldEntryID {
		t.Fatalf("expected old entry %d to remain, got %d", oldEntryID, got)
	}
	if got := len(cronScheduler.cron.Entries()); got != 1 {
		t.Fatalf("expected old cron entry to remain after invalid replace, got %d entries", got)
	}
}

func TestScheduledTaskSchedulerStopCanBeCalled(t *testing.T) {
	scheduler := NewScheduledTaskScheduler()

	scheduler.Start()
	scheduler.Stop()
	scheduler.Stop()
}

func TestScheduledTaskSchedulerConcurrentAddSameTaskKeepsOneJob(t *testing.T) {
	scheduler := NewScheduledTaskScheduler()
	cronScheduler := scheduler.(*cronScheduledTaskScheduler)
	defer scheduler.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := scheduler.Add(1, "@every 1h", func() {}); err != nil {
				t.Errorf("add scheduled task: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := len(cronScheduler.entries); got != 1 {
		t.Fatalf("expected one tracked job after concurrent add, got %d", got)
	}
	if got := len(cronScheduler.cron.Entries()); got != 1 {
		t.Fatalf("expected one cron entry after concurrent add, got %d", got)
	}
}
