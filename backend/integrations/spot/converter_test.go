package spot

import (
	"encoding/xml"
	"os"
	"testing"
	"time"
)

func TestConvertToSpotPriceList(t *testing.T) {
	// Read the test XML file
	xmlData, err := os.ReadFile("mock/oneDay.xml")
	if err != nil {
		t.Fatalf("Failed to read test XML file: %v", err)
	}

	// Unmarshal the XML data
	var doc PublicationMarketDocument
	err = xml.Unmarshal(xmlData, &doc)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML data: %v", err)
	}

	t.Run("UTC timezone", func(t *testing.T) {
		// Set the period start and end times in UTC
		periodStart, _ := time.Parse(time.RFC3339, "2024-10-22T21:00:00Z")
		periodEnd, _ := time.Parse(time.RFC3339, "2024-10-23T21:00:00Z")

		// Convert to SpotPriceList
		spotPriceList, err := ConvertToSpotPriceList(&doc, periodStart, periodEnd, time.UTC)
		if err != nil {
			t.Fatalf("Failed to convert to SpotPriceList: %v", err)
		}

		// Check the number of prices
		expectedCount := 24 // 24 hours
		if len(spotPriceList.Prices) != expectedCount {
			t.Errorf("Expected %d prices, but got %d", expectedCount, len(spotPriceList.Prices))
		}

		// Check the first price
		expectedFirstPrice := SpotPrice{
			DateTime:  periodStart,
			PriceCkwh: -0.08,
		}
		if spotPriceList.Prices[0] != expectedFirstPrice {
			t.Errorf("Expected first price %+v, but got %+v", expectedFirstPrice, spotPriceList.Prices[0])
		}

		// Check the last price
		expectedLastPrice := SpotPrice{
			DateTime:  periodEnd,
			PriceCkwh: -0.081,
		}
		if spotPriceList.Prices[len(spotPriceList.Prices)-1] != expectedLastPrice {
			t.Errorf("Expected last price %+v, but got %+v", expectedLastPrice, spotPriceList.Prices[len(spotPriceList.Prices)-1])
		}
	})

	t.Run("EEST timezone", func(t *testing.T) {
		// Load EEST timezone
		eest, err := time.LoadLocation("Europe/Helsinki")
		if err != nil {
			t.Fatalf("Failed to load EEST timezone: %v", err)
		}

		// Set the same times but in EEST
		periodStart := time.Date(2024, 10, 23, 0, 0, 0, 0, eest) // 21:00 UTC previous day
		periodEnd := time.Date(2024, 10, 24, 0, 0, 0, 0, eest)   // 21:00 UTC same day

		// Convert to SpotPriceList
		spotPriceList, err := ConvertToSpotPriceList(&doc, periodStart, periodEnd, eest)
		if err != nil {
			t.Fatalf("Failed to convert to SpotPriceList: %v", err)
		}

		// Check the number of prices
		expectedCount := 24 // 24 hours
		if len(spotPriceList.Prices) != expectedCount {
			t.Errorf("Expected %d prices, but got %d", expectedCount, len(spotPriceList.Prices))
		}

		// Check the first price
		expectedFirstPrice := SpotPrice{
			DateTime:  periodStart,
			PriceCkwh: -0.08,
		}
		if spotPriceList.Prices[0] != expectedFirstPrice {
			t.Errorf("Expected first price %+v, but got %+v", expectedFirstPrice, spotPriceList.Prices[0])
		}

		// Verify timezone of all prices
		for i, price := range spotPriceList.Prices {
			if price.DateTime.Location() != eest {
				t.Errorf("Price[%d] has wrong timezone: got %v, want %v",
					i, price.DateTime.Location(), eest)
			}
		}

		// Check the last price
		expectedLastPrice := SpotPrice{
			DateTime:  periodEnd,
			PriceCkwh: -0.081,
		}
		if spotPriceList.Prices[len(spotPriceList.Prices)-1] != expectedLastPrice {
			t.Errorf("Expected last price %+v, but got %+v", expectedLastPrice, spotPriceList.Prices[len(spotPriceList.Prices)-1])
		}
	})
}
