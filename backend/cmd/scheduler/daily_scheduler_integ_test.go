package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Integration tests using FakeClock to drive the real scheduler loop.

func wait() { time.Sleep(1 * time.Millisecond) }

func advanceMinutes(fc *FakeClock, minutes int) {
	for i := 0; i < minutes; i++ {
		fc.Advance(time.Minute)
		wait()
	}
}

func buildScheduler(clock *FakeClock, offCalls, onCalls *int32) *Scheduler {
	s := NewSchedulerWithClock(clock)

	s.AddSchedule(&DailySchedule{
		Name: "Night Lights OFF",
		Trigger: Trigger{Time: func() time.Time {
			cur := clock.Now()
			return time.Date(cur.Year(), cur.Month(), cur.Day(), 7, 30, 0, 0, cur.Location())
		}},
		Action: func(ctx context.Context) error { atomic.AddInt32(offCalls, 1); return nil },
	})

	s.AddSchedule(&DailySchedule{
		Name: "Night Lights ON",
		Trigger: Trigger{Time: func() time.Time {
			cur := clock.Now()
			return time.Date(cur.Year(), cur.Month(), cur.Day(), 16, 30, 0, 0, cur.Location())
		}},
		Action: func(ctx context.Context) error { atomic.AddInt32(onCalls, 1); return nil },
	})

	return s
}

func TestScheduler_Integration_SingleDay(t *testing.T) {
	start := time.Date(2025, 11, 3, 5, 0, 0, 0, time.Local)
	fc := NewFakeClock(start)
	var off, on int32
	s := buildScheduler(fc, &off, &on)
	s.Start()
	defer s.Stop()

	advanceMinutes(fc, 149) // 05:00 -> 07:29
	assert.Equal(t, int32(0), off)
	assert.Equal(t, int32(0), on)

	advanceMinutes(fc, 1) // 07:30
	assert.Equal(t, int32(1), off)
	assert.Equal(t, int32(0), on)

	advanceMinutes(fc, (16-7)*60+(29-30)) // 07:31 -> 16:29
	assert.Equal(t, int32(1), off)
	assert.Equal(t, int32(0), on)

	advanceMinutes(fc, 1) // 16:30
	assert.Equal(t, int32(1), off)
	assert.Equal(t, int32(1), on)

	advanceMinutes(fc, (23-16)*60-30) // 16:31 -> 23:00
	assert.Equal(t, int32(1), off)
	assert.Equal(t, int32(1), on)
}

func TestScheduler_Integration_NextDayReset(t *testing.T) {
	start := time.Date(2025, 11, 3, 6, 0, 0, 0, time.Local)
	fc := NewFakeClock(start)
	var off, on int32
	s := buildScheduler(fc, &off, &on)
	s.Start()
	defer s.Stop()

	advanceMinutes(fc, 90) // 06:00 -> 07:30
	assert.Equal(t, int32(1), off)
	assert.Equal(t, int32(0), on)

	advanceMinutes(fc, (16-7)*60) // 07:30 -> 16:30
	assert.Equal(t, int32(1), off)
	assert.Equal(t, int32(1), on)

	advanceMinutes(fc, (24-16)*60-30+7*60+30) // 16:30 -> midnight -> next 07:30
	assert.Equal(t, int32(2), off)
	assert.Equal(t, int32(1), on)

	advanceMinutes(fc, (16-7)*60) // next 07:30 -> 16:30
	assert.Equal(t, int32(2), off)
	assert.Equal(t, int32(2), on)
}
