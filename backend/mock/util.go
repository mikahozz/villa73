package mock

import (
	"math"
	"time"
)

func GenerateFutureDates(addInterval time.Duration, amount int, roundToNextEven10min bool, roundToNextEvenHour bool) []string {
	var times []string
	t := time.Now()
	minute := 0
	if roundToNextEven10min {
		minute = int(math.Ceil(float64(t.Minute()))/10) * 10
	}
	hour := t.Hour()
	if roundToNextEvenHour {
		hour = t.Hour() + 1
	}

	date := time.Date(t.Year(), t.Month(), t.Day(), hour, minute, 0, 0, t.Location())
	for i := 0; i < amount; i++ {
		times = append(times, date.UTC().Format(time.RFC3339Nano))
		date = date.Add(addInterval)
	}
	return times
}

func ConvertStrArrayToInterface(arr []string) []interface{} {
	interfaces := make([]interface{}, len(arr))
	for i, v := range arr {
		interfaces[i] = v
	}
	return interfaces
}
