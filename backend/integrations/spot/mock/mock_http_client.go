package mock

import (
	"os"
	"time"
)

type MockHTTPClient struct {
	GetFunc func(endpoint string, periodStart, periodEnd time.Time) ([]byte, error)
}

func (m *MockHTTPClient) Get(endpoint string, periodStart, periodEnd time.Time) ([]byte, error) {
	return m.GetFunc(endpoint, periodStart, periodEnd)
}

func NewMockHTTPClient(filename string) *MockHTTPClient {
	return &MockHTTPClient{
		GetFunc: func(endpoint string, periodStart, periodEnd time.Time) ([]byte, error) {
			content, err := os.ReadFile(filename)
			if err != nil {
				return nil, err
			}
			return content, nil
		},
	}
}
