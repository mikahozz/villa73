// Package main implements a time-based daily action scheduler.
//
// # Design overview
//
// Schedules are registered via AddSchedule and the scheduler loop fires every
// minute, calling evaluate(now) on each tick.
//
// evaluate(now) inspects every registered schedule in insertion order.
// Uncategorized schedules are skipped when any of the following gates fail:
//
//  1. Trigger time is not today (sameDay check)
//  2. Trigger time not yet reached (now < trigger instant)
//  3. Filters do not pass
//  4. Already triggered today   (LastTriggered is on the same calendar day)
//  5. Action is already running
//
// Categorized schedules go through a two-phase process:
//
//	Phase 1 – candidate collection (gates 1-3 only, no execution guards):
//	Every schedule whose trigger time has been reached today and whose filters
//	pass is considered a candidate, even if it has already triggered today.
//	Within each category the *last* such candidate in insertion order is the
//	winner.  Applying gates 4-5 is intentionally deferred so that an
//	already-triggered winner keeps suppressing earlier (lower-priority)
//	candidates on subsequent evaluation cycles.
//
//	Phase 2 – winner execution (gates 4-5):
//	The category winner is queued for execution only when it has not already
//	triggered today and is not currently running.
//
// The "last triggerable wins" rule makes it easy to express override logic.
// Example: "morning lights ON at 06:45" followed by "morning lights OFF at
// sunrise" share a category.  When sunrise is before 06:45 the OFF schedule
// is added later in the list, so it wins and lights are never turned on.
// When sunrise is after 06:45 the ON schedule wins instead.
//
// Actions are executed synchronously within evaluate; the scheduler loop
// therefore blocks until all actions for a given tick complete.
// Action errors prevent LastTriggered from being stamped, causing the action
// to be retried on the next evaluation cycle (once per minute).
package main

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// --- Clock abstraction ---

// Clock abstracts time operations so tests can inject a fake clock.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

// RealClock delegates to the standard library.
type RealClock struct{}

func (rc *RealClock) Now() time.Time                         { return time.Now() }
func (rc *RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// --- Schedule definition types ---

type FilterType string
type Comparator string
type AndOrType string

const (
	FilterDate FilterType = "date"

	LessThan    Comparator = "less_than"
	GreaterThan Comparator = "greater_than"
	Equal       Comparator = "equal"

	AND AndOrType = "and"
	OR  AndOrType = "or"
)

// Trigger holds a function that returns the trigger instant for the current day.
// The function is called on every evaluation cycle, which lets dynamic triggers
// (e.g. today's sunrise time) stay up to date.
type Trigger struct {
	Time func() time.Time
}

// Filter constrains when a schedule is eligible to run.
type Filter struct {
	Type       FilterType
	Date       time.Time
	Comparator Comparator
}

// DailySchedule describes a single recurring action and its trigger conditions.
//
// Category groups related schedules so that only the last triggerable one
// in insertion order runs per evaluation cycle (see package-level doc).
// Leave Category empty if a schedule should always run independently.
//
// LastTriggered is stamped only on successful completion; a failing action
// leaves it zero so the scheduler retries on the next cycle.
type DailySchedule struct {
	Name          string
	Category      string
	Trigger       Trigger
	FilterLogic   AndOrType
	Filters       []Filter
	Action        func(context.Context) error
	LastTriggered time.Time
	running       bool // guarded by Scheduler.mu
}

// --- Scheduler lifecycle ---

// Scheduler holds the registered schedules and drives the evaluation loop.
type Scheduler struct {
	schedules []*DailySchedule
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	clock     Clock
}

// NewScheduler creates a Scheduler backed by the real system clock.
func NewScheduler() *Scheduler {
	return NewSchedulerWithClock(&RealClock{})
}

// NewSchedulerWithClock creates a Scheduler with a custom clock (useful in tests).
func NewSchedulerWithClock(clock Clock) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		schedules: make([]*DailySchedule, 0),
		ctx:       ctx,
		cancel:    cancel,
		clock:     clock,
	}
}

// AddSchedule registers a schedule. Schedules are evaluated in insertion order.
func (s *Scheduler) AddSchedule(schedule *DailySchedule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.schedules = append(s.schedules, schedule)
	s.logScheduleAdded(schedule)
}

// Start launches the scheduler loop in a background goroutine.
func (s *Scheduler) Start() {
	s.wg.Add(1)
	s.logStart()
	go s.run()
}

// Stop signals the scheduler to stop and waits for the loop to exit.
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
}

// run is the main scheduler loop: evaluate once per minute.
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

// --- Evaluation ---

