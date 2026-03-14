package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMorningLightsAfterSunriseEndsOff(t *testing.T) {
	now := time.Date(2026, 3, 14, 7, 0, 0, 0, time.Local)
	trigMorningOn := func() time.Time {
		return time.Date(now.Year(), now.Month(), now.Day(), 6, 45, 0, 0, time.Local)
	}
	trigSunriseOff := func() time.Time {
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
		Action:   act("ON", "on", 40*time.Millisecond),
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
	if len(executed) != 2 || executed[0] != "ON" || executed[1] != "OFF" {
		t.Fatalf("expected deterministic execution order [ON OFF], got %v", executed)
	}
}
