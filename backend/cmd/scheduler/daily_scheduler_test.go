package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createSchedule(t func() time.Time, fn func(context.Context) error) *Scheduler {
	scheduler := NewScheduler()

	schedule := &DailySchedule{
		Name: "Test Schedule",
		Trigger: Trigger{
			Time: t,
		},
		Action: fn,
	}
	scheduler.AddSchedule(schedule)

	return scheduler
}

func TestSunsetSchedule(t *testing.T) {
	// Track if action was called
	var actionCalled bool
	var mu sync.Mutex

	dummyAction := func(ctx context.Context) error {
		mu.Lock()
		actionCalled = true
		mu.Unlock()
		return nil
	}

	t.Run("Should execute at 18:30", func(t *testing.T) {
		now := time.Now()
		actionCalled = false
		scheduleTime := func() time.Time {
			return time.Date(now.Year(), now.Month(), now.Day(), 18, 30, 0, 0, time.Local)
		}
		testTime := time.Date(now.Year(), now.Month(), now.Day(), 18, 30, 0, 0, time.Local)
		scheduler := createSchedule(scheduleTime, dummyAction)
		scheduler.evaluate(testTime)

		time.Sleep(100 * time.Millisecond)

		assert.True(t, actionCalled, "Action should have been executed at 18:30")
	})

	t.Run("Should execute at 18:50", func(t *testing.T) {
		now := time.Now()
		actionCalled = false
		scheduleTime := func() time.Time {
			return time.Date(now.Year(), now.Month(), now.Day(), 18, 30, 0, 0, time.Local)
		}
		testTime := time.Date(now.Year(), now.Month(), now.Day(), 18, 50, 0, 0, time.Local)
		scheduler := createSchedule(scheduleTime, dummyAction)
		scheduler.evaluate(testTime)

		time.Sleep(100 * time.Millisecond)

		assert.True(t, actionCalled, "Action should have been executed at 18:50")
	})

	t.Run("Should not execute at 18:00", func(t *testing.T) {
		now := time.Now()
		actionCalled = false
		scheduleTime := func() time.Time {
			return time.Date(now.Year(), now.Month(), now.Day(), 18, 30, 0, 0, time.Local)
		}
		testTime := time.Date(now.Year(), now.Month(), now.Day(), 18, 00, 0, 0, time.Local)
		scheduler := createSchedule(scheduleTime, dummyAction)
		scheduler.evaluate(testTime)

		time.Sleep(100 * time.Millisecond)

		assert.False(t, actionCalled, "Action should not have been executed at 18:00")
	})
	t.Run("Should not execute second time same day but day after", func(t *testing.T) {
		now := time.Now()
		actionCalled = false
		scheduleTime := func() time.Time {
			return time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, time.Local)
		}
		testTime := time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, time.Local)
		scheduler := createSchedule(scheduleTime, dummyAction)
		scheduler.evaluate(testTime)

		time.Sleep(100 * time.Millisecond)

		assert.True(t, actionCalled, "Action should have been executed first time")

		actionCalled = false
		testTime = time.Date(now.Year(), now.Month(), now.Day(), 21, 0, 0, 0, time.Local)
		scheduler.evaluate(testTime)

		time.Sleep(100 * time.Millisecond)

		assert.False(t, actionCalled, "Action should NOT have been executed second time same day")

		tomorrowTestTime := testTime.Add(24 * time.Hour)
		scheduler.evaluate(tomorrowTestTime)

		time.Sleep(100 * time.Millisecond)

		assert.True(t, actionCalled, "Action should have been executed next day")
	})
}

func TestOfflineActionRetry(t *testing.T) {
	now := time.Now()
	scheduleTime := func() time.Time {
		return time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.Local)
	}
	attempts := 0
	failing := func(ctx context.Context) error {
		attempts++
		return assert.AnError
	}
	scheduler := createSchedule(scheduleTime, failing)
	scheduler.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.Local))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, attempts)
	if !scheduler.schedules[0].LastTriggered.IsZero() {
		t.Fatalf("expected LastTriggered zero after failure")
	}
	scheduler.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 12, 1, 0, 0, time.Local))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, attempts)
	if !scheduler.schedules[0].LastTriggered.IsZero() {
		t.Fatalf("expected LastTriggered still zero after second failure")
	}
}

func TestCategorySupersede(t *testing.T) {
	now := time.Now()
	// Both schedules trigger at 10:00; evaluation at 10:05 -> both eligible
	trig := func() time.Time { return time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local) }
	executed := []string{}
	var mu sync.Mutex
	makeAction := func(name string) func(context.Context) error {
		return func(ctx context.Context) error {
			mu.Lock()
			executed = append(executed, name)
			mu.Unlock()
			return nil
		}
	}
	scheduler := NewScheduler()
	scheduler.AddSchedule(&DailySchedule{Name: "First", Category: "group1", Trigger: Trigger{Time: trig}, Action: makeAction("First")})
	scheduler.AddSchedule(&DailySchedule{Name: "Second", Category: "group1", Trigger: Trigger{Time: trig}, Action: makeAction("Second")})
	scheduler.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 10, 5, 0, 0, time.Local))
	time.Sleep(100 * time.Millisecond)
	// Expect only last (Second) executed
	assert.Equal(t, []string{"Second"}, executed)
	if !scheduler.schedules[1].LastTriggered.IsZero() && !scheduler.schedules[0].LastTriggered.IsZero() {
		t.Fatalf("expected only last schedule LastTriggered to be set")
	}
}

