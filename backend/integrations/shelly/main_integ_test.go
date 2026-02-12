package shelly

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShellySwitchIntegration(t *testing.T) {
	c := GetClient()
	if c == nil {
		t.Skip("client not configured")
	}
	t.Run("Status when OFF", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		st, err := c.GetStatus(ctx)
		if err != nil { // Device offline; skip functional mutation tests
			t.Skipf("device unreachable, skipping status mutation tests: %v", err)
			return
		}
		first, err := c.Set(ctx, !st.Output, true, 8*time.Second)
		require.NoError(t, err)
		require.True(t, first.Output == !st.Output, "First failed")
		time.Sleep(2 * time.Second)
		second, err := c.Set(ctx, st.Output, true, 8*time.Second)
		require.NoError(t, err)
		require.True(t, second.Output == st.Output, "Second failed")
	})

	t.Run("Offline host returns error", func(t *testing.T) {
		// Use an unroutable address for quick failure
		offClient := NewShellyClient("http://127.0.0.1:59999", &http.Client{Timeout: 1 * time.Second})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := offClient.GetStatus(ctx)
		require.Error(t, err, "expected error for offline host")
		_, err = offClient.Set(ctx, true, false, 2*time.Second)
		require.Error(t, err, "expected error for offline host set")
	})
}
