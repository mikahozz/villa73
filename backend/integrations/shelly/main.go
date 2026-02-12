// Package shelly provides a Shelly Outdoor Plug integration for checking the status of the plug and switching it on or off.
// RPC API reference (Shelly Gen2 devices):
//
//	Get status:  /rpc/Switch.GetStatus?id=<switchID>
//	Set state:   /rpc/Switch.Set?id=<switchID>&on={true|false}
//
// A typical status response contains fields like:
//
//	{"id":0, "output":true, ...}
//
// Only the "output" field is used here.
//
// The client exposes GetStatus and Set methods. Set can optionally verify the state
// by polling until the device reports the desired value or a timeout occurs.
//
// NOTE: This implementation assumes Shelly Gen2 RPC endpoints. Adjust endpoints or
// JSON parsing if using a different firmware/device generation.
package shelly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/mikahozz/gohome/config"
	"github.com/rs/zerolog/log"
)

// SwitchStatus represents the minimal status information we care about.
type SwitchStatus struct {
	ID     int  `json:"id"`
	Output bool `json:"output"`
}

// ShellyClient encapsulates interaction with a Shelly outdoor plug / switch.
type ShellyClient struct {
	baseURL string
	http    *http.Client
	pollInt time.Duration
	mu      sync.Mutex // serialize Set operations to avoid overlapping state changes
}

func TurnOff(ctx context.Context) error {
	c := GetClient()
	if c == nil {
		err := errors.New("missing SHELLY_BASE_URL")
		log.Error().Err(err).Str("event", "shelly_client_missing_env").Msg("SHELLY_BASE_URL not set; skipping Shelly actions")
		return err
	}
	_, err := c.Set(ctx, false, true, 10*time.Second)
	if err != nil {
		log.Error().Err(err).Str("event", "shelly_turn_off_error").Msg("failed to turn off Shelly plug")
		return err
	}
	log.Info().Str("event", "shelly_turn_off_success").Msg("Shelly plug turned OFF")
	return nil
}

func TurnOn(ctx context.Context) error {
	c := GetClient()
	if c == nil {
		err := errors.New("missing SHELLY_BASE_URL")
		log.Error().Err(err).Str("event", "shelly_client_missing_env").Msg("SHELLY_BASE_URL not set; skipping Shelly actions")
		return err
	}
	_, err := c.Set(ctx, true, true, 10*time.Second)
	if err != nil {
		log.Error().Err(err).Str("event", "shelly_turn_on_error").Msg("failed to turn on Shelly plug")
		return err
	}
	log.Info().Str("event", "shelly_turn_on_success").Msg("Shelly plug turned ON")
	return nil
}

func GetClient() *ShellyClient {
	config.LoadEnv()
	baseURL := os.Getenv("SHELLY_BASE_URL")
	if baseURL == "" {
		log.Error().Str("event", "shelly_client_missing_env").Msg("SHELLY_BASE_URL not set; skipping Shelly actions")
		return nil
	}
	// In this simplified plug use-case switch id is always 0.
	return NewShellyClient(baseURL, nil)
}

// NewShellyClient constructs a new client. httpClient may be nil (defaults applied).
func NewShellyClient(baseURL string, httpClient *http.Client) *ShellyClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &ShellyClient{baseURL: trimTrailingSlash(baseURL), http: httpClient, pollInt: 200 * time.Millisecond}
}

func trimTrailingSlash(s string) string {
	if len(s) > 0 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}

// GetStatus retrieves the current switch status.
func (c *ShellyClient) GetStatus(ctx context.Context) (SwitchStatus, error) {
	endpoint := fmt.Sprintf("%s/rpc/Switch.GetStatus?id=0", c.baseURL)
	log.Info().Str("event", "shelly_get_status").Msg("Sending query to Shelly: " + endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return SwitchStatus{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return SwitchStatus{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return SwitchStatus{}, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	var status SwitchStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return SwitchStatus{}, err
	}
	return status, nil
}

// Set changes the switch output state. If verify is true, it polls until the
// desired state is reported or timeout expires. Returns final observed status.
func (c *ShellyClient) Set(ctx context.Context, on bool, verify bool, timeout time.Duration) (SwitchStatus, error) {
	// Ensure only one Set (including verification polling) runs at a time to avoid races
	c.mu.Lock()
	defer c.mu.Unlock()
	endpoint := fmt.Sprintf("%s/rpc/Switch.Set?id=0", c.baseURL)
	u, err := url.Parse(endpoint)
	if err != nil {
		return SwitchStatus{}, err
	}
	q := u.Query()
	if on {
		q.Set("on", "true")
	} else {
		q.Set("on", "false")
	}
	u.RawQuery = q.Encode()
	log.Info().Str("event", "shelly_set").Msg("Sending query to Shelly: " + u.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return SwitchStatus{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return SwitchStatus{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return SwitchStatus{}, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	// We ignore the response body of Set since status will be fetched below.
	if !verify {
		return c.GetStatus(ctx)
	}
	deadline := time.Now().Add(timeout)
	for {
		st, err := c.GetStatus(ctx)
		if err == nil && st.Output == on {
			return st, nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return SwitchStatus{}, err
			}
			return st, errors.New("verification timeout waiting for desired state")
		}
		select {
		case <-ctx.Done():
			return SwitchStatus{}, ctx.Err()
		case <-time.After(c.pollInt):
		}
	}
}
