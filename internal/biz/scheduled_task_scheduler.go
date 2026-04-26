package biz

import (
	"sync"

	"github.com/robfig/cron/v3"
)

type cronScheduledTaskScheduler struct {
	cron    *cron.Cron
	parser  cron.Parser
	mu      sync.Mutex
	entries map[int64]cron.EntryID
}

func NewScheduledTaskScheduler() ScheduledTaskScheduler {
	parser := cron.NewParser(
		cron.Minute |
			cron.Hour |
			cron.Dom |
			cron.Month |
			cron.Dow |
			cron.Descriptor,
	)
	return &cronScheduledTaskScheduler{
		cron:    cron.New(cron.WithParser(parser)),
		parser:  parser,
		entries: map[int64]cron.EntryID{},
	}
}

func (s *cronScheduledTaskScheduler) Add(taskID int64, cronExpr string, fn func()) error {
	schedule, err := s.parser.Parse(cronExpr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if oldEntryID, ok := s.entries[taskID]; ok {
		s.cron.Remove(oldEntryID)
	}
	entryID := s.cron.Schedule(schedule, cron.FuncJob(fn))
	s.entries[taskID] = entryID
	return nil
}

func (s *cronScheduledTaskScheduler) Remove(taskID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, ok := s.entries[taskID]
	if !ok {
		return
	}
	s.cron.Remove(entryID)
	delete(s.entries, taskID)
}

func (s *cronScheduledTaskScheduler) Start() {
	s.cron.Start()
}

func (s *cronScheduledTaskScheduler) Stop() {
	s.cron.Stop()
}