func TestCategorySupersedeLatestTimeWins(t *testing.T) {
	now := time.Now()
	// Two schedules in same category with different trigger times; added in reverse chronological order
	trigLate := func() time.Time { return time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, time.Local) }
	trigEarly := func() time.Time { return time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local) }
	var mu sync.Mutex
	executed := []string{}
	act := func(name string) func(context.Context) error {
		return func(ctx context.Context) error {
			mu.Lock()
			executed = append(executed, name)
			mu.Unlock()
			return nil
		}
	}
	s := NewScheduler()
	// Add early (later in list) after late to ensure ordering alone doesn't decide
	s.AddSchedule(&DailySchedule{Name: "Late", Category: "cat1", Trigger: Trigger{Time: trigLate}, Action: act("Late")})
	s.AddSchedule(&DailySchedule{Name: "Early", Category: "cat1", Trigger: Trigger{Time: trigEarly}, Action: act("Early")})
	// Evaluate after both times passed
	s.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, time.Local))
	time.Sleep(100 * time.Millisecond)
	// Expect only Late executed because its trigger time (15:00) is later than 10:00
	assert.Equal(t, []string{"Late"}, executed)
	if s.schedules[0].LastTriggered.IsZero() || !s.schedules[1].LastTriggered.IsZero() {
		// schedules[0] is Late, schedules[1] is Early
		if s.schedules[1].LastTriggered.IsZero() {
			// ok
		} else {
			t.Fatalf("expected Early not to have LastTriggered set")
		}
	}
}

func TestEarlierScheduleDoesNotRunAfterLaterTriggered(t *testing.T) {
	now := time.Now()
	trigEarly := func() time.Time { return time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.Local) }
	trigLate := func() time.Time { return time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, time.Local) }
	var mu sync.Mutex
	executed := []string{}
	act := func(name string) func(context.Context) error {
		return func(ctx context.Context) error { mu.Lock(); executed = append(executed, name); mu.Unlock(); return nil }
	}
	s := NewScheduler()
	s.AddSchedule(&DailySchedule{Name: "Early", Category: "lights", Trigger: Trigger{Time: trigEarly}, Action: act("Early")})
	s.AddSchedule(&DailySchedule{Name: "Late", Category: "lights", Trigger: Trigger{Time: trigLate}, Action: act("Late")})

	// Morning: only early should run (late not yet time)
	s.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 8, 5, 0, 0, time.Local))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, []string{"Early"}, executed)

	// Evening after both times: only late should run, early must NOT re-run
	s.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 19, 0, 0, 0, time.Local))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, []string{"Early", "Late"}, executed)

	// Another later evaluation: early must remain suppressed
	s.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, time.Local))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, []string{"Early", "Late"}, executed)
}

// TestTimezoneTrigger ensures that a trigger time expressed in a non-UTC location
// fires at the correct absolute instant even if the evaluation 'now' is in UTC.
// Previously the scheduler only compared Hour/Minute fields, causing times whose
// zone hour differed to never match until hours aligned incorrectly.
func TestTimezoneTrigger(t *testing.T) {
	// Helsinki location
	helsinki, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}
	// Sunset example: 16:08:19 local Helsinki
	triggerInstant := time.Date(2025, 11, 8, 16, 8, 19, 0, helsinki)

	// Action records execution time
	var executedAt time.Time
	act := func(ctx context.Context) error { executedAt = time.Now(); return nil }
	s := NewScheduler()
	s.AddSchedule(&DailySchedule{Name: "Sunset", Trigger: Trigger{Time: func() time.Time { return triggerInstant }}, Action: act})

	// Evaluate a moment BEFORE the trigger in UTC equivalent (triggerInstant in UTC is 14:08:19)
	beforeUTC := triggerInstant.In(time.UTC).Add(-time.Second) // 14:08:18 UTC
	s.evaluate(beforeUTC)
	time.Sleep(25 * time.Millisecond)
	if !executedAt.IsZero() {
		t.Fatalf("action executed too early at %v", executedAt)
	}

	// Evaluate exactly at the trigger absolute instant in UTC
	atUTC := triggerInstant.In(time.UTC) // 14:08:19 UTC
	s.evaluate(atUTC)
	time.Sleep(25 * time.Millisecond)
	if executedAt.IsZero() {
		t.Fatalf("expected action to execute at trigger instant")
	}
	// Ensure schedule LastTriggered set
	if s.schedules[0].LastTriggered.IsZero() {
		t.Fatalf("expected LastTriggered to be set")
	}
}
