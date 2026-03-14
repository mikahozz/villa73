package main

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Clock interface for dependency injection in tests
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

// RealClock uses actual system time
type RealClock struct{}

func (rc *RealClock) Now() time.Time {
	return time.Now()
}

func (rc *RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

type FilterType string

const (
	FilterDate FilterType = "date"
)

type AndOrType string

const (
	AND AndOrType = "and"
	OR  AndOrType = "or"
)

type Trigger struct {
	Time func() time.Time
}

type Comparator string

const (
	LessThan    Comparator = "less_than"
	GreaterThan Comparator = "greater_than"
	Equal       Comparator = "equal"
)

type Filter struct {
	Type       FilterType
	Date       time.Time
	Comparator Comparator
}

type DailySchedule struct {
	Name          string
	Category      string // optional grouping; due schedules in the same category run in insertion order
	Trigger       Trigger
	FilterLogic   AndOrType
	Filters       []Filter
	Action        func(context.Context) error
	LastTriggered time.Time
	running       bool
}

const maxCategoryRunsPerEvaluation = 4

type Scheduler struct {
	schedules []*DailySchedule
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	clock     Clock
}

// NewScheduler creates a new scheduler instance with real clock
func NewScheduler() *Scheduler {
	return NewSchedulerWithClock(&RealClock{})
}

// NewSchedulerWithClock creates a new scheduler instance with custom clock
func NewSchedulerWithClock(clock Clock) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		schedules: make([]*DailySchedule, 0),
		ctx:       ctx,
		cancel:    cancel,
		clock:     clock,
	}
}

// AddSchedule adds a schedule to the scheduler
func (s *Scheduler) AddSchedule(schedule *DailySchedule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.schedules = append(s.schedules, schedule)
	s.logScheduleAdded(schedule)
}

// Start begins running the scheduler
func (s *Scheduler) Start() {
	s.wg.Add(1)
	s.logStart()
	go s.run()
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			s.logStop()
			return
		case <-s.clock.After(1 * time.Minute):
			s.evaluate(s.clock.Now())
		}
	}
}

// evaluate checks all schedules and executes due schedules in insertion order.
// Category schedules are allowed to override each other within the same cycle,
// but only if the trigger belongs to the current day and the per-cycle category
// execution cap has not been reached.
func (s *Scheduler) evaluate(now time.Time) {
	s.mu.RLock()
	schedules := make([]*DailySchedule, len(s.schedules))
	copy(schedules, s.schedules)
	s.mu.RUnlock()

	categoryRuns := make(map[string]int)
	for _, sch := range schedules {
		t := sch.Trigger.Time()
		reason := "trigger_time_not_reached"

		if hasTriggeredThisPeriod(sch, now) {
			reason = "already_triggered_today"
			s.logScheduleSkip(sch, now, t, reason)
			continue
		}

		if !sameDay(t, now) {
			reason = "trigger_not_due_today"
			s.logScheduleSkip(sch, now, t, reason)
			continue
		}

		if !s.shouldTrigger(sch, now) {
			s.logScheduleSkip(sch, now, t, reason)
			continue
		}

		if !s.filtersPass(sch, now) {
			reason = "filters_not_passed"
			s.logScheduleSkip(sch, now, t, reason)
			continue
		}

		if s.isRunning(sch) {
			reason = "action_in_progress"
			s.logScheduleSkip(sch, now, t, reason)
			continue
		}

		if sch.Category != "" && categoryRuns[sch.Category] >= maxCategoryRunsPerEvaluation {
			reason = "category_run_limit_reached"
			s.logScheduleSkip(sch, now, t, reason)
			continue
		}

		if sch.Category != "" {
			categoryRuns[sch.Category]++
		}

		s.logScheduleTrigger(sch, now)
		s.executeSchedule(sch, now)
	}
}

func (s *Scheduler) isRunning(schedule *DailySchedule) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return schedule.running
}

func (s *Scheduler) executeSchedule(schedule *DailySchedule, now time.Time) {
	start := s.clock.Now()

	s.mu.Lock()
	schedule.running = true
	s.mu.Unlock()

	s.logActionStart(schedule, start)
	defer func() {
		s.mu.Lock()
		schedule.running = false
		s.mu.Unlock()
		if r := recover(); r != nil {
			s.logActionPanic(schedule, r)
		}
	}()

	if err := schedule.Action(s.ctx); err != nil {
		log.Error().Err(err).Str("event", "action_error").Str("schedule", schedule.Name).Msg("action failed; will retry next cycle")
		return
	}

	s.mu.Lock()
	schedule.LastTriggered = now
	s.mu.Unlock()
	s.logActionFinish(schedule, start)
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() &&
		a.Month() == b.Month() &&
		a.Day() == b.Day()
}

