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
	Category      string // optional grouping; only last eligible schedule in same category runs per evaluation cycle
	Trigger       Trigger
	FilterLogic   AndOrType
	Filters       []Filter
	Action        func(context.Context) error
	LastTriggered time.Time
}

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

// evaluate checks all schedules and executes matching ones
func (s *Scheduler) evaluate(now time.Time) {
	s.mu.RLock()
	schedules := make([]*DailySchedule, len(s.schedules))
	copy(schedules, s.schedules)
	s.mu.RUnlock()

	// Track, per category, the latest trigger time that has already fired today.
	triggeredMax := make(map[string]time.Time)
	for _, sch := range schedules {
		if sch.Category == "" || sch.LastTriggered.IsZero() || !hasTriggeredThisPeriod(sch, now) {
			continue
		}
		t := sch.Trigger.Time()
		if prev, ok := triggeredMax[sch.Category]; !ok || t.After(prev) {
			triggeredMax[sch.Category] = t
		}
	}

	// Determine candidate per category for this evaluation.
	candidates := make(map[string]*DailySchedule)
	candidateTime := make(map[string]time.Time)
	eligible := make(map[*DailySchedule]bool)
	for _, sch := range schedules {
		// Basic eligibility (time reached & not triggered today)
		if !s.shouldTrigger(sch, now) {
			continue
		}
		if !s.filtersPass(sch, now) {
			continue
		}
		t := sch.Trigger.Time()
		cat := sch.Category
		eligible[sch] = true
		// If a schedule already fired today in this category with time >= t, suppress (we only allow later times).
		if cat != "" {
			if firedT, ok := triggeredMax[cat]; ok && (t.Before(firedT) || t.Equal(firedT)) {
				eligible[sch] = false
				continue
			}
			prevT, ok := candidateTime[cat]
			if !ok || t.After(prevT) || t.Equal(prevT) { // tie -> later-added wins
				candidates[cat] = sch
				candidateTime[cat] = t
			}
		} else {
			// No category: treat each as its own candidate under a unique key (use name)
			candidates[sch.Name] = sch
		}
	}

	for _, sch := range schedules {
		catKey := sch.Category
		if catKey == "" {
			catKey = sch.Name
		}
		isWinner := candidates[catKey] == sch && eligible[sch]
		if isWinner {
			s.logScheduleTrigger(sch, now)
			go func(sch *DailySchedule) {
				start := s.clock.Now()
				s.logActionStart(sch, start)
				defer func() {
					if r := recover(); r != nil {
						s.logActionPanic(sch, r)
					}
				}()
				if err := sch.Action(s.ctx); err != nil {
					log.Error().Err(err).Str("event", "action_error").Str("schedule", sch.Name).Msg("action failed; will retry next cycle")
				} else {
					s.mu.Lock()
					sch.LastTriggered = now
					s.mu.Unlock()
					s.logActionFinish(sch, start)
				}
			}(sch)
			continue
		}
		// Derive skip reason
		reason := "trigger_time_not_reached"
		if hasTriggeredThisPeriod(sch, now) {
			reason = "already_triggered_today"
		} else if eligible[sch] { // eligible but not winner
			reason = "superseded_by_later_schedule"
		} else if s.shouldTrigger(sch, now) && sch.Category != "" {
			// suppressed due to later already triggered
			t := sch.Trigger.Time()
			if firedT, ok := triggeredMax[sch.Category]; ok && (t.Before(firedT) || t.Equal(firedT)) {
				reason = "earlier_than_triggered_later_schedule"
			}
		} else if s.shouldTrigger(sch, now) && !s.filtersPass(sch, now) {
			reason = "filters_not_passed"
		}
		triggerT := sch.Trigger.Time()
		s.logScheduleSkip(sch, now, triggerT, reason)
	}
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