// evaluate checks every registered schedule against now and executes the ones
// that are due.  See the package-level doc for the full gate and category logic.
func (s *Scheduler) evaluate(now time.Time) {
	s.mu.RLock()
	schedules := make([]*DailySchedule, len(s.schedules))
	copy(schedules, s.schedules)
	s.mu.RUnlock()

	// categoryCandidates holds the latest triggerable schedule per category.
	// categoryOrder preserves the first-seen insertion order for deterministic execution.
	categoryCandidates := make(map[string]*DailySchedule)
	categoryOrder := make([]string, 0)
	queue := make([]*DailySchedule, 0)

	for _, sch := range schedules {
		triggerTime := sch.Trigger.Time()

		if !sameDay(triggerTime, now) {
			s.logScheduleSkip(sch, now, triggerTime, "trigger_not_due_today")
			continue
		}
		if now.Before(triggerTime) {
			s.logScheduleSkip(sch, now, triggerTime, "trigger_time_not_reached")
			continue
		}
		if !s.filtersPass(sch, now) {
			s.logScheduleSkip(sch, now, triggerTime, "filters_not_passed")
			continue
		}

		if sch.Category == "" {
			// For uncategorized schedules apply execution guards immediately.
			if hasTriggeredToday(sch, now) {
				s.logScheduleSkip(sch, now, triggerTime, "already_triggered_today")
				continue
			}
			if s.isRunning(sch) {
				s.logScheduleSkip(sch, now, triggerTime, "action_in_progress")
				continue
			}
			queue = append(queue, sch)
			continue
		}

		// For categorized schedules: track all time-eligible candidates regardless of
		// whether they have already triggered today.  Winner selection must consider
		// every schedule that has reached its trigger time so that an already-triggered
		// winner continues to suppress earlier (lower-priority) candidates on subsequent
		// evaluation cycles.  Execution guards are applied to the winner only, below.
		if _, seen := categoryCandidates[sch.Category]; !seen {
			categoryOrder = append(categoryOrder, sch.Category)
		}
		categoryCandidates[sch.Category] = sch
	}

	// For each category, queue the winning candidate only when it has not already
	// run today and is not currently executing.
	for _, cat := range categoryOrder {
		sch := categoryCandidates[cat]
		triggerTime := sch.Trigger.Time()
		if hasTriggeredToday(sch, now) {
			s.logScheduleSkip(sch, now, triggerTime, "already_triggered_today")
			continue
		}
		if s.isRunning(sch) {
			s.logScheduleSkip(sch, now, triggerTime, "action_in_progress")
			continue
		}
		queue = append(queue, sch)
	}

	for _, sch := range queue {
		s.logScheduleTrigger(sch, now)
		s.executeSchedule(sch, now)
	}
}

// hasTriggeredToday reports whether the schedule has already run on the same
// calendar day as now (using the schedule's own timezone if any).
func hasTriggeredToday(schedule *DailySchedule, now time.Time) bool {
	lt := schedule.LastTriggered
	return !lt.IsZero() &&
		lt.Year() == now.Year() &&
		lt.Month() == now.Month() &&
		lt.Day() == now.Day()
}

// sameDay reports whether two instants fall on the same calendar day.
func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// isRunning reports whether the schedule's action goroutine is still executing.
func (s *Scheduler) isRunning(schedule *DailySchedule) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return schedule.running
}

// executeSchedule runs the action synchronously.
// LastTriggered is stamped only on success; a failed action is retried next cycle.
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
		log.Error().Err(err).Str("event", "action_error").Str("schedule", schedule.Name).
			Msg("action failed; will retry next cycle")
		return
	}

	s.mu.Lock()
	schedule.LastTriggered = now
	s.mu.Unlock()
	s.logActionFinish(schedule, start)
}

// --- Filter evaluation ---

// filtersPass returns true when the schedule's filters allow execution at now.
// An empty filter list is always a pass.
func (s *Scheduler) filtersPass(schedule *DailySchedule, now time.Time) bool {
	if len(schedule.Filters) == 0 {
		return true
	}
	if schedule.FilterLogic == OR {
		for _, f := range schedule.Filters {
			if s.filterPass(f, now) {
				return true
			}
		}
		return false
	}
	// Default: AND — all filters must pass.
	for _, f := range schedule.Filters {
		if !s.filterPass(f, now) {
			return false
		}
	}
	return true
}

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

// --- Logging helpers ---

func (s *Scheduler) logScheduleAdded(schedule *DailySchedule) {
	evt := log.Info().Str("event", "schedule_added").Str("name", schedule.Name).
		Str("trigger_time", schedule.Trigger.Time().Format(time.RFC3339)).
		Int("filters", len(schedule.Filters))
	if schedule.Category != "" {
		evt = evt.Str("category", schedule.Category)
	}
	evt.Msg("schedule registered")
}

func (s *Scheduler) logStart() {
	s.mu.RLock()
	names := make([]string, 0, len(s.schedules))
	for _, sch := range s.schedules {
		names = append(names, sch.Name)
	}
	cnt := len(s.schedules)
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
	evt := log.Debug().Str("event", "schedule_skip").Str("name", schedule.Name).
		Str("reason", reason).Time("now", now).Time("trigger_time", triggerT).
		Time("last_triggered", schedule.LastTriggered)
	if schedule.Category != "" {
		evt = evt.Str("category", schedule.Category)
	}
	evt.Msg("schedule not executed")
}

func (s *Scheduler) logActionStart(schedule *DailySchedule, start time.Time) {
	log.Info().Str("event", "action_start").Str("schedule", schedule.Name).Time("start", start).Msg("action started")
}

func (s *Scheduler) logActionFinish(schedule *DailySchedule, start time.Time) {
	log.Info().Str("event", "action_finish").Str("schedule", schedule.Name).
		Dur("duration", time.Since(start)).Msg("action finished")
}

func (s *Scheduler) logActionPanic(schedule *DailySchedule, r interface{}) {
	log.Error().Str("event", "action_panic").Str("schedule", schedule.Name).
		Interface("panic", r).Bytes("stack", debug.Stack()).Msg("schedule action panic")
}