// shouldTrigger checks if the trigger condition is met
func (s *Scheduler) shouldTrigger(schedule *DailySchedule, now time.Time) bool {
	if hasTriggeredThisPeriod(schedule, now) {
		return false
	}
	t := schedule.Trigger.Time()
	// Compare absolute instants instead of naive hour/minute fields which break across timezones.
	// Trigger when now >= t.
	return !now.Before(t)
}

func hasTriggeredThisPeriod(schedule *DailySchedule, now time.Time) bool {
	if schedule.LastTriggered.IsZero() {
		return false
	}
	return schedule.LastTriggered.Year() == now.Year() &&
		schedule.LastTriggered.Month() == now.Month() &&
		schedule.LastTriggered.Day() == now.Day()
}

// filtersPass checks if all filters pass according to logic type
func (s *Scheduler) filtersPass(schedule *DailySchedule, now time.Time) bool {
	if len(schedule.Filters) == 0 {
		return true
	}

	if schedule.FilterLogic == OR {
		for _, filter := range schedule.Filters {
			if s.filterPass(filter, now) {
				return true
			}
		}
		return false
	}

	// Default to AND logic
	for _, filter := range schedule.Filters {
		if !s.filterPass(filter, now) {
			return false
		}
	}
	return true
}

// filterPass checks if a single filter passes
func (s *Scheduler) filterPass(filter Filter, now time.Time) bool {
	switch filter.Type {
	case FilterDate:
		switch filter.Comparator {
		case Equal:
			return now.Year() == filter.Date.Year() &&
				now.Month() == filter.Date.Month() &&
				now.Day() == filter.Date.Day()
		case LessThan:
			return now.Before(filter.Date)
		case GreaterThan:
			return now.After(filter.Date)
		default:
			log.Info().Msg("No filter matched for: " + filter.Date.String())
		}
	}
	return true
}

// --- Logging helpers (centralized formatting) ---
func (s *Scheduler) logScheduleAdded(schedule *DailySchedule) {
	trigInfo := schedule.Trigger.Time().Format(time.RFC3339)
	evt := log.Info().Str("event", "schedule_added").Str("name", schedule.Name).Str("trigger_time", trigInfo).Int("filters", len(schedule.Filters))
	if schedule.Category != "" {
		evt = evt.Str("category", schedule.Category)
	}
	evt.Msg("schedule registered")
}

func (s *Scheduler) logStart() {
	s.mu.RLock()
	cnt := len(s.schedules)
	names := make([]string, 0, cnt)
	for _, sch := range s.schedules {
		names = append(names, sch.Name)
	}
	s.mu.RUnlock()
	log.Info().Str("event", "scheduler_start").Int("schedule_count", cnt).Strs("schedules", names).Msg("scheduler started")
}

func (s *Scheduler) logStop() {
	log.Info().Str("event", "scheduler_stop").Msg("scheduler stopping")
}

func (s *Scheduler) logScheduleTrigger(schedule *DailySchedule, now time.Time) {
	evt := log.Info().Str("event", "schedule_trigger").Str("name", schedule.Name).Time("now", now)
	if schedule.Category != "" {
		evt = evt.Str("category", schedule.Category)
	}
	evt.Msg("executing schedule action")
}

func (s *Scheduler) logScheduleSkip(schedule *DailySchedule, now, triggerT time.Time, reason string) {
	evt := log.Debug().Str("event", "schedule_skip").Str("name", schedule.Name).Str("reason", reason).Time("now", now).Time("trigger_time", triggerT).Time("last_triggered", schedule.LastTriggered)
	if schedule.Category != "" {
		evt = evt.Str("category", schedule.Category)
	}
	evt.Msg("schedule not executed")
}

func (s *Scheduler) logActionStart(schedule *DailySchedule, start time.Time) {
	log.Info().Str("event", "action_start").Str("schedule", schedule.Name).Time("start", start).Msg("action started")
}

func (s *Scheduler) logActionFinish(schedule *DailySchedule, start time.Time) {
	dur := time.Since(start)
	log.Info().Str("event", "action_finish").Str("schedule", schedule.Name).Dur("duration", dur).Msg("action finished")
}

func (s *Scheduler) logActionPanic(schedule *DailySchedule, r interface{}) {
	log.Error().Str("event", "action_panic").Str("schedule", schedule.Name).Interface("panic", r).Bytes("stack", debug.Stack()).Msg("schedule action panic")
}
