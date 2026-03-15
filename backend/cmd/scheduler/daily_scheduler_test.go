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
		currentEvalTime := time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, time.Local)
		scheduleTime := func() time.Time {
			return time.Date(currentEvalTime.Year(), currentEvalTime.Month(), currentEvalTime.Day(), 20, 0, 0, 0, time.Local)
		}
		testTime := currentEvalTime
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
		currentEvalTime = tomorrowTestTime
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
	assert.Equal(t, []string{"Second"}, executed)
	assert.True(t, scheduler.schedules[0].LastTriggered.IsZero(), "expected first schedule LastTriggered to remain unset")
	assert.False(t, scheduler.schedules[1].LastTriggered.IsZero(), "expected second schedule LastTriggered to be set")
}

func TestCategoryKeepsEarlierCandidateIfLaterNotTriggerable(t *testing.T) {
	now := time.Now()
	trigDue := func() time.Time { return time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local) }
	trigFuture := func() time.Time { return time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, time.Local) }

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
	s.AddSchedule(&DailySchedule{Name: "Due", Category: "cat1", Trigger: Trigger{Time: trigDue}, Action: act("Due")})
	s.AddSchedule(&DailySchedule{Name: "Future", Category: "cat1", Trigger: Trigger{Time: trigFuture}, Action: act("Future")})

	s.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, time.Local))
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, []string{"Due"}, executed)
	assert.False(t, s.schedules[0].LastTriggered.IsZero(), "expected due schedule LastTriggered to be set")
	assert.True(t, s.schedules[1].LastTriggered.IsZero(), "expected future schedule LastTriggered to remain unset")
}

func TestCategoryInsertionOrderOverridesTriggerTime(t *testing.T) {
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
	assert.Equal(t, []string{"Early"}, executed)
	assert.True(t, s.schedules[0].LastTriggered.IsZero(), "expected late schedule LastTriggered to remain unset")
	assert.False(t, s.schedules[1].LastTriggered.IsZero(), "expected early schedule LastTriggered to be set")
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

// TestWinnerSuppressesLoserAfterWinnerHasTriggered is a regression for the bug
// where the category loser would fire on the second evaluation cycle once the
// winner had already run.
//
// Scenario: "ON at 06:45" (first) and "OFF at sunrise 07:00" (second) share a
// category.  At 07:05 OFF wins (last in insertion order) and runs.  On the next
// cycle at 07:06, OFF has already triggered today so it used to be skipped before
// category-winner selection, leaving ON as the sole candidate — which then
// incorrectly ran.  After the fix, OFF is still the category winner and suppresses
// ON even after OFF has triggered.
func TestWinnerSuppressesLoserAfterWinnerHasTriggered(t *testing.T) {
	now := time.Now()
	trigOn := func() time.Time { return time.Date(now.Year(), now.Month(), now.Day(), 6, 45, 0, 0, time.Local) }
	trigOff := func() time.Time { return time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, time.Local) }

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
	s.AddSchedule(&DailySchedule{Name: "ON at 06:45", Category: "morning_lights", Trigger: Trigger{Time: trigOn}, Action: act("ON")})
	s.AddSchedule(&DailySchedule{Name: "OFF at sunrise", Category: "morning_lights", Trigger: Trigger{Time: trigOff}, Action: act("OFF")})

	// First cycle: both are due; OFF wins (last in insertion order) and runs.
	s.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 7, 5, 0, 0, time.Local))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, []string{"OFF"}, executed, "first cycle: only OFF (winner) should run")
	assert.True(t, s.schedules[0].LastTriggered.IsZero(), "ON LastTriggered must remain unset — it was suppressed")
	assert.False(t, s.schedules[1].LastTriggered.IsZero(), "OFF LastTriggered must be set")

	// Second cycle: OFF is still the winner; it has already triggered so nothing runs.
	// ON must NOT run even though its own LastTriggered is still zero.
	s.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 7, 6, 0, 0, time.Local))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, []string{"OFF"}, executed, "second cycle: ON must not run — OFF is still the category winner")

	// Third cycle: same result; continues to stay suppressed for the rest of the day.
	s.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.Local))
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, []string{"OFF"}, executed, "third cycle: still suppressed for the rest of the day")
}

