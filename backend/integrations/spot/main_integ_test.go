package spot

import (
	"testing"
	"time"
)

func TestGetPrices_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name       string
		timeFormat string
		location   *time.Location
	}{
		{
			name:       "UTC timezone",
			timeFormat: "utc",
			location:   time.UTC,
		},
		{
			name:       "Helsinki timezone",
			timeFormat: "Europe/Helsinki",
			location:   mustLoadLocation("Europe/Helsinki"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var start, end time.Time
			if tt.location == time.UTC {
				start, _ = time.Parse(time.RFC3339, "2024-10-22T21:00:00Z")
				end, _ = time.Parse(time.RFC3339, "2024-10-23T21:00:00Z")
			} else {
				// For Helsinki timezone, use local midnight-to-midnight
				start = time.Date(2024, 10, 23, 0, 0, 0, 0, tt.location)
				end = time.Date(2024, 10, 24, 0, 0, 0, 0, tt.location)
			}

			prices, err := GetPrices(start, end, tt.location)
			if err != nil {
				t.Fatalf("GetPrices failed: %v", err)
			}

			if len(prices.Prices) == 0 {
				t.Fatal("Expected prices to be returned, got empty list")
			}

			// Check that we have prices for each hour
			expectedHours := int(end.Sub(start).Hours())
			if len(prices.Prices) != expectedHours {
				t.Errorf("Expected %d hourly prices, got %d", expectedHours, len(prices.Prices))
			}

			// Verify that all prices are in the correct timezone
			for i, price := range prices.Prices {
				if price.DateTime.Location() != tt.location {
					t.Errorf("Price[%d] has wrong timezone: got %v, want %v",
						i, price.DateTime.Location(), tt.location)
				}

				// Verify price is reasonable (between -1000 and 1000 cents/kWh)
				if price.PriceCkwh < -1000 || price.PriceCkwh > 1000 {
					t.Errorf("Price[%d] seems unreasonable: %.2f cents/kWh",
						i, price.PriceCkwh)
				}
			}

			// Verify prices are in chronological order
			for i := 1; i < len(prices.Prices); i++ {
				if !prices.Prices[i].DateTime.After(prices.Prices[i-1].DateTime) {
					t.Errorf("Prices not in chronological order at index %d: "+
						"%v not after %v", i,
						prices.Prices[i].DateTime,
						prices.Prices[i-1].DateTime)
				}
			}

			// Print first and last price for manual verification
			t.Logf("First price: %v %.3f cents/kWh",
				prices.Prices[0].DateTime,
				prices.Prices[0].PriceCkwh)
			t.Logf("Last price: %v %.3f cents/kWh",
				prices.Prices[len(prices.Prices)-1].DateTime,
				prices.Prices[len(prices.Prices)-1].PriceCkwh)
		})
	}
}

// Helper function to load timezone location
func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}
