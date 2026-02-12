package spot

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

func parseISO8601Duration(duration string) (time.Duration, error) {
	// Remove the "PT" prefix
	duration = strings.TrimPrefix(duration, "PT")

	// Check if the duration ends with "M" for minutes
	if strings.HasSuffix(duration, "M") {
		minutes, err := strconv.Atoi(strings.TrimSuffix(duration, "M"))
		if err != nil {
			return 0, fmt.Errorf("invalid duration format: %s", duration)
		}
		return time.Duration(minutes) * time.Minute, nil
	}

	// Check if the duration ends with "H" for hours
	if strings.HasSuffix(duration, "H") {
		hours, err := strconv.Atoi(strings.TrimSuffix(duration, "H"))
		if err != nil {
			return 0, fmt.Errorf("invalid duration format: %s", duration)
		}
		return time.Duration(hours) * time.Hour, nil
	}

	return 0, fmt.Errorf("unsupported duration format: %s", duration)
}

func ConvertToSpotPriceList(doc *PublicationMarketDocument, periodStart, periodEnd time.Time, location *time.Location) (*SpotPriceList, error) {
	var spotPrices []SpotPrice

	for _, ts := range doc.TimeSeries {
		start, err := time.Parse("2006-01-02T15:04Z", ts.Period.TimeInterval.Start)
		if err != nil {
			return nil, fmt.Errorf("error parsing start time: %w", err)
		}

		resolution, err := parseISO8601Duration(ts.Period.Resolution)
		if err != nil {
			return nil, fmt.Errorf("error parsing resolution: %w", err)
		}

		for _, point := range ts.Period.Points {
			dateTime := start.Add(time.Duration(point.Position-1) * resolution)
			localDateTime := dateTime.In(location)

			// Include times that are >= start and <= end
			if localDateTime.Before(periodStart) || localDateTime.After(periodEnd) {
				continue
			}

			priceWithVat := point.Price * vatMultiplier(point.Price, localDateTime)
			price := math.Round(priceWithVat*100) / 1000
			spotPrices = append(spotPrices, SpotPrice{
				DateTime:  localDateTime,
				PriceCkwh: price,
			})
		}
	}

	sort.Slice(spotPrices, func(i, j int) bool {
		return spotPrices[i].DateTime.Before(spotPrices[j].DateTime)
	})

	return &SpotPriceList{Prices: spotPrices}, nil
}

type VATRate struct {
	EffectiveFrom string
	Multiplier    float64
}

var vatRates = []VATRate{
	{EffectiveFrom: "2024-09-01", Multiplier: 1.255},
	{EffectiveFrom: "2013-01-01", Multiplier: 1.24},
	{EffectiveFrom: "2010-07-01", Multiplier: 1.23},
	{EffectiveFrom: "1994-06-01", Multiplier: 1.22},
}

func vatMultiplier(price float64, dt time.Time) float64 {
	if price < 0 {
		return 1
	}
	priceDate := dt.Format("2006-01-02")
	for _, rate := range vatRates {
		if priceDate >= rate.EffectiveFrom {
			return rate.Multiplier
		}
	}
	panic(fmt.Sprintf("No VAT multiplier found for price date: %s", priceDate))
}
