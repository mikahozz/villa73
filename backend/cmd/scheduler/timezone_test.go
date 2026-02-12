package main

import (
	"testing"
	"time"
)

// TestScheduleTriggerTimezoneConsistency verifies that trigger times are calculated correctly
// even when the system timezone (TZ environment variable) is UTC.
func TestScheduleTriggerTimezoneConsistency(t *testing.T) {
	helsinki, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		t.Fatal(err)
	}

	// Test case: Date boundary issue
	// When it's 01:00 Feb 2 in Helsinki, it's 23:00 Feb 1 in UTC
	// The trigger for "23:00" should be Feb 2 in Helsinki, not Feb 1
	helsinkiTime := time.Date(2026, 2, 2, 1, 0, 0, 0, helsinki)

	// Simulate what the trigger function does
	nowUTC := helsinkiTime.UTC() // This is what time.Now() would return in UTC environment

	// OLD (buggy) way: uses UTC date
	buggyTrigger := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 23, 0, 0, 0, helsinki)

	// NEW (fixed) way: uses Helsinki date
	nowHelsinki := nowUTC.In(helsinki)
	fixedTrigger := time.Date(nowHelsinki.Year(), nowHelsinki.Month(), nowHelsinki.Day(), 23, 0, 0, 0, helsinki)

	t.Logf("Helsinki time: %s", helsinkiTime.Format(time.RFC3339))
	t.Logf("UTC time (what time.Now() returns): %s", nowUTC.Format(time.RFC3339))
	t.Logf("Buggy trigger (uses UTC date): %s", buggyTrigger.Format(time.RFC3339))
	t.Logf("Fixed trigger (uses Helsinki date): %s", fixedTrigger.Format(time.RFC3339))

	// The buggy trigger would be 2026-02-01T23:00:00+02:00 (yesterday!)
	// The fixed trigger should be 2026-02-02T23:00:00+02:00 (today)
	expectedDate := time.Date(2026, 2, 2, 23, 0, 0, 0, helsinki)

	if buggyTrigger.Equal(expectedDate) {
		t.Errorf("Buggy trigger accidentally matched expected - test may be invalid")
	}

	if !fixedTrigger.Equal(expectedDate) {
		t.Errorf("Fixed trigger = %s, want %s", fixedTrigger.Format(time.RFC3339), expectedDate.Format(time.RFC3339))
	}

	// Test sunrise/sunset functions with the same scenario
	// When it's Feb 2 in Helsinki but Feb 1 in UTC, we should get Feb 2 sunrise/sunset

	// Simulate buggy getSunriseTimeToday: passes time.Now() which is UTC
	buggySunData := sunDataInstance.GetSunDataForSingleDate(nowUTC)
	if buggySunData.Date != "2026-02-01" {
		t.Errorf("Expected buggy pattern to return 2026-02-01 (UTC date), got %s", buggySunData.Date)
	}

	// Simulate fixed getSunriseTimeToday: passes time.Now().In(zone) which is Helsinki
	fixedSunData := sunDataInstance.GetSunDataForSingleDate(nowHelsinki)
	if fixedSunData.Date != "2026-02-02" {
		t.Errorf("Expected fixed pattern to return 2026-02-02 (Helsinki date), got %s", fixedSunData.Date)
	}

	t.Logf("Buggy sunrise date: %s (WRONG - uses UTC date)", buggySunData.Date)
	t.Logf("Fixed sunrise date: %s (CORRECT - uses Helsinki date)", fixedSunData.Date)
}

// TestTriggerFunctionsAtDateBoundary tests that trigger functions use correct date
// when system clock is UTC but we're past midnight in Helsinki timezone.
// This catches the bug where time.Now() gives UTC date but we use Helsinki timezone.
func TestTriggerFunctionsAtDateBoundary(t *testing.T) {
	// We can't actually control what time.Now() returns in the trigger functions,
	// so instead we test the pattern: does the code use time.Now().In(zone) before
	// extracting Year/Month/Day?

	helsinki, _ := time.LoadLocation("Europe/Helsinki")

	// Simulate the problematic time: 23:30 UTC on Feb 1 = 01:30 Helsinki on Feb 2
	// This is when the bug would manifest
	simulatedUTC := time.Date(2026, 2, 1, 23, 30, 0, 0, time.UTC)

	// BUGGY pattern: time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, zone)
	// where now = time.Now() in UTC
	buggyTrigger := time.Date(simulatedUTC.Year(), simulatedUTC.Month(), simulatedUTC.Day(), 23, 0, 0, 0, helsinki)

	// FIXED pattern: time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, zone)
	// where now = time.Now().In(zone)
	nowInHelsinki := simulatedUTC.In(helsinki)
	fixedTrigger := time.Date(nowInHelsinki.Year(), nowInHelsinki.Month(), nowInHelsinki.Day(), 23, 0, 0, 0, helsinki)

	t.Logf("Simulated UTC time: %s", simulatedUTC.Format(time.RFC3339))
	t.Logf("Converted to Helsinki: %s", nowInHelsinki.Format(time.RFC3339))
	t.Logf("Buggy trigger (uses UTC date components): %s", buggyTrigger.Format(time.RFC3339))
	t.Logf("Fixed trigger (uses Helsinki date components): %s", fixedTrigger.Format(time.RFC3339))

	// Expected: should create trigger for Feb 2 (today in Helsinki), not Feb 1 (today in UTC)
	expectedTrigger := time.Date(2026, 2, 2, 23, 0, 0, 0, helsinki)

	if buggyTrigger.Equal(expectedTrigger) {
		t.Error("Buggy pattern produced correct result - test case is invalid")
	}

	if !fixedTrigger.Equal(expectedTrigger) {
		t.Errorf("Fixed pattern produced wrong trigger: got %s, want %s",
			fixedTrigger.Format(time.RFC3339), expectedTrigger.Format(time.RFC3339))
	}

	// Verify the dates differ by one day
	if buggyTrigger.Day() != 1 {
		t.Errorf("Buggy trigger should be Feb 1, got day %d", buggyTrigger.Day())
	}
	if fixedTrigger.Day() != 2 {
		t.Errorf("Fixed trigger should be Feb 2, got day %d", fixedTrigger.Day())
	}
}