func TestSunriseBefore645ExecutesInInsertionOrder(t *testing.T) {
	now := time.Now()
	trigMorningOn := func() time.Time { return time.Date(now.Year(), now.Month(), now.Day(), 6, 45, 0, 0, time.Local) }
	trigSunriseOff := func() time.Time { return time.Date(now.Year(), now.Month(), now.Day(), 6, 30, 0, 0, time.Local) }
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
	s.AddSchedule(&DailySchedule{Name: "Morning lights ON at 6:45", Category: "night_lights", Trigger: Trigger{Time: trigMorningOn}, Action: act("ON")})
	s.AddSchedule(&DailySchedule{Name: "Morning lights OFF at sunrise", Category: "night_lights", Trigger: Trigger{Time: trigSunriseOff}, Action: act("OFF")})

	s.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, time.Local))
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, []string{"OFF"}, executed)
	assert.True(t, s.schedules[0].LastTriggered.IsZero(), "expected ON schedule LastTriggered to remain unset")
	assert.False(t, s.schedules[1].LastTriggered.IsZero(), "expected OFF schedule LastTriggered to be set")
}

// TestSunriseAfter645EndsOff is a regression for the scenario where sunrise
// falls AFTER 06:45, making both schedules due in the same cycle.  The OFF
// action (added last) must win even when the ON action has an artificial delay —
// confirming that category winner selection is resolved before any action starts.
func TestSunriseAfter645EndsOff(t *testing.T) {
	now := time.Date(2026, 3, 14, 7, 0, 0, 0, time.Local)
	trigMorningOn := func() time.Time {
		return time.Date(now.Year(), now.Month(), now.Day(), 6, 45, 0, 0, time.Local)
	}
	trigSunriseOff := func() time.Time {
		// Sunrise at 06:41 — before 06:45 in wall-clock terms but added later,
		// so it is the last triggerable candidate and must win.
		return time.Date(now.Year(), now.Month(), now.Day(), 6, 41, 43, 0, time.Local)
	}

	var mu sync.Mutex
	finalState := "unknown"
	executed := []string{}

	act := func(name, state string, delay time.Duration) func(context.Context) error {
		return func(ctx context.Context) error {
			time.Sleep(delay)
			mu.Lock()
			executed = append(executed, name)
			finalState = state
			mu.Unlock()
			return nil
		}
	}

	s := NewScheduler()
	s.AddSchedule(&DailySchedule{
		Name:     "Morning lights ON at 6:45",
		Category: "night_lights",
		Trigger:  Trigger{Time: trigMorningOn},
		Action:   act("ON", "on", 40*time.Millisecond), // delay must not affect selection
	})
	s.AddSchedule(&DailySchedule{
		Name:     "Morning lights OFF at sunrise",
		Category: "night_lights",
		Trigger:  Trigger{Time: trigSunriseOff},
		Action:   act("OFF", "off", 0),
	})

	s.evaluate(now)

	mu.Lock()
	defer mu.Unlock()
	if finalState != "off" {
		t.Fatalf("expected daytime lights to end OFF, got %q; executed=%v", finalState, executed)
	}
	if len(executed) != 1 || executed[0] != "OFF" {
		t.Fatalf("expected only the last triggerable category candidate [OFF], got %v", executed)
	}
}

func TestOverdueSchedulesRunOnlyForCurrentDay(t *testing.T) {
	now := time.Now()
	var called bool
	act := func(ctx context.Context) error {
		called = true
		return nil
	}

	s := NewScheduler()
	s.AddSchedule(&DailySchedule{
		Name:     "Yesterday schedule",
		Category: "overdue",
		Trigger: Trigger{
			Time: func() time.Time {
				yesterday := now.Add(-24 * time.Hour)
				return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 10, 0, 0, 0, time.Local)
			},
		},
		Action: act,
	})

	s.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.Local))
	time.Sleep(50 * time.Millisecond)

	assert.False(t, called, "expected prior-day overdue schedule not to run")
	assert.True(t, s.schedules[0].LastTriggered.IsZero(), "expected LastTriggered to remain zero")
}

func TestCategoryRunsOnlyLastTriggerableCandidatePerEvaluation(t *testing.T) {
	now := time.Now()
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
	for i := 0; i < 5; i++ {
		name := "Schedule " + string(rune('A'+i))
		s.AddSchedule(&DailySchedule{
			Name:     name,
			Category: "cap",
			Trigger: Trigger{
				Time: func() time.Time {
					return time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, time.Local)
				},
			},
			Action: act(name),
		})
	}

	s.evaluate(time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.Local))
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, []string{"Schedule E"}, executed)
	for i := 0; i < 4; i++ {
		assert.True(t, s.schedules[i].LastTriggered.IsZero(), "expected non-winning candidate LastTriggered to remain unset")
	}
	assert.False(t, s.schedules[4].LastTriggered.IsZero(), "expected last triggerable candidate to execute")
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
