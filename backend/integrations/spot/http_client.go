package spot

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type HTTPClient interface {
	Get(endpoint string, periodStart, periodEnd time.Time) ([]byte, error)
}

type DefaultHTTPClient struct {
	apiKey string
}

func NewDefaultHTTPClient(apiKey string) *DefaultHTTPClient {
	return &DefaultHTTPClient{
		apiKey: apiKey,
	}
}

func (c *DefaultHTTPClient) Get(endpoint string, periodStart, periodEnd time.Time) ([]byte, error) {
	apiURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid API endpoint: %w", err)
	}

	params := url.Values{}
	params.Add("securityToken", c.apiKey)
	params.Add("documentType", "A44")
	params.Add("in_Domain", "10YFI-1--------U")
	params.Add("out_Domain", "10YFI-1--------U")
	params.Add("periodStart", periodStart.UTC().Format("200601021504"))
	params.Add("periodEnd", periodEnd.UTC().Format("200601021504"))

	apiURL.RawQuery = params.Encode()
	fmt.Println("Requesting url:", apiURL.String())

	resp, err := http.Get(apiURL.String())
	if err != nil {
		return nil, fmt.Errorf("error making API request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
