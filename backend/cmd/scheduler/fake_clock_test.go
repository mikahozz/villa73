package main

import (
	"sync"
	"time"
)

// FakeClock is a test implementation of the Clock interface.
// It allows manual control of "current time" and deterministic firing
// of one-shot timers created via After(). Used by integration tests
// to drive the scheduler without real waiting.
type FakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

// fakeTimer models a single pending After() call.
// Purpose:
//   - Store its absolute deadline.
//   - Provide a channel the scheduler is blocked on.
//
// Firing:
//   - When FakeClock.Advance moves now >= deadline, the channel
//     receives the (new) current time and the timer is removed.
type fakeTimer struct {
	deadline time.Time
	ch       chan time.Time
}

// NewFakeClock returns a FakeClock starting at the given time.
func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{now: start, timers: []*fakeTimer{}}
}

// Now returns the current fake time.
func (fc *FakeClock) Now() time.Time {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.now
}

// After schedules a one-shot timer that will fire once fake time reaches deadline.
func (fc *FakeClock) After(d time.Duration) <-chan time.Time {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	t := &fakeTimer{
		deadline: fc.now.Add(d),
		ch:       make(chan time.Time, 1), // buffered to avoid blocking on rapid advances
	}
	fc.timers = append(fc.timers, t)
	return t.ch
}

// Advance moves time forward and fires all timers whose deadlines have passed.
func (fc *FakeClock) Advance(d time.Duration) {
	fc.mu.Lock()
	fc.now = fc.now.Add(d)
	cur := fc.now
	var remaining []*fakeTimer
	for _, tm := range fc.timers {
		if !cur.Before(tm.deadline) { // cur >= deadline
			tm.ch <- cur
		} else {
			remaining = append(remaining, tm)
		}
	}
	fc.timers = remaining
	fc.mu.Unlock()
}
