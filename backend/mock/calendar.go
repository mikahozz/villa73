package mock

import (
	"encoding/json"
	"time"
)

type Event struct {
	Title     string    `json:"title"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

func Events() (string, error) {
	now := time.Now()
	events := []Event{
		{
			Title:     "Mock Event 1",
			StartTime: now.Add(2 * time.Hour),
			EndTime:   now.Add(3 * time.Hour),
		},
		{
			Title:     "Mock Event 2",
			StartTime: now.Add(24 * time.Hour),
			EndTime:   now.Add(25 * time.Hour),
		},
	}

	data, err := json.Marshal(events)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
