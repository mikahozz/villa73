package spot

import (
	"testing"
	"time"

	"github.com/mikahozz/gohome/integrations/spot/mock"
)

func TestGetSpotPrices(t *testing.T) {
	mockClient := mock.NewMockHTTPClient("mock/oneDay.xml")
	spotService := NewSpotService(mockClient, "http://mock.api")

	periodStart, _ := time.Parse(time.RFC3339, "2024-10-22T21:00:00Z")
	periodEnd, _ := time.Parse(time.RFC3339, "2024-10-23T21:00:00Z")

	prices, err := spotService.GetSpotPrices(periodStart, periodEnd)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(prices.Prices) == 0 {
		t.Error("Expected non-empty price list")
	}
}

func TestGetSpotPrices_NoData(t *testing.T) {
	mockClient := mock.NewMockHTTPClient("mock/noData_200.xml")
	spotService := NewSpotService(mockClient, "http://mock.api")

	periodStart := time.Now().AddDate(0, 0, 2)
	periodEnd := periodStart.AddDate(0, 0, 1)

	_, err := spotService.GetSpotPrices(periodStart, periodEnd)

	noDataErr, ok := err.(*NoDataError)
	if !ok {
		t.Fatalf("Expected NoDataError, got %T", err)
	}

	expectedTextPrefix := "No matching data found for Data item ENERGY_PRICES and interval"
	if len(noDataErr.Message) < len(expectedTextPrefix) || noDataErr.Message[:len(expectedTextPrefix)] != expectedTextPrefix {
		t.Errorf("Expected error text to start with '%s', got '%s'", expectedTextPrefix, noDataErr.Message)
	}
}
