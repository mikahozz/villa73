package sun

import (
	"testing"
	"time"
)

// TestNewSunData verifies that sun data loads correctly from JSON file and builds the internal index.
// It checks that all 366 days (including leap day) are loaded and can be looked up.
func TestNewSunData(t *testing.T) {

	sunData, err := NewSunData()
	if err != nil {
		t.Fatalf("NewSunData failed: %v", err)
	}

	// Should have 366 records (365 days + leap day)
	if len(sunData.byMonthDay) != 366 {
		t.Errorf("Expected 366 records (including leap day), got %d", len(sunData.byMonthDay))
	}

	// Verify Jan 1 data exists
	jan1 := sunData.GetSunDataForSingleDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if jan1 == nil || jan1.Date != "2026-01-01" {
		t.Errorf("Jan 1 data incorrect: %+v", jan1)
	}

	// Verify leap day (Feb 29) exists
	leapDay := sunData.GetSunDataForSingleDate(time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC))
	if leapDay == nil || leapDay.Date != "2024-02-29" {
		t.Errorf("Leap day data incorrect: %+v", leapDay)
	}
}

// TestGetSunDataForSingleDate_YearIndependence verifies that sun data lookup works regardless of year.
// The system matches only by month and day, so querying Feb 1 in any year returns the same sun times.
func TestGetSunDataForSingleDate_YearIndependence(t *testing.T) {

	sunData, err := NewSunData()
	if err != nil {
		t.Fatalf("NewSunData failed: %v", err)
	}

	// Query the same month-day (Feb 1) in different years
	data2024 := sunData.GetSunDataForSingleDate(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	data2025 := sunData.GetSunDataForSingleDate(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	data2026 := sunData.GetSunDataForSingleDate(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))

	if data2024 == nil || data2025 == nil || data2026 == nil {
		t.Fatalf("Failed to get data for different years")
	}

	// The sunrise/sunset times should have the same time-of-day, just different dates
	if data2024.Sunrise.Hour() != data2025.Sunrise.Hour() ||
		data2024.Sunrise.Hour() != data2026.Sunrise.Hour() {
		t.Errorf("Sunrise hours differ across years: 2024=%d, 2025=%d, 2026=%d",
			data2024.Sunrise.Hour(), data2025.Sunrise.Hour(), data2026.Sunrise.Hour())
	}

	// But the dates should reflect the queried year
	if data2024.Date != "2024-02-01" {
		t.Errorf("Expected date 2024-02-01, got %s", data2024.Date)
	}
	if data2025.Date != "2025-02-01" {
		t.Errorf("Expected date 2025-02-01, got %s", data2025.Date)
	}
	if data2026.Date != "2026-02-01" {
		t.Errorf("Expected date 2026-02-01, got %s", data2026.Date)
	}
}

// TestGetSunDataForSingleDate_ParsedTimesHaveCorrectYear verifies that returned time.Time values
// contain the queried date's year, not the year from the JSON file (which was 2025).
// This was a bug where times would be in 2025 regardless of query year.
func TestGetSunDataForSingleDate_ParsedTimesHaveCorrectYear(t *testing.T) {

	sunData, err := NewSunData()
	if err != nil {
		t.Fatalf("NewSunData failed: %v", err)
	}

	// Query Feb 1, 2026
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	data := sunData.GetSunDataForSingleDate(now)

	// The returned times should be in 2026, not 2025
	if data.Sunrise.Year() != 2026 {
		t.Errorf("Sunrise year should be 2026, got %d", data.Sunrise.Year())
	}
	if data.Sunset.Year() != 2026 {
		t.Errorf("Sunset year should be 2026, got %d", data.Sunset.Year())
	}

	// Also verify that the times are reasonable (not in the past by more than a day from query date)
	dayBefore := now.Add(-24 * time.Hour)
	if data.Sunrise.Before(dayBefore) {
		t.Errorf("Sunrise is more than 24 hours before query date: %v", data.Sunrise)
	}
	if data.Sunset.Before(dayBefore) {
		t.Errorf("Sunset is more than 24 hours before query date: %v", data.Sunset)
	}
}

// TestGetSunDataForSingleDate_LeapDay verifies that Feb 29 data exists and can be queried.
func TestGetSunDataForSingleDate_LeapDay(t *testing.T) {

	sunData, err := NewSunData()
	if err != nil {
		t.Fatalf("NewSunData failed: %v", err)
	}

	// Query Feb 29 in a leap year (2024)
	leapDay := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	data := sunData.GetSunDataForSingleDate(leapDay)

	if data.Date != "2024-02-29" {
		t.Errorf("Expected date 2024-02-29, got %s", data.Date)
	}

	// Verify it has reasonable data (should be between Feb 28 and Mar 1)
	if data.Sunrise.IsZero() || data.Sunset.IsZero() {
		t.Errorf("Leap day data missing sunrise or sunset")
	}

	// Sunrise should be before sunset
	if !data.Sunset.After(data.Sunrise) {
		t.Errorf("Sunset should be after sunrise for leap day")
	}
}

// TestGetSunDataForDateRange verifies that querying a date range returns correct results.
func TestGetSunDataForDateRange(t *testing.T) {

	sunData, err := NewSunData()
	if err != nil {
		t.Fatalf("NewSunData failed: %v", err)
	}

	// Query a 3-day range
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	results := sunData.GetSunDataForDateRange(start, end)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Verify dates are sequential
	expectedDates := []string{"2026-01-01", "2026-01-02", "2026-01-03"}
	for i, result := range results {
		if result.Date != expectedDates[i] {
			t.Errorf("Expected date %s at position %d, got %s", expectedDates[i], i, result.Date)
		}
	}

	// Query single day by using zero value for end date
	singleResult := sunData.GetSunDataForDateRange(start, time.Time{})
	if len(singleResult) != 1 {
		t.Errorf("Expected 1 result for single day query, got %d", len(singleResult))
	}
	if singleResult[0].Date != "2026-01-01" {
		t.Errorf("Expected date 2026-01-01, got %s", singleResult[0].Date)
	}
}

// TestGetSunDataForSingleDate_TimezoneAndOffset verifies that times are parsed with correct timezone.
func TestGetSunDataForSingleDate_TimezoneAndOffset(t *testing.T) {

	sunData, err := NewSunData()
	if err != nil {
		t.Fatalf("NewSunData failed: %v", err)
	}

	now := time.Now()
	data := sunData.GetSunDataForSingleDate(now)

	// Basic sanity: sunset should be after sunrise
	if !data.Sunset.After(data.Sunrise) {
		t.Errorf("Sunset (%v) should be after sunrise (%v)", data.Sunset, data.Sunrise)
	}

	// Timezone should be Europe/Helsinki
	locRise := data.Sunrise.Location()
	locSet := data.Sunset.Location()
	if locRise.String() != data.Timezone || locSet.String() != data.Timezone {
		t.Errorf("Unexpected location names: sunrise=%s sunset=%s expected=%s",
			locRise.String(), locSet.String(), data.Timezone)
	}

	// Verify UTC offset matches
	_, riseOffset := data.Sunrise.Zone()
	_, setOffset := data.Sunset.Zone()
	expectedSeconds := data.UTCOffset * 60
	if riseOffset != expectedSeconds || setOffset != expectedSeconds {
		t.Errorf("Unexpected offsets: sunrise=%d sunset=%d expected=%d",
			riseOffset, setOffset, expectedSeconds)
	}
}

// TestNewSunData_AllDaysPresent verifies that all 366 days of the year are present in the data.
func TestNewSunData_AllDaysPresent(t *testing.T) {

	sunData, err := NewSunData()
	if err != nil {
		t.Fatalf("NewSunData failed: %v", err)
	}

	// Test every single day of a leap year
	year := 2024 // Leap year
	for month := 1; month <= 12; month++ {
		daysInMonth := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()
		for day := 1; day <= daysInMonth; day++ {
			date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
			data := sunData.GetSunDataForSingleDate(date) // Will panic if missing

			if data == nil {
				t.Errorf("Missing data for %s", date.Format("2006-01-02"))
			}
			if data.Sunrise.IsZero() {
				t.Errorf("Missing sunrise for %s", date.Format("2006-01-02"))
			}
		}
	}
}
