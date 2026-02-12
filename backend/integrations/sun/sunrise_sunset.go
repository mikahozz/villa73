package sun

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type monthDay struct {
	Month int
	Day   int
}

// SunData represents the overall structure of the sun data JSON
type SunData struct {
	byMonthDay map[monthDay]*dailySunDataRaw
}

// dailySunDataRaw represents raw sun data from JSON (internal use only)
type dailySunDataRaw struct {
	Month      int    `json:"month"`
	Day        int    `json:"day"`
	Sunrise    string `json:"sunrise"`
	Sunset     string `json:"sunset"`
	Dawn       string `json:"dawn"`
	Dusk       string `json:"dusk"`
	SolarNoon  string `json:"solar_noon"`
	GoldenHour string `json:"golden_hour"`
	DayLength  string `json:"day_length"`
	Timezone   string `json:"timezone"`
	UTCOffset  int    `json:"utc_offset"`
}

// DailySunData represents the sun data for a single day
type DailySunData struct {
	Date       string    `json:"date"`
	Sunrise    time.Time `json:"sunrise"`
	Sunset     time.Time `json:"sunset"`
	Dawn       time.Time `json:"dawn"`
	Dusk       time.Time `json:"dusk"`
	SolarNoon  time.Time `json:"solar_noon"`
	GoldenHour time.Time `json:"golden_hour"`
	DayLength  string    `json:"day_length"` // Keep as string, it's a duration like "10:23:53"
	Timezone   string    `json:"timezone"`
	UTCOffset  int       `json:"utc_offset"`
}

//go:embed sun_helsinki.json
var sunDataJSON []byte

// LoadSunData loads sun data from a JSON file
func NewSunData() (*SunData, error) {
	// Temporary struct for unmarshaling
	var temp struct {
		Results []dailySunDataRaw `json:"results"`
	}
	if err := json.Unmarshal(sunDataJSON, &temp); err != nil {
		return nil, err
	}

	// Build the map
	sunData := &SunData{
		byMonthDay: make(map[monthDay]*dailySunDataRaw, len(temp.Results)),
	}
	for i := range temp.Results {
		key := monthDay{Month: temp.Results[i].Month, Day: temp.Results[i].Day}
		sunData.byMonthDay[key] = &temp.Results[i]
	}

	// Validate we have all 366 days (including leap day)
	if len(sunData.byMonthDay) != 366 {
		return nil, fmt.Errorf("sun data incomplete: expected 366 days, got %d", len(sunData.byMonthDay))
	}

	return sunData, nil
}

// Helper to parse time string for a specific date
func parseTimeForDate(date time.Time, timeStr string, timezone string) time.Time {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc, _ = time.LoadLocation("Europe/Helsinki")
	}

	dateStr := date.Format("2006-01-02")
	tm, err := time.ParseInLocation("2006-01-02 3:04:05 PM", fmt.Sprintf("%s %s", dateStr, timeStr), loc)
	if err != nil {
		log.Error().Err(err).Str("timeStr", timeStr).Msg("Error parsing sun time")
		return time.Time{}
	}
	return tm
}

// GetSunDataForDateRange returns sun data for a date range in converted format.
// If endDate is zero value or equal to startDate, returns only startDate.
func (s *SunData) GetSunDataForDateRange(startDate time.Time, endDate time.Time) []DailySunData {
	// If no end date or same as start, only return single day
	if endDate.IsZero() || endDate.Equal(startDate) {
		endDate = startDate
	}

	var results []DailySunData

	// Iterate through each day in the range
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		// Look up sun data for this month-day
		key := monthDay{Month: int(d.Month()), Day: d.Day()}
		dailyData := s.byMonthDay[key]

		if dailyData == nil {
			// Skip dates not in our data
			continue
		}

		// Convert to the format with actual date
		converted := DailySunData{
			Date:       d.Format("2006-01-02"),
			Sunrise:    parseTimeForDate(d, dailyData.Sunrise, dailyData.Timezone),
			Sunset:     parseTimeForDate(d, dailyData.Sunset, dailyData.Timezone),
			Dawn:       parseTimeForDate(d, dailyData.Dawn, dailyData.Timezone),
			Dusk:       parseTimeForDate(d, dailyData.Dusk, dailyData.Timezone),
			SolarNoon:  parseTimeForDate(d, dailyData.SolarNoon, dailyData.Timezone),
			GoldenHour: parseTimeForDate(d, dailyData.GoldenHour, dailyData.Timezone),
			DayLength:  dailyData.DayLength,
			Timezone:   dailyData.Timezone,
			UTCOffset:  dailyData.UTCOffset,
		}

		results = append(results, converted)
	}

	return results
}

func (s *SunData) GetSunDataForSingleDate(date time.Time) *DailySunData {
	sunData := s.GetSunDataForDateRange(date, time.Time{})
	return &sunData[0]
}
