package main

import (
	"os"
	"testing"
	"time"
)

// TestSunriseSunsetTimezoneIntegration verifies that getSunriseTimeToday() and getSunsetTimeToday()
// return data for the correct date when running in UTC environment but needing Helsinki date.
// This test catches the bug where time.Now() returns UTC time and GetSunDataForSingleDate
// extracts Month/Day from that UTC time instead of Helsinki time.
func TestSunriseSunsetTimezoneIntegration(t *testing.T) {
	// Save original TZ
	originalTZ := os.Getenv("TZ")
	defer func() {
		if originalTZ == "" {
			os.Unsetenv("TZ")
		} else {
			os.Setenv("TZ", originalTZ)
		}
	}()

	// This test only makes sense when it's actually different dates in UTC vs Helsinki
	// We can't control time.Now() in the functions, so we verify the current behavior
	helsinki, _ := time.LoadLocation("Europe/Helsinki")

	// Set system to UTC
	os.Setenv("TZ", "UTC")

	nowUTC := time.Now()
	nowHelsinki := nowUTC.In(helsinki)

	t.Logf("Current UTC time: %s (date: %s)", nowUTC.Format(time.RFC3339), nowUTC.Format("2006-01-02"))
	t.Logf("Current Helsinki time: %s (date: %s)", nowHelsinki.Format(time.RFC3339), nowHelsinki.Format("2006-01-02"))

	// Get sunrise/sunset using current functions
	sunrise := getSunriseTimeToday()
	sunset := getSunsetTimeToday()

	t.Logf("Sunrise returned: %s (date: %s)", sunrise.Format(time.RFC3339), sunrise.Format("2006-01-02"))
	t.Logf("Sunset returned: %s (date: %s)", sunset.Format(time.RFC3339), sunset.Format("2006-01-02"))

	// The critical check: if dates differ between UTC and Helsinki,
	// sunrise/sunset MUST match Helsinki date, not UTC date
	if nowUTC.Format("2006-01-02") != nowHelsinki.Format("2006-01-02") {
		t.Logf("Dates differ - UTC: %s, Helsinki: %s",
			nowUTC.Format("2006-01-02"), nowHelsinki.Format("2006-01-02"))

		// Without the fix, sunrise/sunset will match UTC date (WRONG)
		// With the fix, they should match Helsinki date (CORRECT)

		if sunrise.Format("2006-01-02") == nowUTC.Format("2006-01-02") {
			t.Errorf("BUG DETECTED: Sunrise date %s matches UTC date instead of Helsinki date %s",
				sunrise.Format("2006-01-02"), nowHelsinki.Format("2006-01-02"))
		}

		if sunset.Format("2006-01-02") == nowUTC.Format("2006-01-02") {
			t.Errorf("BUG DETECTED: Sunset date %s matches UTC date instead of Helsinki date %s",
				sunset.Format("2006-01-02"), nowHelsinki.Format("2006-01-02"))
		}

		// Verify they match Helsinki date
		if sunrise.Format("2006-01-02") != nowHelsinki.Format("2006-01-02") {
			t.Errorf("Sunrise date %s does not match Helsinki date %s",
				sunrise.Format("2006-01-02"), nowHelsinki.Format("2006-01-02"))
		}

		if sunset.Format("2006-01-02") != nowHelsinki.Format("2006-01-02") {
			t.Errorf("Sunset date %s does not match Helsinki date %s",
				sunset.Format("2006-01-02"), nowHelsinki.Format("2006-01-02"))
		}
	} else {
		t.Logf("Dates are same in UTC and Helsinki (%s) - test would not catch the bug at this time",
			nowUTC.Format("2006-01-02"))
		t.Log("Run this test between 22:00-23:59 UTC to catch the date boundary bug")
	}
}
