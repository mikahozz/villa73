package cal

import (
	"testing"
	"time"
)

func TestGetFamilyCalendarEventsIntegration(t *testing.T) {
	// from := DateOffset{Months: -6}
	// to := DateOffset{Months: 6}
	from := DateOffset{Days: 0}
	to := DateOffset{Days: 7}
	events, err := GetFamilyCalendarEvents(from, to)
	if err != nil {
		t.Fatalf("GetFamilyCalendarEvents failed with error: %v", err)
	}

	if len(events) == 0 {
		t.Error("Expected to get at least one event, got zero")
	}

	for _, event := range events {
		if event.Uid == "" {
			t.Error("Expected event UID to be non-empty, got empty string")
		}
		if event.Start.Equal(time.Time{}) {
			t.Error("Expected event start time to be non-zero, got zero")
		}
		if event.End.Equal(time.Time{}) {
			t.Error("Expected event end time to be non-zero, got zero")
		}
		if event.Summary == "" {
			t.Error("Expected event summary to be non-empty, got empty string")
		}
	}
}
